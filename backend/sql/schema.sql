-- 不分区埋点事件表
CREATE TABLE IF NOT EXISTS track_events_unpartitioned (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100),
    user_id VARCHAR(100),
    event_type VARCHAR(50) NOT NULL,
    element_path TEXT,
    page_path TEXT,
    referrer TEXT,
    metadata JSONB DEFAULT '{}'::JSONB,
    user_agent TEXT,
    ip_address VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    custom_properties JSONB DEFAULT '{}'::JSONB,
    platform VARCHAR(20),
    device_info JSONB DEFAULT '{}'::JSONB,
    event_duration INTEGER DEFAULT 0,
    event_source VARCHAR(20),
    app_version VARCHAR(20)
);

-- 为常用查询创建索引
CREATE INDEX IF NOT EXISTS idx_track_events_unpart_created_at ON track_events_unpartitioned(created_at);
CREATE INDEX IF NOT EXISTS idx_track_events_unpart_event_type ON track_events_unpartitioned(event_type);
CREATE INDEX IF NOT EXISTS idx_track_events_unpart_session_id ON track_events_unpartitioned(session_id);
CREATE INDEX IF NOT EXISTS idx_track_events_unpart_platform ON track_events_unpartitioned(platform); 