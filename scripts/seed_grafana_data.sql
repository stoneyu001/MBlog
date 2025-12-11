-- ============================================
-- MBlog 模拟数据生成脚本
-- ============================================
-- 用途: 为Grafana Dashboard生成演示数据
-- 标识: 所有模拟数据的 user_id 以 'seed_user_' 开头
-- 删除: 见文件末尾的清理脚本
-- ============================================

DO $$
DECLARE
  day_offset INT;
  page_paths TEXT[] := ARRAY[
    '/',
    '/tech/post_1_数据库优化.html',
    '/tech/post_2_前端优化.html',
    '/tech/post_10_前端优化.html',
    '/tech/post_11_微服务.html',
    '/tech/post_13_安全架构.html',
    '/life/睡眠.html',
    '/life/文明6.html',
    '/life/你好我的神.html',
    '/life/2024补文推荐.html'
  ];
  platforms TEXT[] := ARRAY[
    'Windows/Chrome', 
    'Windows/Edge', 
    'macOS/Safari', 
    'macOS/Chrome',
    'Linux/Firefox',
    'Android/Chrome',
    'iOS/Safari'
  ];
  event_types TEXT[] := ARRAY['PAGEVIEW', 'CLICK'];
  page_path TEXT;
  platform TEXT;
  event_type TEXT;
  daily_pageviews INT;
  daily_clicks INT;
  i INT;
  random_hour INT;
  random_minute INT;
  random_second INT;
  event_timestamp TIMESTAMP;
  user_count INT := 50; -- 模拟50个不同用户
  random_user_id TEXT;
  random_duration INT;
BEGIN
  RAISE NOTICE 'Starting seed data generation...';
  
  -- 生成过去30天的数据
  FOR day_offset IN 0..29 LOOP
    -- 每天的PAGEVIEW数量: 20-100条
    daily_pageviews := 20 + floor(random() * 80)::INT;
    
    -- 每天的CLICK数量: PAGEVIEW的10-30%
    daily_clicks := floor(daily_pageviews * (0.1 + random() * 0.2))::INT;
    
    -- RAISE NOTICE 'Generating day % : % PAGEVIEWS, % CLICKS', 30-day_offset, daily_pageviews, daily_clicks;
    
    -- 生成PAGEVIEW事件
    FOR i IN 1..daily_pageviews LOOP
      -- 随机选择页面、平台和用户
      page_path := page_paths[1 + floor(random() * array_length(page_paths, 1))::INT];
      platform := platforms[1 + floor(random() * array_length(platforms, 1))::INT];
      random_user_id := 'seed_user_' || (floor(random() * user_count))::TEXT;
      
      -- 生成随机时间（在当天的0-23点）
      random_hour := floor(random() * 24)::INT;
      random_minute := floor(random() * 60)::INT;
      random_second := floor(random() * 60)::INT;
      event_timestamp := (NOW() - INTERVAL '1 day' * day_offset)::DATE + 
                        (random_hour || ' hours')::INTERVAL + 
                        (random_minute || ' minutes')::INTERVAL +
                        (random_second || ' seconds')::INTERVAL;
      
      -- 停留时间: 10-600秒（大部分在1-5分钟）
      random_duration := 10 + floor(random() * 590)::INT;
      
      INSERT INTO track_event (
        session_id,
        user_id,
        event_type,
        page_path,
        platform,
        created_at,
        event_duration,
        user_agent,
        ip_address,
        metadata,
        custom_properties,
        device_info,
        referrer,
        element_path
      ) VALUES (
        'seed_session_' || gen_random_uuid()::TEXT,  -- ← 标识1: session_id前缀
        random_user_id,                              -- ← 标识2: user_id前缀
        'PAGEVIEW',
        page_path,
        platform,
        event_timestamp,
        random_duration,
        'Mozilla/5.0 (Seed Data Generator)',
        '127.0.0.' || (1 + floor(random() * 254))::TEXT,
        jsonb_build_object(
          'seed_data', true,                         -- ← 标识3: metadata标记
          'title', 'StoneYu Blog',
          'generated_at', NOW()::TEXT
        ),
        '{}',
        '{}',
        '',
        ''
      );
    END LOOP;
    
    -- 生成CLICK事件
    FOR i IN 1..daily_clicks LOOP
      page_path := page_paths[1 + floor(random() * array_length(page_paths, 1))::INT];
      platform := platforms[1 + floor(random() * array_length(platforms, 1))::INT];
      random_user_id := 'seed_user_' || (floor(random() * user_count))::TEXT;
      
      random_hour := floor(random() * 24)::INT;
      random_minute := floor(random() * 60)::INT;
      random_second := floor(random() * 60)::INT;
      event_timestamp := (NOW() - INTERVAL '1 day' * day_offset)::DATE + 
                        (random_hour || ' hours')::INTERVAL + 
                        (random_minute || ' minutes')::INTERVAL +
                        (random_second || ' seconds')::INTERVAL;
      
      INSERT INTO track_event (
        session_id,
        user_id,
        event_type,
        page_path,
        element_path,
        platform,
        created_at,
        event_duration,
        user_agent,
        ip_address,
        metadata,
        custom_properties,
        device_info,
        referrer
      ) VALUES (
        'seed_session_' || gen_random_uuid()::TEXT,
        random_user_id,
        'CLICK',
        page_path,
        'a.nav-link > span',
        platform,
        event_timestamp,
        0,
        'Mozilla/5.0 (Seed Data Generator)',
        '127.0.0.' || (1 + floor(random() * 254))::TEXT,
        jsonb_build_object(
          'seed_data', true,
          'text', 'Example Link',
          'generated_at', NOW()::TEXT
        ),
        '{}',
        '{}',
        '{}'
      );
    END LOOP;
  END LOOP;  -- ← 缺失的循环结束
  
  RAISE NOTICE 'Seed data generation completed!';
END $$;

-- 查看生成的数据统计
SELECT 
  'Seed Data Statistics' as info,
  COUNT(*) as total,
  SUM(CASE WHEN event_type = 'PAGEVIEW' THEN 1 ELSE 0 END) as pageviews,
  SUM(CASE WHEN event_type = 'CLICK' THEN 1 ELSE 0 END) as clicks,
  COUNT(DISTINCT user_id) as unique_users,
  MIN(created_at) as earliest,
  MAX(created_at) as latest
FROM track_event 
WHERE user_id LIKE 'seed_user_%';

-- ============================================
-- 清理脚本 (删除所有模拟数据)
-- ============================================
-- 使用方法:
-- docker-compose exec db psql -U postgres -d blog_db -f /path/to/cleanup_seed_data.sql
-- 
-- 或者直接执行以下SQL:
-- 
-- DELETE FROM track_event WHERE user_id LIKE 'seed_user_%';
-- 
-- 验证删除:
-- SELECT COUNT(*) FROM track_event WHERE user_id LIKE 'seed_user_%';
-- (应该返回 0)
-- ============================================
