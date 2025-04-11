package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // 必须添加的PostgreSQL驱动
)

type Article struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func main() {
	// 初始化数据库连接（直接简单连接）
	db, err := sql.Open("postgres",
		"postgres://postgres:4341289@db:5432/blog_db?sslmode=disable")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 基础路由测试
	r := gin.Default()
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 新增文章路由
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
	// 托管管理后台静态文件（创建backend/static目录）
	r.Static("/admin", "./static")

	// 新增获取全部文章接口（用于前端展示）
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
	// 启动服务
	r.Run(":3000")
}
