package tracking

import (
	"encoding/json"
	"fmt"
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

func (ts *TrackingService) handleUnpartitionedTrackEvent(c *gin.Context) {
	var req UnpartitionedTrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("请求数据解析失败: %v", err)

		// 提供更详细的错误信息
		errorMsg := "无效的请求数据"
		if jsonErr, ok := err.(*json.SyntaxError); ok {
			errorMsg = fmt.Sprintf("JSON格式错误: 位置 %d", jsonErr.Offset)
		} else if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
			errorMsg = fmt.Sprintf("字段类型错误: %s 应该是 %s 类型", jsonErr.Field, jsonErr.Type)
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   errorMsg,
			"details": err.Error(),
		})
		return
	}

	log.Printf("接收到埋点请求: type=%s, session=%s, user_id=%s, path=%s, timestamp=%d",
		req.EventType, req.SessionID, req.UserID, req.PagePath, req.Timestamp)

	// 验证并设置缺失字段的默认值而不是拒绝请求
	if req.EventType == "" {
		req.EventType = "UNKNOWN"
		log.Printf("警告: 事件类型为空，使用默认值: %s", req.EventType)
	}

	// 创建跟踪事件
	event := convertToUnpartitionedTrackEvent(req, c)

	// 发送到跟踪服务
	ts.TrackUnpartitionedEvent(event)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// 处理批量不分区埋点事件
func (ts *TrackingService) handleUnpartitionedBatchEvents(c *gin.Context) {
	// 解析原始JSON
	var events []map[string]interface{}

	body, _ := c.GetRawData()
	log.Printf("接收到原始批量请求数据: %s", string(body))

	if err := json.Unmarshal(body, &events); err != nil {
		log.Printf("解析JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	log.Printf("解析后的events数组: %+v", events)

	if len(events) == 0 {
		log.Printf("警告: 批量请求为空")
		c.JSON(http.StatusOK, gin.H{"status": "success", "processed": 0})
		return
	}

	if len(events) > 200 {
		log.Printf("警告: 批量请求超过200个事件，将只处理前200个")
		events = events[:200]
	}

	log.Printf("收到批量请求，事件数量: %d", len(events))
	validEvents := 0
	invalidEvents := 0

	// 处理每个事件
	for i, eventMap := range events {
		// 详细记录每个事件的原始数据
		log.Printf("事件[%d]原始数据: %+v", i, eventMap)

		// 特别检查platform和event_duration字段
		if platform, exists := eventMap["platform"]; exists {
			log.Printf("事件[%d]包含platform: %v (类型: %T)", i, platform, platform)
		} else {
			log.Printf("警告: 事件[%d]不包含platform字段", i)
		}

		if duration, exists := eventMap["event_duration"]; exists {
			log.Printf("事件[%d]包含event_duration: %v (类型: %T)", i, duration, duration)
		} else {
			log.Printf("警告: 事件[%d]不包含event_duration字段", i)
		}

		// 构建请求结构体
		req := UnpartitionedTrackEventRequest{
			EventType:        getStringWithFallback(eventMap, "event_type", "eventType"),
			SessionID:        getStringWithFallback(eventMap, "session_id", "sessionId"),
			UserID:           getStringWithFallback(eventMap, "user_id", "userId"),
			ElementPath:      getStringWithFallback(eventMap, "element_path", "elementPath"),
			PagePath:         getStringWithFallback(eventMap, "page_path", "pagePath"),
			Referrer:         getStringWithFallback(eventMap, "referrer", "referrer"),
			Timestamp:        getInt64WithFallback(eventMap, "timestamp", "timestamp"),
			Platform:         getStringWithFallback(eventMap, "platform", "platform"),
			EventDuration:    getIntWithFallback(eventMap, "event_duration", "eventDuration"),
			Metadata:         getMapWithFallback(eventMap, "metadata", "metadata"),
			CustomProperties: getMapWithFallback(eventMap, "custom_properties", "customProperties"),
			DeviceInfo:       getMapWithFallback(eventMap, "device_info", "deviceInfo"),
		}

		// 打印请求内容以调试
		log.Printf("处理事件: platform=%s, event_duration=%d, type=%s, path=%s",
			req.Platform, req.EventDuration, req.EventType, req.PagePath)

		// 转换为事件对象并发送
		event := convertToUnpartitionedTrackEvent(req, c)
		log.Printf("转换后的事件对象: platform=%s, event_duration=%d",
			event.Platform, event.EventDuration)
		ts.TrackUnpartitionedEvent(event)
		validEvents++
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"processed": validEvents,
		"invalid":   invalidEvents,
	})
}

// 同时尝试下划线和驼峰两种命名获取map值
func getMapWithFallback(m map[string]interface{}, snakeKey, camelKey string) map[string]interface{} {
	if val, ok := m[snakeKey].(map[string]interface{}); ok {
		return val
	}
	if val, ok := m[camelKey].(map[string]interface{}); ok {
		return val
	}
	return nil
}

// 同时尝试下划线和驼峰两种命名获取字符串值
func getStringWithFallback(m map[string]interface{}, snakeKey, camelKey string) string {
	if val, ok := m[snakeKey].(string); ok && val != "" {
		return val
	}
	if val, ok := m[camelKey].(string); ok {
		return val
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// 同时尝试下划线和驼峰两种命名获取int64值
func getInt64WithFallback(m map[string]interface{}, snakeKey, camelKey string) int64 {
	if val := getInt64(m, snakeKey); val != 0 {
		return val
	}
	return getInt64(m, camelKey)
}

// 同时尝试下划线和驼峰两种命名获取int值
func getIntWithFallback(m map[string]interface{}, snakeKey, camelKey string) int {
	// 先尝试snake_case，检查key是否存在
	if val, exists := m[snakeKey]; exists {
		return convertToInt(val)
	}
	// 再尝试camelCase
	if val, exists := m[camelKey]; exists {
		return convertToInt(val)
	}
	return 0
}

// convertToInt 将interface{}转换为int，支持多种数值类型
func convertToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case int32:
		return int(val)
	default:
		return 0
	}
}

// 跟踪服务状态
func (ts *TrackingService) handleTrackingStatus(c *gin.Context) {
	// 查询数据库中总埋点数量
	var count int64
	err := ts.db.QueryRow("SELECT COUNT(*) FROM track_event").Scan(&count)

	status := "active"
	if err != nil {
		status = "error"
		log.Printf("获取埋点数量失败: %v", err)
	}

	// 使用中国时区的当前时间
	currentTime := time.Now().In(chinaLocation)

	c.JSON(http.StatusOK, gin.H{
		"status":         status,
		"timestamp":      currentTime.Unix(),
		"total_events":   count,
		"timezone":       "Asia/Shanghai",
		"formatted_time": currentTime.Format("2006-01-02 15:04:05"),
	})
}
