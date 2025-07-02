package docker

//
//import (
//	"context"
//	"fmt"
//	"regexp"
//	"strings"
//
//	dockerswarm "github.com/docker/docker/api/types/swarm"
//	"github.com/docker/docker/client"
//	"github.com/elissonalvesilva/releasy/pkg/logger"
//)
//
//type (
//	dockerSwarmClient struct {
//		cli *client.Client
//	}
//
//	DockerClient interface {
//		CreateService(serviceName, version string, image string, replicas uint64, envs []string, port int) error
//		ListServices() ([]string, error)
//		RemoveService(serviceName string, version string) error
//		Close() error
//		GetReplicas(serviceName string) (uint64, error)
//		ListByServiceName(serviceName string) ([]string, error)
//		GetServiceImage(serviceName string) (string, error)
//	}
//)
//
//func NewDockerClient() (*dockerSwarmClient, error) {
//	logger.Info("✅ [Releasy] Docker client created with version negotiation")
//	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
//	if err != nil {
//		return nil, err
//	}
//	return &dockerSwarmClient{cli: cli}, nil
//}
//
//func (c *dockerSwarmClient) CreateService(serviceName string, version string, image string, replicas uint64, envs []string, port int) error {
//	ctx := context.Background()
//
//	safeServiceName := strings.ToLower(strings.TrimSpace(serviceName))
//	safeVersion := strings.ToLower(strings.TrimSpace(version))
//
//	targetName := fmt.Sprintf("%s-%s", safeServiceName, safeVersion)
//	dnsSafe := regexp.MustCompile(`[^a-z0-9-]+`)
//	targetName = dnsSafe.ReplaceAllString(targetName, "-")
//
//	logger.WithField("service", targetName).Info("Creating versioned service")
//
//	spec := dockerswarm.ServiceSpec{
//		Annotations: dockerswarm.Annotations{
//			Name: targetName,
//			Labels: map[string]string{
//				"traefik.enable": "true",
//				fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", targetName): fmt.Sprintf("%d", port),
//				fmt.Sprintf("traefik.http.routers.%s.rule", targetName):                      fmt.Sprintf("Host(`%s.local`)", targetName),
//			},
//		},
//		TaskTemplate: dockerswarm.TaskSpec{
//			ContainerSpec: &dockerswarm.ContainerSpec{
//				Image: image,
//				Env:   envs,
//			},
//		},
//		Mode: dockerswarm.ServiceMode{
//			Replicated: &dockerswarm.ReplicatedService{
//				Replicas: &replicas,
//			},
//		},
//		EndpointSpec: &dockerswarm.EndpointSpec{
//			Ports: []dockerswarm.PortConfig{
//				{
//					TargetPort:    uint32(port),
//					PublishedPort: 0,
//					Protocol:      dockerswarm.PortConfigProtocolTCP,
//				},
//			},
//		},
//		Networks: []dockerswarm.NetworkAttachmentConfig{
//			{
//				Target: "releasy_web",
//			},
//		},
//	}
//
//	_, err := c.cli.ServiceCreate(ctx, spec, dockerswarm.ServiceCreateOptions{})
//	if err != nil {
//		logger.Error("error to create service: ", err)
//		return err
//	}
//
//	logger.WithFields(map[string]interface{}{
//		"service":  serviceName,
//		"image":    image,
//		"envs":     envs,
//		"replicas": replicas,
//	}).Info("service created in Swarm")
//
//	return nil
//}
//
//func (c *dockerSwarmClient) ListServices() ([]string, error) {
//	ctx := context.Background()
//	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
//	if err != nil {
//		logger.Error("error to list services: ", err)
//		return nil, err
//	}
//
//	var names []string
//	for _, s := range services {
//		names = append(names, s.Spec.Name)
//	}
//
//	return names, nil
//}
//
//func (c *dockerSwarmClient) RemoveService(appName string, version string) error {
//	ctx := context.Background()
//	targetName := fmt.Sprintf("%s-%s", appName, version)
//
//	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
//	if err != nil {
//		logger.Error("error to find service for removal: ", err)
//		return err
//	}
//
//	for _, s := range services {
//		if s.Spec.Name == targetName {
//			logger.WithField("service", targetName).Info("Removing versioned service")
//			return c.cli.ServiceRemove(ctx, s.ID)
//		}
//	}
//
//	logger.WithField("service", targetName).Warn("Versioned service not found for removal")
//	return nil
//}
//
//func (c *dockerSwarmClient) Close() error {
//	return c.cli.Close()
//}
//
//func (c *dockerSwarmClient) GetReplicas(serviceName string) (uint64, error) {
//	ctx := context.Background()
//
//	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
//	if err != nil {
//		logger.Error("error to find service for replica check: ", err)
//		return 0, err
//	}
//
//	for _, s := range services {
//		if s.Spec.Name == serviceName {
//			if s.Spec.Mode.Replicated != nil && s.Spec.Mode.Replicated.Replicas != nil {
//				replicas := *s.Spec.Mode.Replicated.Replicas
//				logger.WithFields(map[string]interface{}{
//					"service":  serviceName,
//					"replicas": replicas,
//				}).Info("Current replicas found")
//				return replicas, nil
//			}
//			break
//		}
//	}
//
//	logger.WithField("service", serviceName).Warn("Service not found for replica check")
//	return 0, nil
//}
//
//func (c *dockerSwarmClient) ListByServiceName(serviceName string) ([]string, error) {
//	ctx := context.Background()
//
//	services, err := c.cli.ServiceList(ctx, dockerswarm.ServiceListOptions{})
//	if err != nil {
//		logger.Error("error to find service for replica check: ", err)
//		return nil, err
//	}
//
//	var names []string
//	for _, s := range services {
//		if s.Spec.Name == serviceName {
//			names = append(names, s.Spec.Name)
//		}
//	}
//
//	return names, nil
//}
//
//func (c *dockerSwarmClient) GetServiceImage(serviceName string) (string, error) {
//	ctx := context.Background()
//
//	service, _, err := c.cli.ServiceInspectWithRaw(ctx, serviceName, dockerswarm.ServiceInspectOptions{})
//	if err != nil {
//		return "", err
//	}
//
//	return service.Spec.TaskTemplate.ContainerSpec.Image, nil
//}
