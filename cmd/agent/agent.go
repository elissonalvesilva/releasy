package main

import (
	"log"
	"os"

	"github.com/elissonalvesilva/releasy/internal/agent"
	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/internal/traefik"
	"github.com/elissonalvesilva/releasy/pkg/httpclient"
)

func main() {
	redisAddr := os.Getenv("RELEASY_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname: %v", err)
	}

	networkName := getenv("RELEASY_NETWORK", "releasy_network")
	dynamicFilePath := getenv("TRAEFIK_DYNAMIC_FILE", "./dynamic.yml")

	streams := store.NewStreamsStore(redisAddr)
	if err := streams.Ping(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}
	log.Println("Connected to Redis:", redisAddr)

	dockerClient, err := docker.NewDockerClient(networkName)
	if err != nil {
		log.Fatalf("Docker init failed: %v", err)
	}
	defer dockerClient.Close()
	log.Println("Docker client ready on network:", networkName)

	httpClient := httpclient.New()
	healthChecker := healthcheck.NewHTTPHealthChecker(httpClient)

	traefikClient := traefik.NewClient(dynamicFilePath)

	myAgent := agent.NewAgent(
		"agent-"+hostname,
		"releasy_jobs",
		"releasy-group",
		dockerClient,
		healthChecker,
		traefikClient,
		streams,
	)

	log.Println("Agent ready. Starting worker...")
	myAgent.Start()
}

func getenv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
