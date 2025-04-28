FROM docker.xuanyuan.me/library/nginx:alpine

# 复制已构建的静态文件
COPY docs/.vitepress/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"] 