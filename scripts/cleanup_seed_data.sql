-- ============================================
-- 清理模拟数据脚本
-- ============================================
-- 用途: 删除通过 seed_grafana_data.sql 生成的所有模拟数据
-- 安全性: 只删除标识为模拟数据的记录
-- ============================================

-- 显示将要删除的数据统计
SELECT 
  'Data to be deleted' as info,
  COUNT(*) as total,
  SUM(CASE WHEN event_type = 'PAGEVIEW' THEN 1 ELSE 0 END) as pageviews,
  SUM(CASE WHEN event_type = 'CLICK' THEN 1 ELSE 0 END) as clicks,
  MIN(created_at) as start_date,
  MAX(created_at) as end_date
FROM track_event 
WHERE user_id LIKE 'seed_user_%';

-- 删除所有模拟数据（通过user_id前缀识别）
DELETE FROM track_event 
WHERE user_id LIKE 'seed_user_%';

-- 验证删除结果
SELECT 
  'After deletion' as info,
  COUNT(*) as remaining_seed_data
FROM track_event 
WHERE user_id LIKE 'seed_user_%';

-- 显示剩余的真实数据统计
SELECT 
  'Real data statistics' as info,
  COUNT(*) as total,
  SUM(CASE WHEN event_type = 'PAGEVIEW' THEN 1 ELSE 0 END) as pageviews,
  SUM(CASE WHEN event_type = 'CLICK' THEN 1 ELSE 0 END) as clicks,
  SUM(CASE WHEN event_type = 'REQUEST' THEN 1 ELSE 0 END) as requests,
  COUNT(DISTINCT user_id) as unique_users
FROM track_event;
