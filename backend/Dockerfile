# ============================================
# 第一阶段：构建阶段
# ============================================
FROM golang:1.24.0-alpine AS builder

# 设置工作目录
WORKDIR /build

# 设置 Go 环境变量
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 先复制 go.mod 和 go.sum，利用 Docker 层缓存
COPY go.mod go.sum ./

# 下载依赖（这一层会被缓存，除非 go.mod/go.sum 改变）
RUN go mod download

# 复制源代码
COPY . .

# 编译生成可执行文件
# -ldflags="-w -s" 去除调试信息，减小二进制文件体积
RUN go build -ldflags="-w -s" -o blog .

# ============================================
# 第二阶段：运行阶段
# ============================================
FROM alpine:latest

# 设置时区为上海
ENV TZ=Asia/Shanghai

# 安装必要的运行时依赖
# - ca-certificates: HTTPS 请求需要
# - tzdata: 时区数据
# - nodejs, npm: 用于构建前端站点
# - git: VitePress 构建时需要
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache ca-certificates tzdata nodejs npm git && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    rm -rf /var/cache/apk/*

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件和静态资源
COPY --from=builder /build/blog /app/blog
COPY --from=builder /build/static /app/static

# 设置文件权限
RUN chmod +x /app/blog

# 注意：为了能够写入挂载的 frontend 目录，这里不切换到非 root 用户
# 如果需要更高安全性，可以在宿主机上确保挂载目录权限正确后恢复 appuser

# 暴露端口
EXPOSE 3000

# 健康检查（可选，但推荐）
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/api/ping || exit 1

# 启动应用
CMD ["./blog"]
