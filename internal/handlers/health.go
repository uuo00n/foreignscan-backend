package handlers

import (
	"foreignscan/internal/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary 服务健康检查
// @Description 检查服务是否存活
// @Tags system
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ReadinessCheck godoc
// @Summary 就绪探针
// @Description 检查服务依赖（如数据库）是否就绪
// @Tags system
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /ready [get]
func ReadinessCheck(c *gin.Context) {
	sqlDB, err := database.GetDB().DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "database connection error",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
