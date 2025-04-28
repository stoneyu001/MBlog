# MBlog 项目结构说明

## 项目概述
MBlog是一个博客系统，由前端和后端两部分组成，使用Docker容器化部署。

## 项目结构

```
MBlog/
├── backend/               # 后端Go代码
│   ├── main.go            # 主入口文件
│   ├── Dockerfile         # 后端Docker构建文件
│   ├── pkg/               # 功能包目录
│   └── services/          # 服务层代码
├── frontend/              # 前端代码
│   ├── Dockerfile         # 前端Docker构建文件
│   ├── nginx/             # Nginx配置
│   └── docs/              # 文档网站源码
├── docker/                # Docker相关配置
├── docker-compose.yml     # Docker编排配置
└── .env                   # 环境变量配置
```

## 启动文件说明

### 1. docker-compose.yml
功能: 定义和配置Docker服务，编排多个容器
- 配置后端服务 (Go)
- 配置前端服务 (Nginx)
- 建立容器间的依赖关系
- 定义端口映射和数据卷

### 2. backend/main.go
功能: Go后端的入口文件
- 初始化数据库连接
- 设置路由和中间件
- 提供API接口服务
- 处理文件管理和追踪功能

### 3. backend/Dockerfile
功能: 构建后端Docker镜像
- 设置Go环境和依赖
- 复制源代码并编译
- 准备静态文件目录
- 配置容器启动命令

### 4. frontend/Dockerfile
功能: 构建前端Docker镜像
- 设置Node.js环境
- 安装前端依赖
- 构建静态网站文件
- 配置Nginx服务静态文件

### 5. .env
功能: 环境变量配置文件
- 设置端口号
- 配置数据库连接信息
- 定义全局环境变量

## 启动方式

### 生产环境启动
```bash
# 后台启动所有服务
docker-compose up -d
```

## 访问地址
- 后端API: http://localhost:3000
- 前端页面: http://localhost:80
- 管理界面: http://localhost:3000/admin 