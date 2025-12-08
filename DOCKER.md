# Docker 部署详细文档

本文档提供 MBlog 项目 Docker 部署的详细说明，适合需要深入了解容器化架构的开发者和运维人员。

## 📦 容器架构

### 服务组成

MBlog 使用 3 个 Docker 容器协同工作：

| 服务 | 容器名 | 镜像 | 端口映射 | 说明 |
|------|--------|------|----------|------|
| frontend | mblog_frontend | 自建（nginx:alpine） | 80:80 | 静态文件服务 |
| backend | mblog_backend | 自建（alpine:latest） | 3000:3000 | Go API 服务 |
| db | mblog_db | postgres:15-alpine | 5432:5432 | PostgreSQL 数据库 |

### 网络架构

所有服务运行在自定义桥接网络 `mblog_network` 中，实现服务间隔离和内部通信。

```
┌─────────────────────────────────────────────────┐
│              Docker Network                     │
│           (mblog_network - bridge)              │
│                                                 │
│  ┌─────────────┐                               │
│  │  Frontend   │                               │
│  │  (Nginx)    │ ◄── HTTP Requests             │
│  └──────┬──────┘                               │
│         │                                       │
│         ▼                                       │
│  ┌─────────────┐                               │
│  │   Backend   │                               │
│  │  (Go API)   │ ◄── API Calls                 │
│  └──────┬──────┘                               │
│         │                                       │
│         ▼                                       │
│  ┌─────────────┐                               │
│  │  Database   │                               │
│  │ (PostgreSQL)│ ◄── SQL Queries               │
│  └─────────────┘                               │
└─────────────────────────────────────────────────┘
```

## 🏗️ Dockerfile 详解

### Backend Dockerfile

**多阶段构建策略**：

#### 第一阶段：构建 (Builder)

```dockerfile
FROM golang:1.24.0-alpine AS builder
```

- **基础镜像**：`golang:1.24.0-alpine`（约 300MB）
- **目的**：编译 Go 代码
- **优化**：
  - 使用 GOPROXY 加速依赖下载
  - 分离 `go.mod`/`go.sum` 和源码复制，利用 Docker 层缓存
  - `go mod download` 单独一层，依赖不变时可复用
  - 编译参数 `-ldflags="-w -s"` 去除调试信息，减小二进制体积

#### 第二阶段：运行 (Runtime)

```dockerfile
FROM alpine:latest
```

- **基础镜像**：`alpine:latest`（约 5MB）
- **最终镜像大小**：约 20-30MB（vs 单阶段构建 300MB+）
- **安全性**：
  - 非 root 用户运行（`appuser`，UID 1000）
  - 最小化攻击面（只包含必要的运行时依赖）
- **运行时依赖**：
  - `ca-certificates`：HTTPS 请求需要
  - `tzdata`：时区支持
  - `wget`：健康检查使用

**构建优化对比**：

| 优化项 | 优化前 | 优化后 | 提升 |
|--------|--------|--------|------|
| 镜像体积 | ~300MB | ~25MB | 92% ↓ |
| 构建速度（依赖不变） | 3-5分钟 | 30秒 | 83% ↑ |
| 安全风险 | 高（root用户） | 低（非root） | - |

### Frontend Dockerfile

**多阶段构建策略**：

#### 第一阶段：构建 VitePress

```dockerfile
FROM node:20-alpine AS builder
```

- **功能**：编译 VitePress 静态文件
- **优化**：
  - 使用国内 npm 镜像源加速
  - `npm ci --only=production` 精确安装生产依赖
  - 分离 `package.json` 和源码复制

#### 第二阶段：Nginx 服务

```dockerfile
FROM nginx:alpine
```

- **最终镜像大小**：约 40MB
- **包含**：编译后的静态文件 + Nginx 配置
- **不包含**：Node.js 运行时、node_modules

## 🔄 服务依赖与启动顺序

### 依赖链

```
db (健康检查) ──> backend (健康检查) ──> frontend
```

### 健康检查机制

#### 数据库健康检查

```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres -d blog_db"]
  interval: 10s      # 每 10 秒检查一次
  timeout: 5s        # 超时时间 5 秒
  retries: 5         # 失败 5 次后标记为 unhealthy
  start_period: 30s  # 启动后 30 秒内失败不计入 retries
```

#### 后端健康检查

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget --spider http://localhost:3000/api/ping"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 40s
```

#### 前端健康检查

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget --spider http://localhost/"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 10s
```

### depends_on 条件依赖

```yaml
backend:
  depends_on:
    db:
      condition: service_healthy  # 等待数据库健康检查通过
```

**启动时间线**：

```
t=0s    : 数据库启动
t=5s    : 数据库初始化
t=10s   : 数据库健康检查通过 ✓
t=10s   : 后端开始启动
t=15s   : 后端应用启动完成
t=50s   : 后端健康检查通过 ✓
t=50s   : 前端开始启动
t=52s   : Nginx 启动完成
t=60s   : 前端健康检查通过 ✓
```

## 💾 数据持久化

### 数据卷管理

```yaml
volumes:
  pg_data:
    driver: local
```

**存储位置**：
- **Windows**：`\\wsl$\docker-desktop-data\data\docker\volumes\mblog_pg_data`
- **Linux**：`/var/lib/docker/volumes/mblog_pg_data`
- **macOS**：`~/Library/Containers/com.docker.docker/Data/vms/0/`

### 卷操作

#### 查看卷

```bash
# 列出所有卷
docker volume ls

# 查看卷详情
docker volume inspect mblog_pg_data

# 查看卷占用空间
docker system df -v
```

#### 备份卷

```bash
# 方法 1：使用 pg_dump（推荐）
docker-compose exec db pg_dump -U postgres blog_db > backup.sql

# 方法 2：导出整个卷
docker run --rm -v mblog_pg_data:/data -v ${PWD}:/backup alpine \
  tar czf /backup/pg_data_backup.tar.gz -C /data .
```

#### 恢复卷

```bash
# 方法 1：从 SQL 文件恢复
docker-compose exec -T db psql -U postgres blog_db < backup.sql

# 方法 2：恢复整个卷
docker run --rm -v mblog_pg_data:/data -v ${PWD}:/backup alpine \
  tar xzf /backup/pg_data_backup.tar.gz -C /data
```

#### 迁移卷

```bash
# 迁移到另一台机器
# 1. 源机器导出
docker-compose exec db pg_dumpall -U postgres > full_backup.sql

# 2. 复制 full_backup.sql 到目标机器

# 3. 目标机器导入
docker-compose exec -T db psql -U postgres < full_backup.sql
```

## 🌐 网络配置

### 自定义网络

```yaml
networks:
  mblog_network:
    driver: bridge
```

**优势**：
- ✅ 服务间通过服务名通信（自动 DNS 解析）
- ✅ 与其他 Docker 网络隔离
- ✅ 可自定义 IP 段和子网掩码

### 服务间通信

容器内部通过**服务名**访问：

```go
// backend/main.go
dbHost := "db"  // 自动解析为数据库容器 IP
dbURL := "postgres://postgres:password@db:5432/blog_db"
```

### 网络调试

```bash
# 查看网络详情
docker network inspect mblog_mblog_network

# 测试服务间连通性
docker-compose exec backend ping db
docker-compose exec backend wget -O- http://db:5432

# 查看容器 IP
docker-compose exec backend ip addr show
```

## 🔧 高级配置

### 环境变量优先级

```
1. docker-compose.yml 中的 environment
2. .env 文件
3. Dockerfile 中的 ENV
4. 系统默认值
```

### 资源限制

在 `docker-compose.yml` 中添加资源限制：

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### 日志管理

配置日志驱动和大小限制：

```yaml
services:
  backend:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 重启策略

```yaml
restart: unless-stopped
```

| 策略 | 说明 |
|------|------|
| `no` | 不自动重启 |
| `always` | 总是重启 |
| `on-failure` | 仅失败时重启 |
| `unless-stopped` | 总是重启，除非手动停止 |

## 🐛 故障排查

### 常用调试命令

```bash
# 查看容器状态
docker-compose ps

# 查看资源使用
docker stats

# 进入容器 Shell
docker-compose exec backend sh
docker-compose exec db bash

# 查看容器详细信息
docker inspect mblog_backend

# 查看网络连接
docker-compose exec backend netstat -tuln
```

### 日志分析

```bash
# 查看实时日志
docker-compose logs -f

# 查看最近 100 行日志
docker-compose logs --tail=100

# 查看特定时间的日志
docker-compose logs --since="2025-12-08T10:00:00"

# 查看错误日志
docker-compose logs | grep -i error
```

### 性能监控

```bash
# 查看容器资源使用
docker stats mblog_backend mblog_frontend mblog_db

# 查看磁盘使用
docker system df

# 清理未使用的资源
docker system prune -a
```

## 🚀 生产环境部署

### 安全加固

1. **使用 secrets 管理敏感信息**

```yaml
services:
  backend:
    secrets:
      - db_password
secrets:
  db_password:
    file: ./secrets/db_password.txt
```

2. **限制容器权限**

```yaml
services:
  backend:
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    read_only: true
```

3. **使用私有镜像仓库**

```yaml
services:
  backend:
    image: registry.example.com/mblog-backend:latest
```

### HTTPS 配置

添加 Nginx SSL 配置：

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
    
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    
    # ... 其他配置
}
```

### 自动化部署

使用 Docker Compose + CI/CD：

```bash
# .github/workflows/deploy.yml
- name: Deploy to Production
  run: |
    docker-compose pull
    docker-compose up -d --no-deps --build backend
```

## 📊 监控和观测

### Prometheus + Grafana

添加监控服务：

```yaml
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    
  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    volumes:
      - grafana_data:/var/lib/grafana
```

### 健康检查端点

后端已提供 `/api/ping` 端点用于健康检查和监控。

## 🔄 更新和维护

### 更新镜像

```bash
# 拉取最新镜像
docker-compose pull

# 重新构建和启动
docker-compose up -d --build

# 清理旧镜像
docker image prune -a
```

### 版本管理

建议在 `docker-compose.yml` 中固定镜像版本：

```yaml
services:
  db:
    image: postgres:15.3-alpine  # 固定版本，避免意外升级
```

---

📝 **文档贡献**：如果您发现文档有误或需要补充，欢迎提交 PR！
