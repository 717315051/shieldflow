#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 卸载脚本
# 用法: sudo bash scripts/uninstall.sh [--purge-data]
# 功能: 停止服务 → 移除程序/配置 → 可选删除数据库
# ============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

INSTALL_PREFIX="/usr/local/shieldflow"
CONF_DIR="/etc/shieldflow"
LOG_DIR="/var/log/shieldflow"
CACHE_DIR="/var/cache/shieldflow"
BACKUP_DIR="/var/backups/shieldflow"
DB_NAME="shieldflow_cdn"
DB_USER="shieldflow"
PURGE_DATA=false
KEEP_NGINX="${ShieldFlow_KEEP_NGINX:-false}"

[[ $EUID -eq 0 ]] || { error "请以 root 身份运行"; exit 1; }

while [[ $# -gt 0 ]]; do
    case "$1" in
        --purge-data) PURGE_DATA=true; shift ;;
        --help|-h) echo "用法: $0 [--purge-data]"; exit 0 ;;
        *) error "未知参数: $1"; exit 1 ;;
    esac
done

# 停止 Supervisor 服务
stop_services() {
    log "停止 Supervisor 服务..."
    for group in shieldflow-master shieldflow-edge shieldflow-dns-sync shieldflow-log-server; do
        supervisorctl stop "${group}:*" 2>/dev/null || true
        supervisorctl remove "${group}" 2>/dev/null || true
    done
    # 移除配置
    for f in shieldflow-master shieldflow-edge shieldflow-dns-sync shieldflow-log-server; do
        rm -f "/etc/supervisord.d/${f}.conf" "/etc/supervisor/conf.d/${f}.conf"
    done
    supervisorctl reread 2>/dev/null || true
    supervisorctl update 2>/dev/null || true
}

# 停止 Nginx 站点
remove_nginx() {
    log "移除 Nginx 配置..."
    rm -f /etc/nginx/conf.d/shieldflow.conf /etc/nginx/sites-enabled/shieldflow.conf
    if [[ "$KEEP_NGINX" != "true" ]]; then
        systemctl stop nginx 2>/dev/null || true
        systemctl disable nginx 2>/dev/null || true
    else
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
    fi
}

# 移除程序和目录
remove_files() {
    log "移除程序文件..."
    rm -rf "$INSTALL_PREFIX"
    rm -rf "$LOG_DIR" "$CACHE_DIR"
    # 保留备份目录
    if [[ "$PURGE_DATA" == "true" ]]; then
        rm -rf "$BACKUP_DIR"
    fi
}

# 可选删除数据库
remove_database() {
    if [[ "$PURGE_DATA" == "true" ]]; then
        warn "正在删除数据库 ${DB_NAME}（--purge-data）..."
        sudo -u postgres psql -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || warn "删除数据库失败"
        sudo -u postgres psql -c "DROP USER IF EXISTS ${DB_USER};" 2>/dev/null || true
        warn "正在删除 ClickHouse 数据库..."
        clickhouse-client -q "DROP DATABASE IF EXISTS ${DB_NAME}" 2>/dev/null || true
    fi
}

# 移除配置
remove_configs() {
    log "移除配置文件..."
    if [[ "$PURGE_DATA" == "true" ]]; then
        rm -rf "$CONF_DIR"
    else
        warn "保留配置目录 ${CONF_DIR}（如需删除使用 --purge-data）"
    fi
}

main() {
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}   ShieldFlow 卸载脚本${NC}"
    echo -e "${RED}========================================${NC}"
    stop_services
    remove_nginx
    remove_files
    remove_database
    remove_configs
    echo ""
    log "卸载完成。"
    [[ "$PURGE_DATA" == "true" ]] && log "所有数据已清除。" || warn "数据库和配置已保留（--purge-data 可彻底清除）"
}

main "$@"
