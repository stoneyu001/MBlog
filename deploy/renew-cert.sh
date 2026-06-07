#!/bin/bash
# ============================================
# SSL 证书续签脚本
# ============================================
# 调用方式：
#   - systemd timer 自动调用
#   - 手动：sudo bash renew-cert.sh
# ============================================

set -euo pipefail

DOMAIN="www.stoneyu.fun"
LOG_FILE="/var/log/certbot-renew.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "========== 证书续签检查 [${DOMAIN}] =========="

if ! command -v certbot &> /dev/null; then
    log "错误: certbot 未安装"
    exit 1
fi

log "执行 certbot renew..."
if certbot renew --quiet --deploy-hook "systemctl reload nginx" >> "$LOG_FILE" 2>&1; then
    log "续签检查完成（证书未到期则无操作）"
else
    log "续签失败"
    exit 1
fi

log "========== 结束 =========="
