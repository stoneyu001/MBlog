#!/bin/bash
# ============================================
# 一键配置 SSL 证书自动续签
# ============================================
#
# 此脚本会：
# 1. 部署续签脚本到服务器
# 2. 配置 systemd timer 实现自动续签
# 3. 设置日志轮转避免日志文件过大
#
# 使用方法：sudo bash setup-auto-renew.sh
# ============================================

set -euo pipefail

# --- 配置区域 ---
DOMAIN="www.stoneyu.fun"
PROJECT_DIR="$HOME/MBlog"

echo "============================================"
echo "  MBlog SSL 证书自动续签配置"
echo "  域名: $DOMAIN"
echo "============================================"
echo ""

# --- 检查 root 权限 ---
if [ "$EUID" -ne 0 ]; then
    echo "❌ 请使用 sudo 运行此脚本"
    exit 1
fi

# --- 检查 certbot ---
if ! command -v certbot &> /dev/null; then
    echo "❌ certbot 未安装，正在安装..."
    apt update && apt install -y certbot python3-certbot-nginx
    echo "✅ certbot 安装完成"
fi

# --- 获取脚本所在目录 ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RENEW_SCRIPT="$SCRIPT_DIR/renew-cert.sh"

if [ ! -f "$RENEW_SCRIPT" ]; then
    echo "❌ 找不到续签脚本: $RENEW_SCRIPT"
    exit 1
fi

# --- 设置脚本权限 ---
chmod +x "$RENEW_SCRIPT"
echo "✅ 已设置脚本执行权限"

# ============================================
# 方案一：使用 systemd timer（推荐）
# ============================================
echo ""
echo "📦 配置 systemd timer..."

# 创建 service 单元
cat > /etc/systemd/system/certbot-renew.service << 'EOF'
[Unit]
Description=Certbot SSL Certificate Renewal
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --deploy-hook "systemctl reload nginx"
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# 创建 timer 单元（每天凌晨 3:30 和下午 15:30 各检查一次）
cat > /etc/systemd/system/certbot-renew.timer << 'EOF'
[Unit]
Description=Run Certbot SSL Renewal Twice Daily

[Timer]
# 每天凌晨 3:30 执行
OnCalendar=*-*-* 03:30:00
# 每天下午 15:30 执行
OnCalendar=*-*-* 15:30:00
# 添加随机延迟（0~30分钟），避免大量服务器同时请求 Let's Encrypt
RandomizedDelaySec=1800
# 如果错过了执行时间（如服务器关机），启动后立即补执行
Persistent=true

[Install]
WantedBy=timers.target
EOF

# 重载 systemd 并启用 timer
systemctl daemon-reload
systemctl enable certbot-renew.timer
systemctl start certbot-renew.timer

echo "✅ systemd timer 已配置并启用"

# ============================================
# 方案二：同时配置 crontab 作为备用
# ============================================
echo ""
echo "📦 配置 crontab 备用方案..."

CRON_JOB="30 3 * * * /usr/bin/certbot renew --quiet --deploy-hook \"systemctl reload nginx\" >> /var/log/certbot-renew.log 2>&1"

# 检查是否已存在 certbot 的 cron 任务
if crontab -l 2>/dev/null | grep -q "certbot renew"; then
    echo "⚠️  crontab 中已存在 certbot 续签任务，跳过添加"
else
    # 添加 cron 任务
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
    echo "✅ crontab 任务已添加"
fi

# ============================================
# 配置日志轮转
# ============================================
echo ""
echo "📦 配置日志轮转..."

cat > /etc/logrotate.d/certbot-renew << 'EOF'
/var/log/certbot-renew.log {
    monthly
    rotate 6
    compress
    delaycompress
    missingok
    notifempty
    create 0640 root root
}
EOF

echo "✅ 日志轮转已配置"

# ============================================
# 验证配置
# ============================================
echo ""
echo "============================================"
echo "  ✅ 自动续签配置完成！"
echo "============================================"
echo ""
echo "📋 配置摘要:"
echo "  • systemd timer: 每天 03:30 和 15:30 各检查一次"
echo "  • crontab 备用:  每天 03:30 检查一次"
echo "  • 日志文件:      /var/log/certbot-renew.log"
echo "  • 日志轮转:      每月轮转，保留 6 个月"
echo ""
echo "📋 验证命令:"
echo "  查看 timer 状态:  systemctl status certbot-renew.timer"
echo "  查看下次执行时间: systemctl list-timers certbot-renew.timer"
echo "  查看 crontab:    crontab -l"
echo "  手动测试续签:    sudo certbot renew --dry-run"
echo "  查看证书状态:    sudo certbot certificates"
echo ""

# 显示 timer 状态
echo "📋 当前 timer 状态:"
systemctl list-timers certbot-renew.timer --no-pager
echo ""

# 立即做一次 dry-run 测试
echo "🧪 执行 dry-run 测试..."
if certbot renew --dry-run 2>&1; then
    echo ""
    echo "✅ dry-run 测试通过，证书续签配置正常！"
else
    echo ""
    echo "⚠️  dry-run 测试失败，请检查以下事项："
    echo "  1. 宿主机 Nginx 是否正常运行"
    echo "  2. 域名 $DOMAIN 的 DNS 是否正确指向此服务器"
    echo "  3. 端口 80/443 是否开放"
    echo "  4. 运行 'certbot certificates' 查看证书详情"
fi
