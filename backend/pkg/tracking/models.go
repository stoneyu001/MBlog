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
	// 创建按日期分区的父表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS track_events (
		id BIGSERIAL,
		session_id VARCHAR(64) NOT NULL,
		user_id VARCHAR(64),
		event_type VARCHAR(32) NOT NULL,
		element_path TEXT,
		page_path TEXT,
		referrer TEXT,
		metadata JSONB,
		user_agent TEXT,
		ip_address VARCHAR(45),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (id, created_at)
	) PARTITION BY RANGE (created_at);
	`)

	if err != nil {
		return err
	}

	// 创建当天的分区
	today := time.Now().Format("20060102")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("20060102")

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS track_events_` + today + ` 
	PARTITION OF track_events
	FOR VALUES FROM ('` + today + ` 00:00:00') TO ('` + tomorrow + ` 00:00:00');
	`)

	if err != nil {
		return err
	}

	// 创建索引
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS idx_track_session ON track_events (session_id);
	CREATE INDEX IF NOT EXISTS idx_track_type ON track_events (event_type);
	CREATE INDEX IF NOT EXISTS idx_track_time ON track_events USING BRIN (created_at);
	`)

	return err
}
