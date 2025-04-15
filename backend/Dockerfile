# 使用官方 Go 轻量镜像
FROM golang:1.24.0-alpine

# 设置 Go 模块代理
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 设置容器内工作目录
WORKDIR /app

# 优先复制依赖清单（利用层缓存优化）
COPY go.mod go.sum ./
RUN go mod download

# 创建并准备静态文件目录
RUN mkdir -p /app/static

# 复制静态文件
COPY static/admin.html /app/static/index.html

# 复制其他源代码
COPY . .

# 安装 Node.js 和 npm（用于构建前端）
RUN apk add --no-cache nodejs npm

# 构建可执行文件（禁用CGO确保静态二进制）
RUN CGO_ENABLED=0 GOOS=linux go build -o /blog

# 声明服务端口
EXPOSE 3000

# 启动命令
CMD ["/blog"]
