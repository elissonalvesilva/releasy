package controllers

import (
	"net/http"

	"github.com/elissonalvesilva/releasy/internal/service/bluegreen"
	"github.com/gin-gonic/gin"
)

type DeployController struct {
	blueGreen bluegreen.BlueGreen
}

func NewDeployController(bg bluegreen.BlueGreen) *DeployController {
	return &DeployController{
		blueGreen: bg,
	}
}

func (c *DeployController) Deploy(ctx *gin.Context) {
	var cmd bluegreen.BlueGreenCommand

	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	if err := c.blueGreen.Execute(cmd); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to execute deploy",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "deploy executed successfully",
	})
}
