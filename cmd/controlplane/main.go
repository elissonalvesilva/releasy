package main

import (
	"github.com/elissonalvesilva/releasy/internal/api"
	"github.com/elissonalvesilva/releasy/internal/service/deployment"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"os"
)

func main() {
	redisAddr := os.Getenv("RELEASY_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	port := os.Getenv("RELEASY_PORT")
	if port == "" {
		port = ":3344"
	}

	streamsStore := store.NewStreamsStore(redisAddr)

	deploymentService := deployment.NewDeploymentService(streamsStore)
	server := api.NewAPI(streamsStore, deploymentService)

	if err := server.Run(port); err != nil {
		logger.WithError(err).Fatal("API server crashed")
	}
}
