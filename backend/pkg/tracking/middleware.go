package tracking

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"strings"
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

// 确保数据库连接使用UTF8编码
func (ts *TrackingService) ensureUTF8Encoding() error {
	if ts.db != nil {
		_, err := ts.db.Exec("SET client_encoding = 'UTF8'")
		if err != nil {
			log.Printf("设置数据库客户端编码失败: %v", err)
			return err
		}
	}
	return nil
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

	// 确保必要字段有值
	if req.EventType == "" {
		req.EventType = "UNKNOWN"
		log.Printf("警告: 事件类型为空，使用默认值")
	}

	// 统一处理所有URL相关字段的解码
	// 处理页面路径
	pagePath := req.PagePath
	if pagePath == "" {
		pagePath = "/"
		log.Printf("警告: 页面路径为空，使用默认值: /")
	} else {
		decodedPath, err := url.QueryUnescape(pagePath)
		if err != nil {
			log.Printf("页面路径URL解码失败: %v, 使用原始路径: %s", err, pagePath)
		} else {
			if decodedPath != pagePath {
				log.Printf("页面路径URL解码: %s -> %s", pagePath, decodedPath)
			}
			pagePath = decodedPath
		}
	}

	// 处理元素路径
	elementPath := req.ElementPath
	if elementPath != "" {
		decodedPath, err := url.QueryUnescape(elementPath)
		if err != nil {
			log.Printf("元素路径URL解码失败: %v, 使用原始路径: %s", err, elementPath)
		} else {
			if decodedPath != elementPath {
				log.Printf("元素路径URL解码: %s -> %s", elementPath, decodedPath)
			}
			elementPath = decodedPath
		}
	}

	// 处理来源URL
	referrer := req.Referrer
	if referrer != "" {
		decodedURL, err := url.QueryUnescape(referrer)
		if err != nil {
			log.Printf("来源URL解码失败: %v, 使用原始URL: %s", err, referrer)
		} else {
			if decodedURL != referrer {
				log.Printf("来源URL解码: %s -> %s", referrer, decodedURL)
			}
			referrer = decodedURL
		}
	}

	// 处理自定义属性中的URL
	customProps := req.CustomProperties
	// 检查并解码自定义属性中的URL字段
	for key, value := range customProps {
		if strValue, ok := value.(string); ok {
			if strings.Contains(key, "url") || strings.Contains(key, "path") || strings.Contains(key, "link") {
				decodedValue, err := url.QueryUnescape(strValue)
				if err == nil && decodedValue != strValue {
					log.Printf("自定义属性URL解码 [%s]: %s -> %s", key, strValue, decodedValue)
					customProps[key] = decodedValue
				}
			}
		}
	}

	// JSON字段转换
	metadata := convertMapToString(req.Metadata)
	customPropsStr := convertMapToString(customProps)
	deviceInfo := convertMapToString(req.DeviceInfo)

	// 创建事件对象
	event := &UnpartitionedTrackEvent{
		SessionID:        req.SessionID,
		UserID:           req.UserID,
		EventType:        req.EventType,
		ElementPath:      elementPath,
		PagePath:         pagePath,
		Referrer:         referrer,
		Metadata:         metadata,
		UserAgent:        c.Request.UserAgent(),
		IPAddress:        c.ClientIP(),
		CreatedAt:        eventTime,
		CustomProperties: customPropsStr,
		Platform:         req.Platform,
		DeviceInfo:       deviceInfo,
		EventDuration:    req.EventDuration,
	}

	// 记录事件处理日志
	log.Printf("事件处理完成: type=%s, path=%s, element=%s",
		event.EventType, event.PagePath, event.ElementPath)

	return event
}

// 将map转换为JSON字符串，确保中文正确处理
func convertMapToString(data map[string]interface{}) string {
	if len(data) == 0 {
		return `{}`
	}

	// 使用带有中文处理的JSON编码器
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // 防止中文被转义
	if err := encoder.Encode(data); err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return `{}`
	}

	// 去除encoder自动添加的换行符
	return strings.TrimSpace(buffer.String())
}

// TrackingMiddleware 跟踪中间件
func (ts *TrackingService) TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 确保UTF8编码
		if err := ts.ensureUTF8Encoding(); err != nil {
			log.Printf("设置UTF8编码失败: %v", err)
		}

		// 只对/api/tracking/batch端点进行特殊处理
		if c.Request.URL.Path == "/api/tracking/batch" {
			c.Next()
			return
		}

		// 过滤掉静态资源请求
		if shouldSkipRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 从请求头中获取设备指纹和会话ID
		deviceFingerprint := c.GetHeader("X-Device-Fingerprint")
		sessionID := c.GetHeader("X-Session-ID")

		// 对页面路径进行URL解码
		pagePath := c.Request.URL.Path
		decodedPagePath, err := url.QueryUnescape(pagePath)
		if err != nil {
			log.Printf("中间件URL解码失败: %v, 使用原始路径: %s", err, pagePath)
		} else {
			if decodedPagePath != pagePath {
				log.Printf("中间件URL解码: %s -> %s", pagePath, decodedPagePath)
			}
			pagePath = decodedPagePath
		}

		// 获取请求方法和查询参数
		method := c.Request.Method
		query := c.Request.URL.RawQuery

		// 对其他请求自动记录REQUEST事件
		event := &UnpartitionedTrackEvent{
			EventType:        "REQUEST",
			PagePath:         pagePath,
			UserAgent:        c.Request.UserAgent(),
			IPAddress:        c.ClientIP(),
			CreatedAt:        time.Now().In(chinaLocation),
			CustomProperties: createCustomProperties(method, query),
			UserID:           deviceFingerprint,   // 使用设备指纹作为user_id
			SessionID:        sessionID,           // 使用会话ID
			Referrer:         c.Request.Referer(), // 添加来源页面
		}

		// 如果没有设备指纹，使用一个临时ID
		if event.UserID == "" {
			event.UserID = "auto_generated_" + c.ClientIP()
			log.Printf("自动生成用户ID: %s", event.UserID)
		}

		// 如果没有会话ID，生成一个临时ID
		if event.SessionID == "" {
			event.SessionID = "auto_" + time.Now().Format("20060102150405") + "_" + c.ClientIP()
			log.Printf("自动生成会话ID: %s", event.SessionID)
		}

		log.Printf("自动跟踪请求: method=%s, path=%s, query=%s, user_id=%s, session_id=%s",
			method, event.PagePath, query, event.UserID, event.SessionID)

		ts.TrackUnpartitionedEvent(event)
		c.Next()
	}
}

// 判断是否应该跳过该请求的跟踪
func shouldSkipRequest(path string) bool {
	// 静态资源后缀
	staticExtensions := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif",
		".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot",
	}

	// 静态资源目录
	staticDirs := []string{
		"/static/", "/assets/", "/images/",
		"/css/", "/js/", "/fonts/",
	}

	// 检查路径后缀
	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}

	// 检查静态资源目录
	for _, dir := range staticDirs {
		if strings.HasPrefix(strings.ToLower(path), dir) {
			return true
		}
	}

	return false
}

// 创建自定义属性
func createCustomProperties(method string, query string) string {
	props := map[string]interface{}{
		"auto_tracked": true,
		"method":       method,
	}

	// 如果有查询参数，添加到自定义属性（注意去除敏感信息）
	if query != "" {
		props["has_query"] = true
		// 可以选择是否记录具体的查询参数
		// props["query"] = query
	}

	jsonData, err := json.Marshal(props)
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return `{"auto_tracked":true}`
	}

	return string(jsonData)
}

// BatchTrackingHandler 处理批量埋点请求
func (ts *TrackingService) BatchTrackingHandler(c *gin.Context) {
	// 确保UTF8编码
	if err := ts.ensureUTF8Encoding(); err != nil {
		log.Printf("设置UTF8编码失败: %v", err)
	}

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
