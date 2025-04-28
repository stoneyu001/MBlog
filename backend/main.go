package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"blog/pkg/filemanager"
	"blog/pkg/tracking" // 导入跟踪包

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // 必须添加的PostgreSQL驱动
)

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type Article struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// 文件操作请求结构
type FileRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func main() {
	// 初始化数据库连接（使用端口5432 - 容器内部端口）
	dbHost := getEnv("DB_HOST", "db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPass := getEnv("POSTGRES_PASSWORD", "")
	dbName := getEnv("POSTGRES_DB", "blog_db")

	// 构建连接字符串
	// 注意：密码默认值为空字符串，需要在环境变量中设置
	dbURL := "postgres://" + dbUser
	if dbPass != "" {
		dbURL += ":" + dbPass
	}
	dbURL += "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 初始化跟踪服务
	trackingService := tracking.NewTrackingService(db)

	// 初始化分析服务
	analyticsService := tracking.NewAnalyticsService(db)

	// 初始化文件管理器
	if err := filemanager.Init(); err != nil {
		log.Fatal("初始化文件管理器失败:", err)
	}

	// 基础路由测试
	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 添加跟踪中间件
	r.Use(trackingService.TrackingMiddleware())

	// 注册跟踪API处理程序
	trackingService.RegisterHandlers(r)

	// 注册分析API处理程序
	analyticsService.RegisterHandlers(r.Group("/api/tracking"))

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 旧的文章路由（保留用于兼容）
	r.POST("/articles", func(c *gin.Context) {
		var article Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := db.QueryRow(
			"INSERT INTO articles(title, content) VALUES($1, $2) RETURNING id",
			article.Title, article.Content,
		).Scan(&article.ID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
			return
		}

		c.JSON(http.StatusCreated, article)
	})

	// 文章列表API（保留用于兼容）
	r.GET("/api/articles", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, title, created_at FROM articles")
		if err != nil {
			c.JSON(500, gin.H{"error": "查询失败"})
			return
		}
		defer rows.Close()

		var articles []Article
		for rows.Next() {
			var a Article
			if err := rows.Scan(&a.ID, &a.Title, &a.CreatedAt); err != nil {
				c.JSON(500, gin.H{"error": "数据扫描失败"})
				return
			}
			articles = append(articles, a)
		}

		c.JSON(200, articles)
	})

	// 文件管理相关API
	// 1. 获取所有文件
	r.GET("/api/files", func(c *gin.Context) {
		files, err := filemanager.GetAllFiles()
		if err != nil {
			c.JSON(500, gin.H{"error": "获取文件列表失败"})
			return
		}
		c.JSON(200, files)
	})

	// 2. 获取单个文件内容
	r.GET("/api/files/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		content, err := filemanager.GetFileContent(filename)
		if err != nil {
			c.JSON(404, gin.H{"error": "文件不存在或无法读取"})
			return
		}
		c.String(200, content)
	})

	// 3. 保存文件
	r.POST("/api/files", func(c *gin.Context) {
		var req FileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "请求参数错误"})
			return
		}

		if err := filemanager.SaveFile(req.Filename, req.Content); err != nil {
			c.JSON(500, gin.H{"error": "保存文件失败"})
			return
		}

		// 更新侧边栏配置
		if err := filemanager.UpdateSidebarConfig(); err != nil {
			log.Printf("更新侧边栏配置失败: %v", err)
		}

		c.JSON(200, gin.H{"status": "success"})
	})

	// 4. 删除文件
	r.DELETE("/api/files/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		if err := filemanager.DeleteFile(filename); err != nil {
			c.JSON(500, gin.H{"error": "删除失败"})
			return
		}

		// 更新侧边栏配置
		if err := filemanager.UpdateSidebarConfig(); err != nil {
			log.Printf("更新侧边栏配置失败: %v", err)
		}

		c.JSON(200, gin.H{"status": "success"})
	})

	// 5. 构建站点
	r.POST("/api/build", func(c *gin.Context) {
		if err := filemanager.BuildSite(); err != nil {
			c.JSON(500, gin.H{"error": "构建失败"})
			return
		}
		c.JSON(200, gin.H{"status": "success"})
	})

	// 托管管理界面
	r.StaticFile("/admin", "./static/index.html")
	r.StaticFile("/admin/", "./static/index.html")
	r.Static("/static", "./static")

	// 启动服务
	r.Run(":3000")
}
