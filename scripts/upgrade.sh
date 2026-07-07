#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 升级脚本
# 用法: sudo bash scripts/upgrade.sh [--version <tag>]
# 功能: 备份当前版本 → 拉取新代码 → 编译 → 替换二进制 → 滚动重启
# ============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INSTALL_PREFIX="/usr/local/shieldflow"
BACKUP_DIR="/var/backups/shieldflow"
TARGET_VERSION=""

[[ $EUID -eq 0 ]] || die "请以 root 身份运行"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --version) TARGET_VERSION="$2"; shift 2 ;;
        --help|-h) echo "用法: $0 [--version <git-tag-or-branch>]"; exit 0 ;;
        *) die "未知参数: $1" ;;
    esac
done

# 1. 备份当前版本
backup_current() {
    log "备份当前版本..."
    local ts="$(date +%Y%m%d%H%M%S)"
    local bak="${BACKUP_DIR}/upgrade-${ts}"
    mkdir -p "$bak"
    if [[ -d "${INSTALL_PREFIX}/bin" ]]; then
        cp -r "${INSTALL_PREFIX}/bin" "${bak}/bin"
    fi
    if [[ -d "${INSTALL_PREFIX}/web" ]]; then
        cp -r "${INSTALL_PREFIX}/web" "${bak}/web"
    fi
    # 备份配置
    [[ -d /etc/shieldflow ]] && cp -r /etc/shieldflow "${bak}/etc-shieldflow"
    log "已备份到 ${bak}"
}

# 2. 拉取新代码
pull_code() {
    log "拉取最新代码..."
    cd "$PROJECT_ROOT"
    if [[ -d .git ]]; then
        git fetch --all 2>/dev/null || warn "git fetch 失败"
        if [[ -n "$TARGET_VERSION" ]]; then
            git checkout "$TARGET_VERSION" 2>/dev/null || git checkout "tags/${TARGET_VERSION}" 2>/dev/null || die "无法切换到版本 $TARGET_VERSION"
            log "已切换到版本 $TARGET_VERSION"
        else
            git pull --ff-only 2>/dev/null || warn "git pull 失败"
        fi
    else
        warn "非 git 仓库，使用本地代码"
    fi
}

# 3. 编译
build() {
    log "编译后端..."
    cd "$PROJECT_ROOT"
    export PATH=$PATH:/usr/local/go/bin
    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    mkdir -p "${INSTALL_PREFIX}/bin"
    for comp in backend grpc-server edge dns-sync log-server; do
        log "  编译 $comp..."
        go build -ldflags="-s -w" -o "${INSTALL_PREFIX}/bin/${comp}.new" "./cmd/${comp}" || die "编译 $comp 失败"
        mv "${INSTALL_PREFIX}/bin/${comp}.new" "${INSTALL_PREFIX}/bin/${comp}"
    done
    log "后端编译完成"
}

# 4. 编译前端
build_web() {
    log "编译前端..."
    cd "${PROJECT_ROOT}/web"
    if command -v npm &>/dev/null; then
        npm install --silent 2>/dev/null || npm install || warn "npm install 失败"
        npm run build 2>/dev/null || warn "前端构建失败，保留旧版前端"
        rm -rf "${INSTALL_PREFIX}/web.new"
        cp -r dist "${INSTALL_PREFIX}/web.new"
        rm -rf "${INSTALL_PREFIX}/web"
        mv "${INSTALL_PREFIX}/web.new" "${INSTALL_PREFIX}/web"
    else
        warn "npm 未安装，跳过前端升级"
    fi
}

# 5. 升级配置模板（不覆盖已有）
upgrade_configs() {
    log "更新配置模板（不覆盖已有配置）..."
    for f in backend.yaml grpc.yaml edge.yaml dns-sync.yaml log-server.yaml; do
        if [[ -f "${PROJECT_ROOT}/deploy/${f}" && ! -f "/etc/shieldflow/${f}" ]]; then
            cp "${PROJECT_ROOT}/deploy/${f}" "/etc/shieldflow/${f}"
            warn "已部署新配置 ${f}，请检查"
        fi
    done
    # 更新 supervisor/nginx 模板
    for d in /etc/supervisord.d /etc/supervisor/conf.d; do
        if [[ -d "$d" ]]; then
            for f in shieldflow-master shieldflow-edge shieldflow-dns-sync shieldflow-log-server; do
                [[ -f "${PROJECT_ROOT}/deploy/supervisor/${f}.conf" ]] && cp "${PROJECT_ROOT}/deploy/supervisor/${f}.conf" "${d}/" 2>/dev/null || true
            done
        fi
    done
    [[ -f "${PROJECT_ROOT}/deploy/nginx/shieldflow.conf" ]] && cp "${PROJECT_ROOT}/deploy/nginx/shieldflow.conf" /etc/nginx/conf.d/ 2>/dev/null || true
}

# 6. 执行新 SQL 迁移
run_sql_migrations() {
    log "检查 SQL 迁移..."
    local sql_dir="${PROJECT_ROOT}/sql"
    if [[ -d "$sql_dir" ]]; then
        for f in "$sql_dir"/*.sql; do
            [[ -f "$f" ]] || continue
            case "$(basename "$f")" in
                *clickhouse*)
                    clickhouse-client --multiquery < "$f" 2>/dev/null || warn "ClickHouse SQL $(basename "$f") 执行有警告"
                    ;;
                *)
                    sudo -u postgres psql -d shieldflow_cdn -f "$f" 2>/dev/null || warn "PostgreSQL SQL $(basename "$f") 执行有警告"
                    ;;
            esac
        done
    fi
}

# 7. 滚动重启
restart_services() {
    log "重启服务..."
    supervisorctl reread 2>/dev/null || true
    supervisorctl update 2>/dev/null || true
    for group in shieldflow-master shieldflow-dns-sync shieldflow-log-server; do
        supervisorctl restart "${group}:*" 2>/dev/null || warn "重启 ${group} 失败"
    done
    nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || warn "Nginx reload 失败"
}

main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   ShieldFlow 升级脚本${NC}"
    echo -e "${BLUE}========================================${NC}"
    backup_current
    pull_code
    build
    build_web
    upgrade_configs
    run_sql_migrations
    restart_services
    echo ""
    log "升级完成！"
    log "查看服务状态: supervisorctl status"
    log "如需回滚: 从 ${BACKUP_DIR}/ 恢复旧版本二进制"
}

main "$@"
