package tracking

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 初始化中国时区
var chinaLocation *time.Location

func init() {
	var err error
	// 强制设置为Asia/Shanghai
	chinaLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("加载中国时区失败: %v, 尝试使用UTC+8", err)
		// 如果无法加载时区，则使用固定的UTC+8
		chinaLocation = time.FixedZone("CST", 8*60*60)
	}

	// 输出当前时区和时间，用于调试
	log.Printf("初始化时区: %s, 当前时间: %s", chinaLocation.String(), time.Now().In(chinaLocation).Format("2006-01-02 15:04:05"))
}

// 将请求转换为不分区埋点事件对象
func convertToUnpartitionedTrackEvent(req UnpartitionedTrackEventRequest, c *gin.Context) *UnpartitionedTrackEvent {
	// 处理时间戳
	var eventTime time.Time
	if req.Timestamp > 0 {
		// 使用客户端时间戳，但确保转换为正确的时区
		clientTime := time.UnixMilli(req.Timestamp).In(chinaLocation)
		minTime := time.Now().Add(-24 * time.Hour)
		if clientTime.After(minTime) {
			eventTime = clientTime
		} else {
			eventTime = time.Now().In(chinaLocation)
		}
	} else {
		eventTime = time.Now().In(chinaLocation)
	}

	// 记录当前处理的时间信息，用于调试
	log.Printf("事件时间处理: timestamp=%d, 转换后时间=%s, 时区=%s",
		req.Timestamp, eventTime.Format("2006-01-02 15:04:05"), eventTime.Location().String())

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
	if len(data) == 0 {
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
			CreatedAt:        time.Now().In(chinaLocation),
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
