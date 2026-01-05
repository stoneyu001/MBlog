package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 默认管理员凭据 - 可通过环境变量修改
const (
	DefaultAdminUser     = "admin"
	DefaultAdminPassword = "admin123"
	SessionCookieName    = "mblog_session"
	SessionMaxAge        = 24 * time.Hour
)

// 简单的内存 session 存储
var sessions = make(map[string]time.Time)

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 处理登录请求
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请输入用户名和密码",
		})
		return
	}

	// 验证凭据
	if req.Username == DefaultAdminUser && req.Password == DefaultAdminPassword {
		// 生成 session ID
		sessionID := generateSessionID()
		sessions[sessionID] = time.Now().Add(SessionMaxAge)

		// 设置 cookie
		c.SetCookie(
			SessionCookieName,
			sessionID,
			int(SessionMaxAge.Seconds()),
			"/",
			"",
			false, // 生产环境建议设为 true (HTTPS)
			true,  // HttpOnly
		)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "登录成功",
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户名或密码错误",
		})
	}
}

// Logout 处理登出请求
func Logout(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		delete(sessions, sessionID)
	}

	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已登出",
	})
}

// CheckAuth 检查登录状态
func CheckAuth(c *gin.Context) {
	if isAuthenticated(c) {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"username":      DefaultAdminUser,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
		})
	}
}

// RequireAuth 认证中间件 - 保护需要登录的路由
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthenticated(c) {
			// 如果是页面请求，跳转到登录页
			if c.GetHeader("Accept") == "" || c.GetHeader("Accept") == "text/html" || 
			   c.Request.URL.Path == "/admin" || c.Request.URL.Path == "/admin/" {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			// API 请求返回 401
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请先登录",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// 检查是否已认证
func isAuthenticated(c *gin.Context) bool {
	sessionID, err := c.Cookie(SessionCookieName)
	if err != nil {
		return false
	}

	expiry, exists := sessions[sessionID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(expiry) {
		delete(sessions, sessionID)
		return false
	}

	return true
}

// 生成简单的 session ID
func generateSessionID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(16)
}

// 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
