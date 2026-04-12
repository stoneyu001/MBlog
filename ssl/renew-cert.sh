#!/bin/bash
# ============================================
# SSL 证书自动续签脚本
# ============================================
# 
# 用途：使用 certbot 自动续签 Let's Encrypt SSL 证书
# 续签成功后自动重载宿主机 Nginx 使新证书生效
#
# 使用方法：
#   手动执行: sudo bash /path/to/renew-cert.sh
#   自动定时: 通过 crontab 或 systemd timer 调度
# ============================================

set -euo pipefail

# --- 配置区域 ---
DOMAIN="www.stoneyu.fun"
LOG_FILE="/var/log/certbot-renew.log"
NGINX_RELOAD_CMD="systemctl reload nginx"

# --- 日志函数 ---
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# --- 主流程 ---
log "========== 开始证书续签检查 [$DOMAIN] =========="

# 1. 检查 certbot 是否安装
if ! command -v certbot &> /dev/null; then
    log "❌ 错误: certbot 未安装"
    log "   请先安装: sudo apt install certbot python3-certbot-nginx"
    exit 1
fi

# 2. 查看当前证书状态
log "📋 当前证书状态:"
certbot certificates 2>&1 | tee -a "$LOG_FILE"

# 3. 执行续签（仅在证书即将过期时才会真正续签）
#    --quiet: 静默模式，减少输出
#    --deploy-hook: 续签成功后执行的命令（重载 Nginx）
log "🔄 执行 certbot renew..."
if certbot renew \
    --quiet \
    --deploy-hook "$NGINX_RELOAD_CMD" \
    2>&1 | tee -a "$LOG_FILE"; then
    log "✅ 证书续签检查完成"
else
    log "❌ 证书续签失败，请检查日志: $LOG_FILE"
    exit 1
fi

# 4. 验证证书有效期
log "📋 续签后证书状态:"
certbot certificates 2>&1 | tee -a "$LOG_FILE"

log "========== 证书续签检查结束 =========="
