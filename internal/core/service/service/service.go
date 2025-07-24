package service

import (
	"context"
	"encoding/json"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"github.com/elissonalvesilva/releasy/pkg/utils"
	"github.com/google/uuid"
	"time"
)

type (
	CreateServiceCommand struct {
		Application string   `json:"application"`
		ServiceName string   `json:"service_name"`
		Replicas    int      `json:"replicas"`
		Envs        []string `json:"envs"`
		Image       string   `json:"image"`
		Version     string   `json:"version"`
		Hostname    string   `json:"hostname"`
		MaxWaitTime int      `json:"maxWaitTime"`
	}

	ServiceUsecase interface {
		Create(ctx context.Context, command CreateServiceCommand) error
	}

	ServiceService struct {
		StreamsStore store.Streams
		db           store.DbStore
	}
)

func NewService(streams store.Streams, db store.DbStore) *ServiceService {
	return &ServiceService{
		StreamsStore: streams,
		db:           db,
	}
}

func (s *ServiceService) Create(ctx context.Context, command CreateServiceCommand) error {

	buildedEnvs, err := buildEnvsPayload(command.Envs)
	if err != nil {
		return err
	}

	err = s.db.SaveService(ctx, dto.Service{
		ID:          uuid.NewString(),
		Application: command.Application,
		Name:        command.ServiceName,
		Replicas:    command.Replicas,
		Envs:        buildedEnvs,
		Image:       command.Image,
		Version:     command.Version,
		Weight:      100,
		Hostname:    command.Hostname,
		CreatedAt:   time.Now(),
	})

	if err != nil {
		logger.WithError(err).Info("create service failed")
		return err
	}

	deployment, err := domain.NewDeployment(
		domain.StrategyInitialize,
		domain.ActionDeployCreate,
		command.Application,
		command.ServiceName,
		command.Image,
		command.Version,
		command.Replicas,
		0,
		0,
		command.MaxWaitTime,
		command.Envs,
	)

	if err != nil {
		return err
	}

	dtoDeployment := s.toDTODeployment(*deployment)
	if err = s.db.SaveDeployment(ctx, dtoDeployment); err != nil {
		return err
	}

	payload := s.toCreateServiceStreamData(command)

	err = s.StreamsStore.PublishJob("releasy_jobs", payload)
	if err != nil {
		logger.WithError(err).Info("create service failed")
		return err
	}

	return nil
}

func (s *ServiceService) toCreateServiceStreamData(command CreateServiceCommand) map[string]interface{} {

	deployment, _ := domain.NewDeployment(
		domain.StrategyInitialize,
		domain.ActionDeployCreate,
		command.Application,
		command.ServiceName,
		command.Image,
		command.Version,
		command.Replicas,
		0,
		domain.DefaultHealthCheckIntervalSeconds,
		utils.GetIntOrDefault(command.MaxWaitTime, domain.DefaultMaxWaitTimeSeconds),
		command.Envs,
	)

	deploymentValue := map[string]interface{}{
		"id":                    deployment.ID,
		"application":           deployment.Application,
		"strategy":              deployment.DeploymentStrategy,
		"service_name":          deployment.ServiceName,
		"version":               deployment.Version,
		"image":                 deployment.Image,
		"replicas":              deployment.Replicas,
		"swap_interval":         deployment.SwapInterval,
		"health_check_interval": deployment.HealthCheckInterval,
		"max_wait_time":         deployment.MaxWaitTime,
		"env":                   deployment.Envs,
		"action":                deployment.Action,
		"created_at":            deployment.CreatedAt,
	}

	deploymentJSON, _ := json.Marshal(deploymentValue)

	payload := map[string]interface{}{
		"payload":    string(deploymentJSON),
		"created_at": time.Now().Format(time.RFC3339),
	}

	return payload
}

func (s *ServiceService) toEnvString(envs []string) string {
	var envString string
	for _, env := range envs {
		envString += env + ","
	}
	return envString
}

func (d *ServiceService) toDTODeployment(deployment domain.Deployment) dto.Deployment {
	return dto.Deployment{
		ID:                 deployment.ID,
		Application:        deployment.Application,
		DeploymentStrategy: deployment.DeploymentStrategy,
		ServiceName:        deployment.ServiceName,
		Version:            deployment.Version,
		Image:              deployment.Image,
		Replicas:           deployment.Replicas,
		SwapInterval:       deployment.SwapInterval,
		Envs:               deployment.Envs,
		MaxWaitTime:        deployment.MaxWaitTime,
		Action:             deployment.Action,
		Step:               deployment.Step,
		CreatedAt:          deployment.CreatedAt,
	}
}

func buildEnvsPayload(envs []string) (string, error) {
	envsJSON, err := json.Marshal(envs)

	if err != nil {
		return "", err
	}

	return string(envsJSON), nil
}
