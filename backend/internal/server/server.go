package server

import (
	"database/sql"
	"log"

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

	// 初始化跟踪服务
	trackingService := tracking.NewTrackingService(db)
	analyticsService := tracking.NewAnalyticsService(db)

	// 初始化评论服务
	commentService := comments.NewCommentService(db)
	if err := commentService.Init(); err != nil {
		db.Close()
		return nil, err
	}

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
