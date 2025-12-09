-- 设置客户端编码
SET client_encoding = 'UTF8';

-- 创建 track_event 表
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
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    custom_properties JSONB DEFAULT '{}'::jsonb,
    platform VARCHAR(20),
    device_info JSONB DEFAULT '{}'::jsonb,
    event_duration INTEGER DEFAULT 0,
    device_id VARCHAR(100),
    version VARCHAR(20)
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_track_event_created_at ON track_event(created_at);
CREATE INDEX IF NOT EXISTS idx_track_event_event_type ON track_event(event_type);
CREATE INDEX IF NOT EXISTS idx_track_event_session_id ON track_event(session_id);
CREATE INDEX IF NOT EXISTS idx_track_event_user_id ON track_event(user_id);
CREATE INDEX IF NOT EXISTS idx_track_event_type_created ON track_event(event_type, created_at);
CREATE INDEX IF NOT EXISTS idx_track_event_page_path ON track_event(page_path) WHERE event_type = 'PAGEVIEW';

-- 为 JSONB 字段添加 GIN 索引
CREATE INDEX IF NOT EXISTS idx_track_event_metadata ON track_event USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_track_event_custom_properties ON track_event USING gin (custom_properties);
CREATE INDEX IF NOT EXISTS idx_track_event_device_info ON track_event USING gin (device_info);
