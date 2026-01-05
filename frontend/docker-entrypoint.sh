#!/bin/sh
# ============================================
# Frontend 容器启动脚本
# 功能：确保首次启动时有预构建的静态文件
# ============================================

# 检查挂载的目录是否为空（没有 index.html）
if [ ! -f "/usr/share/nginx/html/index.html" ]; then
    echo "检测到 html 目录为空，正在从预构建目录复制文件..."
    cp -r /app/dist-backup/* /usr/share/nginx/html/
    echo "文件复制完成"
else
    echo "html 目录已有文件，跳过复制"
fi

# 启动 Nginx
exec nginx -g "daemon off;"
