#!/usr/bin/env bash
# ============================================================================
# ShieldFlow Docker 容器入口脚本
# 用法: docker run shieldflow/shieldflow <mode>
#   mode=master   → 启动 backend + grpc-server (主控)
#   mode=edge     → 启动 edge (边缘节点)
#   mode=all      → 启动全部组件
#   mode=backend  → 仅 backend
# ============================================================================
set -e

MODE="${1:-master}"
CONF_DIR="/etc/shieldflow"
DEPLOY_DIR="/opt/shieldflow/deploy"

# 首次启动：从模板生成配置
init_configs() {
    for f in backend.yaml grpc.yaml edge.yaml dns-sync.yaml log-server.yaml; do
        if [[ ! -f "${CONF_DIR}/${f}" && -f "${DEPLOY_DIR}/${f}" ]]; then
            cp "${DEPLOY_DIR}/${f}" "${CONF_DIR}/${f}"
            echo "[entrypoint] 部署默认配置: ${f}"
        fi
    done
    # 生成自签名证书
    if [[ ! -f /etc/shieldflow/certs/shieldflow.crt ]]; then
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /etc/shieldflow/certs/shieldflow.key \
            -out /etc/shieldflow/certs/shieldflow.crt \
            -subj "/CN=shieldflow/O=ShieldFlow" 2>/dev/null
        echo "[entrypoint] 生成自签名证书"
    fi
}

# 启动 supervisor 守护进程
start_supervisor() {
    if [[ ! -f /var/run/supervisord.pid ]]; then
        /usr/bin/supervisord -c /etc/supervisor/supervisord.conf 2>/dev/null || \
        /usr/bin/supervisord -n &
        sleep 1
    fi
}

init_configs

# 部署 supervisor 配置
case "$MODE" in
    master|all)
        cp "${DEPLOY_DIR}/supervisor/shieldflow-master.conf" /etc/supervisor/conf.d/ 2>/dev/null || true
        ;;
esac
case "$MODE" in
    all)
        for f in shieldflow-master shieldflow-edge shieldflow-dns-sync shieldflow-log-server; do
            cp "${DEPLOY_DIR}/supervisor/${f}.conf" /etc/supervisor/conf.d/ 2>/dev/null || true
        done
        ;;
    edge)
        cp "${DEPLOY_DIR}/supervisor/shieldflow-edge.conf" /etc/supervisor/conf.d/ 2>/dev/null || true
        ;;
    backend)
        # 仅 backend：从 master 配置提取
        sed -n '/\[program:shieldflow-backend\]/,/^$/p' "${DEPLOY_DIR}/supervisor/shieldflow-master.conf" \
            > /etc/supervisor/conf.d/shieldflow-backend.conf 2>/dev/null || true
        ;;
esac

# 确保 supervisor 主配置 include
mkdir -p /etc/supervisor/conf.d
if [[ -f /etc/supervisor/supervisord.conf ]]; then
    grep -q 'conf.d' /etc/supervisor/supervisord.conf || \
    printf '\n[include]\nfiles = /etc/supervisor/conf.d/*.conf\n' >> /etc/supervisor/supervisord.conf
fi

start_supervisor
supervisorctl reread 2>/dev/null || true
supervisorctl update 2>/dev/null || true

echo "[entrypoint] ShieldFlow 启动完成 (mode=${MODE})"
echo "[entrypoint] 查看服务: supervisorctl status"

# 容器保持运行：tail 日志
exec tail -f /var/log/shieldflow/*.log 2>/dev/null || sleep infinity
