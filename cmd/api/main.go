package main

import (
	"github.com/elissonalvesilva/releasy/internal/controllers"
	"github.com/elissonalvesilva/releasy/internal/docker"
	"github.com/elissonalvesilva/releasy/internal/healthcheck"
	"github.com/elissonalvesilva/releasy/internal/service/bluegreen"
	"github.com/elissonalvesilva/releasy/pkg/httpclient"
	"github.com/gin-gonic/gin"
)

func main() {
	swarmClient, err := docker.NewDockerClient()
	if err != nil {
		panic(err)
	}
	httpClient := httpclient.New()
	healthChecker := healthcheck.NewHTTPHealthChecker(httpClient)

	blueGreen := bluegreen.NewBlueGreenService(swarmClient, healthChecker)

	controller := controllers.NewDeployController(blueGreen)

	r := gin.Default()
	r.POST("/deploy", controller.Deploy)

	r.Run(":3344")
}
