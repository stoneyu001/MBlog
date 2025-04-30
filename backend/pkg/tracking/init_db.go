package tracking

// 数据库表结构参考文档
// 本文件仅作为了解数据库结构的参考，不包含实际执行代码，因为已手动创建

// 不分区埋点表结构
/*
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
    event_duration INTEGER DEFAULT 0
)
*/

// 新的track_event表结构
/*
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
    version VARCHAR(20)
)
*/

// 新表的索引结构
/*
CREATE INDEX IF NOT EXISTS idx_track_event_created_at ON track_event(created_at)
CREATE INDEX IF NOT EXISTS idx_track_event_event_type ON track_event(event_type)
CREATE INDEX IF NOT EXISTS idx_track_event_session_id ON track_event(session_id)
CREATE INDEX IF NOT EXISTS idx_track_event_user_id ON track_event(user_id)
CREATE INDEX IF NOT EXISTS idx_track_event_platform ON track_event(platform)
CREATE INDEX IF NOT EXISTS idx_track_event_device_id ON track_event(device_id)
*/
