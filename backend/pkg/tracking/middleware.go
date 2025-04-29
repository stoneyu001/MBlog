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
	log.Println("埋点服务中间件已初始化")
	return func(c *gin.Context) {
		c.Next()

		// 删除条件，记录所有请求
		event := &UnpartitionedTrackEvent{
			EventType:        "REQUEST",
			PagePath:         c.Request.URL.Path,
			CreatedAt:        time.Now(),
			Metadata:         `{"source": "middleware"}`,
			CustomProperties: `{"auto_tracked": true}`,
			DeviceInfo:       `{"type": "server"}`,
		}
		ts.TrackUnpartitionedEvent(event)
	}
}
