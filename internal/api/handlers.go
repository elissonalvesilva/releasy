package api

import (
	"github.com/elissonalvesilva/releasy/internal/core/service/deployment"
	"github.com/elissonalvesilva/releasy/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (api *API) deploymentHandler(c *gin.Context) {
	var req deployment.DeploymentCommand

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	jobID, err := api.DeploymentService.Execute(c, req)
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

func (api *API) finishDeploymentHandler(c *gin.Context) {
	jobID := c.Param("job_id")

	err := api.DeploymentService.Finish(c, jobID)
	if err != nil {
		logger.WithError(err).Error("Error executing deployment")
		c.JSON(500, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(201, gin.H{
		"status": "finishing deployment",
	})
}

// func (api *API) rollbackHandler(c *gin.Context) {
// 	jobID := c.Param("job_id")

// 	err := api.DeploymentService.Rollback(c, jobID)
// 	if err != nil {
// 		logger.WithError(err).Error("Error executing rollback")
// 		c.JSON(500, gin.H{"error": "Failed to create job"})
// 		return
// 	}

// 	c.JSON(201, gin.H{
// 		"status": "rolling back deployment",
// 	})
// }

// func (api *API) getDeploymentHandler(c *gin.Context) {
// 	jobID := c.Param("job_id")

// 	deployment, err := api.DeploymentService.GetDeployment(c, jobID)
// 	if err != nil {
// 		logger.WithError(err).Error("Error fetching deployment")
// 		c.JSON(500, gin.H{"error": "Failed to fetch deployment"})
// 		return
// 	}

// 	c.JSON(200, gin.H{
// 		"deployment": deployment,
// 	})
// }

// func (api *API) getDeploymentsHandler(c *gin.Context) {
// 	deployments, err := api.DeploymentService.GetDeployments(c)
// 	if err != nil {
// 		logger.WithError(err).Error("Error fetching deployments")
// 		c.JSON(500, gin.H{"error": "Failed to fetch deployments"})
// 		return
// 	}

// 	c.JSON(200, gin.H{
// 		"deployments": deployments,
// 	})
// }

// service handlers

// func (api *API) createServiceHandler(c *gin.Context) {
// 	var req deployment.ServiceCommand

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(400, gin.H{"error": "Invalid JSON"})
// 		return
// 	}

// 	err := api.DeploymentService.CreateService(c, req)
// 	if err != nil {
// 		logger.WithError(err).Error("Error creating service")
// 		c.JSON(500, gin.H{"error": "Failed to create service"})
// 		return
// 	}

// 	c.JSON(201, gin.H{
// 		"status": "service created",
// 	})
// }

// func (api *API) updateServiceHandler(c *gin.Context) {
// 	var req deployment.ServiceCommand

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(400, gin.H{"error": "Invalid JSON"})
// 		return
// 	}

// 	err := api.DeploymentService.UpdateService(c, req)
// 	if err != nil {
// 		logger.WithError(err).Error("Error updating service")
// 		c.JSON(500, gin.H{"error": "Failed to update service"})
// 		return
// 	}

// 	c.JSON(201, gin.H{
// 		"status": "service updated",
// 	})
// }

// func (api *API) deleteServiceHandler(c *gin.Context) {
// 	var req deployment.ServiceCommand

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(400, gin.H{"error": "Invalid JSON"})
// 		return
// 	}

// 	err := api.DeploymentService.DeleteService(c, req)
// 	if err != nil {
// 		logger.WithError(err).Error("Error deleting service")
// 		c.JSON(500, gin.H{"error": "Failed to delete service"})
// 		return
// 	}

// 	c.JSON(204, gin.H{})
// }
