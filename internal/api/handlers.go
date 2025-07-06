package api

import (
	"github.com/elissonalvesilva/releasy/internal/service/deployment"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (api *API) deploymentHandler(c *gin.Context) {
	var req deployment.DeploymentCommand

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	jobID, err := api.DeploymentService.Execute(req)
	if err != nil {
		logger.WithError(err).Error("Error executing deployment")
		c.JSON(500, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(201, gin.H{
		"status": "deployment created",
		"job_id": jobID,
	})
}
