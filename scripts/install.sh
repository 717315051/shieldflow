#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 一键安装脚本
# 用法: sudo bash scripts/install.sh
# 功能: 系统检查 → 安装依赖 → 编译 → 部署配置 → 初始化数据库 → 注册服务 → 防火墙
# ============================================================================
set -euo pipefail

# ---------- 颜色 ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$*"; exit 1; }

# ---------- 变量 ----------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INSTALL_PREFIX="/usr/local/shieldflow"
CONF_DIR="/etc/shieldflow"
LOG_DIR="/var/log/shieldflow"
CACHE_DIR="/var/cache/shieldflow"
BACKUP_DIR="/var/backups/shieldflow"
DB_NAME="shieldflow_cdn"
DB_USER="shieldflow"
DB_PASS="${ShieldFlow_DB_PASS:-$(openssl rand -hex 16 2>/dev/null || echo 'change_me_123')}"
JWT_SECRET="${ShieldFlow_JWT_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo 'change_me_jwt_secret')}"
ADMIN_USER="${ShieldFlow_ADMIN_USER:-admin}"
ADMIN_PASS="${ShieldFlow_ADMIN_PASS:-admin123}"

[[ $EUID -eq 0 ]] || die "请以 root 身份运行此脚本"

# ---------- 1. 系统要求检查 ----------
check_system() {
    log "检查系统环境..."
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        log "操作系统: $ID $VERSION_ID"
        case "$ID" in
            centos|rhel|rocky|almalinux)
                [[ "${VERSION_ID%%.*}" -ge 7 ]] || die "需要 CentOS 7+"
                OS_FAMILY="rhel"
                ;;
            ubuntu)
                [[ "${VERSION_ID%%.*}" -ge 18 ]] || die "需要 Ubuntu 18+"
                OS_FAMILY="debian"
                ;;
            debian)
                [[ "${VERSION_ID%%.*}" -ge 10 ]] || die "需要 Debian 10+"
                OS_FAMILY="debian"
                ;;
            *) die "不支持的操作系统: $ID"
                ;;
        esac
    else
        die "无法识别操作系统"
    fi
    log "系统检查通过 ($OS_FAMILY)"
}

# ---------- 2. 安装依赖 ----------
install_deps() {
    log "安装系统依赖..."
    if [[ "$OS_FAMILY" == "rhel" ]]; then
        yum install -y epel-release >/dev/null 2>&1 || true
        yum groupinstall -y "Development Tools" >/dev/null 2>&1 || true
        yum install -y git wget curl make openssl-devel gcc clang \
            postgresql-server postgresql-contrib redis nginx \
            supervisor clickhouse-server clickhouse-client >/dev/null 2>&1 || warn "部分依赖安装失败，请检查 yum 源"
    else
        apt-get update -qq
        apt-get install -y build-essential git wget curl make openssl \
            postgresql postgresql-contrib redis-server nginx \
            supervisor clickhouse-server clickhouse-client >/dev/null 2>&1 || warn "部分依赖安装失败，请检查 apt 源"
    fi
}

# ---------- 3. 安装 Go ----------
install_go() {
    if command -v go &>/dev/null && go version | grep -qE 'go1\.(2[2-9]|[3-9])|go[2-9]'; then
        log "Go 已安装: $(go version)"
        return
    fi
    log "安装 Go 1.22+..."
    local go_ver="1.22.5"
    local go_arch="amd64"
    [[ "$(uname -m)" == "aarch64" ]] && go_arch="arm64"
    cd /tmp
    wget -q "https://go.dev/dl/go${go_ver}.linux-${go_arch}.tar.gz" || die "Go 下载失败"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${go_ver}.linux-${go_arch}.tar.gz"
    export PATH=$PATH:/usr/local/go/bin
    grep -q '/usr/local/go/bin' /etc/profile || echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    log "Go $(go version) 安装完成"
}

# ---------- 4. 安装 Node.js ----------
install_node() {
    if command -v node &>/dev/null && node -v | grep -qE 'v(1[8-9]|[2-9][0-9])'; then
        log "Node.js 已安装: $(node -v)"
        return
    fi
    log "安装 Node.js 18+..."
    cd /tmp
    local node_arch="x64"
    [[ "$(uname -m)" == "aarch64" ]] && node_arch="arm64"
    wget -q "https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-${node_arch}.tar.xz" || die "Node.js 下载失败"
    tar -C /usr/local --strip-components=1 -xJf "node-v18.20.4-linux-${node_arch}.tar.xz"
    log "Node.js $(node -v) 安装完成"
}

# ---------- 5. 编译后端 ----------
build_backend() {
    log "编译后端组件..."
    cd "$PROJECT_ROOT"
    export PATH=$PATH:/usr/local/go/bin
    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    mkdir -p "${INSTALL_PREFIX}/bin"
    for comp in backend grpc-server edge dns-sync log-server; do
        log "  编译 $comp..."
        go build -ldflags="-s -w" -o "${INSTALL_PREFIX}/bin/${comp}" "./cmd/${comp}" || die "编译 $comp 失败"
    done
    log "后端编译完成: ${INSTALL_PREFIX}/bin/"
}

# ---------- 6. 编译前端 ----------
build_frontend() {
    log "编译前端..."
    cd "${PROJECT_ROOT}/web"
    command -v npm &>/dev/null || die "npm 未安装"
    npm install --silent 2>/dev/null || npm install || die "npm install 失败"
    npm run build 2>/dev/null || die "前端构建失败"
    rm -rf "${INSTALL_PREFIX}/web"
    cp -r dist "${INSTALL_PREFIX}/web"
    log "前端构建完成: ${INSTALL_PREFIX}/web/"
}

# ---------- 7. 创建目录 ----------
create_dirs() {
    log "创建运行目录..."
    for d in "$CONF_DIR" "$LOG_DIR" "$CACHE_DIR" "$BACKUP_DIR" "${INSTALL_PREFIX}" "${CONF_DIR}/certs"; do
        mkdir -p "$d"
    done
    chmod 750 "$CONF_DIR"
    chmod 750 "$LOG_DIR"
}

# ---------- 8. 部署配置文件模板 ----------
deploy_configs() {
    log "部署配置文件模板..."
    # 已存在则备份
    for f in backend.yaml grpc.yaml edge.yaml dns-sync.yaml log-server.yaml; do
        if [[ -f "${CONF_DIR}/${f}" ]]; then
            cp "${CONF_DIR}/${f}" "${CONF_DIR}/${f}.bak.$(date +%s)"
        fi
        cp "${PROJECT_ROOT}/deploy/${f}" "${CONF_DIR}/${f}" 2>/dev/null || warn "模板 ${f} 不存在"
        chmod 640 "${CONF_DIR}/${f}"
    done
    # 用生成的密码替换占位符
    sed -i "s/CHANGE_ME_STRONG_PASSWORD/${DB_PASS}/g" "${CONF_DIR}"/{backend,grpc,dns-sync}.yaml 2>/dev/null || true
    sed -i "s/CHANGE_ME_TO_RANDOM_64_CHAR_STRING/${JWT_SECRET}/g" "${CONF_DIR}"/{backend,grpc}.yaml 2>/dev/null || true
    log "配置文件已部署到 ${CONF_DIR}/"
}

# ---------- 9. 创建数据库和用户 ----------
init_database() {
    log "初始化 PostgreSQL 数据库..."
    # 启动 PostgreSQL
    if ! systemctl is-active --quiet postgresql 2>/dev/null; then
        if [[ "$OS_FAMILY" == "rhel" ]]; then
            postgresql-setup --initdb 2>/dev/null || true
        else
            pg_ctlcluster 14 main start 2>/dev/null || pg_ctlcluster 15 main start 2>/dev/null || true
        fi
        systemctl enable postgresql >/dev/null 2>&1 || true
        systemctl start postgresql 2>/dev/null || true
    fi
    sleep 2
    if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" 2>/dev/null | grep -q 1; then
        sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME};" 2>/dev/null || warn "创建数据库失败（可能已存在）"
    fi
    sudo -u postgres psql -c "DROP USER IF EXISTS ${DB_USER};" 2>/dev/null || true
    sudo -u postgres psql -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASS}';" 2>/dev/null || warn "创建用户失败"
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};" 2>/dev/null || true
    sudo -u postgres psql -d "${DB_NAME}" -c "GRANT ALL ON SCHEMA public TO ${DB_USER};" 2>/dev/null || true
    log "PostgreSQL 数据库 ${DB_NAME} 初始化完成"
}

# ---------- 10. 执行 SQL 初始化 ----------
init_sql() {
    log "执行 SQL 初始化脚本..."
    local sql_dir="${PROJECT_ROOT}/sql"
    if [[ -f "${sql_dir}/001_init_postgresql.sql" ]]; then
        sudo -u postgres psql -d "${DB_NAME}" -f "${sql_dir}/001_init_postgresql.sql" 2>/dev/null \
            || warn "PostgreSQL SQL 执行有警告（可能部分已存在）"
    fi
    # ClickHouse
    if command -v clickhouse-client &>/dev/null; then
        systemctl enable clickhouse-server >/dev/null 2>&1 || true
        systemctl start clickhouse-server 2>/dev/null || true
        sleep 2
        if [[ -f "${sql_dir}/002_init_clickhouse.sql" ]]; then
            clickhouse-client --multiquery < "${sql_dir}/002_init_clickhouse.sql" 2>/dev/null \
                || warn "ClickHouse SQL 执行失败（可能服务未就绪）"
        fi
    else
        warn "clickhouse-client 未安装，跳过 ClickHouse 初始化"
    fi
}

# ---------- 11. 注册 Supervisor 服务 ----------
setup_supervisor() {
    log "注册 Supervisor 服务..."
    systemctl enable supervisor >/dev/null 2>&1 || supervisord 2>/dev/null || true
    systemctl start supervisor 2>/dev/null || true
    local sup_dir
    for d in /etc/supervisord.d /etc/supervisor/conf.d; do
        [[ -d "$d" ]] && sup_dir="$d" && break
    done
    [[ -z "${sup_dir:-}" ]] && sup_dir="/etc/supervisord.d" && mkdir -p "$sup_dir"
    # 确保主配置 include
    for mainconf in /etc/supervisord.conf /etc/supervisor/supervisord.conf; do
        if [[ -f "$mainconf" ]] && ! grep -q "${sup_dir}" "$mainconf" 2>/dev/null; then
            echo "[include]
files = ${sup_dir}/*.conf" >> "$mainconf"
        fi
    done
    cp "${PROJECT_ROOT}/deploy/supervisor/shieldflow-master.conf" "${sup_dir}/" 2>/dev/null || true
    cp "${PROJECT_ROOT}/deploy/supervisor/shieldflow-dns-sync.conf" "${sup_dir}/" 2>/dev/null || true
    cp "${PROJECT_ROOT}/deploy/supervisor/shieldflow-log-server.conf" "${sup_dir}/" 2>/dev/null || true
    supervisorctl reread 2>/dev/null || true
    supervisorctl update 2>/dev/null || true
    log "Supervisor 服务已注册"
}

# ---------- 12. 配置 Nginx ----------
setup_nginx() {
    log "配置 Nginx 反向代理..."
    systemctl enable nginx >/dev/null 2>&1 || true
    local ngx_conf_dir="/etc/nginx/conf.d"
    [[ -d "$ngx_conf_dir" ]] || ngx_conf_dir="/etc/nginx/sites-enabled"
    mkdir -p "$ngx_conf_dir"
    cp "${PROJECT_ROOT}/deploy/nginx/shieldflow.conf" "${ngx_conf_dir}/shieldflow.conf" 2>/dev/null || true
    # 生成自签名证书（如果没有）
    if [[ ! -f /etc/shieldflow/certs/shieldflow.crt ]]; then
        mkdir -p /etc/shieldflow/certs
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /etc/shieldflow/certs/shieldflow.key \
            -out /etc/shieldflow/certs/shieldflow.crt \
            -subj "/CN=shieldflow/O=ShieldFlow" 2>/dev/null || warn "自签名证书生成失败"
    fi
    nginx -t 2>/dev/null && systemctl restart nginx 2>/dev/null || warn "Nginx 配置有误，请手动检查"
    log "Nginx 配置完成"
}

# ---------- 13. 防火墙配置 ----------
setup_firewall() {
    log "配置防火墙..."
    if command -v firewall-cmd &>/dev/null; then
        for port in 80 443 9527 50051; do
            firewall-cmd --permanent --add-port=${port}/tcp 2>/dev/null || true
        done
        firewall-cmd --reload 2>/dev/null || true
    elif command -v ufw &>/dev/null; then
        for port in 80 443 9527 50051; do
            ufw allow ${port}/tcp 2>/dev/null || true
        done
    else
        warn "未检测到防火墙工具（firewalld/ufw），请手动开放端口: 80 443 9527 50051"
    fi
    log "防火墙配置完成"
}

# ---------- 14. 启动 Redis ----------
setup_redis() {
    log "启动 Redis..."
    systemctl enable redis 2>/dev/null || systemctl enable redis-server 2>/dev/null || true
    systemctl start redis 2>/dev/null || systemctl start redis-server 2>/dev/null || true
}

# ---------- 主流程 ----------
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   ShieldFlow 一键安装脚本 v1.0${NC}"
    echo -e "${BLUE}========================================${NC}"
    check_system
    install_deps
    install_go
    install_node
    create_dirs
    build_backend
    build_frontend
    deploy_configs
    init_database
    init_sql
    setup_redis
    setup_supervisor
    setup_nginx
    setup_firewall

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   ShieldFlow 安装完成!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "  数据库:   ${DB_NAME} (用户 ${DB_USER})"
    echo -e "  配置目录: ${CONF_DIR}"
    echo -e "  日志目录: ${LOG_DIR}"
    echo -e "  程序目录: ${INSTALL_PREFIX}"
    echo -e "  管理后台: https://<服务器IP>"
    echo -e "  默认账号: ${ADMIN_USER} / ${ADMIN_PASS}"
    echo -e "  数据库密码已写入: ${CONF_DIR}/backend.yaml"
    echo ""
    echo -e "${YELLOW}  请尽快修改默认管理员密码并更换正式 TLS 证书!${NC}"
    echo -e "${YELLOW}  管理服务: supervisorctl status${NC}"
    echo -e "${YELLOW}  重载服务: supervisorctl reread && supervisorctl update${NC}"
}

main "$@"
