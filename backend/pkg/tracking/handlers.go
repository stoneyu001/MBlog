package tracking

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 不分区埋点请求数据结构
type UnpartitionedTrackEventRequest struct {
	SessionID        string                 `json:"session_id"`
	UserID           string                 `json:"user_id"`
	EventType        string                 `json:"event_type"`
	ElementPath      string                 `json:"element_path"`
	PagePath         string                 `json:"page_path"`
	Referrer         string                 `json:"referrer"`
	Metadata         map[string]interface{} `json:"metadata"`
	Timestamp        int64                  `json:"timestamp"`         // 前端事件发生的时间戳(毫秒)
	CustomProperties map[string]interface{} `json:"custom_properties"` // 自定义属性
	Platform         string                 `json:"platform"`          // 平台：WEB, IOS, ANDROID等
	DeviceInfo       map[string]interface{} `json:"device_info"`       // 设备信息
	EventDuration    int                    `json:"event_duration"`    // 事件持续时间(毫秒)
	EventSource      string                 `json:"event_source"`      // 事件来源
	AppVersion       string                 `json:"app_version"`       // 应用版本号
}

// 批量不分区埋点请求
type BatchUnpartitionedTrackRequest struct {
	Events []UnpartitionedTrackEventRequest `json:"events"`
}

// RegisterHandlers 注册跟踪相关的API路由
func (ts *TrackingService) RegisterHandlers(router *gin.Engine) {
	trackGroup := router.Group("/api/tracking")
	{
		// 单个事件上报
		trackGroup.POST("/event", ts.handleUnpartitionedTrackEvent)

		// 批量事件上报
		trackGroup.POST("/batch", ts.handleUnpartitionedBatchEvents)

		// 检查跟踪服务状态
		trackGroup.GET("/status", ts.handleTrackingStatus)
	}
}

// 处理单个不分区埋点事件
func (ts *TrackingService) handleUnpartitionedTrackEvent(c *gin.Context) {
	var req UnpartitionedTrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("请求数据解析失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	log.Printf("接收到埋点请求: type=%s, session=%s, path=%s, timestamp=%d",
		req.EventType, req.SessionID, req.PagePath, req.Timestamp)

	// 验证必填字段
	if req.EventType == "" {
		log.Printf("事件类型为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "事件类型不能为空"})
		return
	}

	// 创建跟踪事件
	event := convertToUnpartitionedTrackEvent(req, c)

	// 发送到跟踪服务
	ts.TrackUnpartitionedEvent(event)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// 处理批量不分区埋点事件
func (ts *TrackingService) handleUnpartitionedBatchEvents(c *gin.Context) {
	var batchReq BatchUnpartitionedTrackRequest
	if err := c.ShouldBindJSON(&batchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的批量请求数据"})
		return
	}

	// 验证请求中的事件数量
	if len(batchReq.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量请求不能为空"})
		return
	}

	// 限制单次请求的最大事件数
	if len(batchReq.Events) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次批量请求不能超过200个事件"})
		return
	}

	log.Printf("收到批量请求，事件数量: %d", len(batchReq.Events))
	validEvents := 0

	// 处理每个埋点事件
	for _, req := range batchReq.Events {
		if req.EventType == "" {
			continue // 跳过无效事件
		}

		// 转换为事件对象并发送
		event := convertToUnpartitionedTrackEvent(req, c)
		ts.TrackUnpartitionedEvent(event)
		validEvents++
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "processed": validEvents})
}

// 跟踪服务状态
func (ts *TrackingService) handleTrackingStatus(c *gin.Context) {
	// 查询数据库中总埋点数量
	var count int64
	err := ts.db.QueryRow("SELECT COUNT(*) FROM track_events_unpartitioned").Scan(&count)

	status := "active"
	if err != nil {
		status = "error"
		log.Printf("获取埋点数量失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       status,
		"timestamp":    time.Now().Unix(),
		"total_events": count,
	})
}
