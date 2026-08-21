package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Token 配置常量
const (
	TokenCookieName = "mblog_token"
	TokenExpiration = 24 * time.Hour
	tokenIssuer     = "mblog-backend"
)

// getAdminUser 从环境变量获取管理员用户名
func getAdminUser() string {
	return os.Getenv("ADMIN_USER")
}

// getAdminPassword 从环境变量获取管理员密码
func getAdminPassword() string {
	return os.Getenv("ADMIN_PASSWORD")
}

// getJWTSecret 从环境变量获取 JWT 密钥
func getJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

func cookieSecure() bool {
	value := os.Getenv("COOKIE_SECURE")
	if value == "" {
		return true
	}
	secure, err := strconv.ParseBool(value)
	return err == nil && secure
}

// Claims 自定义 JWT Claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// GenerateToken 生成 JWT Token
func GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(TokenExpiration)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    tokenIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// ParseToken 解析并验证 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(getJWTSecret()), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(tokenIssuer), jwt.WithExpirationRequired())

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Login 处理登录请求
func Login(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请输入用户名和密码",
		})
		return
	}

	// 验证凭据
	validUser := subtle.ConstantTimeCompare([]byte(req.Username), []byte(getAdminUser())) == 1
	validPassword := subtle.ConstantTimeCompare([]byte(req.Password), []byte(getAdminPassword())) == 1
	if validUser && validPassword {
		// 生成 JWT
		tokenString, err := GenerateToken(req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "生成令牌失败",
			})
			return
		}

		// 设置 HttpOnly Cookie
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			TokenCookieName,
			tokenString,
			int(TokenExpiration.Seconds()),
			"/",
			"",
			cookieSecure(),
			true, // HttpOnly
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
	// 清除 Cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(TokenCookieName, "", -1, "/", "", cookieSecure(), true)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已登出",
	})
}

// CheckAuth 检查登录状态
func CheckAuth(c *gin.Context) {
	tokenString, err := c.Cookie(TokenCookieName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}

	claims, err := ParseToken(tokenString)
	if err != nil || subtle.ConstantTimeCompare([]byte(claims.Username), []byte(getAdminUser())) != 1 {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"username":      claims.Username,
	})
}

// RequireAuth 认证中间件 - 保护需要登录的路由
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(TokenCookieName)
		if err != nil {
			handleUnauthorized(c)
			return
		}

		claims, err := ParseToken(tokenString)
		if err != nil || subtle.ConstantTimeCompare([]byte(claims.Username), []byte(getAdminUser())) != 1 {
			handleUnauthorized(c)
			return
		}

		// 将用户信息存入上下文
		c.Set("username", claims.Username)
		c.Next()
	}
}

func handleUnauthorized(c *gin.Context) {
	// 如果是页面请求，跳转到登录页
	acceptsHTML := strings.Contains(strings.ToLower(c.GetHeader("Accept")), "text/html")
	if acceptsHTML || c.Request.URL.Path == "/admin" || c.Request.URL.Path == "/admin/" ||
		c.Request.URL.Path == "/analytics" {
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
}
