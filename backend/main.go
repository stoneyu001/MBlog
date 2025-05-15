package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	// 设置全局默认时区为中国时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("加载中国时区失败: %v, 尝试使用UTC+8", err)
		loc = time.FixedZone("CST", 8*60*60)
	}
	time.Local = loc

	log.Printf("系统时区已设置为: %s, 当前时间: %s",
		time.Local.String(), time.Now().In(time.Local).Format("2006-01-02 15:04:05"))

	// 初始化数据库连接（使用端口5432 - 容器内部端口）
	dbHost := getEnv("DB_HOST", "db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPass := getEnv("POSTGRES_PASSWORD", "4341289")
	dbName := getEnv("POSTGRES_DB", "blog_db")

	// 构建连接字符串
	// 注意：密码默认值为空字符串，需要在环境变量中设置
	dbURL := "postgres://" + dbUser
	if dbPass != "" {
		dbURL += ":" + dbPass
	}
	dbURL += "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable&timezone=Asia/Shanghai"

	log.Printf("数据库连接: %s", dbURL)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 初始化跟踪服务
	trackingService := tracking.NewTrackingService(db)

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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Device-Fingerprint, X-Session-ID")

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
		log.Printf("收到获取文件列表请求")
		files, err := filemanager.GetAllFiles()
		if err != nil {
			log.Printf("获取文件列表失败: %v", err)
			c.JSON(500, gin.H{"error": fmt.Sprintf("获取文件列表失败: %v", err)})
			return
		}
		log.Printf("成功获取文件列表，文件数量: %d", len(files))
		c.JSON(200, files)
	})

	// 2. 获取单个文件内容
	r.GET("/api/files/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		log.Printf("收到获取文件内容请求: %s", filename)
		content, err := filemanager.GetFileContent(filename)
		if err != nil {
			log.Printf("获取文件内容失败: %v", err)
			c.JSON(404, gin.H{"error": fmt.Sprintf("文件不存在或无法读取: %v", err)})
			return
		}
		log.Printf("成功获取文件内容: %s", filename)
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
	r.StaticFile("/admin", "/app/static/admin.html")
	r.StaticFile("/admin/", "/app/static/admin.html")
	r.Static("/static", "/app/static")

	// 启动服务
	r.Run(":3000")
}

// 获取当前工作目录
func getWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	return dir
}

// 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
