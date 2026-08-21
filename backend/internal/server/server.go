package server

import (
	"database/sql"
	"log"
	"time"

	"blog/internal/comments"
	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/repository"
	"blog/internal/router"
	"blog/internal/tracking"

	"github.com/gin-gonic/gin"
)

// Server 应用服务器
type Server struct {
	config          *config.Config
	db              *sql.DB
	engine          *gin.Engine
	trackingService *tracking.TrackingService
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.Config) (*Server, error) {
	// 初始化数据库连接
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return nil, err
	}

	// 初始化跟踪服务（带重试机制）
	maxRetries := 5
	var initErr error
	for i := 0; i < maxRetries; i++ {
		if initErr = tracking.InitSchema(db); initErr != nil {
			log.Printf("初始化埋点数据库失败 (尝试 %d/%d): %v", i+1, maxRetries, initErr)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	if initErr != nil {
		log.Printf("埋点数据库初始化最终失败: %v", initErr)
	}
	trackingRepo := repository.NewTrackingRepo(db)
	trackingService := tracking.NewTrackingService(trackingRepo)
	analyticsService := tracking.NewAnalyticsService(trackingRepo)

	// 初始化评论服务（带重试机制）
	for i := 0; i < maxRetries; i++ {
		if initErr = comments.InitSchema(db); initErr != nil {
			log.Printf("初始化评论数据库失败 (尝试 %d/%d): %v", i+1, maxRetries, initErr)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	if initErr != nil {
		log.Printf("评论数据库初始化最终失败: %v", initErr)
	}
	commentRepo := repository.NewCommentRepo(db)
	commentService := comments.NewCommentService(commentRepo)

	// 设置路由
	engine := router.SetupRouter(trackingService, analyticsService, commentService)

	return &Server{
		config:          cfg,
		db:              db,
		engine:          engine,
		trackingService: trackingService,
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("服务器启动在端口: %s", s.config.Server.Port)
	return s.engine.Run(":" + s.config.Server.Port)
}

// Close 关闭服务器资源
func (s *Server) Close() error {
	if s.trackingService != nil {
		s.trackingService.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
