# ============================================
# 第一阶段:构建阶段
# ============================================
FROM node:20-alpine AS builder

WORKDIR /build

# 安装 git（VitePress 需要用于获取文件元数据）
RUN apk add --no-cache git tzdata

# 设置 npm 镜像源为国内镜像（加速依赖下载）
RUN npm config set registry https://registry.npmmirror.com

# 复制 package.json 和 package-lock.json（利用层缓存）
COPY package*.json ./

# 安装依赖
RUN npm ci --only=production

# 复制源代码
COPY docs ./docs

# 构建 VitePress 静态文件
RUN npm run docs:build

# ============================================
# 第二阶段：运行阶段
# ============================================
FROM nginx:alpine

# 设置时区
ENV TZ=Asia/Shanghai
RUN apk add --no-cache tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 从构建阶段复制生成的静态文件到备份目录
# 这样即使 volume 挂载覆盖 /usr/share/nginx/html，我们仍有备份
COPY --from=builder /build/docs/.vitepress/dist /app/dist-backup

# 复制 nginx 配置
COPY nginx/default.conf /etc/nginx/conf.d/default.conf

# 复制并设置 entrypoint 脚本
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# 使用 entrypoint 脚本启动
ENTRYPOINT ["/docker-entrypoint.sh"]
