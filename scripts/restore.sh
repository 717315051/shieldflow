#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 数据恢复脚本
# 用法: sudo bash scripts/restore.sh --file <backup.tar.gz> [--no-clickhouse]
# 功能: 解包备份 → 恢复 PostgreSQL → 恢复 ClickHouse → 恢复配置
# 警告: 恢复会覆盖现有数据，请先停止服务!
# ============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow-RESTORE]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$*"; exit 1; }

DB_NAME="${ShieldFlow_DB_NAME:-shieldflow_cdn}"
DB_USER="${ShieldFlow_DB_USER:-shieldflow}"
BACKUP_FILE=""
NO_CLICKHOUSE=false
EXTRACT_DIR=""

[[ $EUID -eq 0 ]] || die "请以 root 身份运行"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --file)        BACKUP_FILE="$2"; shift 2 ;;
        --db-name)     DB_NAME="$2"; shift 2 ;;
        --no-clickhouse) NO_CLICKHOUSE=true; shift ;;
        --help|-h)
            cat <<EOF
用法: $0 --file <backup.tar.gz> [选项]
选项:
  --file <path>         备份文件路径（必填）
  --db-name <name>      数据库名 (默认: shieldflow_cdn)
  --no-clickhouse       跳过 ClickHouse 恢复
EOF
            exit 0 ;;
        *) die "未知参数: $1" ;;
    esac
done

[[ -n "$BACKUP_FILE" ]] || die "缺少 --file 参数"
[[ -f "$BACKUP_FILE" ]] || die "备份文件不存在: $BACKUP_FILE"

# ---------- 0. 停止服务 ----------
stop_services() {
    log "停止 ShieldFlow 服务..."
    for group in shieldflow-master shieldflow-edge shieldflow-dns-sync shieldflow-log-server; do
        supervisorctl stop "${group}:*" 2>/dev/null || true
    done
}

# ---------- 1. 解包备份 ----------
extract_backup() {
    log "解包备份..."
    EXTRACT_DIR=$(mktemp -d /tmp/shieldflow-restore.XXXXXX)
    tar -xzf "$BACKUP_FILE" -C "$EXTRACT_DIR"
    # 兼容两种打包结构：直接在根目录 或 在子目录中
    if [[ -d "${EXTRACT_DIR}/$(ls "$EXTRACT_DIR" | head -1)/configs" ]]; then
        EXTRACT_DIR="${EXTRACT_DIR}/$(ls "$EXTRACT_DIR" | head -1)"
    fi
    log "  解包到: ${EXTRACT_DIR}"
}

# ---------- 2. 恢复 PostgreSQL ----------
restore_postgres() {
    local dump_file="${EXTRACT_DIR}/${DB_NAME}.dump"
    if [[ ! -f "$dump_file" ]]; then
        warn "  PostgreSQL 备份文件不存在: $dump_file"
        return
    fi
    log "恢复 PostgreSQL..."
    # 确保数据库和用户存在
    sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" 2>/dev/null | grep -q 1 \
        || sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME};" 2>/dev/null || true
    sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" 2>/dev/null | grep -q 1 \
        || sudo -u postgres psql -c "CREATE USER ${DB_USER};" 2>/dev/null || true
    # 恢复（drop 现有连接，避免冲突）
    sudo -u postgres psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${DB_NAME}' AND pid<>pg_backend_pid();" 2>/dev/null || true
    sudo -u postgres dropdb --if-exists "$DB_NAME" 2>/dev/null || true
    sudo -u postgres createdb "$DB_NAME" 2>/dev/null || true
    sudo -u postgres pg_restore -d "$DB_NAME" --no-owner --no-privileges "$dump_file" 2>/dev/null \
        && log "  PostgreSQL 恢复完成" \
        || warn "  PostgreSQL 恢复有警告（通常可忽略）"
}

# ---------- 3. 恢复 ClickHouse ----------
restore_clickhouse() {
    [[ "$NO_CLICKHOUSE" == "true" ]] && { warn "跳过 ClickHouse 恢复 (--no-clickhouse)"; return; }
    local ch_dir="${EXTRACT_DIR}/clickhouse"
    if [[ ! -d "$ch_dir" ]]; then
        warn "  ClickHouse 备份目录不存在: $ch_dir"
        return
    fi
    log "恢复 ClickHouse..."
    if ! command -v clickhouse-client &>/dev/null; then
        warn "  clickhouse-client 未安装，跳过"
        return
    fi
    clickhouse-client -q "CREATE DATABASE IF NOT EXISTS ${DB_NAME}" 2>/dev/null || true
    local count=0
    for tsv in "$ch_dir"/*.tsv; do
        [[ -f "$tsv" ]] || continue
        local tbl="$(basename "$tsv" .tsv)"
        if clickhouse-client -d "$DB_NAME" -q "INSERT INTO ${tbl} FORMAT TabSeparated" < "$tsv" 2>/dev/null; then
            count=$((count+1))
        else
            warn "  恢复表 ${tbl} 失败（表结构可能不存在，请先执行 SQL 初始化）"
        fi
    done
    log "  ClickHouse 恢复完成: ${count} 张表"
}

# ---------- 4. 恢复配置 ----------
restore_configs() {
    local conf_src="${EXTRACT_DIR}/configs/shieldflow"
    if [[ -d "$conf_src" ]]; then
        log "恢复配置文件..."
        mkdir -p /etc/shieldflow
        cp -rf "${conf_src}/"* /etc/shieldflow/ 2>/dev/null || true
        log "  配置已恢复到 /etc/shieldflow/"
    else
        warn "  备份中无配置目录"
    fi
}

# ---------- 5. 启动服务 ----------
start_services() {
    log "启动 ShieldFlow 服务..."
    for group in shieldflow-master shieldflow-dns-sync shieldflow-log-server; do
        supervisorctl start "${group}:*" 2>/dev/null || warn "  启动 ${group} 失败"
    done
}

# ---------- 6. 清理 ----------
cleanup() {
    [[ -n "$EXTRACT_DIR" ]] && rm -rf "$EXTRACT_DIR"
}

main() {
    log "开始 ShieldFlow 数据恢复: $BACKUP_FILE"
    stop_services
    extract_backup
    restore_postgres
    restore_clickhouse
    restore_configs
    start_services
    cleanup
    echo ""
    log "恢复完成！"
    log "请检查: supervisorctl status"
    log "请检查: curl http://127.0.0.1:8080/api/v1/health"
}

trap cleanup EXIT
main "$@"
