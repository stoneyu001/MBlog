package tracking

import (
	"database/sql"
	"log"
)

// InitSchema 初始化数据库表结构
func InitSchema(db *sql.DB) error {
	log.Println("正在检查并初始化埋点数据库表结构...")

	// 1. 设置客户端编码
	_, err := db.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		return err
	}

	// 2. 创建 track_event 表
	// 注意：移除了特定的 COLLATE "zh-Hans-CN-x-icu"，以确保在不同 PostgreSQL 环境（如 Alpine）下的兼容性
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS track_event (
		id SERIAL PRIMARY KEY,
		session_id VARCHAR(100),
		user_id VARCHAR(100),
		event_type VARCHAR(50) NOT NULL,
		element_path TEXT,
		page_path TEXT,
		referrer TEXT,
		metadata JSONB DEFAULT '{}'::jsonb,
		user_agent TEXT,
		ip_address VARCHAR(50),
		created_at TIMESTAMP NOT NULL,
		custom_properties JSONB DEFAULT '{}'::jsonb,
		platform VARCHAR(20),
		device_info JSONB DEFAULT '{}'::jsonb,
		event_duration INTEGER DEFAULT 0,
		device_id VARCHAR(100),
		version VARCHAR(20),
		device_type VARCHAR(50)
	);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	// 3. 数据库迁移：为已存在的表添加缺失的列
	migrations := []string{
		"ALTER TABLE track_event ADD COLUMN IF NOT EXISTS device_type VARCHAR(50)",
	}
	for _, migrationSQL := range migrations {
		if _, err := db.Exec(migrationSQL); err != nil {
			log.Printf("迁移执行失败 (非致命): %v, SQL: %s", err, migrationSQL)
		}
	}

	// 4. 创建索引
	indices := []string{
		"CREATE INDEX IF NOT EXISTS idx_track_event_created_at ON track_event(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_event_type ON track_event(event_type)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_session_id ON track_event(session_id)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_user_id ON track_event(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_platform ON track_event(platform)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_metadata ON track_event USING gin (metadata)",
		"CREATE INDEX IF NOT EXISTS idx_track_event_custom_properties ON track_event USING gin (custom_properties)",
	}

	for _, indexSQL := range indices {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("创建索引失败 (非致命): %v, SQL: %s", err, indexSQL)
		}
	}

	log.Println("埋点数据库表结构初始化完成")
	return nil
}
