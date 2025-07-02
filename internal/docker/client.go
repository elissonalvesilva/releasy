package docker

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/elissonalvesilva/releasy/pkg/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type (
	dockerClient struct {
		cli *client.Client
	}

	DockerClient interface {
		CreateService(serviceName, version string, image string, replicas uint64, envs []string, port int) error
		ListServices() ([]string, error)
		RemoveService(serviceName string) error
		Close() error
		GetReplicas(serviceName string) (uint64, error)
		ListByServiceName(serviceName string) ([]string, error)
		GetServiceImage(serviceName string) (string, error)
	}
)

func NewDockerClient() (*dockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &dockerClient{cli: cli}, nil
}

func (c *dockerClient) Close() error {
	return c.cli.Close()
}

func (c *dockerClient) CreateService(serviceName string, version string, image string, replicas uint64, envs []string, port int) error {
	ctx := context.Background()

	safeServiceName := strings.ToLower(strings.TrimSpace(serviceName))
	safeVersion := strings.ToLower(strings.TrimSpace(version))
	targetName := fmt.Sprintf("%s-%s", safeServiceName, safeVersion)
	dnsSafe := regexp.MustCompile(`[^a-z0-9-]+`)
	project := "releasy"
	targetName = dnsSafe.ReplaceAllString(project+"-"+targetName, "-")

	logger.WithField("service", targetName).Info("Creating versioned containers")
	logger.WithField("image", image).Info("Pulling image")

	// _, err := c.cli.ImagePull(ctx, image, image2.PullOptions{})
	// if err != nil {
	// 	logger.WithError(err).Error("Failed to pull image")
	// 	return err
	// }

	for i := 0; i < int(replicas); i++ {
		instanceName := fmt.Sprintf("%s-%d", targetName, i+1)
		exposedPort := nat.Port(fmt.Sprintf("%d/tcp", port))

		resp, err := c.cli.ContainerCreate(ctx,
			&container.Config{
				Image:        image,
				Env:          envs,
				ExposedPorts: nat.PortSet{exposedPort: struct{}{}},
				Labels: map[string]string{
					"traefik.enable": "true",
					fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", targetName): fmt.Sprintf("%d", port),
				},
			},
			&container.HostConfig{
				NetworkMode: "releasy_releasy_network",
			},
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					"releasy_releasy_network": {
						Aliases: []string{targetName},
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
			"image":     image,
			"port":      port + int(i),
		}).Info("Container created and started")
	}

	return nil
}

// ListServices lista todos os containers em execução (nomes)
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

func (c *dockerClient) RemoveService(serviceName string) error {
	ctx := context.Background()

	targetPrefix := serviceName

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

func (c *dockerClient) GetReplicas(serviceName string) (uint64, error) {
	ctx := context.Background()
	targetPrefix := strings.ToLower(serviceName)

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers for replicas")
		return 0, err
	}

	var count uint64
	for _, cont := range containers {
		for _, name := range cont.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(cleanName, targetPrefix) {
				count++
			}
		}
	}

	logger.WithFields(map[string]interface{}{
		"service":  serviceName,
		"replicas": count,
	}).Info("Replicas counted")

	return count, nil
}

func (c *dockerClient) ListByServiceName(serviceName string) ([]string, error) {
	ctx := context.Background()
	targetPrefix := strings.ToLower(serviceName)

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to list containers by service name")
		return nil, err
	}

	var names []string
	for _, cont := range containers {
		for _, name := range cont.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(cleanName, targetPrefix) {
				names = append(names, cleanName)
			}
		}
	}

	return names, nil
}

func (c *dockerClient) GetServiceImage(serviceName string) (string, error) {
	ctx := context.Background()
	targetPrefix := strings.ToLower(serviceName)

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logger.WithError(err).Error("Failed to inspect containers for image")
		return "", err
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(cleanName, targetPrefix) {
				return cont.Image, nil
			}
		}
	}

	return "", fmt.Errorf("no container found for %s", serviceName)
}
