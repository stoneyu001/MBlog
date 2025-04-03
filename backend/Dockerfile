# 使用官方 Go 轻量镜像
FROM golang:1.24.0-alpine

# 设置容器内工作目录
WORKDIR /app

# 优先复制依赖清单（利用层缓存优化）
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 构建可执行文件（禁用CGO确保静态二进制）
RUN CGO_ENABLED=0 GOOS=linux go build -o /blog

# 声明服务端口
EXPOSE 3000

# 启动命令
CMD ["/blog"]
