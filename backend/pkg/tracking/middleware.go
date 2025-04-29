package tracking

import (
	"encoding/json"
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
