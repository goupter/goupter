package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/response"
)

func Health(c *gin.Context) {
	response.Success(c, gin.H{
		"service": "api",
		"status":  "ok",
	})
}
