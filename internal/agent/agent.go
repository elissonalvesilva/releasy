package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/elissonalvesilva/releasy/pkg/logger"

	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/internal/traefik"
	"github.com/go-redis/redis/v8"
)

type Agent struct {
	AgentName     string
	StreamName    string
	GroupName     string
	DockerClient  docker.DockerClient
	HealthChecker healthcheck.HealthChecker
	TraefikClient traefik.TraefikInterface
	Stream        store.Streams
}

type Deployment struct {
	JobID               string `json:"job_id"`
	DeploymentStrategy  string `json:"strategy"`
	ServiceName         string `json:"service_name"`
	Version             string `json:"version"`
	Image               string `json:"image"`
	Replicas            int    `json:"replicas"`
	SwapInterval        int    `json:"swap_interval"`
	HealthCheckInterval int    `json:"health_check_interval"`
	MaxWaitTime         int    `json:"max_wait_time"`
	Envs                string `json:"env"`
}

func NewAgent(agentName, streamName, groupName string, dockerClient docker.DockerClient, healthChecker healthcheck.HealthChecker, traefikClient traefik.TraefikInterface, stream store.Streams) *Agent {
	return &Agent{
		AgentName:     agentName,
		StreamName:    streamName,
		GroupName:     groupName,
		DockerClient:  dockerClient,
		HealthChecker: healthChecker,
		TraefikClient: traefikClient,
		Stream:        stream,
	}
}

func (a *Agent) Start() {
	ctx := context.Background()
	logger.Info(fmt.Sprintf("[Agent] %s started - watching stream: %s", a.AgentName, a.StreamName))

	for {
		messages, err := a.Stream.ReadJob(a.StreamName, a.GroupName, a.AgentName, 5*time.Second)
		if err != nil {
			logger.WithError(err).Error("Error reading job")
			time.Sleep(2 * time.Second)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		for _, msg := range messages {
			var deploy Deployment
			if err := parseMessage(msg, &deploy); err != nil {
				logger.Error(fmt.Sprintf("[Agent] Failed to parse job: %v", err))
				continue
			}

			logger.Info(fmt.Sprintf("[Agent] JobID=%s Strategy=%s Service=%s", deploy.JobID, deploy.DeploymentStrategy, deploy.ServiceName))

			var procErr error
			switch deploy.DeploymentStrategy {
			case domain.StrategyBlueGreen:
				procErr = a.blueGreen(ctx, deploy)
			default:
				log.Printf("[Agent] Unknown strategy: %s", deploy.DeploymentStrategy)
			}

			if procErr != nil {
				log.Printf("[Agent] Job %s failed: %v", deploy.JobID, procErr)
				continue
			}

			if err := a.Stream.AckJob(a.StreamName, a.GroupName, msg.ID); err != nil {
				log.Printf("[Agent] Ack failed: %v", err)
			} else {
				log.Printf("[Agent] Job %s ACK done", deploy.JobID)
			}
		}
	}
}

func parseMessage(msg redis.XMessage, deploy *Deployment) error {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return fmt.Errorf("missing payload")
	}
	return json.Unmarshal([]byte(raw), deploy)
}

func (a *Agent) blueGreen(ctx context.Context, deploy Deployment) error {
	const port = 8080

	if err := a.DockerClient.CreateService(
		deploy.ServiceName,
		deploy.Version,
		deploy.Image,
		uint64(deploy.Replicas),
		parseEnvString(deploy.Envs),
		port,
	); err != nil {
		return fmt.Errorf("create slot: %w", err)
	}

	if err := a.TraefikClient.EnsureRouter(
		deploy.ServiceName,
		fmt.Sprintf("Host(`%s.local`)", deploy.ServiceName),
	); err != nil {
		return fmt.Errorf("ensure router: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, time.Duration(deploy.MaxWaitTime)*time.Second)
	defer cancel()

	slotName := fmt.Sprintf("%s-%s", deploy.ServiceName, deploy.Version)
	if err := a.HealthChecker.Ping(ctxPing, slotName, port, deploy.HealthCheckInterval); err != nil {
		logger.Info(fmt.Sprintf("[Agent] Rolling back slot %s", slotName))
		_ = a.DockerClient.RemoveSlot(deploy.ServiceName, deploy.Version)
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	oldSlot, err := a.TraefikClient.GetCurrentSlot(deploy.ServiceName)
	if err != nil {
		return fmt.Errorf("get current slot: %w", err)
	}
	logger.Info(fmt.Sprintf("[Agent] Current slot is %s", oldSlot))

	if err := a.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
		{Name: deploy.ServiceName + "-" + oldSlot, Weight: 80},
		{Name: slotName, Weight: 20},
	}); err != nil {
		return fmt.Errorf("insert weighted: %w", err)
	}

	oldWeight := 80
	newWeight := 20
	logger.Info(fmt.Sprintf("[Agent] Swap step: %d%% old, %d%% new", oldWeight, newWeight))
	for oldWeight > 0 {
		time.Sleep(time.Duration(deploy.SwapInterval) * time.Second)
		oldWeight -= 20
		if oldWeight < 0 {
			oldWeight = 0
		}
		newWeight = 100 - oldWeight

		if err := a.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
			{Name: deploy.ServiceName + "-" + oldSlot, Weight: oldWeight},
			{Name: slotName, Weight: newWeight},
		}); err != nil {
			return fmt.Errorf("swap step failed: %w", err)
		}

		logger.Info(fmt.Sprintf("[Agent] Swap step: %d%% old, %d%% new", oldWeight, newWeight))
	}

	if err := a.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
		{Name: slotName, Weight: 100},
	}); err != nil {
		return fmt.Errorf("cleanup weighted: %w", err)
	}

	if err := a.DockerClient.RemoveSlot(deploy.ServiceName, oldSlot); err != nil {
		logger.Warn(fmt.Sprintf("[Agent] Failed to remove old slot: %v", err))
	}

	if err := a.TraefikClient.PointRouterTo(deploy.ServiceName, deploy.Version); err != nil {
		return fmt.Errorf("point router failed: %w", err)
	}

	logger.Info(fmt.Sprintf("[Agent] BlueGreen rollout finished for %s", deploy.ServiceName))
	return nil
}

func parseEnvString(envs string) []string {
	var out []string
	_ = json.Unmarshal([]byte(envs), &out)
	return out
}
