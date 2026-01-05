# MBlog - 现代化的博客管理系统

一个基于 Go + VitePress + PostgreSQL 的全栈博客系统，支持 Docker 一键部署，内置管理后台与访问统计。

## ✨ 特性

- 🚀 **开箱即用**：Docker Compose 一键启动，无需复杂配置。
- 📝 **文章管理**：支持 Markdown 文章的 CRUD 与批量导入。
- 🔐 **安全认证**：内置美观的管理员登录页面。
- 📊 **访问统计**：实时跟踪页面访问量与用户行为。
- 💬 **评论系统**：支持用户互动评论。

##  快速开始

### 1. 准备环境
确保已安装 [Docker](https://www.docker.com/) 和 [Docker Compose](https://docs.docker.com/compose/install/)。

### 2. 启动服务
```bash
# 克隆项目
git clone <your-repo-url>
cd MBlog

# 启动（首次启动会自动构建）
docker-compose up -d
```

### 3. 访问应用
- **博客首页**：`http://localhost`
- **管理后台**：`http://localhost/admin`
- **默认账号**：`admin` / `admin123`

## 📚 常用命令

| 命令 | 说明 |
|------|------|
| `docker-compose up -d` | 后台启动服务 |
| `docker-compose down` | 停止服务 |
| `docker-compose logs -f` | 查看实时日志 |
| `docker-compose up -d --build` | 重新构建并启动 |

## ⚙️ 进阶配置

- **修改密码**：在 `backend/internal/middleware/auth.go` 中修改默认凭据。
- **自定义端口**：复制 `.env.example` 为 `.env` 并修改 `FRONTEND_PORT` 等变量。
- **数据持久化**：数据库数据存储在 Docker 卷 `pg_data` 中，重启不丢失。

## 🔒 配置 HTTPS (SSL)

生产环境建议启用 HTTPS，以下是使用 Let's Encrypt 免费证书的配置步骤：

### 1. 修改 Docker 端口

编辑 `.env` 文件，将前端端口改为 `8080`（避免与宿主机 Nginx 冲突）：
```bash
FRONTEND_PORT=8080
```

重启 Docker 容器：
```bash
docker-compose down && docker-compose up -d
```

### 2. 安装宿主机 Nginx

```bash
sudo apt update && sudo apt install nginx -y
```

### 3. 申请 SSL 证书

```bash
# 安装 Certbot
sudo apt install certbot -y

# 申请证书（替换 your-domain.com 为你的域名）
sudo certbot certonly --standalone -d your-domain.com -d www.your-domain.com
```

### 4. 配置 Nginx 反向代理

创建配置文件 `/etc/nginx/conf.d/blog.conf`：

```nginx
# HTTP 跳转 HTTPS
server {
    listen 80;
    server_name your-domain.com www.your-domain.com;
    return 301 https://$host$request_uri;
}

# HTTPS 配置
server {
    listen 443 ssl;
    server_name your-domain.com www.your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 5. 启用配置

```bash
sudo nginx -t && sudo systemctl reload nginx
```

---
💡 **提示**：生产环境部署建议修改 `.env` 中的数据库密码并开启 HTTPS。