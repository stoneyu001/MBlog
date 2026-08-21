package middleware

import "github.com/gin-gonic/gin"

// NoStore 防止登录态页面和管理接口被浏览器或中间代理缓存。
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}
