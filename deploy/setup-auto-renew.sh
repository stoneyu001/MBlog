#!/bin/bash
# ============================================
# SSL 证书自动续签配置（仅 systemd timer）
# ============================================
set -euo pipefail

if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RENEW_SCRIPT="$SCRIPT_DIR/renew-cert.sh"

if [ ! -f "$RENEW_SCRIPT" ]; then
    echo "找不到续签脚本: $RENEW_SCRIPT"
    exit 1
fi

chmod +x "$RENEW_SCRIPT"

echo "安装 certbot（如未安装）..."
if ! command -v certbot &> /dev/null; then
    apt update && apt install -y certbot python3-certbot-nginx
fi

# --- systemd service（调用 renew-cert.sh）---
cat > /etc/systemd/system/certbot-renew.service << EOF
[Unit]
Description=Certbot SSL Certificate Renewal
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=${RENEW_SCRIPT}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# --- systemd timer（每天 03:30 和 15:30 各检查一次）---
cat > /etc/systemd/system/certbot-renew.timer << 'EOF'
[Unit]
Description=Run Certbot SSL Renewal Twice Daily

[Timer]
OnCalendar=*-*-* 03:30:00
OnCalendar=*-*-* 15:30:00
RandomizedDelaySec=1800
Persistent=true

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now certbot-renew.timer

# --- logrotate（按月轮转，保留 6 个月）---
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

echo ""
echo "============================================"
echo "  自动续签配置完成"
echo "============================================"
echo ""
echo "  timer 状态: systemctl status certbot-renew.timer"
echo "  下次执行:   systemctl list-timers certbot-renew.timer"
echo "  手动测试:   sudo certbot renew --dry-run"
echo "  证书状态:   sudo certbot certificates"
echo ""
