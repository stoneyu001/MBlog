package comments

import (
	"database/sql"
	"log"
)

// InitSchema 初始化评论系统数据库表
func InitSchema(db *sql.DB) error {
	log.Println("正在检查并初始化评论数据库表结构...")

	// 1. 设置客户端编码
	_, err := db.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		return err
	}

	// 2. 创建 comments 表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		article_id VARCHAR(100) NOT NULL,
		nickname VARCHAR(100) NOT NULL,
		email VARCHAR(100),
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		ip_address VARCHAR(50),
		status VARCHAR(20) DEFAULT 'approved',
		reply_to INTEGER DEFAULT NULL,
		user_agent TEXT
	);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	// 3. 创建索引
	indices := []string{
		"CREATE INDEX IF NOT EXISTS idx_comments_article_id ON comments(article_id)",
		"CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_comments_status ON comments(status)",
		"CREATE INDEX IF NOT EXISTS idx_comments_reply_to ON comments(reply_to)",
	}

	for _, indexSQL := range indices {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("创建评论索引失败 (非致命): %v, SQL: %s", err, indexSQL)
		}
	}

	log.Println("评论数据库表结构初始化完成")
	return nil
}
