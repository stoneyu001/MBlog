package tracking

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/url"
	"strings"
	"sync"
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

	// 时区初始化完成

	// 启动后台缓存清理任务
	go backgroundCacheCleaner()
}

// backgroundCacheCleaner 后台定时清理过期缓存
func backgroundCacheCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 后台缓存清理器已启动

	for range ticker.C {
		cleanExpiredCache()
	}
}

// cleanExpiredCache 清理超过1小时的过期缓存
func cleanExpiredCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	now := time.Now().UnixMilli()

	for sid, data := range lastEventCache {
		if now-data.Timestamp > 3600000 { // 1小时 = 3600000毫秒
			delete(lastEventCache, sid)
		}
	}

	// 缓存已清理
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

// 添加新的结构体用于缓存上一次事件信息
type LastEventInfo struct {
	Timestamp int64
	EventType string
	SessionID string
}

// 用于存储每个会话的最后一次事件信息
var (
	lastEventCache = make(map[string]LastEventInfo)
	cacheMutex     sync.RWMutex
)

func getLastEventInfo(sessionID string) (LastEventInfo, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	info, exists := lastEventCache[sessionID]
	return info, exists
}

func updateLastEventInfo(sessionID string, info LastEventInfo) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	lastEventCache[sessionID] = info
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
			// timestamp过旧，使用服务器时间
			eventTime = time.Now().In(chinaLocation)
		}
	} else {
		// timestamp无效，使用服务器时间
		eventTime = time.Now().In(chinaLocation)
	}

	// 确保必要字段有值
	if req.EventType == "" {
		req.EventType = "UNKNOWN"

	}

	// 验证并智能处理platform字段
	validPlatforms := map[string]bool{
		"Windows": true, "macOS": true, "Linux": true,
		"Android": true, "iOS": true, "Unknown": true,
	}

	// 智能解析platform：处理前端发送的"OS/Browser"格式
	var extractedBrowser string

	// 如果包含"/"，说明是"OS/Browser"格式，需要拆分
	if strings.Contains(req.Platform, "/") {
		parts := strings.Split(req.Platform, "/")
		if len(parts) >= 2 {
			req.Platform = parts[0]     // OS部分作为platform
			extractedBrowser = parts[1] // Browser部分单独保存

		}
	}

	// 如果platform为空、unknown或不在有效列表中，则从UserAgent提取
	if req.Platform == "" || req.Platform == "unknown" || !validPlatforms[req.Platform] {
		// 记录原始值（如果有）
		if req.Platform != "" && req.Platform != "unknown" {

		}

		// 从user_agent中提取平台信息
		userAgent := c.Request.UserAgent()
		platform, browser := extractPlatformFromUA(userAgent)
		req.Platform = platform
		extractedBrowser = browser // 使用UserAgent提取的浏览器

	}

	// 将浏览器信息保存到metadata中
	if extractedBrowser != "" {
		if req.Metadata == nil {
			req.Metadata = make(map[string]interface{})
		}
		// 只在metadata中没有browser字段时才设置
		if _, exists := req.Metadata["browser"]; !exists {
			req.Metadata["browser"] = extractedBrowser
		}
		req.Metadata["user_agent"] = c.Request.UserAgent()

	}

	// 优化event_duration计算：计算任意两个事件之间的时间差
	if req.EventDuration <= 0 {
		// 获取上一次事件信息
		lastEvent, exists := getLastEventInfo(req.SessionID)

		if exists && req.Timestamp > lastEvent.Timestamp {
			durationMs := req.Timestamp - lastEvent.Timestamp

			// 计算所有事件之间的持续时间（不限制事件类型）
			// 上限设为1小时（3600秒），避免异常值
			req.EventDuration = int(math.Min(float64(durationMs)/1000.0, 3600.0))

			// 计算事件持续时间
			req.EventDuration = int(math.Min(float64(durationMs)/1000.0, 3600.0))
		}
	}

	// 始终更新最后一次事件信息（用于下次计算）
	updateLastEventInfo(req.SessionID, LastEventInfo{
		Timestamp: req.Timestamp,
		EventType: req.EventType,
		SessionID: req.SessionID,
	})

	// 统一处理所有URL相关字段的解码
	// 处理页面路径
	pagePath := req.PagePath
	if pagePath == "" {
		pagePath = "/"

	} else {
		decodedPath, err := url.QueryUnescape(pagePath)
		if err != nil {
			// URL解码失败
		} else {
			pagePath = decodedPath
		}
	}

	// 处理元素路径
	elementPath := req.ElementPath
	if elementPath != "" {
		decodedPath, err := url.QueryUnescape(elementPath)
		if err != nil {
			// URL解码失败
		} else {
			elementPath = decodedPath
		}
	}

	// 处理来源URL
	referrer := req.Referrer
	if referrer != "" {
		decodedURL, err := url.QueryUnescape(referrer)
		if err != nil {
			// URL解码失败
		} else {
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
					customProps[key] = decodedValue
				}
			}
		}
	}

	// 处理metadata中的URL字段
	metadataMap := req.Metadata
	for key, value := range metadataMap {
		if strValue, ok := value.(string); ok {
			// 检查是否是URL相关字段
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "url") ||
				strings.Contains(lowerKey, "path") ||
				strings.Contains(lowerKey, "link") ||
				strings.Contains(lowerKey, "href") {
				decodedValue, err := url.QueryUnescape(strValue)
				if err == nil && decodedValue != strValue {
					metadataMap[key] = decodedValue
				}
			}
		}
	}

	// 清理device_info中的冗余和敏感数据
	deviceInfoMap := req.DeviceInfo
	if deviceInfoMap != nil {
		// 移除可能的敏感字段
		sensitiveKeys := []string{"password", "token", "secret", "key", "auth"}
		for _, sensitiveKey := range sensitiveKeys {
			delete(deviceInfoMap, sensitiveKey)
		}

		// 清理字符串值
		for key, value := range deviceInfoMap {
			if strValue, ok := value.(string); ok {
				deviceInfoMap[key] = cleanString(strValue)
			}
		}
	}

	// 应用字符串清理到主要字段
	req.SessionID = cleanString(req.SessionID)
	req.UserID = cleanString(req.UserID)
	req.EventType = cleanString(req.EventType)
	pagePath = cleanString(pagePath)
	elementPath = cleanString(elementPath)
	referrer = cleanString(referrer)
	req.Platform = cleanString(req.Platform)

	// JSON字段转换（使用已清理的数据）
	metadata := convertMapToString(metadataMap)
	customPropsStr := convertMapToString(customProps)
	deviceInfo := convertMapToString(deviceInfoMap)

	// 从 UserAgent 解析设备类型
	deviceType := extractDeviceTypeFromUA(c.Request.UserAgent())

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
		DeviceType:       deviceType,
	}

	return event
}

// 将map转换为JSON字符串，确保中文正确处理
func convertMapToString(data map[string]interface{}) string {
	// len() 对 nil map 返回 0，所以只需检查 len
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

		// 过滤掉静态资源请求和健康检查请求
		if shouldSkipRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 过滤掉健康检查、认证检查等非业务请求
		skipPaths := []string{
			"/api/ping",
			"/api/auth/check",
			"/favicon.ico",
		}
		for _, skipPath := range skipPaths {
			if c.Request.URL.Path == skipPath {
				c.Next()
				return
			}
		}

		// 从请求头中获取设备指纹和会话ID
		deviceFingerprint := c.GetHeader("X-Device-Fingerprint")
		sessionID := c.GetHeader("X-Session-ID")

		// 对页面路径进行URL解码
		pagePath := c.Request.URL.Path
		decodedPagePath, err := url.QueryUnescape(pagePath)
		if err != nil {
			// URL解码失败，使用原始路径
		} else {
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

		}

		// 如果没有会话ID，生成一个临时ID
		if event.SessionID == "" {
			event.SessionID = "auto_" + time.Now().Format("20060102150405") + "_" + c.ClientIP()

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

// 修改辅助函数
func extractPlatformFromUA(ua string) (platform, browser string) {
	ua = strings.ToLower(ua)

	// 检测操作系统
	switch {
	case strings.Contains(ua, "windows"):
		platform = "Windows"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os x"):
		platform = "macOS"
	case strings.Contains(ua, "linux"):
		platform = "Linux"
	case strings.Contains(ua, "android"):
		platform = "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		platform = "iOS"
	default:
		platform = "Unknown"
	}

	// 检测浏览器（按优先级排序）
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "edg/") && !strings.Contains(ua, "edge/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/"):
		browser = "Safari"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr/"):
		browser = "Opera"
	default:
		browser = "Unknown"
	}

	return platform, browser
}

// extractDeviceTypeFromUA 从 User-Agent 中解析设备类型
func extractDeviceTypeFromUA(ua string) string {
	ua = strings.ToLower(ua)

	// 检测平板设备（先于手机，因为某些平板UA也包含mobile）
	if strings.Contains(ua, "ipad") ||
		(strings.Contains(ua, "android") && !strings.Contains(ua, "mobile")) ||
		strings.Contains(ua, "tablet") {
		return "Tablet"
	}

	// 检测手机设备
	if strings.Contains(ua, "iphone") ||
		strings.Contains(ua, "ipod") ||
		strings.Contains(ua, "android") && strings.Contains(ua, "mobile") ||
		strings.Contains(ua, "windows phone") ||
		strings.Contains(ua, "blackberry") ||
		strings.Contains(ua, "mobile") {
		return "Mobile"
	}

	// 默认为桌面设备
	return "Desktop"
}

// cleanString 清理字符串，去除首尾空格、换行符等
func cleanString(s string) string {
	if s == "" {
		return s
	}
	// 去除首尾空格
	s = strings.TrimSpace(s)
	// 去除多余的换行符和回车符
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// 压缩多个连续空格为单个空格
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
