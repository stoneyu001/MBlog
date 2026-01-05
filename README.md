# MBlog - 博客管理系统

一个现代化的博客管理系统，采用 Go + VitePress + PostgreSQL 技术栈，支持 Docker 一键部署。

## ✨ 特性

- 🚀 **开箱即用**：使用 Docker Compose 一键启动所有服务
- 📝 **文章管理**：提供完整的博客文章 CRUD 功能
- 💬 **评论系统**：内置评论功能，支持用户互动
- 📊 **访问统计**：实时跟踪文章访问量和用户行为
- 🎨 **现代化前端**：基于 VitePress 构建的文档站点
- 🔒 **安全可靠**：容器化部署，数据持久化存储

## 🔧 技术栈

- **后端**：Go 1.24 + Gin Framework
- **前端**：VitePress + Vue 3
- **数据库**：PostgreSQL 15
- **部署**：Docker + Docker Compose
- **Web 服务器**：Nginx

## 📋 环境要求

在开始之前，确保你的系统已安装：

- [Docker](https://www.docker.com/get-started) (20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (v2.0+)
- docker-desktop 默认安装在C盘要挺大的空间,请注意

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd MBlog
```

### 2. 配置环境变量

复制环境变量模板并修改配置：

```bash
# Windows PowerShell
Copy-Item .env.example .env

# Linux / macOS
cp .env.example .env
```

编辑 `.env` 文件，**务必修改数据库密码**：

```env
# 端口配置（可根据需要调整）
FRONTEND_PORT=80
BACKEND_PORT=3000
DB_PORT=5432

# 数据库配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password_here  # ⚠️ 请修改为强密码！
POSTGRES_DB=blog_db
```

### 3. 启动服务

```bash
# 前台启动（查看实时日志）
docker-compose up

# 或者后台启动
docker-compose up -d
```

首次启动会自动：
- 📦 下载所需的 Docker 镜像
- 🔨 构建前端和后端应用
- 🗄️ 初始化 PostgreSQL 数据库
- ✅ 启动所有服务

### 4. 配置管理后台密码

复制密码模板并设置登录凭据：

```bash
# Windows PowerShell
Copy-Item .htpasswd.example .htpasswd

# Linux / macOS
cp .htpasswd.example .htpasswd
```

默认凭据：
- **用户名**：`admin`
- **密码**：`admin123`

> ⚠️ **生产环境请务必修改密码！** 使用 `htpasswd -c .htpasswd admin` 命令重新生成。

### 5. 访问应用

服务启动成功后，可以通过以下地址访问：

- **前端网站**：http://localhost （端口 80）
- **管理界面**：http://localhost/admin（需要登录）
- **API 测试**：http://localhost/api/ping

## 📚 常用命令

```bash
# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs backend
docker-compose logs frontend
docker-compose logs db

# 实时跟踪日志
docker-compose logs -f

# 停止服务
docker-compose down

# 停止服务并删除数据卷（⚠️ 会删除数据库数据）
docker-compose down -v

# 重启服务
docker-compose restart

# 重新构建并启动
docker-compose up --build

# 进入容器
docker-compose exec backend sh
docker-compose exec db psql -U postgres -d blog_db
```

## 📁 项目结构

```
MBlog/
├── backend/                # Go 后端服务
│   ├── main.go            # 主程序入口
│   ├── Dockerfile         # 后端 Docker 构建文件
│   ├── go.mod             # Go 依赖管理
│   ├── pkg/               # 业务逻辑包
│   │   ├── comments/      # 评论系统
│   │   ├── filemanager/   # 文件管理
│   │   └── tracking/      # 访问统计
    A[用户] -->|HTTP:80| B[Frontend<br/>Nginx]
    A -->|HTTP:3000| C[Backend<br/>Go API]
    C -->|5432| D[(PostgreSQL<br/>Database)]
    B -.依赖.-> C
    C -.依赖.-> D
```

服务启动顺序：
1. 🗄️ **PostgreSQL** 启动并通过健康检查
2. 🔧 **Backend** 等待数据库就绪后启动
3. 🎨 **Frontend** 等待后端就绪后启动

## ⚙️ 高级配置

### 修改端口

如果默认端口被占用，可以修改 `.env` 文件中的端口配置：

```env
# 例如：将前端端口改为 8080
FRONTEND_PORT=8080

# 将后端端口改为 8000
BACKEND_PORT=8000

# 将数据库端口改为 5433
DB_PORT=5433
```

修改后重启服务：

```bash
docker-compose down
docker-compose up -d
```

### 数据持久化

数据库数据存储在 Docker 命名卷 `pg_data` 中，即使容器删除，数据也会保留。

查看卷信息：
```bash
docker volume ls
docker volume inspect mblog_pg_data
```

备份数据库：
```bash
docker-compose exec db pg_dump -U postgres blog_db > backup.sql
```

恢复数据库：
```bash
docker-compose exec -T db psql -U postgres blog_db < backup.sql
```

### 开发模式

如果需要本地开发和热重载：

```bash
# 后端开发（在 backend 目录）
cd backend
go run main.go

# 前端开发（在 frontend 目录）
cd frontend
npm install
npm run dev
```

## ❓ 常见问题

### Q1: 端口被占用怎么办？

**错误信息**：`Bind for 0.0.0.0:80 failed: port is already allocated`

**解决方案**：修改 `.env` 文件中的对应端口号，然后重新启动服务。

### Q2: 数据库连接失败

**错误信息**：`connection refused` 或 `could not connect to database`

**解决方案**：
1. 确保数据库服务已启动：`docker-compose ps`
2. 检查数据库健康状态：`docker-compose exec db pg_isready`
3. 查看数据库日志：`docker-compose logs db`

### Q3: 前端无法访问后端 API

**解决方案**：
1. 检查后端服务状态：`docker-compose ps backend`
2. 测试后端 API：`curl http://localhost:3000/api/ping`
3. 检查网络配置：`docker network ls`

### Q4: 如何完全重置环境？

```bash
# 停止所有服务并删除容器、网络、卷
docker-compose down -v

# 删除所有镜像（可选）
docker-compose down --rmi all -v

# 重新启动
docker-compose up --build
```

### Q5: 忘记数据库密码怎么办？

1. 修改 `.env` 文件中的密码
2. 删除数据库卷：`docker-compose down -v`
3. 重新启动：`docker-compose up -d`

⚠️ **注意**：删除卷会丢失所有数据！

## 🔒 安全建议

- ✅ 使用强密码（至少 16 位，包含大小写字母、数字、特殊字符）
- ✅ 不要将 `.env` 文件提交到版本控制系统
- ✅ 生产环境使用 HTTPS
- ✅ 定期备份数据库
- ✅ 及时更新 Docker 镜像和依赖

## 📖 API 文档

### 主要接口

#### 健康检查
```http
GET /api/ping
```

#### 文件管理
```http
GET  /api/files          # 获取文件列表
GET  /api/files/:path    # 获取文件内容
POST /api/files          # 保存文件
DELETE /api/files/:path  # 删除文件
```

#### 评论系统
```http
GET  /api/comments/:pageId      # 获取评论
POST /api/comments              # 发表评论
```

#### 统计追踪
```http
GET  /api/visitors/stats        # 访问统计
POST /api/track                 # 记录访问
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

[MIT License](LICENSE)

---

💡 **提示**：如有问题，请先查看 [常见问题](#-常见问题) 或提交 Issue。