package dto

import "time"

type (
	Service struct {
		ID          string    `json:"id"`
		Application string    `json:"application"`
		Name        string    `json:"name"`
		Version     string    `json:"version"`
		Image       string    `json:"image"`
		Replicas    int       `json:"replicas"`
		Envs        string    `json:"envs"`
		Weight      int       `json:"weight"`
		Hostname    string    `json:"hostname"`
		CreatedAt   time.Time `json:"created_at"`
	}
)
