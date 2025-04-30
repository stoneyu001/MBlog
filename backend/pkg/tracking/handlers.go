package tracking

import (
	"encoding/json"
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
	// 解析原始JSON
	var rawData map[string]interface{}
	body, _ := c.GetRawData()
	if err := json.Unmarshal(body, &rawData); err != nil {
		log.Printf("解析JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 从events字段获取事件数组
	eventsRaw, ok := rawData["events"]
	if !ok || eventsRaw == nil {
		log.Printf("请求中没有events字段")
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求中缺少events字段"})
		return
	}

	// 将事件数组转换为[]map[string]interface{}
	eventsArray, ok := eventsRaw.([]interface{})
	if !ok {
		log.Printf("events字段不是数组")
		c.JSON(http.StatusBadRequest, gin.H{"error": "events字段不是数组"})
		return
	}

	if len(eventsArray) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量请求不能为空"})
		return
	}

	if len(eventsArray) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次批量请求不能超过200个事件"})
		return
	}

	log.Printf("收到批量请求，事件数量: %d", len(eventsArray))
	validEvents := 0

	// 处理每个事件
	for _, eventRaw := range eventsArray {
		eventMap, ok := eventRaw.(map[string]interface{})
		if !ok {
			log.Printf("跳过非对象事件")
			continue
		}

		// 从map中提取驼峰命名的字段
		eventType, _ := eventMap["eventType"].(string)
		if eventType == "" {
			log.Printf("跳过无效事件: 事件类型为空")
			continue
		}

		// 构建请求结构体
		req := UnpartitionedTrackEventRequest{
			EventType:   eventType,
			SessionID:   getString(eventMap, "sessionId"),
			UserID:      getString(eventMap, "userId"),
			ElementPath: getString(eventMap, "elementPath"),
			PagePath:    getString(eventMap, "pagePath"),
			Referrer:    getString(eventMap, "referrer"),
			Timestamp:   getInt64(eventMap, "timestamp"),
		}

		// 处理metadata
		if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
			req.Metadata = metadata
		}

		// 打印请求内容以调试
		log.Printf("处理事件: %+v", req)

		// 转换为事件对象并发送
		event := convertToUnpartitionedTrackEvent(req, c)
		ts.TrackUnpartitionedEvent(event)
		validEvents++
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"processed": validEvents,
	})
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
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
