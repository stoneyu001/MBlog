package tracking

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 前端埋点请求数据结构
type TrackEventRequest struct {
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	ElementPath string                 `json:"element_path"`
	PagePath    string                 `json:"page_path"`
	Referrer    string                 `json:"referrer"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   int64                  `json:"timestamp"` // 前端事件发生的时间戳(毫秒)
}

// 批量埋点请求
type BatchTrackRequest struct {
	Events []TrackEventRequest `json:"events"`
}

// RegisterHandlers 注册跟踪相关的API路由
func (ts *TrackingService) RegisterHandlers(router *gin.Engine) {
	trackGroup := router.Group("/api/tracking")
	{
		// 单个事件上报
		trackGroup.POST("/event", ts.handleTrackEvent)

		// 批量事件上报
		trackGroup.POST("/batch", ts.handleBatchEvents)

		// 检查跟踪服务状态
		trackGroup.GET("/status", ts.handleTrackingStatus)
	}
}

// 处理单个埋点事件
func (ts *TrackingService) handleTrackEvent(c *gin.Context) {
	var req TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 验证必填字段
	if req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "事件类型不能为空"})
		return
	}

	// 创建跟踪事件
	event := convertToTrackEvent(req, c)

	// 发送到跟踪服务
	ts.TrackEvent(event)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// 处理批量埋点事件
func (ts *TrackingService) handleBatchEvents(c *gin.Context) {
	var batchReq BatchTrackRequest
	if err := c.ShouldBindJSON(&batchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的批量请求数据"})
		return
	}

	if len(batchReq.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量事件不能为空"})
		return
	}

	// 批量处理事件
	for _, req := range batchReq.Events {
		if req.EventType == "" {
			continue // 跳过无效事件
		}

		event := convertToTrackEvent(req, c)
		ts.TrackEvent(event)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(batchReq.Events),
	})
}

// 跟踪服务状态
func (ts *TrackingService) handleTrackingStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "active",
		"timestamp": time.Now().Unix(),
	})
}

// 将请求转换为跟踪事件
func convertToTrackEvent(req TrackEventRequest, c *gin.Context) *TrackEvent {
	// 如果请求未提供会话ID，则获取或生成一个
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = getSessionID(c)
	}

	// 处理时间戳
	createdAt := time.Now()
	if req.Timestamp > 0 {
		// 使用客户端提供的时间戳，但限制不能早于24小时前
		clientTime := time.UnixMilli(req.Timestamp)
		minTime := time.Now().Add(-24 * time.Hour)
		if clientTime.After(minTime) {
			createdAt = clientTime
		}
	}

	// 序列化元数据
	metadataJSON, _ := json.Marshal(req.Metadata)

	return &TrackEvent{
		SessionID:   sessionID,
		UserID:      req.UserID,
		EventType:   req.EventType,
		ElementPath: req.ElementPath,
		PagePath:    req.PagePath,
		Referrer:    req.Referrer,
		Metadata:    string(metadataJSON),
		UserAgent:   c.Request.UserAgent(),
		IPAddress:   getClientIP(c),
		CreatedAt:   createdAt,
	}
}
