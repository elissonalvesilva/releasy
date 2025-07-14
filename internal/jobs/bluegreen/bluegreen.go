package bluegreen

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"github.com/elissonalvesilva/releasy/internal/store"
	"strconv"
	"strings"
	"time"

	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/traefik"
	"github.com/elissonalvesilva/releasy/pkg/logger"
)

type Handler struct {
	DockerClient  docker.DockerClient
	TraefikClient traefik.TraefikInterface
	HealthChecker healthcheck.HealthChecker
	db            store.DbStore
}

func New(
	dockerClient docker.DockerClient,
	traefikClient traefik.TraefikInterface,
	healthChecker healthcheck.HealthChecker,
	db store.DbStore,
) *Handler {
	return &Handler{
		DockerClient:  dockerClient,
		TraefikClient: traefikClient,
		HealthChecker: healthChecker,
		db:            db,
	}
}

func (h *Handler) Run(ctx context.Context, deploy *dto.Deployment) error {

	switch deploy.Action {
	case domain.ActionDeployCreate:
		return h.executeCreateBlueGreen(ctx, deploy)
	case domain.ActionDeployRollback:
		return h.executeRollback(ctx, deploy)
	case domain.ActionDeployFinish:
		return h.executeFinishBlueGreen(ctx, deploy)
	default:
		return fmt.Errorf("invalid action: %s", deploy.Action)
	}
}

func (h *Handler) executeCreateBlueGreen(ctx context.Context, deploy *dto.Deployment) error {
	port := extractPort(parseEnvString(deploy.Envs))

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepCreatingInfra); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	if err := h.DockerClient.CreateService(
		deploy.ServiceName,
		deploy.Version,
		deploy.Image,
		uint64(deploy.Replicas),
		parseEnvString(deploy.Envs),
		port,
		false,
	); err != nil {
		return fmt.Errorf("create slot: %w", err)
	}

	if err := h.TraefikClient.EnsureRouter(
		deploy.ServiceName,
		fmt.Sprintf("Host(`%s.local`)", deploy.ServiceName),
	); err != nil {
		return fmt.Errorf("ensure router: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, time.Duration(deploy.MaxWaitTime)*time.Second)
	defer cancel()

	slotName := fmt.Sprintf("%s-%s", deploy.ServiceName, deploy.Version)
	if err := h.HealthChecker.Ping(ctxPing, slotName, port, deploy.HealthCheckInterval); err != nil {
		_ = h.DockerClient.RemoveSlot(deploy.ServiceName, deploy.Version)
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	oldSlot, err := h.TraefikClient.GetCurrentSlot(deploy.ServiceName)
	if err != nil {
		return fmt.Errorf("get current slot: %w", err)
	}
	logger.Info(fmt.Sprintf("[BlueGreen] Current slot is %s", oldSlot))

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepSwapTraffic); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	if err := h.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
		{Name: deploy.ServiceName + "-" + oldSlot, Weight: 80},
		{Name: slotName, Weight: 20},
	}); err != nil {
		return fmt.Errorf("insert weighted: %w", err)
	}

	oldWeight := 80
	newWeight := 20

	logger.Info(fmt.Sprintf("[BlueGreen] Candidate weight is %d and Current is %d", newWeight, oldWeight))
	for oldWeight > 0 {
		time.Sleep(time.Duration(deploy.SwapInterval) * time.Second)
		oldWeight -= 20
		if oldWeight < 0 {
			oldWeight = 0
		}
		newWeight = 100 - oldWeight

		if err := h.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
			{Name: deploy.ServiceName + "-" + oldSlot, Weight: oldWeight},
			{Name: slotName, Weight: newWeight},
		}); err != nil {
			return fmt.Errorf("swap step failed: %w", err)
		}

		logger.Info(fmt.Sprintf("[BlueGreen] Candidate weight is %d and Current is %d", newWeight, oldWeight))
	}

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepEffective); err != nil {
		return fmt.Errorf("update deployment step: %w", err)
	}

	logger.Info(fmt.Sprintf("[BlueGreen] Deployment effective for %s and deployment_id: %s", deploy.ServiceName, deploy.ID))

	return nil
}

func (h *Handler) executeFinishBlueGreen(ctx context.Context, deploy *dto.Deployment) error {
	if err := h.updateDeploymentStep(ctx, deploy, domain.StepFinishing); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	oldSlot, err := h.TraefikClient.GetCurrentSlot(deploy.ServiceName)
	if err != nil {
		logger.WithError(err).Error("get current slot")
		return fmt.Errorf("get current slot: %w", err)
	}
	logger.Info(fmt.Sprintf("[BlueGreen] Current slot is %s", oldSlot))

	slotName := fmt.Sprintf("%s-%s", deploy.ServiceName, deploy.Version)
	if err := h.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
		{Name: slotName, Weight: 100},
	}); err != nil {
		return fmt.Errorf("cleanup weighted: %w", err)
	}

	if err := h.DockerClient.RemoveSlot(deploy.ServiceName, oldSlot); err != nil {
		logger.Warn(fmt.Sprintf("[BlueGreen] Failed to remove old slot: %v", err))
	}

	if err := h.TraefikClient.PointRouterTo(deploy.ServiceName, deploy.Version); err != nil {
		logger.Warn(fmt.Sprintf("[BlueGreen] Failed to point router: %v", err))
		return fmt.Errorf("point router failed: %w", err)
	}

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepFinished); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	logger.Info(fmt.Sprintf("[BlueGreen] Rollout finished for %s", deploy.ServiceName))
	return nil
}

func (h *Handler) executeRollback(ctx context.Context, deploy *dto.Deployment) error {
	return nil
}

func (h *Handler) updateDeploymentStep(ctx context.Context, deploy *dto.Deployment, step string) error {
	deploy.Step = step
	if err := h.db.UpdateDeploymentStep(ctx, deploy.ID, step); err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}
	return nil
}

func parseEnvString(envs string) []string {
	var out []string
	_ = json.Unmarshal([]byte(envs), &out)
	return out
}

func extractPort(envs []string) int {
	for _, env := range envs {
		if strings.HasPrefix(env, "APP_PORT=") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				if portVal, err := strconv.Atoi(parts[1]); err == nil {
					return portVal
				}
			}
		}
	}
	return domain.DefaultServicePort
}
