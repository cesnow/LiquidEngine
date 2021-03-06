package HealthFoundation

import (
	"github.com/gin-gonic/gin"
)

func Routers(gin *gin.Engine) {
	HealthRouters := gin.Group("/health")
	HealthRouters.Use()
	{
		HealthRouters.GET("/readiness", HealthReadiness)
		HealthRouters.GET("/ping", HealthPing)
	}
}
