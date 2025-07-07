package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (api *API) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}

func (api *API) registerRoutes() {
	api.Router.GET("/ping", api.healthHandler)
	api.Router.POST("/deployment", api.deploymentHandler)
	api.Router.PUT("/deployment/finish/:job_id", api.finishDeploymentHandler)
}
