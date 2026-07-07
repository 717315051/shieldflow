#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 数据备份脚本
# 用法: sudo bash scripts/backup.sh [--output <dir>]
# 功能: PostgreSQL 备份 → ClickHouse 备份 → 配置文件备份 → 打包到 /var/backups/shieldflow/
# 建议: 加入 crontab 每日执行  0 2 * * * /root/shieldflow/scripts/backup.sh
# ============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow-BACKUP]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$*"; exit 1; }

DB_NAME="${ShieldFlow_DB_NAME:-shieldflow_cdn}"
BACKUP_ROOT="/var/backups/shieldflow"
RETAIN_DAYS="${ShieldFlow_BACKUP_RETAIN:-30}"
TS="$(date +%Y%m%d_%H%M%S)"
BACKUP_DIR="${BACKUP_ROOT}/${TS}"

[[ $EUID -eq 0 ]] || die "请以 root 身份运行"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --output) BACKUP_DIR="$2"; shift 2 ;;
        --help|-h) echo "用法: $0 [--output <dir>]  (默认: ${BACKUP_ROOT}/<timestamp>)"; exit 0 ;;
        *) die "未知参数: $1" ;;
    esac
done

mkdir -p "$BACKUP_DIR"

# ---------- 1. PostgreSQL 备份 ----------
backup_postgres() {
    log "备份 PostgreSQL ($DB_NAME)..."
    if command -v pg_dump &>/dev/null; then
        if sudo -u postgres pg_dump -Fc "$DB_NAME" > "${BACKUP_DIR}/${DB_NAME}.dump" 2>/dev/null; then
            log "  PostgreSQL 备份完成: ${DB_NAME}.dump ($(du -h "${BACKUP_DIR}/${DB_NAME}.dump" | awk '{print $1}'))"
        else
            warn "  PostgreSQL 备份失败（数据库可能未运行）"
        fi
    else
        warn "  pg_dump 未安装，跳过 PostgreSQL 备份"
    fi
}

# ---------- 2. ClickHouse 备份 ----------
backup_clickhouse() {
    log "备份 ClickHouse ($DB_NAME)..."
    if command -v clickhouse-client &>/dev/null; then
        # 获取所有表
        local tables
        tables="$(clickhouse-client -q "SHOW TABLES FROM ${DB_NAME}" 2>/dev/null || true)"
        if [[ -z "$tables" ]]; then
            warn "  ClickHouse 数据库 $DB_NAME 无表或不可用"
            return
        fi
        local ch_dir="${BACKUP_DIR}/clickhouse"
        mkdir -p "$ch_dir"
        local count=0
        while IFS= read -r tbl; do
            [[ -z "$tbl" ]] && continue
            if clickhouse-client -q "SELECT * FROM ${DB_NAME}.${tbl} FORMAT TabSeparated" > "${ch_dir}/${tbl}.tsv" 2>/dev/null; then
                count=$((count+1))
            else
                warn "  导出表 ${tbl} 失败"
            fi
        done <<< "$tables"
        log "  ClickHouse 备份完成: ${count} 张表 → ${ch_dir}/"
    else
        warn "  clickhouse-client 未安装，跳过 ClickHouse 备份"
    fi
}

# ---------- 3. 配置文件备份 ----------
backup_configs() {
    log "备份配置文件..."
    local conf_dir="${BACKUP_DIR}/configs"
    mkdir -p "$conf_dir"
    [[ -d /etc/shieldflow ]] && cp -r /etc/shieldflow "${conf_dir}/shieldflow" 2>/dev/null || warn "  /etc/shieldflow 不存在"
    # Supervisor 配置
    for d in /etc/supervisord.d /etc/supervisor/conf.d; do
        [[ -d "$d" ]] && cp -r "$d" "${conf_dir}/$(basename $d)" 2>/dev/null || true
    done
    # Nginx 配置
    [[ -f /etc/nginx/conf.d/shieldflow.conf ]] && cp /etc/nginx/conf.d/shieldflow.conf "${conf_dir}/" 2>/dev/null || true
    log "  配置文件备份完成: ${conf_dir}/"
}

# ---------- 4. 打包 ----------
package() {
    log "打包备份..."
    local tarball="${BACKUP_ROOT}/shieldflow-backup-${TS}.tar.gz"
    tar -czf "$tarball" -C "${BACKUP_ROOT}" "${TS}" 2>/dev/null && rm -rf "$BACKUP_DIR"
    log "  打包完成: ${tarball} ($(du -h "$tarball" | awk '{print $1}'))"
}

# ---------- 5. 清理旧备份 ----------
cleanup() {
    log "清理 ${RETAIN_DAYS} 天前的旧备份..."
    find "$BACKUP_ROOT" -name "shieldflow-backup-*.tar.gz" -mtime +${RETAIN_DAYS} -delete 2>/dev/null || true
    local removed
    removed=$(find "$BACKUP_ROOT" -maxdepth 1 -type d -name "20*" -mtime +${RETAIN_DAYS} 2>/dev/null | wc -l)
    find "$BACKUP_ROOT" -maxdepth 1 -type d -name "20*" -mtime +${RETAIN_DAYS} -exec rm -rf {} \; 2>/dev/null || true
    log "  清理完成"
}

main() {
    log "开始 ShieldFlow 数据备份 @ ${TS}"
    backup_postgres
    backup_clickhouse
    backup_configs
    package
    cleanup
    log "备份完成: ${BACKUP_ROOT}/shieldflow-backup-${TS}.tar.gz"
}

main "$@"
