# MBlog

基于 **Go + VitePress + PostgreSQL** 的个人博客系统，支持 Docker 一键部署。

## 🚀 快速部署到服务器

### 前提条件

- 一台 Linux 云服务器（Ubuntu/Debian）
- 域名已解析到服务器 IP（`www.stoneyu.fun`）
- 服务器已安装 Docker 和 Docker Compose

### 一键部署

```bash
# 1. 克隆项目
git clone https://github.com/stoneyu001/MBlog.git
cd MBlog

# 2. 配置环境变量
cp .env.example .env
nano .env  # 修改数据库密码

# 3. 一键部署（包括 Docker 构建、Nginx SSL 配置、证书申请、自动续签）
sudo bash deploy/deploy.sh
```

部署完成后访问：
- 🌐 博客首页：`https://www.stoneyu.fun`
- 🔧 管理后台：`https://www.stoneyu.fun/admin`（账号：`admin` / `admin123`）

### 更新部署

```bash
cd ~/MBlog
git pull
sudo bash deploy/deploy.sh --update
```

### 仅重启服务

```bash
cd ~/MBlog
sudo bash deploy/deploy.sh --restart
```

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

## 项目结构

```
MBlog/
├── backend/            # Go 后端 API
├── frontend/           # VitePress 前端
│   ├── docs/           # 博客内容与配置
│   └── nginx/          # 容器内 Nginx 配置
├── deploy/             # 部署相关文件
│   ├── deploy.sh           # 一键部署脚本
│   ├── nginx-host.conf     # 宿主机 Nginx SSL 配置
│   ├── renew-cert.sh       # 证书续签脚本
│   └── setup-auto-renew.sh # 自动续签配置
├── docker-compose.yml  # Docker 编排
├── .env.example        # 环境变量模板
└── README.md
```

## 架构说明

```
用户浏览器
    ↓ HTTPS (443)
宿主机 Nginx (SSL 终结)
    ↓ proxy_pass (8080)
Docker: frontend Nginx (静态文件 + 反向代理)
    ↓ proxy_pass (3000)
Docker: backend Go API
    ↓ (5432)
Docker: PostgreSQL
```

## 常用命令

```bash
# 部署相关
sudo bash deploy/deploy.sh              # 首次完整部署
sudo bash deploy/deploy.sh --update     # 更新代码后重新构建
sudo bash deploy/deploy.sh --restart    # 仅重启服务

# Docker 相关
docker compose logs -f           # 查看日志
docker compose ps                # 查看容器状态
docker compose down              # 停止所有服务
docker compose up -d --build     # 重新构建并启动

# SSL 相关
sudo certbot certificates        # 查看证书状态
sudo certbot renew --dry-run     # 测试续签
sudo systemctl status certbot-renew.timer  # 查看自动续签状态
```

## 配置

1. 复制 `.env.example` 为 `.env`
2. 修改数据库密码和端口
3. 运行 `sudo bash deploy/deploy.sh` 完成全部配置

## License

MIT