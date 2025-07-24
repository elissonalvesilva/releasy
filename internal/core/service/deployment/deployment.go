package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"github.com/elissonalvesilva/releasy/internal/store"
	"time"
)

type (
	DeploymentCommand struct {
		DeploymentStrategy  string   `json:"strategy"`
		Application         string   `json:"application"`
		ServiceName         string   `json:"service_name"`
		Replicas            int      `json:"replicas"`
		Image               string   `json:"image"`
		SwapInterval        int      `json:"swap_interval,omitempty"`
		HealthCheckInterval int      `json:"health_check_interval,omitempty"`
		Envs                []string `json:"envs,omitempty"`
		MaxWaitTime         int      `json:"max_wait_time,omitempty"`
		Version             string   `json:"version"`
		Action              string   `json:"action,omitempty"`
	}

	Deployment interface {
		Execute(ctx context.Context, command DeploymentCommand) (string, error)
		Finish(ctx context.Context, jobId string) error
	}

	DeploymentService struct {
		StreamsStore store.Streams
		db           store.DbStore
	}
)

func NewDeploymentService(streams store.Streams, db store.DbStore) *DeploymentService {
	return &DeploymentService{
		StreamsStore: streams,
		db:           db,
	}
}

func (d *DeploymentService) Execute(ctx context.Context, command DeploymentCommand) (string, error) {
	service, err := d.getService(ctx, command.Application, command.ServiceName)
	if err != nil {
		return "", err
	}

	if command.Action == "" {
		command.Action = domain.ActionDeployCreate
	}

	deployment, err := domain.NewDeployment(
		command.DeploymentStrategy,
		command.Action,
		command.Application,
		command.ServiceName,
		command.Image,
		command.Version,
		service.Replicas,
		command.SwapInterval,
		command.HealthCheckInterval,
		command.MaxWaitTime,
		command.Envs,
	)

	if err != nil {
		return "", err
	}

	deploymentJSON, err := d.toDeploymentStreamData(*deployment)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"payload":    string(deploymentJSON),
		"created_at": time.Now().Format(time.RFC3339),
	}

	dtoDeployment := d.toDTODeployment(*deployment)
	if err := d.db.SaveDeployment(ctx, dtoDeployment); err != nil {
		return "", err
	}

	if err := d.StreamsStore.PublishJob("releasy_jobs", payload); err != nil {
		return "", err
	}

	return deployment.ID, nil
}

func (d *DeploymentService) Finish(ctx context.Context, jobId string) error {
	deployment, err := d.db.GetDeploymentByID(ctx, jobId)
	if err != nil {
		return err
	}

	if deployment.Step != domain.StepEffective {
		return fmt.Errorf("deployment is not effective")
	}

	deployment.Action = domain.ActionDeployFinish

	job := map[string]interface{}{
		"id":           deployment.ID,
		"service_name": deployment.ServiceName,
		"strategy":     deployment.Strategy,
		"image":        deployment.Image,
		"action":       deployment.Action,
		"version":      deployment.Version,
		"created_at":   time.Now().Format(time.RFC3339),
	}

	deploymentJSON, err := json.Marshal(job)

	payload := map[string]interface{}{
		"payload":    string(deploymentJSON),
		"created_at": time.Now().Format(time.RFC3339),
	}

	if err := d.StreamsStore.PublishJob("releasy_jobs", payload); err != nil {
		return err
	}

	return nil
}

func (d *DeploymentService) toDTODeployment(deployment domain.Deployment) dto.Deployment {
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

func (d *DeploymentService) toDeploymentStreamData(deployment domain.Deployment) ([]byte, error) {
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

	deploymentJSON, err := json.Marshal(deploymentValue)
	if err != nil {
		return []byte(""), err
	}

	return deploymentJSON, nil
}

func (d *DeploymentService) getService(ctx context.Context, application, serviceName string) (*dto.Service, error) {
	return d.db.GetService(ctx, application, serviceName)
}
