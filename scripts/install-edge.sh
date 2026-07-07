#!/usr/bin/env bash
# ============================================================================
# ShieldFlow 边缘节点安装脚本
# 用法: sudo bash scripts/install-edge.sh --node-id <ID> --master <IP:PORT> [选项]
# 功能: 安装 edge 二进制 → 配置 edge.yaml → 注册 Supervisor → 连接主控验证
# ============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ShieldFlow-EDGE]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INSTALL_PREFIX="/usr/local/shieldflow"
CONF_DIR="/etc/shieldflow"
LOG_DIR="/var/log/shieldflow"
CACHE_DIR="/var/cache/shieldflow"

NODE_ID=""
MASTER_ADDR=""
NODE_REGION="${NODE_REGION:-cn-east-1}"
LICENSE_KEY="${LICENSE_KEY:-}"
NODE_TOKEN="${NODE_TOKEN:-}"
HTTP_PORT="${HTTP_PORT:-80}"
HTTPS_PORT="${HTTPS_PORT:-443}"
SKIP_BUILD="${SKIP_BUILD:-false}"
BINARY_ONLY="${BINARY_ONLY:-false}"

[[ $EUID -eq 0 ]] || die "请以 root 身份运行"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --node-id)    NODE_ID="$2"; shift 2 ;;
        --master)     MASTER_ADDR="$2"; shift 2 ;;
        --region)     NODE_REGION="$2"; shift 2 ;;
        --license)    LICENSE_KEY="$2"; shift 2 ;;
        --token)      NODE_TOKEN="$2"; shift 2 ;;
        --http-port)  HTTP_PORT="$2"; shift 2 ;;
        --https-port) HTTPS_PORT="$2"; shift 2 ;;
        --skip-build) SKIP_BUILD="true"; shift ;;
        --binary-only) BINARY_ONLY="true"; shift ;;
        --help|-h)
            cat <<EOF
用法: $0 --node-id <ID> --master <IP:PORT> [选项]
选项:
  --node-id <ID>        节点 ID（必填，唯一）
  --master <IP:PORT>    主控 gRPC 地址，如 1.2.3.4:50051（必填）
  --region <region>     节点区域 (默认: cn-east-1)
  --license <key>       License Key
  --token <token>       节点认证 Token
  --http-port <port>    HTTP 端口 (默认: 80)
  --https-port <port>   HTTPS 端口 (默认: 443)
  --skip-build          跳过编译（已有二进制时使用）
  --binary-only         仅安装二进制（不配置 supervisor）
EOF
            exit 0 ;;
        *) die "未知参数: $1" ;;
    esac
done

[[ -n "$NODE_ID" ]]    || die "缺少 --node-id"
[[ -n "$MASTER_ADDR" ]] || die "缺少 --master"

# ---------- 1. 系统检查 ----------
check_system() {
    log "检查系统..."
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        case "$ID" in
            centos|rhel|rocky|almalinux|ubuntu|debian) log "系统: $ID $VERSION_ID ✓" ;;
            *) die "不支持的操作系统: $ID" ;;
        esac
    else
        die "无法识别操作系统"
    fi
}

# ---------- 2. 安装依赖 ----------
install_deps() {
    log "安装依赖..."
    if command -v yum &>/dev/null; then
        yum install -y supervisor 2>/dev/null || true
    elif command -v apt-get &>/dev/null; then
        apt-get update -qq 2>/dev/null || true
        apt-get install -y supervisor 2>/dev/null || true
    fi
    command -v supervisorctl &>/dev/null || warn "supervisor 未安装，请手动安装"
}

# ---------- 3. 创建目录 ----------
create_dirs() {
    log "创建目录..."
    for d in "$CONF_DIR" "$CONF_DIR/certs" "$LOG_DIR" "$CACHE_DIR" "$INSTALL_PREFIX/bin"; do
        mkdir -p "$d"
    done
}

# ---------- 4. 编译或复制 edge 二进制 ----------
install_binary() {
    if [[ "$SKIP_BUILD" == "true" ]] && [[ -f "${INSTALL_PREFIX}/bin/edge" ]]; then
        log "跳过编译，使用已有 edge 二进制"
        return
    fi
    if [[ -f "${PROJECT_ROOT}/cmd/edge/main.go" ]]; then
        log "编译 edge 二进制..."
        command -v go &>/dev/null || die "Go 未安装"
        cd "$PROJECT_ROOT"
        export PATH=$PATH:/usr/local/go/bin
        export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
        go build -ldflags="-s -w" -o "${INSTALL_PREFIX}/bin/edge" "./cmd/edge" || die "编译 edge 失败"
        log "edge 二进制编译完成"
    else
        die "找不到 edge 源码 (${PROJECT_ROOT}/cmd/edge/)，且无预编译二进制"
    fi
}

# ---------- 5. 生成 edge.yaml ----------
generate_config() {
    log "生成 edge.yaml..."
    local template="${PROJECT_ROOT}/deploy/edge.yaml"
    local conf="${CONF_DIR}/edge.yaml"
    if [[ -f "$template" ]]; then
        cp "$template" "$conf"
    else
        # 内联生成最小配置
        cat > "$conf" <<EOF
node:
  id: "${NODE_ID}"
  region: "${NODE_REGION}"
  license_key: "${LICENSE_KEY}"
grpc:
  server: "${MASTER_ADDR}"
  tls: false
  token: "${NODE_TOKEN}"
proxy:
  http_port: ${HTTP_PORT}
  https_port: ${HTTPS_PORT}
  cert_file: /etc/shieldflow/certs/edge.crt
  key_file: /etc/shieldflow/certs/edge.key
waf:
  enabled: true
  mode: block
  threshold: 60
cache:
  enabled: true
  path: ${CACHE_DIR}
  max_size: "50GB"
  ttl: "10m"
  compress: gzip
ddos:
  enabled: true
  max_connections_per_ip: 100
cc:
  enabled: true
  global_rate_limit: 1000
  global_window: "1s"
  challenge_type: js
bot:
  enabled: true
  allow_search_engines: true
  block_scanners: true
  block_scrapers: false
  block_no_ua: true
origins: []
EOF
    fi
    # 替换占位符
    sed -i "s/edge-default-01/${NODE_ID}/g" "$conf"
    sed -i "s/cn-east-1/${NODE_REGION}/g" "$conf"
    sed -i "s|127.0.0.1:50051|${MASTER_ADDR}|g" "$conf"
    sed -i "s/CHANGE_ME_LICENSE_KEY/${LICENSE_KEY}/g" "$conf"
    sed -i "s/CHANGE_ME_NODE_TOKEN/${NODE_TOKEN}/g" "$conf"
    sed -i "s/http_port: 80/http_port: ${HTTP_PORT}/g" "$conf"
    sed -i "s/https_port: 443/https_port: ${HTTPS_PORT}/g" "$conf"
    chmod 640 "$conf"
    log "配置已生成: ${conf}"
}

# ---------- 6. 生成自签名证书 ----------
gen_certs() {
    if [[ -f "${CONF_DIR}/certs/edge.crt" ]]; then
        log "证书已存在，跳过"
        return
    fi
    log "生成自签名证书..."
    mkdir -p "${CONF_DIR}/certs"
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "${CONF_DIR}/certs/edge.key" \
        -out "${CONF_DIR}/certs/edge.crt" \
        -subj "/CN=${NODE_ID}/O=ShieldFlow" 2>/dev/null || warn "证书生成失败"
}

# ---------- 7. 注册 Supervisor ----------
setup_supervisor() {
    [[ "$BINARY_ONLY" == "true" ]] && { log "仅二进制模式，跳过 Supervisor"; return; }
    log "注册 Supervisor..."
    local sup_dir
    for d in /etc/supervisord.d /etc/supervisor/conf.d; do
        [[ -d "$d" ]] && sup_dir="$d" && break
    done
    [[ -z "${sup_dir:-}" ]] && sup_dir="/etc/supervisord.d" && mkdir -p "$sup_dir"
    # 部署 edge supervisor 配置
    if [[ -f "${PROJECT_ROOT}/deploy/supervisor/shieldflow-edge.conf" ]]; then
        cp "${PROJECT_ROOT}/deploy/supervisor/shieldflow-edge.conf" "${sup_dir}/"
    else
        cat > "${sup_dir}/shieldflow-edge.conf" <<EOF
[group:shieldflow-edge]
programs=shieldflow-edge
priority=10

[program:shieldflow-edge]
command=${INSTALL_PREFIX}/bin/edge -config ${CONF_DIR}/edge.yaml
directory=${INSTALL_PREFIX}
user=root
autostart=true
autorestart=true
startsecs=5
stdout_logfile=${LOG_DIR}/edge.stdout.log
stderr_logfile=${LOG_DIR}/edge.stderr.log
stdout_logfile_maxbytes=200MB
stdout_logfile_backups=20
stderr_logfile_maxbytes=200MB
stderr_logfile_backups=20
EOF
    fi
    systemctl enable supervisor 2>/dev/null || supervisord 2>/dev/null || true
    systemctl start supervisor 2>/dev/null || true
    supervisorctl reread 2>/dev/null || true
    supervisorctl update 2>/dev/null || true
    sleep 3
    supervisorctl start "shieldflow-edge:shieldflow-edge" 2>/dev/null || true
    log "Supervisor 服务已注册"
}

# ---------- 8. 连接主控验证 ----------
verify_master() {
    log "验证主控连接..."
    local host="${MASTER_ADDR%%:*}"
    local port="${MASTER_ADDR##*:}"
    [[ "$port" == "$MASTER_ADDR" ]] && port=50051
    if command -v nc &>/dev/null; then
        if nc -z -w 5 "$host" "$port" 2>/dev/null; then
            log "主控 ${MASTER_ADDR} 连接成功 ✓"
        else
            warn "无法连接主控 ${MASTER_ADDR}，请检查网络/防火墙（端口 ${port}）"
        fi
    else
        warn "nc 未安装，跳过连通性测试"
    fi
    # 检查本地健康检查端口
    sleep 2
    if command -v curl &>/dev/null; then
        if curl -s -m 3 "http://127.0.0.1:${HTTP_PORT}/ping" 2>/dev/null | grep -q "ok\|pong\|alive"; then
            log "Edge 健康检查正常 ✓"
        else
            warn "Edge 健康检查未就绪（可能正在启动）"
        fi
    fi
}

# ---------- 9. 防火墙 ----------
setup_firewall() {
    for port in "$HTTP_PORT" "$HTTPS_PORT" 9527; do
        if command -v firewall-cmd &>/dev/null; then
            firewall-cmd --permanent --add-port=${port}/tcp 2>/dev/null || true
        elif command -v ufw &>/dev/null; then
            ufw allow ${port}/tcp 2>/dev/null || true
        fi
    done
    command -v firewall-cmd &>/dev/null && firewall-cmd --reload 2>/dev/null || true
    log "防火墙已配置 (端口 ${HTTP_PORT}/${HTTPS_PORT}/9527)"
}

main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   ShieldFlow 边缘节点安装${NC}"
    echo -e "${BLUE}   节点ID: ${NODE_ID}${NC}"
    echo -e "${BLUE}   主控:   ${MASTER_ADDR}${NC}"
    echo -e "${BLUE}========================================${NC}"
    check_system
    install_deps
    create_dirs
    install_binary
    gen_certs
    generate_config
    setup_supervisor
    setup_firewall
    verify_master
    echo ""
    log "边缘节点安装完成!"
    log "节点ID: ${NODE_ID}  区域: ${NODE_REGION}"
    log "管理: supervisorctl status shieldflow-edge:shieldflow-edge"
    log "日志: tail -f ${LOG_DIR}/edge.stderr.log"
}

main "$@"
