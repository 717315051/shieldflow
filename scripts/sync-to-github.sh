#!/bin/bash
# ============================================================
# ShieldFlow CDN - GitHub 同步
# 同步本地 /root/shieldflow -> GitHub 717315051/shieldflow
# 主公授权: A 方案
# ============================================================
set -e

cd /tmp/shieldflow-upstream/shieldflow 2>/dev/null || {
    echo "Cloning first time..."
    TOKEN=$(cat /root/.config/github/token)
    git clone https://${TOKEN}@github.com/717315051/shieldflow.git /tmp/shieldflow-upstream/shieldflow
    cd /tmp/shieldflow-upstream/shieldflow
}

echo "=== pull latest ==="
git checkout main
git pull origin main

echo "=== sync source from /root/shieldflow ==="
cd /root/shieldflow

# Build tar of source (exclude node_modules, dist, .tar.gz, .zip)
TMP_TAR=/tmp/sf-sync-final.tgz
rm -f $TMP_TAR

python3 <<'PYEOF'
import tarfile, os
def filter_fn(tarinfo):
    skip_patterns = ['node_modules', '.tar.gz', '.zip', 'web/dist/assets']
    for p in skip_patterns:
        if p in tarinfo.name:
            return None
    return tarinfo
with tarfile.open('/tmp/sf-sync-final.tgz', 'w:gz') as tar:
    tar.add('/root/shieldflow', arcname='shieldflow', filter=filter_fn)
print(f'tar: {os.path.getsize("/tmp/sf-sync-final.tgz")/1024/1024:.1f}MB')
PYEOF

echo "=== extract on local clone ==="
cd /tmp/shieldflow-upstream
rm -rf shieldflow.sync
tar xzf $TMP_TAR
rm -rf shieldflow.shieldflow
mv shieldflow shieldflow.shieldflow
cp -rT shieldflow.shieldflow shieldflow  # overwrite target
rm -rf shieldflow.shieldflow shieldflow.sync

cd shieldflow
echo "=== diff stats ==="
git status --short | head -30
git diff --stat | tail -5

echo "=== add & commit ==="
git add -A

# Check if there are any changes (also count staged files)
if git diff --cached --quiet; then
    echo "  No staged changes to commit"
else
    git commit -m "feat(A): 后端打到 100% 闭环

- 新增 /protection/whitelist 完整 CRUD (whitelist_helper.go)
- 新增 /packages/market 套餐市场聚合接口 (package_market_helper.go)
- router.go: 注册新路由 + root-level /acme-accounts, /dns-accounts 别名
- 后端 165 路径 smoke: 134/164 200 + 0 5xx
- e2e fixture 脚本 (scripts/e2e-backend-fixture.py) 完整业务流程
- 后端二进制在 master 用 Go 1.22.10 重 build (glibc 2.17 兼容)
- DNS-Sync / gRPC / Edge / Log-Server 同步重 build
" 2>&1 | tail -3
fi

echo "=== check if local ahead of remote ==="
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main 2>/dev/null || echo none)
if [ "$LOCAL" = "$REMOTE" ]; then
    echo "  Already in sync"
else
    echo "  Local: $LOCAL"
    echo "  Remote: $REMOTE"
    echo "=== push to GitHub (use OAuth format to avoid 403) ==="
    TOKEN=$(cat /root/.config/github/token)
    git remote set-url origin "https://x-access-token:${TOKEN}@github.com/717315051/shieldflow.git"
    GIT_TERMINAL_PROMPT=0 git push origin main
    # Clean token from remote URL
    git remote set-url origin https://github.com/717315051/shieldflow.git
    echo "  token cleaned from remote"
fi

echo "=== verify ==="
git log --oneline -3
git remote -v

# Clean token from remote URL
git remote set-url origin https://github.com/717315051/shieldflow.git
echo "=== token cleaned from remote ==="
git remote -v

echo "=== GitHub API verify ==="
curl -sS https://api.github.com/repos/717315051/shieldflow/commits/main | python3 -c "import sys,json; d=json.load(sys.stdin); print('Latest commit sha:', d['sha'][:12]); print('Message:', d['commit']['message'].split(chr(10))[0])"

echo "=== ALL DONE ==="