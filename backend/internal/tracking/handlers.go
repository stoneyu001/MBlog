package tracking

import (
	"blog/internal/middleware"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	EventDuration    int                    `json:"event_duration"`    // 页面停留时间（秒，仅 PAGE_LEAVE）
}

// 批量不分区埋点请求
type BatchUnpartitionedTrackRequest struct {
	Events []UnpartitionedTrackEventRequest `json:"events"`
}

var allowedTrackEventTypes = map[string]bool{
	"PAGEVIEW":   true,
	"PAGE_LEAVE": true,
	"CLICK":      true,
	"EXPOSURE":   true,
	"CUSTOM":     true,
}

func validateTrackRequest(req *UnpartitionedTrackEventRequest) error {
	req.EventType = strings.ToUpper(strings.TrimSpace(req.EventType))
	req.PagePath = strings.TrimSpace(req.PagePath)
	if !allowedTrackEventTypes[req.EventType] {
		return fmt.Errorf("不支持的事件类型")
	}
	if req.PagePath == "" || len(req.PagePath) > 2048 {
		return fmt.Errorf("页面路径为空或过长")
	}
	if len(req.SessionID) > 100 || len(req.UserID) > 100 {
		return fmt.Errorf("会话或访客标识过长")
	}
	if len(req.ElementPath) > 4096 || len(req.Referrer) > 4096 {
		return fmt.Errorf("事件字段过长")
	}
	if req.EventDuration < 0 || req.EventDuration > 3600 {
		return fmt.Errorf("页面停留时间超出范围")
	}
	return nil
}

// RegisterHandlers 注册跟踪相关的API路由
func (ts *TrackingService) RegisterHandlers(router *gin.Engine) {
	trackGroup := router.Group("/api/tracking")
	{
		// 单个事件上报
		trackGroup.POST("/event", middleware.RateLimit(120, time.Minute), ts.handleUnpartitionedTrackEvent)

		// 批量事件上报
		trackGroup.POST("/batch", middleware.RateLimit(120, time.Minute), ts.handleUnpartitionedBatchEvents)

		// 服务状态包含内部事件总量，只对管理员开放。
		trackGroup.GET("/status", middleware.RequireAuth(), middleware.NoStore(), ts.handleTrackingStatus)
	}
}

func (ts *TrackingService) handleUnpartitionedTrackEvent(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
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

		c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
		return
	}

	if err := validateTrackRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	// 解析原始JSON
	var events []map[string]interface{}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
	body, err := c.GetRawData()
	if err != nil {
		log.Printf("读取请求体失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求数据失败"})
		return
	}
	if err := json.Unmarshal(body, &events); err != nil {
		log.Printf("解析JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if len(events) == 0 {
		log.Printf("警告: 批量请求为空")
		c.JSON(http.StatusOK, gin.H{"status": "success", "processed": 0})
		return
	}

	log.Printf("收到批量请求，事件数量: %d", len(events))
	validEvents := 0
	invalidEvents := 0

	if len(events) > 200 {
		log.Printf("警告: 批量请求超过200个事件，将只处理前200个")
		invalidEvents = len(events) - 200
		events = events[:200]
	}

	// 处理每个事件
	for _, eventMap := range events {
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

		if err := validateTrackRequest(&req); err != nil {
			invalidEvents++
			continue
		}

		// 转换为事件对象并发送
		event := convertToUnpartitionedTrackEvent(req, c)
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
	count, err := ts.repo.CountAll()
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
