package domain

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
)

type (
	Deployment struct {
		JobID               string
		DeploymentStrategy  string
		ServiceName         string
		Replicas            int
		Image               string
		SwapInterval        int
		HealthCheckInterval int
		Envs                string
		MaxWaitTime         int
		Version             string
	}
)

var (
	ErrDeploymentNameIsInvalid = errors.New("deployment name is invalid")
)

const (
	StrategyBlueGreen     = "blue_green"
	StrategyRollingUpdate = "rolling_update"
	StrategyCanary        = "canary"
	StrategyAllIn         = "all_in"
)

var allowed = map[string]bool{
	StrategyBlueGreen:     true,
	StrategyRollingUpdate: true,
	StrategyCanary:        true,
	StrategyAllIn:         true,
}

func NewDeployment(deploymentStrategy, serviceName, image, version string, replicas, swapInterval, healthCheckInterval, maxWaitTime int, envs []string) (*Deployment, error) {
	if deploymentStrategy == "" || !isValidStrategy(deploymentStrategy) {
		return nil, ErrDeploymentNameIsInvalid
	}

	jobID := uuid.NewString()

	buildedEnvs, err := buildEnvsPayload(envs)
	if err != nil {
		return nil, err
	}

	return &Deployment{
		JobID:               jobID,
		DeploymentStrategy:  deploymentStrategy,
		ServiceName:         serviceName,
		Replicas:            replicas,
		Image:               image,
		SwapInterval:        swapInterval,
		HealthCheckInterval: healthCheckInterval,
		Envs:                buildedEnvs,
		MaxWaitTime:         maxWaitTime,
		Version:             version,
	}, nil
}

func isValidStrategy(strategy string) bool {
	return allowed[strategy]
}

func buildEnvsPayload(envs []string) (string, error) {
	envsJSON, err := json.Marshal(envs)

	if err != nil {
		return "", err
	}

	return string(envsJSON), nil
}
