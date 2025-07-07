package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"time"

	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/jobs/bluegreen"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/internal/traefik"
	"github.com/elissonalvesilva/releasy/pkg/logger"
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
	db            store.DbStore

	blueGreenJob *bluegreen.Handler
}

func NewAgent(
	agentName, streamName, groupName string,
	dockerClient docker.DockerClient,
	healthChecker healthcheck.HealthChecker,
	traefikClient traefik.TraefikInterface,
	stream store.Streams,
	db store.DbStore,
) *Agent {
	return &Agent{
		AgentName:     agentName,
		StreamName:    streamName,
		GroupName:     groupName,
		DockerClient:  dockerClient,
		HealthChecker: healthChecker,
		TraefikClient: traefikClient,
		Stream:        stream,
		db:            db,

		blueGreenJob: bluegreen.New(dockerClient, traefikClient, healthChecker, db),
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
			deploy, err := parseMessage(msg)
			if err != nil {
				logger.Error(fmt.Sprintf("[Agent] Failed to parse job: %v", err))
				continue
			}

			logger.Info(fmt.Sprintf("[Agent] JobID=%s Strategy=%s Service=%s Action=%s", deploy.ID, deploy.DeploymentStrategy, deploy.ServiceName, deploy.Action))

			var procErr error
			switch deploy.DeploymentStrategy {
			case domain.StrategyBlueGreen:
				procErr = a.blueGreenJob.Run(ctx, deploy)
			default:
				logger.Info(fmt.Sprintf("[Agent] Unknown strategy: %s", deploy.DeploymentStrategy))
			}

			if procErr != nil {
				logger.WithError(err).Error(fmt.Sprintf("[Agent] Job %s failed: %v", deploy.ID, deploy.DeploymentStrategy))
				continue
			}

			if err := a.Stream.AckJob(a.StreamName, a.GroupName, msg.ID); err != nil {
				logger.Error(fmt.Sprintf("[Agent] Job %s ACK failed: %v", deploy.ID, err))
			} else {
				logger.Info(fmt.Sprintf("[Agent] Job %s ACK done", deploy.ID))
			}
		}
	}
}

func parseMessage(msg redis.XMessage) (*dto.Deployment, error) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return nil, fmt.Errorf("missing payload")
	}

	var deploy dto.Deployment
	if err := json.Unmarshal([]byte(raw), &deploy); err != nil {
		return nil, err
	}

	return &deploy, nil
}
