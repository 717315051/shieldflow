#!/bin/bash
# ============================================================
# ShieldFlow CDN - A 方案主脚本
# 主公授权: A — 把后端打到 100% 闭环
# ============================================================
# 步骤:
#   1. 把 /root/shieldflow 源码 sync 到 master
#   2. master 上 Go 1.22.10 重 build 5 个 binary
#   3. 重启 4 个 systemd 服务
#   4. 跑 e2e-backend-fixture.py 创建数据
#   5. 跑 165 路径终极 smoke
#   6. 同步源码到 GitHub
# ============================================================
set -e

cd /root/shieldflow

echo "=========================================="
echo "ShieldFlow CDN - A 方案: 后端 100% 闭环"
echo "=========================================="

# ---- 1. Sync source to master ----
echo ""
echo ">>> 1. Sync source to master"
# scp the new/updated files
SSH="sshpass -p <MASTER_SSH_PASS_REDACTED> ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR"
SCP="sshpass -p <MASTER_SSH_PASS_REDACTED> scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -C"

# All changed files
for f in internal/handlers/router.go \
         internal/handlers/whitelist_helper.go \
         internal/handlers/package_market_helper.go \
         scripts/e2e-backend-fixture.py; do
    [ -f "$f" ] || continue
    $SCP "$f" "root@82.158.224.144:/opt/shieldflow-src/$f" || true
done
# also build-all.sh
$SCP bin/build-all.sh root@82.158.224.144:/opt/shieldflow-src/bin/build-all.sh

# ---- 2. Build all binaries on master ----
echo ""
echo ">>> 2. Build all on master"
sshpass -p <MASTER_SSH_PASS_REDACTED> ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR root@82.158.224.144 'chmod +x /opt/shieldflow-src/bin/build-all.sh && /opt/shieldflow-src/bin/build-all.sh' 2>&1 | tail -40

# ---- 3. Verify backend up ----
echo ""
echo ">>> 3. Verify backend"
sleep 2
curl -sS --max-time 5 http://82.158.224.144:8080/api/v1/health
echo

# ---- 4. Run e2e fixture (create real data) ----
echo ""
echo ">>> 4. Run e2e fixture (create real data, fill all id=1 etc)"
python3 scripts/e2e-backend-fixture.py 2>&1 | tail -80

# ---- 5. Final smoke ----
echo ""
echo ">>> 5. Final 165-path smoke"
# (Already part of fixture STEP 15)
# We just print the summary

# ---- 6. GitHub sync ----
echo ""
echo ">>> 6. GitHub sync"
chmod +x scripts/sync-to-github.sh
bash scripts/sync-to-github.sh 2>&1 | tail -20

echo ""
echo "=========================================="
echo "A 方案完成"
echo "=========================================="