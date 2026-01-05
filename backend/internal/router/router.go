package router

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"blog/pkg/comments"
	"blog/pkg/tracking"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置并返回配置好的 Gin 路由器
func SetupRouter(
	trackingService *tracking.TrackingService,
	analyticsService *tracking.AnalyticsService,
	commentService *comments.CommentService,
) *gin.Engine {
	r := gin.Default()

	// 注册全局中间件
	r.Use(middleware.CORS())
	r.Use(trackingService.TrackingMiddleware())

	// 创建处理器
	fileHandler := handler.NewFileHandler()
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	healthHandler := handler.NewHealthHandler()

	// ============================================
	// 公开路由（无需登录）
	// ============================================
	// 健康检查
	r.GET("/api/ping", healthHandler.Ping)

	// 登录页面和认证 API
	r.StaticFile("/login", "/app/static/login.html")
	r.StaticFile("/login/", "/app/static/login.html")
	r.POST("/api/auth/login", middleware.Login)
	r.POST("/api/auth/logout", middleware.Logout)
	r.GET("/api/auth/check", middleware.CheckAuth)

	// 埋点和评论（由各自的包注册，公开访问）
	trackingService.RegisterHandlers(r)
	commentService.RegisterHandlers(r)

	// 静态资源
	r.StaticFile("/favicon.ico", "/app/static/favicon.ico")
	r.Static("/static", "/app/static")

	// ============================================
	// 受保护路由（需要登录）
	// ============================================
	admin := r.Group("")
	admin.Use(middleware.RequireAuth())
	{
		// 管理页面
		admin.GET("/admin", func(c *gin.Context) {
			c.File("/app/static/admin.html")
		})
		admin.GET("/admin/", func(c *gin.Context) {
			c.File("/app/static/admin.html")
		})
		admin.GET("/analytics", func(c *gin.Context) {
			c.File("/app/static/analytics.html")
		})

		// 文件管理 API
		admin.GET("/api/files", fileHandler.GetAllFiles)
		admin.GET("/api/files/*filename", fileHandler.GetFileContent)
		admin.POST("/api/files", fileHandler.SaveFile)
		admin.DELETE("/api/files/*filename", fileHandler.DeleteFile)
		admin.POST("/api/build", fileHandler.BuildSite)
		admin.POST("/api/upload", fileHandler.UploadFiles)

		// 统计分析 API
		admin.GET("/api/analytics", analyticsHandler.GetFullStats)
	}

	return r
}
