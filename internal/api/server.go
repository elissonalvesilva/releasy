package api

import (
	"github.com/elissonalvesilva/releasy/internal/core/service/deployment"
	"github.com/elissonalvesilva/releasy/internal/store"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"github.com/gin-gonic/gin"
)

type (
	API struct {
		Router            *gin.Engine
		Streams           store.Streams
		DeploymentService *deployment.DeploymentService
	}
)

func NewAPI(streams store.Streams, deploymentService *deployment.DeploymentService) *API {
	r := gin.Default()
	api := &API{
		Router:            r,
		Streams:           streams,
		DeploymentService: deploymentService,
	}
	api.registerRoutes()
	return api
}

func (a *API) Run(addr string) error {
	logger.WithField("addr", addr).Info("Starting releasy control plane API server")
	return a.Router.Run(addr)
}
