package server

import (
	"database/sql"
	"log"
	"time"

	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/router"
	"blog/pkg/comments"
	"blog/pkg/filemanager"
	"blog/pkg/tracking"

	"github.com/gin-gonic/gin"
)

// Server 应用服务器
type Server struct {
	config *config.Config
	db     *sql.DB
	engine *gin.Engine
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
	trackingService := tracking.NewTrackingService(db)
	analyticsService := tracking.NewAnalyticsService(db)

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
	commentService := comments.NewCommentService(db)

	// 初始化文件管理器
	if err := filemanager.Init(); err != nil {
		db.Close()
		return nil, err
	}

	// 设置路由
	engine := router.SetupRouter(trackingService, analyticsService, commentService)

	return &Server{
		config: cfg,
		db:     db,
		engine: engine,
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("服务器启动在端口: %s", s.config.Server.Port)
	return s.engine.Run(":" + s.config.Server.Port)
}

// Close 关闭服务器资源
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
