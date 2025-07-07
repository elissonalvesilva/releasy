package domain

import "time"

type (
	Service struct {
		ID        string
		Name      string
		Version   string
		Image     string
		Replicas  int
		Envs      string
		Weight    int
		Hostname  string
		CreatedAt time.Time
	}
)
