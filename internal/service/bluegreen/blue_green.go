package bluegreen

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/pkg/logger"
)

type (
	BlueGreenCommand struct {
		ServiceName         string   `json:"service_name"`
		Replicas            int      `json:"replicas"`
		Image               string   `json:"image"`
		SwapInterval        int      `json:"swap_interval,omitempty"`
		HealthCheckInterval int      `json:"health_check_interval,omitempty"`
		Env                 []string `json:"env,omitempty"`
		MaxWaitTime         int      `json:"max_wait_time,omitempty"`
		Version             string   `json:"version"`
	}

	service struct {
		ServiceName         string   `json:"service_name"`
		Replicas            uint64   `json:"replicas,omitempty"`
		Image               string   `json:"image"`
		SwapInterval        int      `json:"swap_interval,omitempty"`
		Env                 []string `json:"env,omitempty"`
		HealthCheckInterval int      `json:"health_check_interval,omitempty"`
		AppPort             int      `json:"app_port,omitempty"`
		MaxWaitTime         int      `json:"max_wait_time,omitempty"`
		Version             string   `json:"version"`
	}

	BlueGreen interface {
		Execute(command BlueGreenCommand) error
	}

	blueGreenService struct {
		client            docker.DockerClient
		healthCheckClient healthcheck.HealthChecker
	}
)

const (
	defaultSwapInterval        = 30
	defaultHealthCheckInterval = 5
	defaultPort                = 80
	defaultMaxWaitTime         = 3600 // 1 hour
)

func NewBlueGreenService(client docker.DockerClient, healthCheckClient healthcheck.HealthChecker) BlueGreen {
	return &blueGreenService{
		client:            client,
		healthCheckClient: healthCheckClient,
	}
}

func (b *blueGreenService) service(command BlueGreenCommand) *service {
	var env []string
	var swapInterval int
	var replicas uint64
	var healthCheckInterval int
	var maxWaitTime int

	if command.Env != nil {
		env = command.Env
	}

	if command.SwapInterval != 0 {
		swapInterval = command.SwapInterval
	}

	if command.Replicas != 0 {
		replicas = uint64(command.Replicas)
	}

	if command.HealthCheckInterval != 0 {
		healthCheckInterval = command.HealthCheckInterval
	} else {
		healthCheckInterval = defaultHealthCheckInterval
	}

	if command.MaxWaitTime != 0 {
		maxWaitTime = command.MaxWaitTime
	} else {
		maxWaitTime = defaultMaxWaitTime
	}

	envs, appPort := b.buildEnv(env)

	return &service{
		ServiceName:         command.ServiceName,
		Replicas:            replicas,
		Image:               command.Image,
		SwapInterval:        swapInterval,
		Env:                 envs,
		AppPort:             appPort,
		HealthCheckInterval: healthCheckInterval,
		MaxWaitTime:         maxWaitTime,
		Version:             command.Version,
	}
}

func (b *blueGreenService) buildEnv(customEnv []string) ([]string, int) {
	finalEnv := []string{}
	appPort := defaultPort
	hasPort := false

	for _, envVar := range customEnv {
		finalEnv = append(finalEnv, envVar)

		if strings.HasPrefix(envVar, "APP_PORT=") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				if portVal, err := strconv.Atoi(parts[1]); err == nil {
					appPort = portVal
					hasPort = true
				}
			}
		}
	}

	if !hasPort {
		finalEnv = append(finalEnv, fmt.Sprintf("APP_PORT=%d", defaultPort))
	}

	return finalEnv, appPort
}

// func (b *blueGreenService) Execute(command BlueGreenCommand) error {

// 	service := b.service(command)
// 	err := b.client.CreateService(service.ServiceName, service.Image, service.Replicas, service.Env, service.AppPort)
// 	if err != nil {
// 		return err
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(service.MaxWaitTime)*time.Second)
// 	defer cancel()

// 	err = b.healthCheckClient.Ping(ctx, service.ServiceName, service.AppPort, service.HealthCheckInterval)
// 	if err != nil {
// 		logger.WithField("service", service.ServiceName).WithError(err).Error("Error pinging service")
// 		return err
// 	}

// 	services, err := b.client.ListByServiceName(command.ServiceName)
// 	if err != nil {
// 		return err
// 	}

// 	for _, serviceName := range services {
// 		if serviceName == service.ServiceName {
// 			continue
// 		}

// 		image, err := b.client.GetServiceImage(serviceName)
// 		if err != nil {
// 			logger.WithField("service", serviceName).WithError(err).Warn("Could not get image from old service")
// 			continue
// 		}

// 		logger.WithFields(map[string]interface{}{
// 			"service": serviceName,
// 			"image":   image,
// 		}).Info("Removing old version")

// 		if err := b.client.RemoveService(serviceName, image); err != nil {
// 			logger.WithField("service", serviceName).WithError(err).Warn("Failed to remove old service")
// 		}
// 	}

// 	return nil
// }

func (b *blueGreenService) Execute(command BlueGreenCommand) error {
	logger.WithField("service", command.ServiceName).Info("🚀 Starting Blue/Green deployment")

	service := b.service(command)

	logger.WithFields(map[string]interface{}{
		"service":  service.ServiceName,
		"image":    service.Image,
		"replicas": service.Replicas,
	}).Info("🔨 Creating new service version")

	err := b.client.CreateService(
		service.ServiceName,
		service.Version,
		service.Image,
		service.Replicas,
		service.Env,
		service.AppPort,
	)
	if err != nil {
		logger.WithError(err).Error("❌ Failed to create new service")
		logger.WithError(err).Error("❌ Failed to create new service")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"service":  service.ServiceName,
		"app_port": service.AppPort,
	}).Info("🔍 Running health check")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(service.MaxWaitTime)*time.Second)
	defer cancel()

	targetName := fmt.Sprintf("releasy-%s-%s", strings.ToLower(service.ServiceName), strings.ToLower(service.Version))
	targetName = strings.TrimSpace(targetName)

	err = b.healthCheckClient.Ping(ctx, targetName, service.AppPort, service.HealthCheckInterval)
	if err != nil {
		logger.WithField("service", targetName).
			WithError(err).
			Error("❌ Health check failed")
		return err
	}

	logger.WithField("service", targetName).Info("✅ Health check passed")

	services, err := b.client.ListByServiceName(fmt.Sprintf("releasy-%s", strings.ToLower(service.ServiceName)))
	if err != nil {
		logger.WithError(err).Error("❌ Failed to list services by name")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"new_service": service.ServiceName,
		"found":       services,
	}).Info("🔎 Checking for old versions to remove")

	for _, serviceName := range services {
		if strings.HasPrefix(serviceName, targetName) {
			continue
		}

		image, err := b.client.GetServiceImage(serviceName)
		if err != nil {
			logger.WithField("service", serviceName).WithError(err).
				Warn("⚠️ Could not get image from old service")
			continue
		}

		logger.WithFields(map[string]interface{}{
			"old_service": serviceName,
			"old_image":   image,
		}).Info("🗑️ Removing old version")

		if err := b.client.RemoveService(serviceName); err != nil {
			logger.WithField("service", serviceName).WithError(err).
				Warn("⚠️ Failed to remove old service")
		} else {
			logger.WithField("service", serviceName).Info("✅ Old version removed")
		}
	}

	logger.WithField("service", service.ServiceName).Info("🎉 Blue/Green deployment complete")
	return nil
}
