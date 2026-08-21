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
		article_id VARCHAR(200) NOT NULL,
		nickname VARCHAR(100) NOT NULL,
		email VARCHAR(100),
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		ip_address VARCHAR(50),
		status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
		reply_to INTEGER DEFAULT NULL REFERENCES comments(id) ON DELETE CASCADE,
		user_agent TEXT
	);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	// 3. 对已有数据库执行向后兼容迁移。约束使用 NOT VALID，避免历史脏数据阻断启动。
	migrations := []string{
		"ALTER TABLE comments ALTER COLUMN article_id TYPE VARCHAR(200)",
		"UPDATE comments SET status = 'approved' WHERE status IS NULL OR status = ''",
		"ALTER TABLE comments ALTER COLUMN status SET DEFAULT 'pending'",
		"ALTER TABLE comments ALTER COLUMN status SET NOT NULL",
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'comments_status_check') THEN
				ALTER TABLE comments ADD CONSTRAINT comments_status_check
				CHECK (status IN ('pending', 'approved', 'rejected')) NOT VALID;
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'comments_reply_to_fkey') THEN
				ALTER TABLE comments ADD CONSTRAINT comments_reply_to_fkey
				FOREIGN KEY (reply_to) REFERENCES comments(id) ON DELETE CASCADE NOT VALID;
			END IF;
		END $$`,
	}
	for _, migrationSQL := range migrations {
		if _, err := db.Exec(migrationSQL); err != nil {
			return err
		}
	}

	// 4. 创建索引
	indices := []string{
		"CREATE INDEX IF NOT EXISTS idx_comments_article_id ON comments(article_id)",
		"CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_comments_status ON comments(status)",
		"CREATE INDEX IF NOT EXISTS idx_comments_reply_to ON comments(reply_to)",
		"CREATE INDEX IF NOT EXISTS idx_comments_article_status_created ON comments(article_id, status, created_at DESC)",
	}

	for _, indexSQL := range indices {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("创建评论索引失败 (非致命): %v, SQL: %s", err, indexSQL)
		}
	}

	log.Println("评论数据库表结构初始化完成")
	return nil
}
