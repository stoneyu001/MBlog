package tracking

import (
	"time"
)

// UnpartitionedTrackEvent 表示不分区的埋点事件
type UnpartitionedTrackEvent struct {
	ID               int64     `json:"id"`
	SessionID        string    `json:"session_id"`        // 会话ID
	UserID           string    `json:"user_id"`           // 用户标识（匿名ID或登录ID）
	EventType        string    `json:"event_type"`        // 事件类型：PAGEVIEW, CLICK, API_CALL等
	ElementPath      string    `json:"element_path"`      // 元素路径（用于点击事件）
	PagePath         string    `json:"page_path"`         // 页面路径
	Referrer         string    `json:"referrer"`          // 来源页面
	Metadata         string    `json:"metadata"`          // JSON格式的额外数据
	UserAgent        string    `json:"user_agent"`        // 用户代理
	IPAddress        string    `json:"ip_address"`        // IP地址
	CreatedAt        time.Time `json:"created_at"`        // 创建时间
	CustomProperties string    `json:"custom_properties"` // JSON格式的自定义属性
	Platform         string    `json:"platform"`          // 平台：WEB, IOS, ANDROID等
	DeviceInfo       string    `json:"device_info"`       // JSON格式的设备信息
	EventDuration    int       `json:"event_duration"`    // 事件持续时间(毫秒)
	DeviceID         string    `json:"device_id"`         // 设备ID（与数据库表对齐）
	Version          string    `json:"version"`           // 应用版本（与数据库表对齐）
}
