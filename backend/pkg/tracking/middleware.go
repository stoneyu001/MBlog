package tracking

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 将请求转换为不分区埋点事件对象
func convertToUnpartitionedTrackEvent(req UnpartitionedTrackEventRequest, c *gin.Context) *UnpartitionedTrackEvent {
	// 处理时间戳
	var eventTime time.Time
	if req.Timestamp > 0 {
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

	// JSON字段转换
	metadata := convertMapToString(req.Metadata)
	customProps := convertMapToString(req.CustomProperties)
	deviceInfo := convertMapToString(req.DeviceInfo)

	// 创建事件对象
	event := &UnpartitionedTrackEvent{
		SessionID:        req.SessionID,
		UserID:           req.UserID,
		EventType:        req.EventType,
		ElementPath:      req.ElementPath,
		PagePath:         req.PagePath,
		Referrer:         req.Referrer,
		Metadata:         metadata,
		UserAgent:        c.Request.UserAgent(),
		IPAddress:        c.ClientIP(),
		CreatedAt:        eventTime,
		CustomProperties: customProps,
		Platform:         req.Platform,
		DeviceInfo:       deviceInfo,
		EventDuration:    req.EventDuration,
		EventSource:      req.EventSource,
		AppVersion:       req.AppVersion,
	}

	return event
}

// 将map转换为JSON字符串
func convertMapToString(data map[string]interface{}) string {
	if data == nil || len(data) == 0 {
		return `{}`
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return `{}`
	}

	return string(jsonData)
}

// TrackingMiddleware 跟踪中间件
func (ts *TrackingService) TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只对/api/tracking/batch端点进行特殊处理
		if c.Request.URL.Path == "/api/tracking/batch" {
			c.Next()
			return
		}

		// 对其他请求自动记录REQUEST事件
		event := &UnpartitionedTrackEvent{
			EventType:        "REQUEST",
			PagePath:         c.Request.URL.Path,
			UserAgent:        c.Request.UserAgent(),
			IPAddress:        c.ClientIP(),
			CreatedAt:        time.Now(),
			CustomProperties: `{"auto_tracked": true}`,
		}

		ts.TrackUnpartitionedEvent(event)
		c.Next()
	}
}

// BatchTrackingHandler 处理批量埋点请求
func (ts *TrackingService) BatchTrackingHandler(c *gin.Context) {
	var events []UnpartitionedTrackEventRequest
	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	for _, req := range events {
		event := convertToUnpartitionedTrackEvent(req, c)
		ts.TrackUnpartitionedEvent(event)
	}

	c.JSON(200, gin.H{"status": "success"})
}
