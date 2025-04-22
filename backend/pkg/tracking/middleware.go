package tracking

import (
	"encoding/json"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TrackingMiddleware 创建一个Gin中间件用于记录API调用
func (ts *TrackingService) TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 先执行下一个中间件
		c.Next()

		// 判断是否需要跟踪此请求
		if shouldSkipTracking(c.Request.URL.Path) {
			return
		}

		// 生成或获取会话ID
		sessionID := getSessionID(c)

		// 计算处理时间
		duration := time.Since(startTime)

		// 准备元数据
		metadata := map[string]interface{}{
			"status_code": c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
			"method":      c.Request.Method,
			"query":       c.Request.URL.RawQuery,
			"client_ip":   getClientIP(c),
		}

		// 错误信息
		if len(c.Errors) > 0 {
			metadata["errors"] = c.Errors.String()
		}

		// 序列化元数据
		metadataJSON, _ := json.Marshal(metadata)

		// 创建跟踪事件
		event := &TrackEvent{
			SessionID: sessionID,
			UserID:    getUserID(c),
			EventType: "API_CALL",
			PagePath:  c.Request.URL.Path,
			Referrer:  c.Request.Referer(),
			Metadata:  string(metadataJSON),
			UserAgent: c.Request.UserAgent(),
			IPAddress: getClientIP(c),
			CreatedAt: time.Now(),
		}

		// 发送到跟踪服务
		ts.TrackEvent(event)
	}
}

// shouldSkipTracking 判断是否应该跳过某些路径的跟踪
func shouldSkipTracking(path string) bool {
	// 跳过静态资源和健康检查等路径
	skipPaths := []string{
		"/static/",
		"/favicon.ico",
		"/health",
		"/metrics",
	}

	for _, p := range skipPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// getSessionID 获取或生成会话ID
func getSessionID(c *gin.Context) string {
	// 从Cookie中获取
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		return sessionID
	}

	// 从请求头获取
	sessionID = c.GetHeader("X-Session-ID")
	if sessionID != "" {
		return sessionID
	}

	// 生成新的会话ID
	sessionID = uuid.New().String()

	// 设置Cookie
	c.SetCookie("session_id", sessionID, 86400*30, "/", "", false, true)

	return sessionID
}

// getUserID 获取用户ID
func getUserID(c *gin.Context) string {
	// 如果已认证用户，可以从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if exists {
		return userID.(string)
	}

	// 匿名用户使用会话ID作为标识
	return getSessionID(c)
}

// getClientIP 获取客户端真实IP
func getClientIP(c *gin.Context) string {
	// 检查代理头
	if xForwardedFor := c.GetHeader("X-Forwarded-For"); xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}

	// 检查其他头
	if clientIP := c.GetHeader("X-Real-IP"); clientIP != "" {
		if net.ParseIP(clientIP) != nil {
			return clientIP
		}
	}

	// 使用RemoteAddr
	clientIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && net.ParseIP(clientIP) != nil {
		return clientIP
	}

	return "unknown"
}
