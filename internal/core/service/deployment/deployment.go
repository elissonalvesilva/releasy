package deployment

import (
	"encoding/json"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/store"
	"time"
)

type (
	DeploymentCommand struct {
		DeploymentStrategy  string   `json:"strategy"`
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
		Execute(command DeploymentCommand) (string, error)
	}

	DeploymentService struct {
		StreamsStore store.Streams
	}
)

func NewDeploymentService(streams store.Streams) *DeploymentService {
	return &DeploymentService{
		StreamsStore: streams,
	}
}

func (d *DeploymentService) Execute(command DeploymentCommand) (string, error) {
	if command.Action == "" {
		command.Action = domain.ActionDeployCreate
	}

	deployment, err := domain.NewDeployment(
		command.DeploymentStrategy,
		command.Action,
		command.ServiceName,
		command.Image,
		command.Version,
		command.Replicas,
		command.SwapInterval,
		command.HealthCheckInterval,
		command.MaxWaitTime,
		command.Envs,
	)

	if err != nil {
		return "", err
	}

	deploymentValue := map[string]interface{}{
		"job_id":                deployment.JobID,
		"strategy":              deployment.DeploymentStrategy,
		"service_name":          deployment.ServiceName,
		"version":               deployment.Version,
		"image":                 deployment.Image,
		"replicas":              deployment.Replicas,
		"swap_interval":         deployment.SwapInterval,
		"health_check_interval": deployment.HealthCheckInterval,
		"max_wait_time":         deployment.MaxWaitTime,
		"env":                   deployment.Envs,
		"created_at":            time.Now().Format(time.RFC3339),
	}

	deploymentJSON, err := json.Marshal(deploymentValue)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"payload":    string(deploymentJSON),
		"created_at": time.Now().Format(time.RFC3339),
	}

	if err := d.StreamsStore.PublishJob("releasy_jobs", payload); err != nil {
		return "", err
	}

	return deployment.JobID, nil
}
