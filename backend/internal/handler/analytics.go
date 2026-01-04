package handler

import (
	"log"

	"blog/pkg/tracking"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler 统计分析处理器
type AnalyticsHandler struct {
	analyticsService *tracking.AnalyticsService
}

// NewAnalyticsHandler 创建统计分析处理器
func NewAnalyticsHandler(analyticsService *tracking.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetFullStats 获取完整统计数据
func (h *AnalyticsHandler) GetFullStats(c *gin.Context) {
	stats, err := h.analyticsService.GetFullStats()
	if err != nil {
		log.Printf("获取统计数据失败: %v", err)
		c.JSON(500, gin.H{"error": "获取统计数据失败"})
		return
	}
	c.JSON(200, stats)
}
