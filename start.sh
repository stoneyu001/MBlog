#!/bin/bash

# ============================================
# MBlog 快速启动脚本
# ============================================

echo "🚀 MBlog 快速启动脚本"
echo "===================="
echo ""

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ 错误：未检测到 Docker，请先安装 Docker"
    echo "   下载地址：https://www.docker.com/get-started"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "❌ 错误：未检测到 Docker Compose，请先安装 Docker Compose"
    echo "   安装文档：https://docs.docker.com/compose/install/"
    exit 1
fi

echo "✅ Docker 环境检测通过"
echo ""

# 检查 .env 文件是否存在
if [ ! -f .env ]; then
    echo "⚠️  未找到 .env 文件，正在从模板创建..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "✅ 已创建 .env 文件"
        echo "⚠️  请编辑 .env 文件，设置数据库密码！"
        echo "   位置: $(pwd)/.env"
        echo ""
        read -p "按回车键继续..."
    else
        echo "❌ 错误：未找到 .env.example 模板文件"
        exit 1
    fi
else
    echo "✅ 找到 .env 配置文件"
fi

echo ""
echo "📦 正在启动服务..."
echo "   1️⃣  PostgreSQL 数据库"
echo "   2️⃣  Go 后端服务"
echo "   3️⃣  Nginx 前端服务"
echo ""

# 启动服务
docker-compose up -d

# 检查启动状态
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ 服务启动成功！"
    echo ""
    echo "📌 访问地址："
    echo "   • 前端网站: http://localhost:$(grep FRONTEND_PORT .env | cut -d '=' -f2)"
    echo "   • 后端 API: http://localhost:$(grep BACKEND_PORT .env | cut -d '=' -f2)"
    echo "   • 管理界面: http://localhost:$(grep BACKEND_PORT .env | cut -d '=' -f2)/admin"
    echo ""
    echo "📊 查看服务状态: docker-compose ps"
    echo "📋 查看服务日志: docker-compose logs -f"
    echo "🛑 停止服务: docker-compose down"
else
    echo ""
    echo "❌ 服务启动失败，请查看错误信息"
    echo "💡 提示："
    echo "   • 检查端口是否被占用"
    echo "   • 查看日志: docker-compose logs"
    exit 1
fi
