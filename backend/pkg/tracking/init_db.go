package tracking

// 数据库表结构参考文档
// 本文件仅作为了解数据库结构的参考，不包含实际执行代码，因为已手动创建

/*
-- 设置数据库默认字符集
SET client_encoding = 'UTF8';

-- 创建track_event表
CREATE TABLE IF NOT EXISTS track_event (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100) COLLATE "zh-Hans-CN-x-icu",
    user_id VARCHAR(100) COLLATE "zh-Hans-CN-x-icu",
    event_type VARCHAR(50) NOT NULL COLLATE "zh-Hans-CN-x-icu",
    element_path TEXT COLLATE "zh-Hans-CN-x-icu",
    page_path TEXT COLLATE "zh-Hans-CN-x-icu",
    referrer TEXT COLLATE "zh-Hans-CN-x-icu",
    metadata JSONB DEFAULT '{}'::jsonb,
    user_agent TEXT COLLATE "zh-Hans-CN-x-icu",
    ip_address VARCHAR(50) COLLATE "zh-Hans-CN-x-icu",
    created_at TIMESTAMP NOT NULL,
    custom_properties JSONB DEFAULT '{}'::jsonb,
    platform VARCHAR(20) COLLATE "zh-Hans-CN-x-icu",
    device_info JSONB DEFAULT '{}'::jsonb,
    event_duration INTEGER DEFAULT 0,
    device_id VARCHAR(100) COLLATE "zh-Hans-CN-x-icu",
    version VARCHAR(20) COLLATE "zh-Hans-CN-x-icu"
) WITH (
    OIDS = FALSE
) TABLESPACE pg_default;

-- 为现有表修改字符集
ALTER TABLE track_event ALTER COLUMN session_id TYPE VARCHAR(100) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN user_id TYPE VARCHAR(100) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN event_type TYPE VARCHAR(50) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN element_path TYPE TEXT COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN page_path TYPE TEXT COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN referrer TYPE TEXT COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN user_agent TYPE TEXT COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN ip_address TYPE VARCHAR(50) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN platform TYPE VARCHAR(20) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN device_id TYPE VARCHAR(100) COLLATE "zh-Hans-CN-x-icu";
ALTER TABLE track_event ALTER COLUMN version TYPE VARCHAR(20) COLLATE "zh-Hans-CN-x-icu";

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_track_event_created_at ON track_event(created_at);
CREATE INDEX IF NOT EXISTS idx_track_event_event_type ON track_event(event_type);
CREATE INDEX IF NOT EXISTS idx_track_event_session_id ON track_event(session_id);
CREATE INDEX IF NOT EXISTS idx_track_event_user_id ON track_event(user_id);
CREATE INDEX IF NOT EXISTS idx_track_event_platform ON track_event(platform);
CREATE INDEX IF NOT EXISTS idx_track_event_device_id ON track_event(device_id);

-- 为JSONB字段添加GIN索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_track_event_metadata ON track_event USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_track_event_custom_properties ON track_event USING gin (custom_properties);
CREATE INDEX IF NOT EXISTS idx_track_event_device_info ON track_event USING gin (device_info);
*/
