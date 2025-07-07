package traefik

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type (
	Client struct {
		dynamicFilePath string
	}

	Config struct {
		HTTP HTTP `yaml:"http"`
	}

	HTTP struct {
		Routers  map[string]Router       `yaml:"routers"`
		Services map[string]ServiceBlock `yaml:"services"`
	}

	Router struct {
		Rule    string `yaml:"rule"`
		Service string `yaml:"service"`
	}

	ServiceBlock struct {
		LoadBalancer *LoadBalancer `yaml:"loadBalancer,omitempty"`
		Weighted     *Weighted     `yaml:"weighted,omitempty"`
	}

	LoadBalancer struct {
		Servers []Server `yaml:"servers"`
	}

	Server struct {
		URL string `yaml:"url"`
	}

	Weighted struct {
		Services []WeightedService `yaml:"services"`
	}

	WeightedService struct {
		Name   string `yaml:"name"`
		Weight int    `yaml:"weight"`
	}

	WeightedBackend struct {
		Name   string
		Weight int
	}

	TraefikInterface interface {
		EnsureRouter(serviceName, rule string) error
		InsertWeightedService(serviceName string, backends []WeightedBackend) error
		GetCurrentSlot(serviceName string) (string, error)
		PointRouterTo(serviceName, slot string) error
	}
)

func NewClient(dynamicFilePath string) *Client {
	return &Client{dynamicFilePath: dynamicFilePath}
}

func (c *Client) load() (*Config, error) {
	file, err := os.ReadFile(c.dynamicFilePath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if len(file) == 0 {
		cfg.HTTP.Routers = map[string]Router{}
		cfg.HTTP.Services = map[string]ServiceBlock{}
		return cfg, nil
	}

	if err := yaml.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	if cfg.HTTP.Routers == nil {
		cfg.HTTP.Routers = map[string]Router{}
	}
	if cfg.HTTP.Services == nil {
		cfg.HTTP.Services = map[string]ServiceBlock{}
	}

	return cfg, nil
}

func (c *Client) save(cfg *Config) error {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(c.dynamicFilePath, out, 0644)
}

func (c *Client) EnsureRouter(serviceName, rule string) error {
	cfg, err := c.load()
	if err != nil {
		return err
	}

	splitName := fmt.Sprintf("%s-svc", serviceName)

	if _, ok := cfg.HTTP.Routers[serviceName]; !ok {
		cfg.HTTP.Routers[serviceName] = Router{
			Rule:    rule,
			Service: splitName,
		}
	}

	return c.save(cfg)
}

func (c *Client) InsertWeightedService(serviceName string, backends []WeightedBackend) error {
	cfg, err := c.load()
	if err != nil {
		return err
	}

	splitName := fmt.Sprintf("%s-svc", serviceName)

	router := cfg.HTTP.Routers[serviceName]
	router.Service = splitName
	cfg.HTTP.Routers[serviceName] = router

	weighted := &Weighted{
		Services: []WeightedService{},
	}

	for _, b := range backends {
		weighted.Services = append(weighted.Services, WeightedService{
			Name:   fmt.Sprintf("%s@docker", b.Name),
			Weight: b.Weight,
		})
	}

	cfg.HTTP.Services[splitName] = ServiceBlock{
		Weighted: weighted,
	}

	return c.save(cfg)
}

func (c *Client) GetCurrentSlot(serviceName string) (string, error) {
	cfg, err := c.load()
	if err != nil {
		return "", err
	}

	r, ok := cfg.HTTP.Routers[serviceName]
	if !ok {
		return "", fmt.Errorf("router %s not found", serviceName)
	}

	if !strings.HasSuffix(r.Service, "-svc") {
		if strings.HasSuffix(r.Service, "-v1") {
			return "v1", nil
		}
		if strings.HasSuffix(r.Service, "-v2") {
			return "v2", nil
		}
		return "v1", nil
	}

	split, ok := cfg.HTTP.Services[r.Service]
	if !ok || split.Weighted == nil || len(split.Weighted.Services) == 0 {
		return "", fmt.Errorf("no split config for %s", serviceName)
	}

	var max WeightedService
	for _, s := range split.Weighted.Services {
		if s.Weight > max.Weight {
			max = s
		}
	}

	// NEW: remove provider suffix if present
	name := strings.Split(max.Name, "@")[0]

	if strings.HasSuffix(name, "-v1") {
		return "v1", nil
	}
	if strings.HasSuffix(name, "-v2") {
		return "v2", nil
	}

	return "", fmt.Errorf("unknown slot")
}

func (c *Client) PointRouterTo(serviceName, slot string) error {
	cfg, err := c.load()
	if err != nil {
		return err
	}

	r, ok := cfg.HTTP.Routers[serviceName]
	if !ok {
		return fmt.Errorf("router %s not found", serviceName)
	}

	r.Service = fmt.Sprintf("%s-svc", serviceName)
	cfg.HTTP.Routers[serviceName] = r

	return c.save(cfg)
}
