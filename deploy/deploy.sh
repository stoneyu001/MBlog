#!/bin/bash
# ============================================
# MBlog 一键部署脚本
# ============================================
#
# 适用场景：
#   1. 首次部署：从 GitHub 克隆后执行
#   2. 更新部署：git pull 后执行
#
# 使用方法：
#   首次部署: sudo bash deploy/deploy.sh
#   更新代码: sudo bash deploy/deploy.sh --update
#   仅重启:   sudo bash deploy/deploy.sh --restart
#
# ============================================

set -uo pipefail

# ============================================
# 配置区域
# ============================================
DOMAIN="www.stoneyu.fun"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NGINX_CONF_NAME="www.stoneyu.fun"
CERTBOT_WEBROOT="/var/www/certbot"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- 工具函数 ---
info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[✅]${NC} $1"; }
warn()    { echo -e "${YELLOW}[⚠️]${NC} $1"; }
error()   { echo -e "${RED}[❌]${NC} $1"; exit 1; }

print_banner() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}  MBlog 一键部署 — ${DOMAIN}${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
}

# ============================================
# 检查前置依赖
# ============================================
check_prerequisites() {
    info "检查前置依赖..."

    # 检查 root 权限
    if [ "$EUID" -ne 0 ]; then
        error "请使用 sudo 运行此脚本: sudo bash deploy/deploy.sh"
    fi

    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        error "Docker 未安装。请先安装: https://docs.docker.com/engine/install/"
    fi
    success "Docker $(docker --version | awk '{print $3}' | tr -d ',')"

    # 检查 Docker Compose
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        error "Docker Compose 未安装"
    fi
    success "Docker Compose 已就绪"

    # 检查 curl（健康检查需要）
    if ! command -v curl &> /dev/null; then
        apt install -y curl
    fi

    # 检查 Nginx (宿主机)
    if ! command -v nginx &> /dev/null; then
        warn "宿主机 Nginx 未安装，正在安装..."
        apt update && apt install -y nginx
        systemctl enable nginx
        success "Nginx 安装完成"
    else
        success "Nginx $(nginx -v 2>&1 | awk -F/ '{print $2}')"
    fi

    # 检查 Certbot 及 nginx 插件
    if ! command -v certbot &> /dev/null; then
        warn "Certbot 未安装，正在安装..."
        apt update && apt install -y certbot python3-certbot-nginx
        success "Certbot 安装完成"
    else
        success "Certbot 已就绪"
    fi

    # 确保 certbot nginx 插件已安装
    if ! python3 -c "import certbot_nginx" &> /dev/null 2>&1; then
        warn "Certbot Nginx 插件未安装，正在安装..."
        apt update && apt install -y python3-certbot-nginx
        success "Certbot Nginx 插件安装完成"
    fi

    echo ""
}

# ============================================
# 配置环境变量
# ============================================
setup_env() {
    info "检查环境变量配置..."

    if [ ! -f "$PROJECT_DIR/.env" ]; then
        warn ".env 文件不存在，从模板创建..."
        cp "$PROJECT_DIR/.env.example" "$PROJECT_DIR/.env"
        warn "⚠️  请编辑 .env 文件修改数据库密码！"
        warn "   运行: nano $PROJECT_DIR/.env"
        warn "   修改完成后重新运行此脚本"
        echo ""
        read -p "是否现在编辑 .env 文件? (y/N): " edit_env
        if [[ "$edit_env" =~ ^[Yy]$ ]]; then
            ${EDITOR:-nano} "$PROJECT_DIR/.env"
        fi
    fi

    # 验证 .env 中密码已修改
    if grep -q "your_secure_password_here" "$PROJECT_DIR/.env" 2>/dev/null; then
        error ".env 中的数据库密码尚未修改，请先编辑 .env 文件"
    fi

    # 验证 JWT 密钥已修改
    if grep -q "your-secret-key-should-be-complex" "$PROJECT_DIR/.env" 2>/dev/null; then
        error ".env 中的 JWT_SECRET 尚未修改，请先编辑 .env 文件"
    fi

    success ".env 配置已就绪"
}

# ============================================
# 部署宿主机 Nginx 配置
# ============================================
setup_nginx() {
    info "部署宿主机 Nginx 配置..."

    local nginx_src="$PROJECT_DIR/deploy/nginx-host.conf"
    local nginx_available="/etc/nginx/sites-available/$NGINX_CONF_NAME"
    local nginx_enabled="/etc/nginx/sites-enabled/$NGINX_CONF_NAME"

    if [ ! -f "$nginx_src" ]; then
        error "找不到 Nginx 配置文件: $nginx_src"
    fi

    # 创建 certbot webroot 目录
    mkdir -p "$CERTBOT_WEBROOT"

    # 首次部署：SSL 证书还不存在时，先部署一个仅 HTTP 的临时配置
    if [ ! -d "/etc/letsencrypt/live/$DOMAIN" ]; then
        info "SSL 证书尚未申请，先部署 HTTP 临时配置..."

        cat > "$nginx_available" << TMPEOF
# 临时配置：仅用于 certbot 首次申请证书
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN stoneyu.fun;

    location /.well-known/acme-challenge/ {
        root $CERTBOT_WEBROOT;
        allow all;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
TMPEOF
    else
        # 证书已存在，部署完整 SSL 配置
        cp "$nginx_src" "$nginx_available"
    fi

    # 创建软链接
    ln -sf "$nginx_available" "$nginx_enabled"

    # 移除默认配置（避免冲突）
    rm -f /etc/nginx/sites-enabled/default

    # 测试配置
    if nginx -t 2>&1; then
        systemctl reload nginx
        success "Nginx 配置已部署并重载"
    else
        error "Nginx 配置测试失败，请检查配置文件"
    fi
}

# ============================================
# 申请/续签 SSL 证书
# ============================================
setup_ssl() {
    info "检查 SSL 证书..."

    if [ ! -d "/etc/letsencrypt/live/$DOMAIN" ]; then
        info "首次申请 SSL 证书..."
        echo ""
        warn "即将为 $DOMAIN 申请 Let's Encrypt 证书"
        warn "请确保域名 DNS 已正确指向此服务器"
        echo ""

        # 使用 webroot 方式申请证书（不修改 Nginx 配置，与手动维护的配置解耦）
        if certbot certonly --webroot \
            -w "$CERTBOT_WEBROOT" \
            -d "$DOMAIN" \
            -d "stoneyu.fun" \
            --non-interactive \
            --agree-tos \
            --email "2065802923@qq.com"; then

            success "SSL 证书申请成功！"

            # 证书申请成功后，部署完整的 SSL Nginx 配置
            info "部署完整 SSL Nginx 配置..."
            cp "$PROJECT_DIR/deploy/nginx-host.conf" "/etc/nginx/sites-available/$NGINX_CONF_NAME"
            nginx -t && systemctl reload nginx
            success "完整 SSL 配置已部署"
        else
            error "SSL 证书申请失败，请检查域名 DNS 配置"
        fi
    else
        success "SSL 证书已存在"

        # 检查证书是否已覆盖裸域名，若未覆盖则扩展证书
        if ! openssl x509 -in "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" -text -noout 2>/dev/null | grep -q "DNS:stoneyu.fun"; then
            info "证书未覆盖裸域名 stoneyu.fun，正在扩展..."
            if certbot certonly --webroot \
                -w "$CERTBOT_WEBROOT" \
                -d "$DOMAIN" \
                -d "stoneyu.fun" \
                --expand \
                --non-interactive \
                --agree-tos \
                --email "2065802923@qq.com"; then
                success "证书已扩展覆盖 stoneyu.fun"
                systemctl reload nginx
            else
                warn "证书扩展失败，stoneyu.fun 可能无法访问"
            fi
        fi

        # 检查是否需要续签
        info "尝试续签..."
        certbot renew --quiet --deploy-hook "systemctl reload nginx" || true
        success "证书续签检查完成"
    fi
}

# ============================================
# 配置自动续签
# ============================================
setup_auto_renew() {
    info "配置 SSL 自动续签..."

    # 检查 systemd timer 是否已配置
    if systemctl is-enabled certbot-renew.timer &> /dev/null; then
        success "自动续签已配置（systemd timer）"
        return
    fi

    # 运行自动续签配置脚本
    if [ -f "$PROJECT_DIR/deploy/setup-auto-renew.sh" ]; then
        chmod +x "$PROJECT_DIR/deploy/setup-auto-renew.sh"
        bash "$PROJECT_DIR/deploy/setup-auto-renew.sh"
    else
        # 后备方案：直接配置 crontab
        local cron_job='30 3 * * * /usr/bin/certbot renew --quiet --deploy-hook "systemctl reload nginx" >> /var/log/certbot-renew.log 2>&1'
        if ! crontab -l 2>/dev/null | grep -q "certbot renew"; then
            (crontab -l 2>/dev/null; echo "$cron_job") | crontab -
        fi
        success "已通过 crontab 配置自动续签"
    fi
}

# ============================================
# 启动 Docker 服务
# ============================================
start_docker() {
    local mode="$1"

    info "启动 Docker 服务..."
    cd "$PROJECT_DIR"

    case "$mode" in
        "build")
            info "构建并启动所有容器（首次部署或代码更新）..."
            $COMPOSE_CMD up -d --build
            ;;
        "restart")
            info "重启所有容器..."
            $COMPOSE_CMD restart
            ;;
        "up")
            info "启动所有容器..."
            $COMPOSE_CMD up -d
            ;;
    esac

    # 等待服务就绪
    info "等待服务启动..."
    sleep 5

    # 检查容器状态
    echo ""
    info "容器状态:"
    $COMPOSE_CMD ps
    echo ""

    # 健康检查
    local max_retries=30
    local retry=0
    info "等待后端服务就绪..."

    while [ $retry -lt $max_retries ]; do
        if curl -sf http://127.0.0.1:8080 > /dev/null 2>&1; then
            success "服务已就绪！"
            return
        fi
        retry=$((retry + 1))
        sleep 2
    done

    warn "服务可能仍在启动中，请使用 'docker compose logs -f' 查看日志"
}

# ============================================
# 打印部署结果
# ============================================
print_result() {
    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}  🎉 MBlog 部署成功！${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "  🌐 博客首页:   https://$DOMAIN"
    echo "  🔧 管理后台:   https://$DOMAIN/admin"
    echo "  🔑 默认账号:   admin / admin123"
    echo ""
    echo "  📋 常用命令:"
    echo "    查看日志:    cd $PROJECT_DIR && $COMPOSE_CMD logs -f"
    echo "    重启服务:    cd $PROJECT_DIR && $COMPOSE_CMD restart"
    echo "    停止服务:    cd $PROJECT_DIR && $COMPOSE_CMD down"
    echo "    更新部署:    cd $PROJECT_DIR && git pull && sudo bash deploy/deploy.sh --update"
    echo "    证书状态:    sudo certbot certificates"
    echo ""
}

# ============================================
# 主流程
# ============================================
main() {
    print_banner

    local mode="${1:-}"

    case "$mode" in
        "--update" | "-u")
            info "模式: 更新部署（重新构建镜像）"
            echo ""
            check_prerequisites
            start_docker "build"
            setup_ssl
            print_result
            ;;
        "--restart" | "-r")
            info "模式: 仅重启服务"
            echo ""
            # 初始化 COMPOSE_CMD
            if docker compose version &> /dev/null; then
                COMPOSE_CMD="docker compose"
            else
                COMPOSE_CMD="docker-compose"
            fi
            start_docker "restart"
            ;;
        *)
            info "模式: 完整部署（首次或重新部署）"
            echo ""
            check_prerequisites
            setup_env
            start_docker "build"
            setup_nginx
            setup_ssl
            setup_auto_renew
            print_result
            ;;
    esac
}

main "$@"
