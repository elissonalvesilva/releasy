package initial

import (
	"context"
	"fmt"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/internal/traefik"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"github.com/elissonalvesilva/releasy/pkg/utils"
	"time"
)

type Handler struct {
	DockerClient  docker.DockerClient
	TraefikClient traefik.TraefikInterface
	HealthChecker healthcheck.HealthChecker
	db            store.DbStore
}

func NewAgent(
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
	_, err := h.getService(ctx, deploy)
	if err != nil {
		logger.Error(ctx, "Failed to get service", "error", err)
		return err
	}

	port := utils.ExtractPort(utils.ParseEnvString(deploy.Envs))

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepCreatingInfra); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	if err := h.DockerClient.CreateService(
		deploy.ServiceName,
		deploy.Version,
		deploy.Image,
		uint64(deploy.Replicas),
		utils.ParseEnvString(deploy.Envs),
		port,
		true,
	); err != nil {
		logger.WithError(err).Error("create service")
		return fmt.Errorf("create slot: %w", err)
	}

	if err := h.TraefikClient.EnsureRouter(
		deploy.ServiceName,
		fmt.Sprintf("Host(`%s.local`)", deploy.ServiceName),
	); err != nil {
		logger.WithError(err).Error("create traefik router")
		return fmt.Errorf("ensure router: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, time.Duration(deploy.MaxWaitTime)*time.Second)
	defer cancel()

	slotName := fmt.Sprintf("%s-%s", deploy.ServiceName, deploy.Version)
	if err := h.HealthChecker.Ping(ctxPing, slotName, port, deploy.HealthCheckInterval); err != nil {
		_ = h.DockerClient.RemoveSlot(deploy.ServiceName, deploy.Version)
		logger.WithError(err).Error("check health check")
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	if err := h.TraefikClient.InsertWeightedService(deploy.ServiceName, []traefik.WeightedBackend{
		{Name: slotName, Weight: 100},
	}); err != nil {
		logger.WithError(err).Error("insert weighted service")
		return fmt.Errorf("cleanup weighted: %w", err)
	}

	if err := h.TraefikClient.PointRouterTo(deploy.ServiceName, deploy.Version); err != nil {
		logger.Warn(fmt.Sprintf("[Inital] Failed to point router: %v", err))
		return fmt.Errorf("point router failed: %w", err)
	}

	if err := h.updateDeploymentStep(ctx, deploy, domain.StepFinished); err != nil {
		logger.WithError(err).Error("update deployment step")
		return fmt.Errorf("update deployment step: %w", err)
	}

	return nil
}

func (h *Handler) getService(ctx context.Context, deploy *dto.Deployment) (*dto.Service, error) {
	logger.Info(deploy)
	service, err := h.db.GetService(ctx, deploy.Application, deploy.ServiceName)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (h *Handler) updateDeploymentStep(ctx context.Context, deploy *dto.Deployment, step string) error {
	deploy.Step = step
	if err := h.db.UpdateDeploymentStep(ctx, deploy.ID, step); err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}
	return nil
}
