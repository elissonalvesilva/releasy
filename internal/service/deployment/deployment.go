package deployment

import (
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"github.com/elissonalvesilva/releasy/internal/store"
	"time"
)

type (
	DeploymentCommand struct {
		DeploymentName      string   `json:"deployment_name"`
		ServiceName         string   `json:"service_name"`
		Replicas            int      `json:"replicas"`
		Image               string   `json:"image"`
		SwapInterval        int      `json:"swap_interval,omitempty"`
		HealthCheckInterval int      `json:"health_check_interval,omitempty"`
		Envs                []string `json:"envs,omitempty"`
		MaxWaitTime         int      `json:"max_wait_time,omitempty"`
		Version             string   `json:"version"`
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
	deployment, err := domain.NewDeployment(
		command.DeploymentName,
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

	payload := map[string]interface{}{
		"job_id":                deployment,
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

	if err := d.StreamsStore.PublishJob(command.DeploymentName, payload); err != nil {
		return "", err
	}

	return deployment.JobID, nil
}
