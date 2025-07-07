package main

import (
	"context"
	"fmt"
	"github.com/elissonalvesilva/releasy/internal/api"
	"github.com/elissonalvesilva/releasy/internal/core/service/deployment"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"log"
	"os"
)

func main() {
	redisAddr := getenv("RELEASY_REDIS_ADDR", "localhost:6379")
	postgresDSN := getenv("RELEASY_POSTGRES_DSN", "postgres://postgres:postgres@localhost:5454/releasy?sslmode=disable")
	port := getenv("RELEASY_PORT", ":3344")

	pg, err := store.NewPgStore(postgresDSN)
	if err != nil {
		log.Fatalf("Postgres connect failed: %v", err)
	}
	if err := pg.InitSchema(context.Background()); err != nil {
		logger.WithError(err).Fatal("InitSchema failed")
	}
	logger.Info(fmt.Sprintf("Postgres schema ensured: %s", postgresDSN))

	streamsStore := store.NewStreamsStore(redisAddr)

	deploymentService := deployment.NewDeploymentService(streamsStore, pg)
	server := api.NewAPI(streamsStore, deploymentService)

	if err := server.Run(port); err != nil {
		logger.WithError(err).Fatal("API server crashed")
	}
}

func getenv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
