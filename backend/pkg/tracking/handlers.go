package tracking

import (
	"encoding/json"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 验证必填字段
	if req.EventType == "" {
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
	if len(batchReq.Events) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次批量请求不能超过1000个事件"})
		return
	}

	// 处理每个埋点事件
	for _, req := range batchReq.Events {
		if req.EventType == "" {
			continue // 跳过无效事件
		}

		// 转换为事件对象并发送
		event := convertToUnpartitionedTrackEvent(req, c)
		ts.TrackUnpartitionedEvent(event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "processed": len(batchReq.Events)})
}

// 跟踪服务状态
func (ts *TrackingService) handleTrackingStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "active",
		"timestamp": time.Now().Unix(),
	})
}

// 将请求转换为不分区埋点事件对象
func convertToUnpartitionedTrackEvent(req UnpartitionedTrackEventRequest, c *gin.Context) *UnpartitionedTrackEvent {
	// 处理时间戳
	var eventTime time.Time
	if req.Timestamp > 0 {
		// 使用客户端提供的时间戳，但限制不能早于24小时前
		clientTime := time.UnixMilli(req.Timestamp)
		minTime := time.Now().Add(-24 * time.Hour)
		if clientTime.After(minTime) {
			eventTime = clientTime
		} else {
			eventTime = time.Now()
		}
	} else {
		eventTime = time.Now()
	}

	// 创建事件对象
	event := &UnpartitionedTrackEvent{
		SessionID:        req.SessionID,
		UserID:           req.UserID,
		EventType:        req.EventType,
		ElementPath:      req.ElementPath,
		PagePath:         req.PagePath,
		Referrer:         req.Referrer,
		Metadata:         convertMapToString(req.Metadata),
		UserAgent:        c.Request.UserAgent(),
		IPAddress:        c.ClientIP(),
		CreatedAt:        eventTime,
		CustomProperties: convertMapToString(req.CustomProperties),
		Platform:         req.Platform,
		DeviceInfo:       convertMapToString(req.DeviceInfo),
		EventDuration:    req.EventDuration,
		EventSource:      req.EventSource,
		AppVersion:       req.AppVersion,
	}

	return event
}

// 将map转换为JSON字符串
func convertMapToString(data map[string]interface{}) string {
	if data == nil {
		return "{}"
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}

	return string(jsonData)
}

// TrackingMiddleware 跟踪中间件
func (ts *TrackingService) TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行请求
		c.Next()

		// 不跟踪API请求，只跟踪页面请求
		if c.Request.Method != "GET" || c.Writer.Status() >= 400 {
			return
		}

		// 跟踪信息
		path := c.Request.URL.Path
		// 只记录前端页面访问
		if !isAPIPath(path) {
			// 获取会话ID
			sessionID, _ := c.Cookie("session_id")

			event := &UnpartitionedTrackEvent{
				SessionID:   sessionID,
				EventType:   "PAGEVIEW",
				PagePath:    path,
				Referrer:    c.GetHeader("Referer"),
				UserAgent:   c.Request.UserAgent(),
				IPAddress:   c.ClientIP(),
				CreatedAt:   time.Now(),
				Platform:    "WEB",
				EventSource: "SERVER",
			}
			ts.TrackUnpartitionedEvent(event)
		}
	}
}

// 判断是否为API路径
func isAPIPath(path string) bool {
	// API路径通常以/api开头
	return len(path) >= 4 && path[0:4] == "/api"
}
