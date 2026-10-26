# ============================================
# 第一阶段:构建阶段
# ============================================
FROM node:20-alpine AS builder

WORKDIR /build

# 安装 git（VitePress 需要用于获取文件元数据）
RUN apk add --no-cache git tzdata

# 安装 pnpm 并设置镜像源
RUN npm install -g pnpm@9.10.0 && pnpm config set registry https://registry.npmmirror.com

# 复制 pnpm 配置和 lock 文件
COPY package.json pnpm-lock.yaml ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制源代码
COPY docs ./docs

# 构建 VitePress 静态文件
RUN pnpm run docs:build

# ============================================
# 第二阶段：运行阶段
# ============================================
FROM nginx:alpine

# 设置时区
ENV TZ=Asia/Shanghai
RUN apk add --no-cache tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 静态产物直接固化到运行镜像。
COPY --from=builder /build/docs/.vitepress/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY nginx/default.conf /etc/nginx/conf.d/default.conf

# 暴露端口
EXPOSE 80
