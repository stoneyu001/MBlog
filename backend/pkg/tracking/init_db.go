package tracking

import (
	"database/sql"
	"log"
)

// InitDB 初始化数据库表结构
func InitDB(db *sql.DB) error {
	log.Println("初始化埋点数据库表...")

	// 创建不分区埋点表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS track_events_unpartitioned (
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
		event_source VARCHAR(20),
		app_version VARCHAR(20)
	)
	`)

	if err != nil {
		log.Printf("创建埋点表失败: %v", err)
		return err
	}

	// 创建索引
	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS idx_track_events_unpart_created_at ON track_events_unpartitioned(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_track_events_unpart_event_type ON track_events_unpartitioned(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_track_events_unpart_session_id ON track_events_unpartitioned(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_track_events_unpart_platform ON track_events_unpartitioned(platform)`,
	}

	for _, query := range indexQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("创建索引失败: %v", err)
			// 继续创建其他索引，而不是立即返回
		}
	}

	log.Println("埋点数据库表初始化完成")
	return nil
}
