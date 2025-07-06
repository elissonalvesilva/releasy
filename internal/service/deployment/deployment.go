package deployment

import (
	"encoding/json"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/google/uuid"
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
	jobID := uuid.NewString()

	envs, err := d.buildEnvsPayload(command.Envs)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"job_id":                jobID,
		"service_name":          command.ServiceName,
		"version":               command.Version,
		"image":                 command.Image,
		"replicas":              command.Replicas,
		"swap_interval":         command.SwapInterval,
		"health_check_interval": command.HealthCheckInterval,
		"max_wait_time":         command.MaxWaitTime,
		"env":                   envs,
		"created_at":            time.Now().Format(time.RFC3339),
	}

	if err := d.StreamsStore.PublishJob(command.DeploymentName, payload); err != nil {
		return "", err
	}

	return jobID, nil
}

func (d *DeploymentService) buildEnvsPayload(envs []string) (string, error) {
	envsJSON, err := json.Marshal(envs)

	if err != nil {
		return "", err
	}

	return string(envsJSON), nil
}
