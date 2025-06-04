# 使用官方 Go 轻量镜像作为构建环境
FROM docker.xuanyuan.me/library/golang:1.24.0-alpine AS builder
# FROM golang:1.24.0-alpine AS builder

# 设置工作目录
WORKDIR /build

# 设置Go代理为七牛云
ENV GOPROXY=https://goproxy.cn,direct

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -o blog

# 使用轻量级镜像运行
FROM docker.xuanyuan.me/library/alpine:latest
# FROM alpine:latest

# 设置时区为中国时区
ENV TZ=Asia/Shanghai

# 使用国内镜像源并安装必要的包
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache tzdata && \
    apk add --no-cache nodejs npm git && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/blog /app/
COPY --from=builder /build/static /app/static

EXPOSE 3000

CMD ["./blog"]
