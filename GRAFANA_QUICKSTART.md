# Grafana 已成功部署! 🎉

## 访问信息

- **Grafana URL**: http://localhost:3001
- **用户名**: admin
- **密码**: admin123

## 查看 Dashboard

1. 打开浏览器访问: http://localhost:3001
2. 使用上述账号密码登录
3. 点击左侧菜单 "Dashboards" 或直接访问: http://localhost:3001/d/mblog-analytics

## 当前 Dashboard 包含的图表

✅ **核心指标**:
- 总访问量 (近30天)
- 独立访客数

✅ **趋势分析**:
- 访问趋势图 (时间序列)

✅ **内容分析**:
- 热门页面 (Top 5)
- 平台分布 (Windows/Linux/Mac等)

## 服务状态

```bash
# 查看所有服务状态
docker-compose ps

# 查看 Grafana 日志
docker-compose logs -f grafana

# 重启 Grafana
docker-compose restart grafana

# 停止 Grafana
docker-compose stop grafana
```

## 下一步 (可选)

### 1. 添加更多图表
在 Grafana 中可以手动添加:
- 浏览器分布
- 事件类型分布  
- 访问时长分布
- 用户路径分析

### 2. 配置告警
在 Dashboard 中为任何指标设置告警规则，例如:
- 访问量异常下降
- 跳出率过高
- 错误率增加

### 3. 优化数据库 (推荐)
运行优化脚本添加索引:

```bash
# 连接到数据库
docker exec -it mblog_db psql -U postgres -d blog_db

# 执行以下 SQL
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_track_event_type_created 
ON track_event(event_type, created_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_track_event_user_created 
ON track_event(user_id, created_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_track_event_page_path 
ON track_event(page_path) WHERE event_type = 'PAGEVIEW';
```

## 故障排除

### Grafana 无法连接数据库
```bash
# 检查数据库是否运行
docker-compose ps db

# 查看 Grafana 错误日志
docker-compose logs grafana | grep -i error
```

### Dashboard 没有数据
确保:
1. 后端服务正在运行并采集数据
2. PostgreSQL 中 `track_event` 表有数据
3. 数据源配置正确 (检查 grafana/provisioning/datasources/postgres.yml)

---

**恭喜!** 您已成功将数据分析系统从自研代码迁移到 Grafana! 🚀
