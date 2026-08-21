package router

import (
	"log"
	"os"
	"strings"
	"time"

	"blog/internal/comments"
	"blog/internal/handler"
	"blog/internal/middleware"
	"blog/internal/tracking"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置并返回配置好的 Gin 路由器
func SetupRouter(
	trackingService *tracking.TrackingService,
	analyticsService *tracking.AnalyticsService,
	commentService *comments.CommentService,
) *gin.Engine {
	r := gin.Default()

	trustedProxies := []string{"127.0.0.1", "172.16.0.0/12"}
	if configured := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); configured != "" {
		trustedProxies = strings.Split(configured, ",")
		for index := range trustedProxies {
			trustedProxies[index] = strings.TrimSpace(trustedProxies[index])
		}
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Printf("可信代理配置无效，将禁用代理转发头: %v", err)
		_ = r.SetTrustedProxies(nil)
	}

	// 创建处理器
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	healthHandler := handler.NewHealthHandler()

	// ============================================
	// 公开路由（无需登录）
	// ============================================
	// 健康检查
	r.GET("/api/ping", healthHandler.Ping)

	// 登录页面和认证 API
	r.GET("/login", middleware.NoStore(), func(c *gin.Context) { c.File("/app/static/login.html") })
	r.GET("/login/", middleware.NoStore(), func(c *gin.Context) { c.File("/app/static/login.html") })
	r.POST("/api/auth/login", middleware.NoStore(), middleware.RateLimit(5, 10*time.Minute), middleware.Login)
	r.POST("/api/auth/logout", middleware.NoStore(), middleware.Logout)
	r.GET("/api/auth/check", middleware.NoStore(), middleware.CheckAuth)

	// 埋点和评论（由各自的包注册，公开访问）
	trackingService.RegisterHandlers(r)
	commentService.RegisterHandlers(r)

	// 静态资源
	r.StaticFile("/favicon.ico", "/app/static/favicon.ico")

	// ============================================
	// 受保护路由（需要登录）
	// ============================================
	admin := r.Group("")
	admin.Use(middleware.RequireAuth())
	admin.Use(middleware.NoStore())
	{
		// 管理页面
		admin.GET("/admin", func(c *gin.Context) {
			c.File("/app/static/analytics.html")
		})
		admin.GET("/admin/", func(c *gin.Context) {
			c.File("/app/static/analytics.html")
		})
		admin.GET("/analytics", func(c *gin.Context) {
			c.File("/app/static/analytics.html")
		})

		// 统计分析 API
		admin.GET("/api/analytics", analyticsHandler.GetFullStats)
	}

	return r
}
