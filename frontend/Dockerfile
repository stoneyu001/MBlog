FROM docker.xuanyuan.me/library/node:18-alpine

WORKDIR /app

# 复制 package.json 和 package-lock.json（如果存在）
COPY package*.json ./

# 安装依赖
RUN npm install

# 复制源代码（包括 .vitepress 目录）
COPY . .

# 构建文档
RUN npm run docs:build

# 使用 nginx 来服务静态文件
FROM docker.xuanyuan.me/library/nginx:alpine
COPY --from=0 /app/docs/.vitepress/dist /usr/share/nginx/html
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"] 