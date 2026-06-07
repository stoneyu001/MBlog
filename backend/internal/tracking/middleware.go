package tracking

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// chinaLocation 中国时区（Asia/Shanghai）
var chinaLocation *time.Location

func init() {
	var err error
	chinaLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("加载中国时区失败: %v, 尝试使用UTC+8", err)
		chinaLocation = time.FixedZone("CST", 8*60*60)
	}
	go backgroundCacheCleaner()
}

// === 会话缓存（用于 event_duration 计算）===

// LastEventInfo 缓存每个会话的最后一次事件信息
type LastEventInfo struct {
	Timestamp int64
	EventType string
	SessionID string
}

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

func backgroundCacheCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		cleanExpiredCache()
	}
}

func cleanExpiredCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	now := time.Now().UnixMilli()
	for sid, data := range lastEventCache {
		if now-data.Timestamp > 3600000 {
			delete(lastEventCache, sid)
		}
	}
}
// === 中间件 ===

// TrackingMiddleware 自动埋点中间件：为每个页面请求记录 REQUEST 事件
func (ts *TrackingService) TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过埋点批量上报端点自身
		if c.Request.URL.Path == "/api/tracking/batch" {
			c.Next()
			return
		}

		// 跳过静态资源和健康检查
		if shouldSkipRequest(c.Request.URL.Path) {
			c.Next()
			return
		}

		skipPaths := map[string]bool{
			"/api/ping": true, "/api/auth/check": true, "/favicon.ico": true,
		}
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 构建 REQUEST 事件
		deviceFingerprint := c.GetHeader("X-Device-Fingerprint")
		sessionID := c.GetHeader("X-Session-ID")

		pagePath := c.Request.URL.Path
		if decoded, err := url.QueryUnescape(pagePath); err == nil {
			pagePath = decoded
		}

		ua := c.Request.UserAgent()
		platform, _ := extractPlatformFromUA(ua)

		event := &UnpartitionedTrackEvent{
			EventType:        "REQUEST",
			PagePath:         pagePath,
			UserAgent:        ua,
			IPAddress:        c.ClientIP(),
			CreatedAt:        time.Now().In(chinaLocation),
			CustomProperties: createCustomProperties(c.Request.Method, c.Request.URL.RawQuery),
			UserID:           deviceFingerprint,
			SessionID:        sessionID,
			Referrer:         c.Request.Referer(),
			Platform:         platform,
			DeviceType:       extractDeviceTypeFromUA(ua),
		}

		if event.UserID == "" {
			event.UserID = "auto_generated_" + c.ClientIP()
		}
		if event.SessionID == "" {
			event.SessionID = "auto_" + time.Now().Format("20060102150405") + "_" + c.ClientIP()
		}

		log.Printf("自动跟踪请求: method=%s, path=%s, user_id=%s, session_id=%s",
			c.Request.Method, event.PagePath, event.UserID, event.SessionID)

		ts.TrackUnpartitionedEvent(event)
		c.Next()
	}
}

func shouldSkipRequest(path string) bool {
	staticExts := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif",
		".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot",
	}
	for _, ext := range staticExts {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	staticDirs := []string{
		"/static/", "/assets/", "/images/", "/css/", "/js/", "/fonts/",
	}
	for _, dir := range staticDirs {
		if strings.HasPrefix(strings.ToLower(path), dir) {
			return true
		}
	}
	return false
}

func createCustomProperties(method string, query string) string {
	props := map[string]interface{}{
		"auto_tracked": true,
		"method":       method,
	}
	if query != "" {
		props["has_query"] = true
	}
	jsonData, err := json.Marshal(props)
	if err != nil {
		return `{"auto_tracked":true}`
	}
	return string(jsonData)
}
