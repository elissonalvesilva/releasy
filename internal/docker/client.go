package docker

import (
	"context"
	"fmt"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	image2 "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"io"
	"regexp"
	"strings"
)

type (
	dockerClient struct {
		cli         *client.Client
		networkName string
	}

	DockerClient interface {
		CreateService(serviceName, slot, image string, replicas uint64, envs []string, port int) error
		ListServices() ([]string, error)
		RemoveSlot(serviceName, slot string) error
		Close() error
		GetReplicas(serviceName, slot string) (uint64, error)
		ListBySlot(serviceName, slot string) ([]string, error)
		GetServiceImage(serviceName, slot string) (string, error)
	}
)

func NewDockerClient(networkName string) (*dockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &dockerClient{cli: cli, networkName: networkName}, nil
}

func (c *dockerClient) Close() error {
	return c.cli.Close()
}

func (c *dockerClient) CreateService(serviceName, slot string, image string, replicas uint64, envs []string, port int) error {
	ctx := context.Background()

	base := strings.ToLower(strings.TrimSpace(serviceName))
	slot = strings.ToLower(strings.TrimSpace(slot))
	targetName := fmt.Sprintf("%s-%s", base, slot)
	safeName := regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(targetName, "-")

	logger.WithFields(map[string]interface{}{
		"service": safeName,
		"image":   image,
		"slot":    slot,
	}).Info("Creating containers")

	_, err := c.cli.ImageInspect(ctx, image)
	if err != nil {
		if errdefs.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Image %s not found locally, pulling...", image))
			rc, pullErr := c.cli.ImagePull(ctx, image, image2.PullOptions{})
			if pullErr != nil {
				logger.WithError(pullErr).Error("Failed to pull image")
				return pullErr
			}
			defer rc.Close()
			io.Copy(io.Discard, rc) // Descomente se quiser garantir que consome tudo
		} else {
			logger.WithError(err).Error("Failed to inspect image")
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Image %s found locally, skip pulling", image))
	}

	for i := 0; i < int(replicas); i++ {
		instanceName := fmt.Sprintf("%s-%d", safeName, i+1)
		exposedPort := nat.Port(fmt.Sprintf("%d/tcp", port))

		labels := map[string]string{
			"traefik.enable": "true",
			fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", safeName): fmt.Sprintf("%d", port),
		}

		resp, err := c.cli.ContainerCreate(ctx,
			&container.Config{
				Image:        image,
				Env:          envs,
				ExposedPorts: nat.PortSet{exposedPort: struct{}{}},
				Labels:       labels,
				//Healthcheck: &container.HealthConfig{
				//	Test:     []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:%d/ping || exit 1", port)},
				//	Interval: 30 * time.Second,
				//	Timeout:  5 * time.Second,
				//	Retries:  10,
				//},
			},
			&container.HostConfig{
				NetworkMode: container.NetworkMode(c.networkName),
			},
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					c.networkName: {
						Aliases: []string{safeName},
					},
				},
			},
			nil,
			instanceName,
		)
		if err != nil {
			logger.WithError(err).Error("Failed to create container")
			return err
		}

		if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			logger.WithError(err).Error("Failed to start container")
			return err
		}

		logger.WithFields(map[string]interface{}{
			"container": instanceName,
			"slot":      slot,
			"port":      port,
		}).Info("Container created & started")
	}

	return nil
}

func (c *dockerClient) ListServices() ([]string, error) {
	ctx := context.Background()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers")
		return nil, err
	}

	var names []string
	for _, cont := range containers {
		if len(cont.Names) > 0 {
			names = append(names, strings.TrimPrefix(cont.Names[0], "/"))
		}
	}

	return names, nil
}

func (c *dockerClient) RemoveSlot(serviceName, slot string) error {
	ctx := context.Background()
	targetPrefix := fmt.Sprintf("%s-%s", strings.ToLower(serviceName), strings.ToLower(slot))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers for removal")
		return err
	}

	var found bool
	for _, cont := range containers {
		for _, name := range cont.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(cleanName, targetPrefix) {
				logger.WithField("container", cleanName).Info("Removing container")
				if cont.State == "running" {
					_ = c.cli.ContainerStop(ctx, cont.ID, container.StopOptions{})
				}
				err := c.cli.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true})
				if err != nil {
					logger.WithError(err).Errorf("Failed to remove container: %s", cleanName)
				}
				found = true
			}
		}
	}

	if !found {
		logger.WithField("prefix", targetPrefix).Warn("No containers found to remove")
	}

	return nil
}

func (c *dockerClient) GetReplicas(serviceName, slot string) (uint64, error) {
	ctx := context.Background()
	targetPrefix := fmt.Sprintf("%s-%s", strings.ToLower(serviceName), strings.ToLower(slot))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers for replicas")
		return 0, err
	}

	var count uint64
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.HasPrefix(strings.TrimPrefix(name, "/"), targetPrefix) {
				count++
			}
		}
	}

	logger.WithFields(map[string]interface{}{
		"service":  serviceName,
		"slot":     slot,
		"replicas": count,
	}).Info("Replicas counted")

	return count, nil
}

func (c *dockerClient) ListBySlot(serviceName, slot string) ([]string, error) {
	ctx := context.Background()
	targetPrefix := fmt.Sprintf("%s-%s", strings.ToLower(serviceName), strings.ToLower(slot))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers by slot")
		return nil, err
	}

	var names []string
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.HasPrefix(strings.TrimPrefix(name, "/"), targetPrefix) {
				names = append(names, name)
			}
		}
	}

	return names, nil
}

func (c *dockerClient) GetServiceImage(serviceName, slot string) (string, error) {
	ctx := context.Background()
	targetPrefix := fmt.Sprintf("%s-%s", strings.ToLower(serviceName), strings.ToLower(slot))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to inspect containers for image")
		return "", err
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.HasPrefix(strings.TrimPrefix(name, "/"), targetPrefix) {
				return cont.Image, nil
			}
		}
	}

	return "", fmt.Errorf("no container found for %s-%s", serviceName, slot)
}
