package domain

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

type (
	Deployment struct {
		ID                  string
		DeploymentStrategy  string
		ServiceName         string
		Replicas            int
		Image               string
		SwapInterval        int
		HealthCheckInterval int
		Envs                string
		MaxWaitTime         int
		Version             string
		Action              string
		Step                string
		CreatedAt           time.Time
	}
)

var (
	ErrDeploymentNameIsInvalid = errors.New("deployment name is invalid")
	ErrActionIsInvalid         = errors.New("action is invalid")
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

const (
	ActionDeployCreate   = "create"
	ActionDeployFinish   = "finish"
	ActionDeployRollback = "rollback"
)

var allowedActions = map[string]bool{
	ActionDeployCreate:   true,
	ActionDeployFinish:   true,
	ActionDeployRollback: true,
}

const (
	StepCreating      = "creating"
	StepCreatingInfra = "creating_infra"
	StepSwapTraffic   = "swap_traffic"
	StepFinishing     = "finishing"
	StepRollback      = "rollback"
	StepRunning       = "running"
	StepEffective     = "effective"
	StepFinished      = "finished"
	StepFailed        = "failed"
	StepRollBacking   = "rollbacking"
)

const (
	DefaultServicePort = 8080
)

func NewDeployment(deploymentStrategy, action, serviceName, image, version string, replicas, swapInterval, healthCheckInterval, maxWaitTime int, envs []string) (*Deployment, error) {
	if deploymentStrategy == "" || !isValidStrategy(deploymentStrategy) {
		return nil, ErrDeploymentNameIsInvalid
	}

	if action == "" || !isValidAction(action) {
		return nil, ErrActionIsInvalid
	}

	jobID := uuid.NewString()

	buildedEnvs, err := buildEnvsPayload(envs)
	if err != nil {
		return nil, err
	}

	return &Deployment{
		ID:                  jobID,
		DeploymentStrategy:  deploymentStrategy,
		ServiceName:         serviceName,
		Replicas:            replicas,
		Image:               image,
		SwapInterval:        swapInterval,
		HealthCheckInterval: healthCheckInterval,
		Envs:                buildedEnvs,
		MaxWaitTime:         maxWaitTime,
		Version:             version,
		Action:              action,
		Step:                StepCreating,
		CreatedAt:           time.Now(),
	}, nil
}

func isValidStrategy(strategy string) bool {
	return allowed[strategy]
}

func isValidAction(action string) bool {
	return allowedActions[action]
}

func buildEnvsPayload(envs []string) (string, error) {
	envsJSON, err := json.Marshal(envs)

	if err != nil {
		return "", err
	}

	return string(envsJSON), nil
}
