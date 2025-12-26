package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	// 导入评论系统包
	"blog/pkg/comments"
	"blog/pkg/filemanager"
	"blog/pkg/tracking"

	// 导入跟踪包
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

// 文件操作请求结构
type FileRequest struct {
	Filename         string `json:"filename"`
	Content          string `json:"content"`
	OriginalFilename string `json:"originalFilename"` // 用于重命名检测
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
	dbUser := getEnv("DB_USER", "postgres")
	// 密码必须从环境变量获取，不提供默认值
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = os.Getenv("POSTGRES_PASSWORD")
		if dbPass == "" {
			log.Fatal("数据库密码未设置: 请在环境变量中设置 DB_PASSWORD 或 POSTGRES_PASSWORD")
		}
	}
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

	// 配置数据库连接池
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接验证失败:", err)
	}

	log.Printf("数据库连接池配置完成: 最大连接=%d, 空闲连接=%d, 生命周期=%v",
		25, 5, 5*time.Minute)

	// 初始化跟踪服务
	trackingService := tracking.NewTrackingService(db)

	// 初始化评论服务
	commentService := comments.NewCommentService(db)
	if err := commentService.Init(); err != nil {
		log.Fatal("初始化评论服务失败:", err)
	}

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

	// 注册评论API处理程序
	commentService.RegisterHandlers(r)

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
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
	r.GET("/api/files/*filename", func(c *gin.Context) {
		filename := c.Param("filename")
		// 移除路径参数前面的斜杠
		if len(filename) > 0 && filename[0] == '/' {
			filename = filename[1:]
		}
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

		// 检查是否需要重命名（删除旧文件）
		if req.OriginalFilename != "" && req.OriginalFilename != req.Filename {
			log.Printf("检测到重命名操作: %s -> %s", req.OriginalFilename, req.Filename)
			// 尝试删除旧文件
			// 注意：我们需要处理 OriginalFilename，因为它可能是绝对路径或相对路径
			// filemanager.DeleteFile 会自动处理前缀
			if err := filemanager.DeleteFile(req.OriginalFilename); err != nil {
				log.Printf("重命名时删除旧文件失败: %v", err)
				// 这里我们可以选择报错，或者继续保存新文件但保留旧文件
				// 为了数据安全，我们继续保存新文件，但记录错误
			}
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
	r.DELETE("/api/files/*filename", func(c *gin.Context) {
		filename := c.Param("filename")
		log.Printf("收到删除文件请求，原始路径: %s", filename)

		// 移除路径参数前面的斜杠
		if len(filename) > 0 && filename[0] == '/' {
			filename = filename[1:]
			log.Printf("处理后的文件路径: %s", filename)
		}

		// 尝试删除文件
		if err := filemanager.DeleteFile(filename); err != nil {
			log.Printf("删除文件失败: %v", err)
			c.JSON(500, gin.H{"error": fmt.Sprintf("删除失败: %v", err)})
			return
		}

		// 更新侧边栏配置
		if err := filemanager.UpdateSidebarConfig(); err != nil {
			log.Printf("更新侧边栏配置失败: %v", err)
		}

		log.Printf("文件删除成功: %s", filename)
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
