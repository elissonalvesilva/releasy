package dto

import "time"

type (
	Deployment struct {
		ID                  string    `json:"id"`
		DeploymentStrategy  string    `json:"strategy"`
		ServiceName         string    `json:"service_name"`
		Version             string    `json:"version"`
		Image               string    `json:"image"`
		Replicas            int       `json:"replicas"`
		SwapInterval        int       `json:"swap_interval"`
		HealthCheckInterval int       `json:"health_check_interval"`
		MaxWaitTime         int       `json:"max_wait_time"`
		Envs                string    `json:"env"`
		Action              string    `json:"action"`
		Step                string    `json:"step"`
		CreatedAt           time.Time `json:"created_at"`
		UpdatedAt           time.Time `json:"updated_at"`
	}
)
