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

	// 注册中间件
	r.Use(middleware.CORS())
	r.Use(trackingService.TrackingMiddleware())

	// 创建处理器
	fileHandler := handler.NewFileHandler()
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	healthHandler := handler.NewHealthHandler()

	// 注册路由
	// 健康检查
	r.GET("/api/ping", healthHandler.Ping)

	// 文件管理
	r.GET("/api/files", fileHandler.GetAllFiles)
	r.GET("/api/files/*filename", fileHandler.GetFileContent)
	r.POST("/api/files", fileHandler.SaveFile)
	r.DELETE("/api/files/*filename", fileHandler.DeleteFile)
	r.POST("/api/build", fileHandler.BuildSite)
	r.POST("/api/upload", fileHandler.UploadFiles)

	// 统计分析
	r.GET("/api/analytics", analyticsHandler.GetFullStats)

	// 埋点和评论（由各自的包注册）
	trackingService.RegisterHandlers(r)
	commentService.RegisterHandlers(r)

	// 静态文件
	r.StaticFile("/admin", "/app/static/admin.html")
	r.StaticFile("/admin/", "/app/static/admin.html")
	r.StaticFile("/analytics", "/app/static/analytics.html")
	r.StaticFile("/favicon.ico", "/app/static/favicon.ico")
	r.Static("/static", "/app/static")

	return r
}
