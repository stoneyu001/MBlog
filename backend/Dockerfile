# 使用官方 Go 轻量镜像作为构建环境
FROM docker.xuanyuan.me/library/golang:1.24.0-alpine as builder

# 设置工作目录
WORKDIR /build

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -o blog

# 使用轻量级镜像运行
FROM docker.xuanyuan.me/library/alpine:latest

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/blog /app/
COPY --from=builder /build/static /app/static

EXPOSE 3000

CMD ["./blog"]
