# 部署搭建教程

> ShieldFlow CDN 完整部署指南，涵盖一键脚本、Docker、手动三种方式。

## 部署架构

```
                         ┌─────────────────────────────────┐
                         │          ShieldFlow 主控          │
                         │  ┌──────────┐  ┌─────────────┐  │
                         │  │ Backend  │  │ gRPC Server │  │
                         │  │  :8080   │  │   :50051    │  │
                         │  └──────────┘  └─────────────┘  │
                         │  ┌──────────┐  ┌─────────────┐  │
                         │  │ DNS Sync │  │  Web 前端   │  │
                         │  │  :9528   │  │  (Nginx)   │  │
                         │  └──────────┘  └─────────────┘  │
                         │  PostgreSQL  ·  Redis  ·  CH    │
                         └──────────────┬──────────────────┘
                                        │ gRPC 下发配置
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
              ┌─────┴─────┐     ┌──────┴──────┐    ┌──────┴──────┐
              │  Edge 节点 │     │  Edge 节点  │    │  Edge 节点  │
              │  :80/:443  │     │  :80/:443   │    │  :80/:443   │
              │  (北京)    │     │  (上海)     │    │  (海外)     │
              └────────────┘     └─────────────┘    └─────────────┘
                                        │
                              ┌─────────┴─────────┐
                              │  日志服务器（可选）  │
                              │    :9529           │
                              │  ClickHouse 写入   │
                              └───────────────────┘
```

**组件说明**：

| 组件 | 端口 | 职责 | 部署位置 |
|------|------|------|---------|
| Backend API | 8080 | 管理后台接口 | 主控服务器 |
| gRPC Server | 50051 | 配置下发/日志收集 | 主控服务器 |
| DNS Sync | 9528 | DNS CNAME 同步 | 主控服务器 |
| Web 前端 | 80/443 | 管理界面 | 主控服务器 (Nginx) |
| Edge 节点 | 80/443 | CDN 加速+防护 | 各边缘服务器 |
| Log Server | 9529 | 日志接收+写入 | 独立服务器（可选） |
| PostgreSQL | 5432 | 配置数据 | 主控服务器 |
| ClickHouse | 9000 | 日志数据 | 主控/日志服务器 |
| Redis | 6379 | 缓存/会话 | 主控服务器 |

---

## 系统要求

### 硬件要求

| 角色 | CPU | 内存 | 磁盘 | 带宽 | 说明 |
|------|-----|------|------|------|------|
| 主控（最小） | 2核 | 4GB | 50GB | 10Mbps | 测试/小规模 |
| 主控（推荐） | 4核 | 8GB | 100GB | 50Mbps | 生产环境 |
| 边缘节点（最小） | 2核 | 2GB | 20GB | 10Mbps | 小流量站点 |
| 边缘节点（推荐） | 4核 | 4GB | 50GB | 100Mbps+ | 生产环境 |
| 日志服务器 | 4核 | 8GB | 500GB+ | 50Mbps | 大规模日志 |

### 软件要求

| 软件 | 最低版本 | 推荐版本 | 安装命令 |
|------|---------|---------|---------|
| OS | CentOS 7 / Ubuntu 18.04 | Ubuntu 22.04 | - |
| Go | 1.22 | 1.22+ | `wget https://go.dev/dl/go1.22.linux-amd64.tar.gz` |
| Node.js | 18 | 20 LTS | `curl -fsSL https://deb.nodesource.com/setup_20.x \| bash -` |
| PostgreSQL | 14 | 15+ | `apt install postgresql` |
| ClickHouse | 23.3 | 24+ | 参见下方安装步骤 |
| Redis | 6 | 7+ | `apt install redis-server` |
| Supervisor | 4+ | 4+ | `apt install supervisor` |
| Nginx | 1.18 | 1.24+ | `apt install nginx` |

### 网络端口

| 端口 | 服务 | 方向 | 说明 |
|------|------|------|------|
| 80 | Edge/HTTP | 公网入 | CDN 加速流量 |
| 443 | Edge/HTTPS | 公网入 | CDN 加密流量 |
| 8080 | Backend API | 内网 | 管理接口（建议不公开） |
| 50051 | gRPC Server | 内网→边缘 | 配置下发 |
| 9528 | DNS Sync | 内网 | DNS 同步 API |
| 9529 | Log Server | 边缘→日志服务器 | 日志上报 |
| 5432 | PostgreSQL | 内网 | 数据库 |
| 9000 | ClickHouse | 内网 | 日志数据库 |
| 6379 | Redis | 内网 | 缓存 |

---

## 方式一：一键脚本部署（推荐）

### 1.1 主控安装

```bash
# 下载项目
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 执行一键安装
sudo bash scripts/install.sh
```

安装脚本自动完成：
1. ✅ 系统检查（OS、内核、依赖）
2. ✅ 安装 Go、Node.js（如未安装）
3. ✅ 编译后端二进制（backend、grpc-server、dns-sync、log-server）
4. ✅ 编译前端（npm install + npm run build）
5. ✅ 安装 PostgreSQL、ClickHouse、Redis
6. ✅ 创建数据库和用户
7. ✅ 导入 SQL 初始化脚本
8. ✅ 复制配置文件到 `/etc/shieldflow/`
9. ✅ 复制二进制到 `/usr/local/shieldflow/bin/`
10. ✅ 注册 Supervisor 服务
11. ✅ 配置 Nginx 反向代理
12. ✅ 配置防火墙规则

**安装过程中需要交互输入**：
- PostgreSQL 数据库密码
- Redis 密码（可选）
- JWT Secret（可自动生成）
- 管理员初始密码

**安装完成后**：
```bash
# 检查服务状态
supervisorctl status shieldflow-master:*

# 访问管理后台
# http://<服务器IP>/
# 默认管理员: admin / <安装时设置的密码>
```

### 1.2 边缘节点安装

**方式 A：通过主控生成安装命令**

在管理后台 → 节点管理 → 添加节点 → 点击"安装"按钮，系统生成安装命令：

```bash
curl -fsSL http://<主控IP>:8080/api/v1/install/edge.sh | \
  NODE_ID=edge-01 \
  LICENSE_KEY=your-license-key \
  GRPC_SERVER=<主控IP>:50051 \
  bash
```

**方式 B：手动执行安装脚本**

```bash
# 在边缘节点服务器上
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

sudo bash scripts/install-edge.sh \
  --node-id edge-01 \
  --license-key your-license-key \
  --grpc-server <主控IP>:50051 \
  --region cn-east-1
```

边缘节点安装脚本自动完成：
1. ✅ 编译 edge 二进制
2. ✅ 复制配置到 `/etc/shieldflow/edge.yaml`
3. ✅ 注册 Supervisor 服务（shieldflow-edge）
4. ✅ 自动连接主控 gRPC 注册
5. ✅ 下载域名配置并启动代理

### 1.3 验证安装

```bash
# 主控服务状态
supervisorctl status shieldflow-master:*
# 预期输出:
# shieldflow-master:shieldflow-backend    RUNNING   pid 12345
# shieldflow-master:shieldflow-grpc-server RUNNING   pid 12346
# shieldflow-master:shieldflow-dns-sync   RUNNING   pid 12347

# 边缘节点状态
supervisorctl status shieldflow-edge:*
# 预期输出:
# shieldflow-edge:shieldflow-edge          RUNNING   pid 12345

# API 健康检查
curl http://localhost:8080/api/v1/health
# 预期: {"code":0,"message":"success","data":{"status":"ok"}}

# gRPC 端口检查
curl http://localhost:50051/health
# 预期: {"status":"ok"}

# 数据库连接
psql -U shieldflow -d shieldflow_cdn -c "SELECT count(*) FROM users;"
# 预期: 1 (默认管理员)

# ClickHouse 连接
clickhouse-client --query "SHOW TABLES FROM shieldflow_cdn;"
# 预期: 6 张日志表
```

---

## 方式二：Docker 部署

### 2.1 docker-compose 一键启动

```bash
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 修改配置（可选）
cp deploy/backend.yaml deploy/backend.yaml.bak
# 编辑 docker-compose.yml 中的环境变量

# 启动全部服务
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f backend
```

docker-compose.yml 包含以下服务：
- `postgres`: PostgreSQL 15
- `clickhouse`: ClickHouse 24
- `redis`: Redis 7
- `backend`: API 服务
- `grpc-server`: gRPC 服务
- `dns-sync`: DNS 同步
- `log-server`: 日志服务器
- `web`: Nginx + 前端静态文件

### 2.2 单独容器构建

```bash
# 构建镜像
docker build -t shieldflow:latest .

# 运行后端
docker run -d --name shieldflow-backend \
  -p 8080:8080 \
  -v /etc/shieldflow/backend.yaml:/etc/shieldflow/backend.yaml \
  -v /var/log/shieldflow:/var/log/shieldflow \
  shieldflow:latest \
  /usr/local/shieldflow/bin/backend -config /etc/shieldflow/backend.yaml

# 运行边缘节点
docker run -d --name shieldflow-edge \
  -p 80:80 -p 443:443 \
  -v /etc/shieldflow/edge.yaml:/etc/shieldflow/edge.yaml \
  shieldflow:latest \
  /usr/local/shieldflow/bin/edge -config /etc/shieldflow/edge.yaml
```

---

## 方式三：手动部署

### 3.1 编译二进制

```bash
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 编译全部
make build

# 或单独编译
go build -o bin/backend ./cmd/backend
go build -o bin/grpc-server ./cmd/grpc-server
go build -o bin/edge ./cmd/edge
go build -o bin/dns-sync ./cmd/dns-sync
go build -o bin/log-server ./cmd/log-server

# 编译前端
cd web && npm install && npm run build && cd ..

# 二进制文件在 bin/ 目录
ls -la bin/
```

### 3.2 安装数据库

#### PostgreSQL

```bash
# Ubuntu/Debian
apt update && apt install -y postgresql postgresql-contrib

# CentOS/RHEL
yum install -y postgresql-server postgresql-contrib
postgresql-setup --initdb

# 启动
systemctl enable postgresql
systemctl start postgresql

# 创建数据库和用户
sudo -u postgres psql <<EOF
CREATE USER shieldflow WITH PASSWORD 'YOUR_STRONG_PASSWORD';
CREATE DATABASE shieldflow_cdn OWNER shieldflow;
GRANT ALL PRIVILEGES ON DATABASE shieldflow_cdn TO shieldflow;
\c shieldflow_cdn
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
EOF

# 导入表结构
psql -U shieldflow -d shieldflow_cdn -f sql/001_init_postgresql.sql
```

#### ClickHouse

```bash
# 安装
apt-get install -y apt-transport-https ca-certificates dirmngr
GPG_KEY=$(curl -fsSL https://packages.clickhouse.com/rpm/lts/repodata/repomd.xml.key)
apt-key adv --keyserver keyserver.ubuntu.com --recv E0C56BD4

echo "deb https://packages.clickhouse.com/deb stable main" > /etc/apt/sources.list.d/clickhouse.list
apt-get update
apt-get install -y clickhouse-server clickhouse-client

# 启动
systemctl enable clickhouse-server
systemctl start clickhouse-server

# 创建数据库并导入
clickhouse-client --query "CREATE DATABASE IF NOT EXISTS shieldflow_cdn"
clickhouse-client --multiquery < sql/002_init_clickhouse.sql
```

#### Redis

```bash
apt install -y redis-server
systemctl enable redis-server
systemctl start redis-server
```

### 3.3 配置文件

```bash
# 创建目录
mkdir -p /etc/shieldflow /var/log/shieldflow /var/cache/shieldflow
mkdir -p /usr/local/shieldflow/bin

# 复制配置模板
cp deploy/backend.yaml /etc/shieldflow/
cp deploy/grpc.yaml /etc/shieldflow/
cp deploy/edge.yaml /etc/shieldflow/
cp deploy/dns-sync.yaml /etc/shieldflow/
cp deploy/log-server.yaml /etc/shieldflow/

# 复制二进制
cp bin/* /usr/local/shieldflow/bin/

# 编辑配置（必须修改密码等敏感信息）
vi /etc/shieldflow/backend.yaml
vi /etc/shieldflow/grpc.yaml
```

**必须修改的配置项**：

| 文件 | 配置项 | 说明 |
|------|--------|------|
| backend.yaml | `database.password` | PostgreSQL 密码 |
| backend.yaml | `jwt.secret` | JWT 签名密钥（随机64字符） |
| backend.yaml | `clickhouse.password` | ClickHouse 密码（如有） |
| backend.yaml | `redis.password` | Redis 密码（如有） |
| grpc.yaml | `database.password` | 同上 |
| edge.yaml | `node.license_key` | 节点 License |
| edge.yaml | `grpc.server` | 主控 gRPC 地址 |
| edge.yaml | `grpc.token` | 节点通信 Token |

### 3.4 配置 Supervisor

```bash
# 安装
apt install -y supervisor

# 复制配置
cp deploy/supervisor/*.conf /etc/supervisor/conf.d/

# 重载
supervisorctl reread
supervisorctl update

# 启动
supervisorctl start shieldflow-master:*
supervisorctl start shieldflow-edge:*
```

### 3.5 配置 Nginx

```bash
# 安装
apt install -y nginx

# 复制配置
cp deploy/nginx/shieldflow.conf /etc/nginx/sites-available/
ln -s /etc/nginx/sites-available/shieldflow.conf /etc/nginx/sites-enabled/

# 编辑配置（修改 server_name 和 SSL 证书路径）
vi /etc/nginx/sites-available/shieldflow.conf

# 测试配置
nginx -t

# 重载
systemctl reload nginx
```

Nginx 配置说明：
- 前端静态文件：`/usr/local/shieldflow/web/dist/`
- API 代理：`/api/` → `http://127.0.0.1:8080`
- gRPC 代理：`/grpc/` → `http://127.0.0.1:50051`（可选）
- SSL：建议用 Let's Encrypt 或自签证书

---

## 数据库初始化

### PostgreSQL

```bash
# 1. 创建用户和数据库
sudo -u postgres psql
CREATE USER shieldflow WITH PASSWORD 'YOUR_PASSWORD';
CREATE DATABASE shieldflow_cdn OWNER shieldflow;
\q

# 2. 导入表结构
psql -U shieldflow -d shieldflow_cdn -f sql/001_init_postgresql.sql

# 3. 验证
psql -U shieldflow -d shieldflow_cdn -c "\dt"
# 预期: 24 张表

# 4. 默认管理员
psql -U shieldflow -d shieldflow_cdn -c "SELECT username, role FROM users;"
# 预期: admin | admin
```

### ClickHouse

```bash
# 1. 创建数据库
clickhouse-client --query "CREATE DATABASE IF NOT EXISTS shieldflow_cdn"

# 2. 导入表结构
clickhouse-client --multiquery < sql/002_init_clickhouse.sql

# 3. 验证
clickhouse-client --query "SHOW TABLES FROM shieldflow_cdn"
# 预期: 6 张表
```

---

## 边缘节点独立部署

边缘节点通常部署在不同区域的服务器上。

### 步骤

```bash
# 1. 在主控管理后台添加节点
#    管理后台 → 节点管理 → 添加节点
#    填写: 节点名称、IP、区域、分组
#    获得: License Key

# 2. 在边缘服务器上安装
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 编译（只需 edge 组件）
go build -o bin/edge ./cmd/edge

# 3. 配置
mkdir -p /etc/shieldflow /var/log/shieldflow /var/cache/shieldflow
cp deploy/edge.yaml /etc/shieldflow/edge.yaml

# 编辑配置
vi /etc/shieldflow/edge.yaml
# 修改:
#   node.id: edge-beijing-01
#   node.license_key: <主控分配的 License>
#   grpc.server: <主控IP>:50051
#   grpc.token: <主控分配的 Token>

# 4. 安装 Supervisor
apt install -y supervisor
cp deploy/supervisor/zycdn-edge.conf /etc/supervisor/conf.d/
supervisorctl reread && supervisorctl update

# 5. 启动
supervisorctl start shieldflow-edge:*

# 6. 验证
# 在主控管理后台 → 节点管理 → 看到节点状态变为 "online"
```

### 多区域部署建议

| 区域 | 节点数量 | 带宽 | 说明 |
|------|---------|------|------|
| 华东 | 2+ | 100Mbps+ | 主要服务区域 |
| 华南 | 1+ | 50Mbps+ | 备用区域 |
| 华北 | 1+ | 50Mbps+ | 备用区域 |
| 海外 | 1+ | 50Mbps+ | 国际流量 |

---

## DNS 同步配置

### Cloudflare

1. 登录 Cloudflare → My Profile → API Tokens → Create Token
2. 权限：Zone:DNS:Edit, Zone:Zone:Read
3. 在管理后台 → 系统设置 → DNS 设置 → 添加 DNS 账户

```yaml
# /etc/shieldflow/dns-sync.yaml
providers:
  cloudflare:
    enabled: true
    api_token: "your-cloudflare-api-token"
```

### 阿里云 DNS

1. 登录阿里云 → AccessKey 管理 → 创建 AccessKey
2. 权限：AliyunDNSFullAccess
3. 在管理后台添加 DNS 账户

```yaml
providers:
  aliyun:
    enabled: true
    access_key_id: "your-access-key-id"
    access_key_secret: "your-access-key-secret"
```

### 腾讯云 DNSPod

1. 登录腾讯云 → 访问管理 → API 密钥管理
2. 在管理后台添加 DNS 账户

```yaml
providers:
  tencent:
    enabled: true
    secret_id: "your-secret-id"
    secret_key: "your-secret-key"
```

### 验证 DNS 同步

```bash
# 手动触发同步
curl -X POST http://localhost:9528/sync

# 查看同步报告
curl http://localhost:9528/report | python3 -m json.tool

# 检查 DNS 记录
dig example.com CNAME
# 预期: 指向 shieldflow 分配的 CNAME 地址
```

---

## 日志服务器独立部署（可选）

适用于大规模日志场景（v1.2.0+），将日志处理从主控分离。

```bash
# 1. 在独立服务器上
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

go build -o bin/log-server ./cmd/log-server

# 2. 配置
mkdir -p /etc/shieldflow /var/log/shieldflow
cp deploy/log-server.yaml /etc/shieldflow/

vi /etc/shieldflow/log-server.yaml
# 修改:
#   server.port: 9529
#   server.auth_token: "your-secret-token"
#   clickhouse.host: <clickhouse-server-ip>
#   clickhouse.password: <password>

# 3. 安装 ClickHouse（如日志服务器独立部署 CH）
# 参见上方 ClickHouse 安装步骤

# 4. 配置 Supervisor
cp deploy/supervisor/zycdn-log-server.conf /etc/supervisor/conf.d/
supervisorctl reread && supervisorctl update

# 5. 在主控配置中指向日志服务器
# 管理后台 → 系统设置 → gRPC 配置
#   日志上传模式: log_server
#   日志服务器地址: http://<日志服务器IP>:9529
#   日志服务器 Token: <上面设置的 auth_token>

# 6. 验证
curl http://localhost:9529/health
# 预期: {"status":"ok","clickhouse":"ok"}
```

---

## SSL 证书配置

### ACME 自动申请

1. 在管理后台 → 系统设置 → ACME 设置
2. 配置 ACME 目录 URL：
   - Let's Encrypt: `https://acme-v02.api.letsencrypt.org/directory`
   - ZeroSSL: `https://acme.zerossl.com/v2/DV`
3. 配置邮箱
4. 添加 ACME 账户

### 为域名申请证书

1. 管理后台 → SSL 证书 → 申请证书
2. 选择域名、验证方式（DNS-01 / HTTP-01）
3. 选择 ACME 账户
4. 提交申请，系统自动完成验证和签发

### 手动上传证书

```bash
# 在管理后台 → SSL 证书 → 上传
# 填写: 证书名称、域名
# 上传: cert.pem（证书文件）+ key.pem（私钥文件）
```

---

## 防火墙配置

### UFW (Ubuntu/Debian)

```bash
# 主控服务器
ufw allow 22/tcp          # SSH
ufw allow 80/tcp          # HTTP (Nginx)
ufw allow 443/tcp         # HTTPS (Nginx)
ufw allow 8080/tcp        # Backend API（建议仅内网）
ufw allow 50051/tcp       # gRPC（仅内网/边缘节点）
ufw allow 9528/tcp        # DNS Sync（仅内网）
ufw allow 9529/tcp        # Log Server（仅边缘节点）
ufw enable

# 边缘节点
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

### firewalld (CentOS/RHEL)

```bash
# 主控
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=8080/tcp
firewall-cmd --permanent --add-port=50051/tcp
firewall-cmd --reload

# 边缘节点
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --reload
```

---

## 服务管理

### Supervisor 命令

```bash
# 查看所有服务
supervisorctl status

# 查看特定组
supervisorctl status shieldflow-master:*
supervisorctl status shieldflow-edge:*

# 启动/停止/重启
supervisorctl start shieldflow-master:shieldflow-backend
supervisorctl stop shieldflow-master:shieldflow-backend
supervisorctl restart shieldflow-master:shieldflow-backend

# 查看日志
supervisorctl tail -f shieldflow-master:shieldflow-backend
tail -f /var/log/shieldflow/backend.log
```

### systemd 命令（数据库）

```bash
# PostgreSQL
systemctl status postgresql
systemctl restart postgresql

# ClickHouse
systemctl status clickhouse-server
systemctl restart clickhouse-server

# Redis
systemctl status redis-server
systemctl restart redis-server

# Nginx
systemctl status nginx
systemctl reload nginx
```

---

## 升级

```bash
# 1. 备份
bash scripts/backup.sh

# 2. 拉取新版本
cd /root/shieldflow
git pull origin main

# 3. 执行升级脚本
bash scripts/upgrade.sh

# 升级脚本自动完成:
# - 编译新二进制
# - 编译前端
# - 数据库迁移（如有新 SQL）
# - 平滑重启服务（先停后启）
# - 验证服务状态

# 4. 边缘节点升级
# 在管理后台 → 节点管理 → 批量升级
# 或手动:
bash scripts/upgrade.sh --component edge
```

---

## 备份与恢复

### 备份

```bash
# 一键备份
bash scripts/backup.sh

# 备份内容:
# - PostgreSQL 全量导出 (pg_dump)
# - ClickHouse 数据备份
# - 配置文件 (/etc/shieldflow/)
# - SSL 证书
# 备份文件: /var/backups/shieldflow/shieldflow_YYYYMMDD_HHMMSS.tar.gz

# 定时备份（crontab）
echo "0 2 * * * /root/shieldflow/scripts/backup.sh" | crontab -
```

### 恢复

```bash
# 恢复
bash scripts/restore.sh /var/backups/shieldflow/shieldflow_20260107_020000.tar.gz

# 恢复过程:
# - 停止服务
# - 恢复 PostgreSQL
# - 恢复 ClickHouse
# - 恢复配置文件
# - 启动服务
# - 验证
```

---

## 常见问题（FAQ）

### Q1: 服务启动失败怎么办？

```bash
# 查看错误日志
supervisorctl tail shieldflow-master:shieldflow-backend stderr
# 或
cat /var/log/shieldflow/backend.log

# 常见原因:
# 1. 数据库连接失败 → 检查 PG 是否运行、密码是否正确
# 2. 端口被占用 → netstat -tlnp | grep 8080
# 3. 配置文件路径错误 → 检查 -config 参数
# 4. 权限问题 → 检查 /var/log/shieldflow 和 /var/cache/shieldflow 权限
```

### Q2: 边缘节点无法连接主控？

```bash
# 1. 检查网络连通性
ping <主控IP>
telnet <主控IP> 50051

# 2. 检查防火墙
# 主控需放行 50051 端口

# 3. 检查 License Key
# 在管理后台 → 节点管理 → 确认 License 正确

# 4. 检查 gRPC Token
# edge.yaml 中的 grpc.token 必须与主控一致
```

### Q3: ClickHouse 写入失败？

```bash
# 1. 检查 CH 状态
systemctl status clickhouse-server

# 2. 检查连接
clickhouse-client --query "SELECT 1"

# 3. 检查表是否存在
clickhouse-client --query "SHOW TABLES FROM shieldflow_cdn"

# 4. 检查磁盘空间
df -h
# ClickHouse 默认在 /var/lib/clickhouse/
```

### Q4: 前端页面空白？

```bash
# 1. 检查前端是否编译
ls /usr/local/shieldflow/web/dist/
# 应有 index.html 和 assets/ 目录

# 2. 检查 Nginx 配置
nginx -t
# 确认 root 指向 dist 目录

# 3. 检查浏览器控制台
# F12 → Console 查看错误

# 4. 重新编译前端
cd web && npm install && npm run build
cp -r dist/ /usr/local/shieldflow/web/dist/
```

### Q5: 域名 CNAME 不生效？

```bash
# 1. 检查 DNS 同步服务
supervisorctl status shieldflow-master:shieldflow-dns-sync

# 2. 手动触发同步
curl -X POST http://localhost:9528/sync

# 3. 检查 DNS 记录
dig <域名> CNAME
nslookup <域名>

# 4. 检查 DNS 服务商配置
# 确认 API Token/Key 权限正确
```

### Q6: SSL 证书申请失败？

```bash
# 1. 检查 ACME 账户配置
# 管理后台 → 系统设置 → ACME

# 2. 检查域名 DNS 是否已指向 CDN
dig <域名> CNAME

# 3. DNS-01 验证需要 DNS 服务商 API
# 确认已添加 DNS 账户

# 4. 查看申请日志
# 管理后台 → SSL 证书 → 申请记录 → 查看日志
```

### Q7: 如何查看实时流量？

```bash
# 1. 管理后台 → 仪表盘 → 查看实时图表

# 2. API 查询
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/traffic/stats?start_time=2026-07-07T00:00:00&end_time=2026-07-07T23:59:59"

# 3. ClickHouse 直接查询
clickhouse-client --query "
  SELECT domain, sum(request_count) as reqs, sum(traffic_bytes) as traffic
  FROM shieldflow_cdn.bandwidth_stats
  WHERE timestamp >= today() - 1
  GROUP BY domain ORDER BY reqs DESC
"
```

### Q8: DDoS 防护不生效？

```bash
# 1. 检查 DDoS 规则是否启用
# 管理后台 → DDoS 防护 → 规则列表

# 2. 检查边缘节点是否加载规则
# 管理后台 → 节点管理 → 节点状态

# 3. 检查 eBPF 程序（需要 root 权限）
# edge 进程需要 CAP_NET_ADMIN 权限

# 4. 查看拦截日志
# 管理后台 → 日志管理 → DDoS 日志
```

### Q9: 如何扩展节点？

```bash
# 1. 在新服务器上安装边缘节点（参见"边缘节点独立部署"）

# 2. 在管理后台添加节点到分组

# 3. 域名配置中选择节点分组（自动负载均衡）

# 4. 验证流量分发
# 管理后台 → 流量统计 → 按节点查看
```

### Q10: 内存不足怎么办？

```bash
# 1. 检查内存使用
free -h
ps aux --sort=-%mem | head -10

# 2. 调整配置
# backend.yaml: 减少 max_open_conns (100→50)
# edge.yaml: 减少 cache.max_size
# ClickHouse: 调整 max_memory_usage

# 3. 添加 Swap
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
```

---

## 性能调优

### PostgreSQL 调优

```bash
# /etc/postgresql/15/main/postgresql.conf
shared_buffers = 2GB              # 25% 内存
effective_cache_size = 6GB        # 75% 内存
work_mem = 64MB
maintenance_work_mem = 512MB
max_connections = 200
wal_buffers = 16MB
random_page_cost = 1.1            # SSD
```

### ClickHouse 调优

```bash
# /etc/clickhouse-server/config.xml
<max_concurrent_queries>100</max_concurrent_queries>
<max_memory_usage>10000000000</max_memory_usage>
<max_thread_pool_size>100</max_thread_pool_size>
```

### Redis 调优

```bash
# /etc/redis/redis.conf
maxmemory 1gb
maxmemory-policy allkeys-lru
timeout 300
tcp-keepalive 60
```

### Edge 节点调优

```yaml
# edge.yaml
cache:
  max_size: "100GB"        # 根据磁盘调整
  ttl: "30m"               # 根据业务调整

proxy:
  max_connections: 10000    # 并发连接
  connect_timeout: "5s"
  read_timeout: "60s"
  write_timeout: "60s"
```

### 系统内核调优

```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.core.netdev_max_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
fs.file-max = 1000000

# 应用
sysctl -p

# 文件描述符限制
echo "* soft nofile 65535" >> /etc/security/limits.conf
echo "* hard nofile 65535" >> /etc/security/limits.conf
```

---

## 技术支持

- **文档**: [README](../README.md) | [开发文档](DEVELOPMENT.md) | [更新日志](../CHANGELOG.md)
- **问题反馈**: [GitHub Issues](https://github.com/717315051/shieldflow/issues)
- **贡献指南**: [CONTRIBUTING.md](../CONTRIBUTING.md)
