package swarm

import (
	"context"

	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/elissonalvesilva/releasy/pkg/logger"
)

type (
	dockerSwarmClient struct {
		cli *client.Client
	}
)

func NewDockerClient() (*dockerSwarmClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &dockerSwarmClient{cli: cli}, nil
}

func (c *dockerSwarmClient) CreateService(serviceName, image string, replicas uint64, envs []string) error {
	ctx := context.Background()

	spec := dockerswarm.ServiceSpec{
		Annotations: dockerswarm.Annotations{
			Name: serviceName,
		},
		TaskTemplate: dockerswarm.TaskSpec{
			ContainerSpec: &dockerswarm.ContainerSpec{
				Image: image,
				Env:   envs,
			},
		},
		Mode: dockerswarm.ServiceMode{
			Replicated: &dockerswarm.ReplicatedService{
				Replicas: &replicas,
			},
		},
	}

	_, err := c.cli.ServiceCreate(ctx, spec, dockerswarm.ServiceCreateOptions{})
	if err != nil {
		logger.Error("error to create service: ", err)
		return err
	}

	logger.WithFields(map[string]interface{}{
		"service":  serviceName,
		"image":    image,
		"envs":     envs,
		"replicas": replicas,
	}).Info("service created in Swarm")

	return nil
}

func (c *dockerSwarmClient) ListServices() ([]string, error) {
	ctx := context.Background()
	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
	if err != nil {
		logger.Error("error to list services: ", err)
		return nil, err
	}

	var names []string
	for _, s := range services {
		names = append(names, s.Spec.Name)
	}

	return names, nil
}

func (c *dockerSwarmClient) RemoveService(serviceName string) error {
	ctx := context.Background()
	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
	if err != nil {
		logger.Error("error to find service for removal: ", err)
		return err
	}

	for _, s := range services {
		if s.Spec.Name == serviceName {
			logger.WithField("service", serviceName).Info("Removing service")
			return c.cli.ServiceRemove(ctx, s.ID)
		}
	}

	logger.WithField("service", serviceName).Warn("Service not found for removal")
	return nil
}

func (c *dockerSwarmClient) Close() error {
	return c.cli.Close()
}

func (c *dockerSwarmClient) GetReplicas(serviceName string) (uint64, error) {
	ctx := context.Background()

	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
	if err != nil {
		logger.Error("error to find service for replica check: ", err)
		return 0, err
	}

	for _, s := range services {
		if s.Spec.Name == serviceName {
			if s.Spec.Mode.Replicated != nil && s.Spec.Mode.Replicated.Replicas != nil {
				replicas := *s.Spec.Mode.Replicated.Replicas
				logger.WithFields(map[string]interface{}{
					"service":  serviceName,
					"replicas": replicas,
				}).Info("Current replicas found")
				return replicas, nil
			}
			break
		}
	}

	logger.WithField("service", serviceName).Warn("Service not found for replica check")
	return 0, nil
}
