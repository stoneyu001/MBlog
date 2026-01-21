# MBlog

基于 **Go + VitePress + PostgreSQL** 的个人博客系统，支持 Docker 一键部署。

## 快速开始

### Linux / macOS

```bash
git clone https://github.com/stoneyu001/MBlog.git
cd MBlog
docker compose up -d
```

### Windows (PowerShell)

```powershell
git clone https://github.com/stoneyu001/MBlog.git
cd MBlog
docker-compose up -d
```

访问地址：
- 博客首页：`http://localhost`
- 管理后台：`http://localhost/admin`（账号：`admin` / `admin123`）

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | VitePress + Vue 3 |
| 后端 | Go (Gin) |
| 数据库 | PostgreSQL |
| 部署 | Docker Compose |

## 核心功能

- 📝 Markdown 文章管理
- 📊 实时访问统计
- 💬 评论系统
- 🔐 管理员认证

## 常用命令

### Linux / macOS

```bash
docker compose up -d          # 启动
docker compose down           # 停止
docker compose logs -f        # 查看日志
docker compose up -d --build  # 重新构建
```

### Windows

```powershell
docker-compose up -d          # 启动
docker-compose down           # 停止
docker-compose logs -f        # 查看日志
docker-compose up -d --build  # 重新构建
```

## 配置

1. 复制 `.env.example` 为 `.env`
2. 修改数据库密码和端口
3. **HTTPS 配置（生产环境推荐）**：
   - 在宿主机安装 Nginx 并申请 SSL 证书（推荐使用 Certbot）
   - 配置 Nginx 反向代理到 Docker 容器端口（默认 8080）
   - 示例 Nginx 配置：
     ```nginx
     server {
         listen 443 ssl;
         server_name your-domain.com;
         ssl_certificate /path/to/cert.pem;
         ssl_certificate_key /path/to/key.pem;
         location / {
             proxy_pass http://127.0.0.1:8080;
         }
     }
     ```

## License

MIT