package tracking

import (
	"database/sql"
	"time"
)

// TrackEvent 表示一个用户行为事件
type TrackEvent struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`   // 会话ID
	UserID      string    `json:"user_id"`      // 用户标识（匿名ID或登录ID）
	EventType   string    `json:"event_type"`   // 事件类型：PAGEVIEW, CLICK, API_CALL等
	ElementPath string    `json:"element_path"` // 元素路径（用于点击事件）
	PagePath    string    `json:"page_path"`    // 页面路径
	Referrer    string    `json:"referrer"`     // 来源页面
	Metadata    string    `json:"metadata"`     // JSON格式的额外数据
	UserAgent   string    `json:"user_agent"`   // 用户代理
	IPAddress   string    `json:"ip_address"`   // IP地址
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
}

// CreateTrackingTables 创建跟踪相关的数据库表
func CreateTrackingTables(db *sql.DB) error {
	// 创建跟踪事件表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS track_events (
		id BIGSERIAL PRIMARY KEY,
		session_id VARCHAR(64) NOT NULL,
		user_id VARCHAR(64),
		event_type VARCHAR(32) NOT NULL,
		element_path TEXT,
		page_path TEXT,
		referrer TEXT,
		metadata JSONB,
		user_agent TEXT,
		ip_address VARCHAR(45),
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- 创建时间索引用于范围查询
	CREATE INDEX IF NOT EXISTS idx_track_time ON track_events (created_at);
	
	-- 创建联合索引用于常用查询
	CREATE INDEX IF NOT EXISTS idx_track_type_time ON track_events (event_type, created_at);
	CREATE INDEX IF NOT EXISTS idx_track_session_time ON track_events (session_id, created_at);
	
	-- 创建表达式索引用于按天统计
	CREATE INDEX IF NOT EXISTS idx_track_date ON track_events (DATE(created_at));
	`)

	return err
}
