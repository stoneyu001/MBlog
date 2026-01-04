package handler

import "github.com/gin-gonic/gin"

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Ping 健康检查
func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}
