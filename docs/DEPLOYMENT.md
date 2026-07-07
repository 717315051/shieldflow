# ShieldFlow CDN 完整部署搭建教程

> **ShieldFlow（盾流）** — 企业级自建 CDN 系统，集成 WAF / DDoS 防护 / CC 防护 / Bot 管理 / 智能缓存 / DNS 同步 / 日志分析 / AI 运维。
>
> 仓库: <https://github.com/717315051/shieldflow>
>
> 本教程面向运维人员，覆盖从零到生产的完整部署流程。所有命令均可直接复制执行。

---

## 目录

1. [部署架构概述](#1-部署架构概述)
2. [系统要求](#2-系统要求)
3. [环境准备](#3-环境准备)
4. [方式一：一键脚本部署（推荐）](#4-方式一一键脚本部署推荐)
5. [方式二：Docker 部署](#5-方式二docker-部署)
6. [方式三：手动部署](#6-方式三手动部署)
7. [数据库初始化](#7-数据库初始化)
8. [配置详解](#8-配置详解)
9. [边缘节点部署](#9-边缘节点部署)
10. [DNS 同步配置](#10-dns-同步配置)
11. [日志服务器部署](#11-日志服务器部署)
12. [SSL 证书配置](#12-ssl-证书配置)
13. [防火墙配置](#13-防火墙配置)
14. [服务管理](#14-服务管理)
15. [升级](#15-升级)
16. [备份与恢复](#16-备份与恢复)
17. [常见问题（FAQ）](#17-常见问题faq)
18. [性能调优建议](#18-性能调优建议)

---

## 1. 部署架构概述

ShieldFlow 采用 **主控 + 边缘节点 + 日志服务器** 的分布式架构，支持单机一体化部署与多节点分布式部署两种模式。

### 1.1 拓扑图

```
                           ┌─────────────────────────────────────────────┐
                           │              ShieldFlow 管理后台              │
                           │   (Nginx 443 → Vue 前端 + /api 反代 :8080)    │
                           └──────────────────────┬──────────────────────┘
                                                  │
                           ┌──────────────────────┴──────────────────────┐
                           │              主控节点 (Master)               │
                           │  ┌────────────┐  ┌────────────┐             │
                           │  │  backend   │  │ grpc-server│             │
                           │  │  API :8080 │  │  gRPC:50051│             │
                           │  └────────────┘  └──────┬─────┘             │
                           │  ┌────────────┐  ┌──────┴─────┐             │
                           │  │ dns-sync   │  │ log-server │             │
                           │  │  :9528     │  │   :9529    │             │
                           │  └────────────┘  └────────────┘             │
                           └──────┬───────────────┬──────────────────────┘
                                  │               │
                    ┌─────────────┴───┐    ┌──────┴────────────────┐
                    │  PostgreSQL 5432 │    │   ClickHouse 9000     │
                    │   shieldflow_cdn │    │   shieldflow_cdn      │
                    │     (24 张表)    │    │    (6 张表)           │
                    └──────────────────┘    └───────────────────────┘
                                  │
                    ┌─────────────┴──────────────┐
                    │       Redis 6379           │
                    │  (缓存 / 会话 / 限流)       │
                    └────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────────────────┐
  │                          边缘节点集群 (Edge)                            │
  │                                                                         │
  │   ┌──────────────┐     ┌──────────────┐     ┌──────────────┐           │
  │   │  Edge Node A │     │  Edge Node B │     │  Edge Node C │           │
  │   │  :80 / :443  │     │  :80 / :443  │     │  :80 / :443  │           │
  │   │  gRPC →主控  │     │  gRPC →主控  │     │  gRPC →主控  │           │
  │   │ WAF/DDoS/CC │     │ WAF/DDoS/CC │     │ WAF/DDoS/CC │           │
  │   └──────────────┘     └──────────────┘     └──────────────┘           │
  │          ↑                                            ↑                  │
  │       用户请求                                    用户请求              │
  └─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 组件说明

| 组件 | 二进制 | 默认端口 | 说明 |
|------|--------|----------|------|
| 后端 API | `backend` | 8080 | RESTful API 服务，管理后台接口 |
| gRPC 服务 | `grpc-server` | 50051 | 与边缘节点通信的控制平面 |
| 边缘节点 | `edge` | 80 / 443 | CDN 代理节点，WAF/DDoS/CC/缓存 |
| DNS 同步 | `dns-sync` | 9528 | Cloudflare/阿里云/腾讯云 DNS 自动同步 |
| 日志服务器 | `log-server` | 9529 | 接收边缘节点日志，写入 ClickHouse |
| PostgreSQL | — | 5432 | 关系型数据存储（24 张表） |
| ClickHouse | — | 9000 | 日志与统计分析（6 张表） |
| Redis | — | 6379 | 缓存、会话、限流 |

### 1.3 部署模式

- **单机一体化**：主控 + 边缘节点 + 数据库全部部署在一台服务器（适合小规模/测试）
- **分布式**：主控单独部署，边缘节点分布在多台服务器（推荐生产环境）
- **日志独立化**：日志服务器独立部署到单独节点（大规模/高可用场景）

---

## 2. 系统要求

### 2.1 硬件要求

| 角色 | CPU | 内存 | 磁盘 | 带宽 | 说明 |
|------|-----|------|------|------|------|
| 主控（最小） | 2 核 | 4 GB | 50 GB | 10 Mbps | 小规模部署最低配置 |
| 主控（推荐） | 4 核 | 8 GB | 100 GB | 50 Mbps | 生产环境推荐 |
| 边缘节点（最小） | 2 核 | 2 GB | 20 GB | 100 Mbps | 低流量节点 |
| 边缘节点（推荐） | 4 核 | 4 GB | 50 GB SSD | 1 Gbps | 高流量节点 |
| 日志服务器 | 4 核 | 8 GB | 200 GB SSD | 100 Mbps | 日志量大的场景 |

> **磁盘说明**：主控使用 SSD 以提升数据库 I/O 性能；边缘节点缓存盘根据缓存大小需求配置，建议 SSD 或 NVMe。

### 2.2 软件要求

| 组件 | 最低版本 | 推荐版本 | 安装方式 |
|------|----------|----------|----------|
| 操作系统 | CentOS 7 / Ubuntu 18.04 / Debian 10 | Ubuntu 22.04 / Debian 12 | — |
| Go | 1.22 | 1.22.5+ | 编译后端 |
| Node.js | 18 | 18.20+ | 编译前端 |
| PostgreSQL | 14 | 15+ | 主数据库 |
| ClickHouse | 23 | 23.11+ | 日志存储 |
| Redis | 6 | 7+ | 缓存 |
| Nginx | 1.18 | 1.25+ | 管理后台反代 |
| Supervisor | 4 | 4+ | 进程管理 |

### 2.3 网络端口要求

| 端口 | 协议 | 方向 | 用途 | 放行范围 |
|------|------|------|------|----------|
| 80 | TCP | 入站 | 边缘节点 HTTP | 公网 |
| 443 | TCP | 入站 | 边缘节点 HTTPS | 公网 |
| 8080 | TCP | 入站 | 后端 API | 内网（经 Nginx） |
| 50051 | TCP | 入站 | gRPC 服务 | 边缘节点 → 主控 |
| 9528 | TCP | 入站 | DNS 同步 | 内网 |
| 9529 | TCP | 入站 | 日志服务器 | 边缘节点 → 日志服务器 |
| 9527 | TCP | 入站 | 健康检查 | 内网 |
| 5432 | TCP | — | PostgreSQL | 仅本地（生产建议内网） |
| 9000 | TCP | — | ClickHouse | 仅本地（生产建议内网） |
| 6379 | TCP | — | Redis | 仅本地 |

> **安全提示**：5432/9000/6379 端口 **严禁对公网开放**，仅限本地或内网访问。

---

## 3. 环境准备

### 3.1 操作系统初始化

以 root 用户登录服务器，执行以下初始化操作：

```bash
# 更新系统
yum update -y          # CentOS/RHEL
# 或
apt-get update && apt-get upgrade -y   # Ubuntu/Debian

# 设置时区
timedatectl set-timezone Asia/Shanghai

# 同步时间
yum install -y chrony && systemctl enable --now chronyd   # CentOS
# 或
apt-get install -y chrony && systemctl enable --now chrony  # Ubuntu/Debian

# 设置主机名（按实际角色命名）
hostnamectl set-hostname shieldflow-master   # 主控
hostnamectl set-hostname shieldflow-edge-01  # 边缘节点

# 关闭 SELinux（CentOS）
setenforce 0
sed -i 's/^SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config

# 设置文件描述符限制
cat >> /etc/security/limits.conf << 'EOF'
* soft nofile 65535
* hard nofile 65535
* soft nproc 65535
* hard nproc 65535
EOF

# 内核参数优化
cat >> /etc/sysctl.conf << 'EOF'
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 1024 65535
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535
fs.file-max = 1048576
EOF
sysctl -p
```

### 3.2 克隆项目代码

```bash
# 安装 git
yum install -y git    # CentOS
# 或
apt-get install -y git  # Ubuntu/Debian

# 克隆仓库
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow
```

### 3.3 安装依赖（手动方式预装）

如不使用一键脚本，需手动安装以下依赖：

```bash
# CentOS/RHEL
yum install -y epel-release
yum groupinstall -y "Development Tools"
yum install -y git wget curl make openssl-devel gcc clang \
    postgresql-server postgresql-contrib redis nginx \
    supervisor clickhouse-server clickhouse-client

# Ubuntu/Debian
apt-get update
apt-get install -y build-essential git wget curl make openssl \
    postgresql postgresql-contrib redis-server nginx \
    supervisor clickhouse-server clickhouse-client
```

### 3.4 安装 Go 1.22+

```bash
GO_VER=1.22.5
GO_ARCH=amd64
[[ "$(uname -m)" == "aarch64" ]] && GO_ARCH=arm64

cd /tmp
wget "https://go.dev/dl/go${GO_VER}.linux-${GO_ARCH}.tar.gz"
rm -rf /usr/local/go
tar -C /usr/local -xzf "go${GO_VER}.linux-${GO_ARCH}.tar.gz"
export PATH=$PATH:/usr/local/go/bin
grep -q '/usr/local/go/bin' /etc/profile || echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# 验证
go version
# 输出示例: go version go1.22.5 linux/amd64
```

### 3.5 安装 Node.js 18+

```bash
NODE_ARCH=x64
[[ "$(uname -m)" == "aarch64" ]] && NODE_ARCH=arm64

cd /tmp
wget "https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-${NODE_ARCH}.tar.xz"
tar -C /usr/local --strip-components=1 -xJf "node-v18.20.4-linux-${NODE_ARCH}.tar.xz"

# 验证
node -v   # v18.20.4
npm -v    # 10.x
```

---

## 4. 方式一：一键脚本部署（推荐）

ShieldFlow 提供一键安装脚本 `scripts/install.sh`，自动完成：系统检查 → 依赖安装 → Go/Node 安装 → 编译后端/前端 → 配置部署 → 数据库初始化 → Supervisor 注册 → Nginx 配置 → 防火墙放行。

### 4.1 主控安装（详细步骤）

#### 步骤 1：克隆代码

```bash
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow
```

#### 步骤 2：（可选）自定义安装参数

安装脚本支持通过环境变量自定义关键参数：

```bash
# 自定义数据库密码（不设置则自动随机生成）
export ShieldFlow_DB_PASS="MyStr0ngDBP@ssw0rd"

# 自定义 JWT 密钥（不设置则自动随机生成）
export ShieldFlow_JWT_SECRET="$(openssl rand -hex 32)"

# 自定义管理员账号
export ShieldFlow_ADMIN_USER="admin"
export ShieldFlow_ADMIN_PASS="Admin@2025"

# 自定义 Go 模块代理（国内推荐）
export GOPROXY="https://goproxy.cn,direct"
```

#### 步骤 3：执行一键安装

```bash
sudo bash scripts/install.sh
```

脚本执行流程（约 5-15 分钟，取决于网络和服务器性能）：

```
1. 检查系统环境 (CentOS/Ubuntu/Debian)
2. 安装系统依赖 (git, gcc, postgresql, redis, nginx, supervisor, clickhouse)
3. 安装 Go 1.22.5 (如未安装)
4. 安装 Node.js 18.20.4 (如未安装)
5. 创建运行目录 (/etc/shieldflow, /var/log/shieldflow, /var/cache/shieldflow)
6. 编译后端 5 个组件 → /usr/local/shieldflow/bin/
7. 编译前端 → /usr/local/shieldflow/web/
8. 部署配置文件模板 → /etc/shieldflow/
9. 初始化 PostgreSQL 数据库 shieldflow_cdn
10. 执行 SQL 初始化脚本 (PostgreSQL 24 张表 + ClickHouse 6 张表)
11. 启动 Redis
12. 注册 Supervisor 服务 (backend + grpc-server)
13. 配置 Nginx 反向代理 + 生成自签名证书
14. 配置防火墙端口放行
```

安装成功后输出：

```
========================================
   ShieldFlow 安装完成!
========================================
  数据库:   shieldflow_cdn (用户 shieldflow)
  配置目录: /etc/shieldflow
  日志目录: /var/log/shieldflow
  程序目录: /usr/local/shieldflow
  管理后台: https://<服务器IP>
  默认账号: admin / admin123
  数据库密码已写入: /etc/shieldflow/backend.yaml

  请尽快修改默认管理员密码并更换正式 TLS 证书!
  管理服务: supervisorctl status
  重载服务: supervisorctl reread && supervisorctl update
```

#### 步骤 4：验证主控安装

```bash
# 检查 Supervisor 服务状态
supervisorctl status
# 预期输出:
# shieldflow-backend                 RUNNING   pid 12345, uptime 0:00:30
# shieldflow-grpc-server             RUNNING   pid 12346, uptime 0:00:30

# 检查后端 API 健康状态
curl -s http://127.0.0.1:8080/api/v1/health
# 预期输出: {"status":"ok"}

# 检查 Nginx
curl -sk https://127.0.0.1/healthz
# 预期输出: ok

# 检查 gRPC 端口监听
ss -tlnp | grep 50051
# 预期: LISTEN  0  128  0.0.0.0:50051

# 检查 PostgreSQL
sudo -u postgres psql -d shieldflow_cdn -c "\dt"
# 预期: 列出 24 张表

# 检查 ClickHouse
clickhouse-client -q "SHOW TABLES FROM shieldflow_cdn"
# 预期: 列出 6 张表
```

### 4.2 边缘节点安装

边缘节点可部署在主控服务器（单机模式）或独立服务器（分布式模式）。

#### 在独立服务器上安装边缘节点

```bash
# 在边缘节点服务器上执行
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 安装边缘节点（需先安装 Go，参考 3.4 节）
sudo bash scripts/install-edge.sh \
  --node-id "edge-bj-01" \
  --master "1.2.3.4:50051" \
  --region "cn-beijing" \
  --http-port 80 \
  --https-port 443
```

参数说明：

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--node-id` | 是 | — | 节点唯一 ID，如 `edge-bj-01` |
| `--master` | 是 | — | 主控 gRPC 地址，如 `1.2.3.4:50051` |
| `--region` | 否 | `cn-east-1` | 节点区域 |
| `--license` | 否 | — | License Key |
| `--token` | 否 | — | 节点认证 Token |
| `--http-port` | 否 | 80 | HTTP 监听端口 |
| `--https-port` | 否 | 443 | HTTPS 监听端口 |
| `--skip-build` | 否 | false | 跳过编译（已有二进制时使用） |
| `--binary-only` | 否 | false | 仅安装二进制，不配置 Supervisor |

#### 验证边缘节点

```bash
# 检查 Supervisor
supervisorctl status shieldflow-edge:shieldflow-edge
# 预期: RUNNING

# 健康检查
curl -s http://127.0.0.1/ping
# 预期: ok / pong / alive

# 检查端口
ss -tlnp | grep -E ':80|:443'

# 查看日志
tail -f /var/log/shieldflow/edge.stderr.log
```

### 4.3 验证安装

完成主控和边缘节点安装后，执行以下全面验证：

```bash
# === 主控验证 ===
# 1. 服务状态
supervisorctl status
# 预期: shieldflow-backend 和 shieldflow-grpc-server 均 RUNNING

# 2. API 健康
curl -s http://127.0.0.1:8080/api/v1/health | python3 -m json.tool

# 3. 管理后台访问
# 浏览器打开 https://<主控IP> ，使用 admin / admin123 登录

# 4. 数据库表
sudo -u postgres psql -d shieldflow_cdn -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
# 预期: 24

clickhouse-client -q "SELECT count() FROM system.tables WHERE database='shieldflow_cdn'"
# 预期: 6

# === 边缘节点验证 ===
# 5. 边缘节点状态
supervisorctl status shieldflow-edge:shieldflow-edge

# 6. 主控连通性（在边缘节点上执行）
nc -zv <主控IP> 50051
# 预期: Connection to <主控IP> 50051 port [tcp/*] succeeded!

# 7. 健康检查
curl -s http://127.0.0.1/ping
```

---

## 5. 方式二：Docker 部署

ShieldFlow 提供完整的 Docker 支持，包含 `Dockerfile`（多阶段构建）和 `docker-compose.yml`（全栈编排）。

### 5.1 docker-compose 一键启动

#### 前置条件

```bash
# 安装 Docker 和 Docker Compose
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

# 验证
docker --version
docker compose version
```

#### 步骤 1：准备配置

```bash
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 创建 .env 文件设置密码（重要！）
cat > .env << 'EOF'
ShieldFlow_DB_PASS=MyStr0ngDBP@ssw0rd
ShieldFlow_JWT_SECRET=your_jwt_secret_hex_64_chars_change_me
ShieldFlow_CH_PASS=
EOF
```

#### 步骤 2：构建镜像并启动

```bash
# 构建镜像（首次约 10-20 分钟）
docker compose build

# 启动全部服务
docker compose up -d

# 查看服务状态
docker compose ps
```

启动的服务列表：

| 容器 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| shieldflow-postgres | postgres:14-alpine | 5432 | PostgreSQL 数据库 |
| shieldflow-clickhouse | clickhouse/clickhouse-server:23.11 | 9000, 8123 | ClickHouse |
| shieldflow-redis | redis:7-alpine | 6379 | Redis |
| shieldflow-master | shieldflow/shieldflow:latest | 8080, 50051 | 主控 (backend + grpc) |
| shieldflow-edge | shieldflow/shieldflow:latest | 80, 443, 9527 | 边缘节点 |
| shieldflow-nginx | nginx:1.25-alpine | 800, 4430 | 管理后台反代 |

> **端口说明**：Docker 模式下 Nginx 管理后台使用 800(HTTP)/4430(HTTPS) 端口，避免与 Edge 的 80/443 冲突。访问 `https://<IP>:4430`。

#### 步骤 3：单独启动特定服务

```bash
# 仅启动主控（不含边缘节点）
docker compose up -d postgres clickhouse redis master nginx

# 仅启动边缘节点
docker compose up -d edge

# 查看日志
docker compose logs -f master
docker compose logs -f edge

# 进入容器
docker exec -it shieldflow-master bash
```

#### 步骤 4：验证

```bash
# 检查所有容器健康状态
docker compose ps

# API 健康
curl -s http://127.0.0.1:8080/api/v1/health

# 管理后台
curl -sk https://127.0.0.1:4430/healthz
```

### 5.2 单独容器部署

如需单独部署某个组件，可使用以下方式：

```bash
# 构建镜像
docker build -t shieldflow/shieldflow:latest .

# 启动 PostgreSQL（独立）
docker run -d --name sf-postgres \
  -e POSTGRES_DB=shieldflow_cdn \
  -e POSTGRES_USER=shieldflow \
  -e POSTGRES_PASSWORD=MyStr0ngDBP@ssw0rd \
  -v sf_pg:/var/lib/postgresql/data \
  -v /root/shieldflow/sql/001_init_postgresql.sql:/docker-entrypoint-initdb.d/001.sql:ro \
  -p 5432:5432 \
  postgres:14-alpine

# 启动 ClickHouse（独立）
docker run -d --name sf-clickhouse \
  -e CLICKHOUSE_DB=shieldflow_cdn \
  -v sf_ch:/var/lib/clickhouse \
  -v /root/shieldflow/sql/002_init_clickhouse.sql:/docker-entrypoint-initdb.d/002.sql:ro \
  -p 9000:9000 -p 8123:8123 \
  clickhouse/clickhouse-server:23.11-alpine

# 启动 Redis（独立）
docker run -d --name sf-redis \
  -v sf_redis:/data \
  -p 6379:6379 \
  redis:7-alpine redis-server --appendonly yes

# 启动主控
docker run -d --name sf-master \
  --link sf-postgres:postgres \
  --link sf-clickhouse:clickhouse \
  --link sf-redis:redis \
  -e ShieldFlow_DB_PASS=MyStr0ngDBP@ssw0rd \
  -e ShieldFlow_JWT_SECRET=your_jwt_secret \
  -v sf_master_conf:/etc/shieldflow \
  -v sf_master_logs:/var/log/shieldflow \
  -p 8080:8080 -p 50051:50051 \
  shieldflow/shieldflow:latest master

# 启动边缘节点
docker run -d --name sf-edge \
  --link sf-master:master \
  -e ShieldFlow_NODE_ID=edge-docker-01 \
  -e ShieldFlow_MASTER_ADDR=master:50051 \
  -v sf_edge_conf:/etc/shieldflow \
  -v sf_edge_logs:/var/log/shieldflow \
  -p 80:80 -p 443:443 \
  shieldflow/shieldflow:latest edge
```

> **注意**：单独容器部署时需手动修改 `/etc/shieldflow/backend.yaml` 中的数据库地址，将 `127.0.0.1` 改为对应容器名或 IP。

---

## 6. 方式三：手动部署

适用于无法使用一键脚本、需要精细控制部署过程的场景。

### 6.1 编译二进制

```bash
cd /root/shieldflow

# 确保 Go 在 PATH 中
export PATH=$PATH:/usr/local/go/bin
export GOPROXY="https://goproxy.cn,direct"

# 创建输出目录
mkdir -p /usr/local/shieldflow/bin

# 编译 5 个后端组件
for comp in backend grpc-server edge dns-sync log-server; do
    echo "编译 $comp..."
    go build -ldflags="-s -w" -o "/usr/local/shieldflow/bin/${comp}" "./cmd/${comp}"
done

# 验证
ls -la /usr/local/shieldflow/bin/
# 预期: backend  grpc-server  edge  dns-sync  log-server
```

### 6.2 编译前端

```bash
cd /root/shieldflow/web

# 安装依赖
npm install

# 构建
npm run build

# 部署到安装目录
rm -rf /usr/local/shieldflow/web
cp -r dist /usr/local/shieldflow/web
```

### 6.3 创建目录结构

```bash
# 配置目录
mkdir -p /etc/shieldflow/certs
chmod 750 /etc/shieldflow

# 日志目录
mkdir -p /var/log/shieldflow
chmod 750 /var/log/shieldflow

# 缓存目录
mkdir -p /var/cache/shieldflow

# 备份目录
mkdir -p /var/backups/shieldflow

# ACME 验证目录
mkdir -p /var/www/acme
```

### 6.4 部署配置文件

```bash
# 复制配置模板
for f in backend.yaml grpc.yaml edge.yaml dns-sync.yaml log-server.yaml; do
    cp /root/shieldflow/deploy/${f} /etc/shieldflow/${f}
    chmod 640 /etc/shieldflow/${f}
done

# 生成随机密码和密钥
DB_PASS=$(openssl rand -hex 16)
JWT_SECRET=$(openssl rand -hex 32)

# 替换占位符
sed -i "s/CHANGE_ME_STRONG_PASSWORD/${DB_PASS}/g" /etc/shieldflow/{backend,grpc,dns-sync}.yaml
sed -i "s/CHANGE_ME_TO_RANDOM_64_CHAR_STRING/${JWT_SECRET}/g" /etc/shieldflow/{backend,grpc}.yaml

echo "数据库密码: ${DB_PASS}"
echo "JWT 密钥: ${JWT_SECRET}"
echo "请妥善保存以上信息！"
```

### 6.5 配置数据库

#### PostgreSQL 初始化

```bash
# CentOS: 初始化数据库
postgresql-setup --initdb

# Ubuntu/Debian: 自动初始化
pg_ctlcluster 14 main start   # 或 15, 取决于安装版本

# 启动并设置开机自启
systemctl enable --now postgresql

# 创建数据库和用户
sudo -u postgres psql << EOF
CREATE DATABASE shieldflow_cdn;
CREATE USER shieldflow WITH PASSWORD '${DB_PASS}';
GRANT ALL PRIVILEGES ON DATABASE shieldflow_cdn TO shieldflow;
\c shieldflow_cdn
GRANT ALL ON SCHEMA public TO shieldflow;
EOF

# 执行 SQL 初始化脚本
sudo -u postgres psql -d shieldflow_cdn -f /root/shieldflow/sql/001_init_postgresql.sql
```

#### ClickHouse 初始化

```bash
systemctl enable --now clickhouse-server

# 执行 SQL 初始化
clickhouse-client --multiquery < /root/shieldflow/sql/002_init_clickhouse.sql

# 验证
clickhouse-client -q "SHOW TABLES FROM shieldflow_cdn"
```

#### Redis 启动

```bash
systemctl enable --now redis      # CentOS
# 或
systemctl enable --now redis-server  # Ubuntu/Debian
```

### 6.6 配置 Supervisor

#### 安装 Supervisor

```bash
# CentOS
yum install -y supervisor
# Ubuntu/Debian
apt-get install -y supervisor

systemctl enable --now supervisor
```

#### 部署 Supervisor 配置

确定 Supervisor 配置目录：

```bash
# 确定 supervisor 配置目录
SUP_DIR=""
for d in /etc/supervisord.d /etc/supervisor/conf.d; do
    [[ -d "$d" ]] && SUP_DIR="$d" && break
done
[[ -z "$SUP_DIR" ]] && SUP_DIR="/etc/supervisord.d" && mkdir -p "$SUP_DIR"
echo "Supervisor 配置目录: $SUP_DIR"
```

> **重要提示**：仓库中 `deploy/supervisor/` 目录下的配置文件命名为 `zycdn-*.conf`，但安装脚本和 Supervisor 进程组名引用的是 `shieldflow-*.conf`。部署时需要重命名复制：

```bash
# 复制并重命名（zycdn → shieldflow）
cp /root/shieldflow/deploy/supervisor/zycdn-master.conf     "${SUP_DIR}/shieldflow-master.conf"
cp /root/shieldflow/deploy/supervisor/zycdn-edge.conf       "${SUP_DIR}/shieldflow-edge.conf"
cp /root/shieldflow/deploy/supervisor/zycdn-dns-sync.conf  "${SUP_DIR}/shieldflow-dns-sync.conf"
cp /root/shieldflow/deploy/supervisor/zycdn-log-server.conf "${SUP_DIR}/shieldflow-log-server.conf"
```

确保主配置文件包含 `include`：

```bash
# 检查主配置是否 include 子配置目录
for mainconf in /etc/supervisord.conf /etc/supervisor/supervisord.conf; do
    if [[ -f "$mainconf" ]] && ! grep -q "${SUP_DIR}" "$mainconf" 2>/dev/null; then
        echo -e "\n[include]\nfiles = ${SUP_DIR}/*.conf" >> "$mainconf"
    fi
done

# 重新加载
supervisorctl reread
supervisorctl update

# 验证
supervisorctl status
```

### 6.7 配置 Nginx

```bash
# 部署 Nginx 配置
cp /root/shieldflow/deploy/nginx/zycdn.conf /etc/nginx/conf.d/shieldflow.conf
# 注意：仓库中 nginx 配置文件名为 zycdn.conf，部署为 shieldflow.conf

# 生成自签名证书（生产环境请替换为正式证书）
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/shieldflow/certs/shieldflow.key \
    -out /etc/shieldflow/certs/shieldflow.crt \
    -subj "/CN=shieldflow/O=ShieldFlow"

# 测试并重启 Nginx
nginx -t
systemctl enable nginx
systemctl restart nginx
```

---

## 7. 数据库初始化

### 7.1 PostgreSQL 建库建表

ShieldFlow PostgreSQL 数据库 `shieldflow_cdn` 包含 **24 张表**：

| 序号 | 表名 | 说明 |
|------|------|------|
| 1 | users | 用户表 |
| 2 | node_groups | 节点分组表 |
| 3 | nodes | CDN 边缘节点表 |
| 4 | packages | 套餐表 |
| 5 | user_packages | 用户套餐表 |
| 6 | domains | 域名表 |
| 7 | certificates | SSL 证书表 |
| 8 | acme_accounts | ACME 账户表 |
| 9 | dns_accounts | DNS 服务商账户表 |
| 10 | blacklists | 黑白名单表 |
| 11 | protection_templates | 防护模板表 |
| 12 | layer4_forwards | 四层转发表 |
| 13 | cache_tasks | 缓存刷新任务表 |
| 14 | system_settings | 系统设置表 |
| 15 | ddos_rules | DDoS 规则表 |
| 16 | ddos_blacklist | DDoS 黑名单表 |
| 17 | ddos_whitelist | DDoS 白名单表 |
| 18 | log_server_config | 日志服务器配置表 |
| 19 | ai_config | AI 配置表 |
| 20 | traffic_packages | 流量包表 |
| 21 | domain_packages | 域名包表 |
| 22 | orders | 购买记录表 |
| 23 | operation_logs | 操作日志表 |
| 24 | balances | 余额表 |

#### 手动初始化

```bash
# 创建数据库和用户
sudo -u postgres psql << 'EOF'
CREATE DATABASE shieldflow_cdn;
CREATE USER shieldflow WITH PASSWORD 'YourStrongPassword';
GRANT ALL PRIVILEGES ON DATABASE shieldflow_cdn TO shieldflow;
\c shieldflow_cdn
GRANT ALL ON SCHEMA public TO shieldflow;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
EOF

# 执行建表脚本
sudo -u postgres psql -d shieldflow_cdn -f /root/shieldflow/sql/001_init_postgresql.sql

# 验证
sudo -u postgres psql -d shieldflow_cdn -c "\dt"
# 预期: 列出 24 张表

sudo -u postgres psql -d shieldflow_cdn -c "SELECT count(*) FROM users;"
# 预期: 1 (默认管理员)

sudo -u postgres psql -d shieldflow_cdn -c "SELECT count(*) FROM system_settings;"
# 预期: 3 (默认系统设置)
```

### 7.2 ClickHouse 建库建表

ShieldFlow ClickHouse 数据库 `shieldflow_cdn` 包含 **6 张表**：

| 序号 | 表名 | 说明 | 分区 | TTL |
|------|------|------|------|-----|
| 1 | access_logs | 七层 HTTP 访问日志 | 按天 | 30 天 |
| 2 | attack_logs | 安全防护攻击日志 | 按月 | 90 天 |
| 3 | ddos_logs | DDoS 防护日志 | 按月 | 90 天 |
| 4 | layer4_logs | 四层转发日志 | 按天 | 30 天 |
| 5 | ai_logs | AI 分析调用日志 | 按月 | 365 天 |
| 6 | bandwidth_stats | 带宽统计（预聚合） | 按月 | 365 天 |

#### 手动初始化

```bash
# 启动 ClickHouse
systemctl enable --now clickhouse-server

# 执行建表脚本
clickhouse-client --multiquery < /root/shieldflow/sql/002_init_clickhouse.sql

# 验证
clickhouse-client -q "SHOW TABLES FROM shieldflow_cdn"
# 预期输出 6 张表

clickhouse-client -q "SELECT count() FROM system.tables WHERE database='shieldflow_cdn'"
# 预期: 6
```

### 7.3 数据库连接配置

确保以下配置文件中的数据库连接信息正确：

```bash
# 检查 backend.yaml 数据库配置
grep -A5 'database:' /etc/shieldflow/backend.yaml
# 预期:
# database:
#   driver: postgres
#   host: 127.0.0.1
#   port: 5432
#   name: shieldflow_cdn
#   user: shieldflow
#   password: <你的密码>

# 检查 ClickHouse 配置
grep -A5 'clickhouse:' /etc/shieldflow/backend.yaml
```

---

## 8. 配置详解

所有配置文件位于 `/etc/shieldflow/`，采用 YAML 格式。

### 8.1 backend.yaml — 后端 API 服务

```yaml
# 部署路径: /etc/shieldflow/backend.yaml
# 组件: cmd/backend → Supervisor: shieldflow-master:shieldflow-backend

server:
  host: 0.0.0.0          # 监听地址，0.0.0.0 表示所有网卡
  port: 8080              # 监听端口
  mode: production        # production / development（开发模式输出更多日志）

database:
  driver: postgres
  host: 127.0.0.1         # PostgreSQL 地址
  port: 5432
  name: shieldflow_cdn
  user: shieldflow
  password: "CHANGE_ME_STRONG_PASSWORD"  # 安装脚本自动替换
  max_open_conns: 100     # 最大连接数
  max_idle_conns: 10      # 空闲连接数

redis:
  host: 127.0.0.1
  port: 6379
  password: ""            # Redis 密码，为空表示无密码
  db: 0                   # Redis 数据库编号

jwt:
  secret: "CHANGE_ME_TO_RANDOM_64_CHAR_STRING"  # 安装脚本自动替换
  expire: 24h             # JWT 过期时间

clickhouse:
  host: 127.0.0.1
  port: 9000
  database: shieldflow_cdn
  username: default
  password: ""

grpc:
  port: 50051             # gRPC 端口（此配置用于 backend 调用 grpc-server）
  tls: false              # 是否启用 TLS
  cert: /etc/shieldflow/certs/server.crt
  key: /etc/shieldflow/certs/server.key

ai:
  provider: openai        # AI 服务商: openai / azure / ollama
  model: gpt-4o-mini      # 模型名称
  api_key: ""             # API Key
  enabled: false          # 是否启用 AI 分析
  base_url: "https://api.openai.com/v1"  # API 地址

acme:
  directory: "https://acme-v02.api.letsencrypt.org/directory"  # ACME 目录 URL
  email: "admin@example.com"  # ACME 注册邮箱

log:
  level: info             # debug / info / warn / error
  format: json            # json / console
  output: /var/log/shieldflow/backend.log
```

### 8.2 grpc.yaml — gRPC 服务

```yaml
# 部署路径: /etc/shieldflow/grpc.yaml
# 组件: cmd/grpc-server → Supervisor: shieldflow-master:shieldflow-grpc-server

server:
  host: 0.0.0.0
  port: 50051
  mode: production

grpc:
  port: 50051
  tls: false              # 生产环境建议启用
  cert: /etc/shieldflow/certs/server.crt
  key: /etc/shieldflow/certs/server.key

database:
  driver: postgres
  host: 127.0.0.1
  port: 5432
  name: shieldflow_cdn
  user: shieldflow
  password: "CHANGE_ME_STRONG_PASSWORD"
  max_open_conns: 50      # gRPC 服务连接数略低于 backend
  max_idle_conns: 5

clickhouse:
  host: 127.0.0.1
  port: 9000
  database: shieldflow_cdn
  username: default
  password: ""

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0

jwt:
  secret: "CHANGE_ME_TO_RANDOM_64_CHAR_STRING"  # 需与 backend.yaml 一致
  expire: 24h

log:
  level: info
  format: json
  output: /var/log/shieldflow/grpc-server.log
```

> **重要**：`grpc.yaml` 中的 `jwt.secret` 必须与 `backend.yaml` 中的 `jwt.secret` 保持一致。

### 8.3 edge.yaml — 边缘节点

```yaml
# 部署路径: /etc/shieldflow/edge.yaml
# 组件: cmd/edge → Supervisor: shieldflow-edge:shieldflow-edge

node:
  id: "edge-default-01"           # 节点唯一 ID（必须唯一）
  region: "cn-east-1"             # 节点区域
  license_key: "CHANGE_ME_LICENSE_KEY"

grpc:
  server: "127.0.0.1:50051"       # 主控 gRPC 地址（独立部署时改为实际 IP）
  tls: false
  cert: /etc/shieldflow/certs/edge.crt
  key: /etc/shieldflow/certs/edge.key
  token: "CHANGE_ME_NODE_TOKEN"   # 节点认证 Token

proxy:
  http_port: 80                   # HTTP 监听端口
  https_port: 443                 # HTTPS 监听端口
  cert_file: /etc/shieldflow/certs/edge.crt
  key_file: /etc/shieldflow/certs/edge.key

waf:
  enabled: true                   # WAF 总开关
  mode: block                     # block(拦截) / detect(仅检测)
  threshold: 60                   # 触发阈值 0-100

cache:
  enabled: true                   # 缓存总开关
  path: /var/cache/shieldflow     # 缓存存储路径
  max_size: "50GB"               # 最大缓存大小
  ttl: "10m"                      # 默认缓存 TTL
  compress: gzip                  # 压缩算法: gzip / br

ddos:
  enabled: true
  max_connections_per_ip: 100     # 单 IP 最大连接数
  new_connections_per_sec: 50     # 单 IP 每秒新建连接数
  max_packets_per_sec: 5000       # 单 IP 每秒最大包数
  auto_ban_enabled: true          # 自动封禁开关
  ban_threshold_connections: 200  # 连接数封禁阈值
  ban_threshold_packets: 10000    # 包速率封禁阈值
  ban_duration_seconds: 3600      # 封禁时长（秒）
  blacklist:                      # 黑名单 CIDR
    - "0.0.0.0/8"
  whitelist:                      # 白名单 CIDR
    - "127.0.0.1/32"

cc:
  enabled: true
  global_rate_limit: 1000         # 全局每秒请求数
  global_window: "1s"             # 统计窗口
  challenge_type: js              # 质询类型: js / captcha / redirect
  waiting_room:
    enabled: false                # 等候室开关
    max_concurrent: 5000          # 最大并发
    base_wait_ms: 1000            # 基础等待时间
    increment_ms: 500             # 递增等待
    max_wait_ms: 30000            # 最大等待

bot:
  enabled: true
  allow_search_engines: true      # 允许搜索引擎
  block_scanners: true            # 拦截扫描器
  block_scrapers: false           # 拦截爬虫
  block_no_ua: true               # 拦截无 UA 请求

# 源站池（示例；实际由主控 gRPC 下发覆盖）
origins:
  - addr: "127.0.0.1:8080"
    weight: 100
    scheme: "http"
    host: "origin.example.com"
```

### 8.4 dns-sync.yaml — DNS 同步服务

```yaml
# 部署路径: /etc/shieldflow/dns-sync.yaml
# 组件: cmd/dns-sync → Supervisor: shieldflow-dns-sync:shieldflow-dns-sync

server:
  port: 9528

database:
  host: localhost
  port: 5432
  name: shieldflow_cdn
  user: shieldflow
  password: your_password      # 需替换为实际密码

sync:
  interval: 300                # 同步间隔（秒），默认 5 分钟
  retry: 3                     # 失败重试次数

providers:
  cloudflare:
    enabled: false             # 启用后需填写 api_token
    api_token: ""
  aliyun:
    enabled: false
    access_key_id: ""
    access_key_secret: ""
  tencent:
    enabled: false
    secret_id: ""
    secret_key: ""

log:
  level: info
  output: /var/log/shieldflow/dns-sync.log
```

### 8.5 log-server.yaml — 日志服务器

```yaml
# 部署路径: /etc/shieldflow/log-server.yaml
# 组件: cmd/log-server → Supervisor: shieldflow-log-server:shieldflow-log-server

server:
  port: 9529
  auth_token: "your-secret-token"   # 鉴权 Token，客户端通过 Authorization: Bearer 携带
                                     # 为空则跳过鉴权（仅内网/开发环境建议）

clickhouse:
  host: localhost
  port: 9000
  database: shieldflow_cdn
  username: default
  password: ""

writer:
  buffer_size: 10000            # 缓冲通道大小（每类日志独立缓冲）
  batch_size: 1000              # 每批写入条数
  flush_interval: 5             # 定时 flush 间隔（秒）
  retry: 3                      # 写入失败重试次数（指数退避）

log:
  level: info
  output: /var/log/shieldflow/log-server.log
```

### 8.6 Nginx 配置 — 管理后台反向代理

```nginx
# 部署路径: /etc/nginx/conf.d/shieldflow.conf
# 仓库模板: deploy/nginx/zycdn.conf

upstream shieldflow_backend {
    server 127.0.0.1:8080 fail_timeout=10s max_fails=3;
    keepalive 32;
}

# HTTP → HTTPS 跳转
server {
    listen 80;
    server_name _;

    # ACME 证书验证
    location /.well-known/acme-challenge/ {
        root /var/www/acme;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS 主服务
server {
    listen 443 ssl http2;
    server_name _;

    ssl_certificate     /etc/shieldflow/certs/shieldflow.crt;
    ssl_certificate_key /etc/shieldflow/certs/shieldflow.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...;
    ssl_prefer_server_ciphers on;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # gzip 压缩
    gzip on;
    gzip_min_length 1k;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript ...;

    client_max_body_size 50m;     # 证书导入等上传

    # 静态前端
    root /usr/local/shieldflow/web;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;   # SPA 路由
    }

    # 静态资源缓存
    location ~* \.(?:js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://shieldflow_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        proxy_connect_timeout 30s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket（实时日志推送）
    location /ws/ {
        proxy_pass http://shieldflow_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 3600s;
    }

    # 健康检查
    location = /healthz {
        access_log off;
        return 200 "ok\n";
        add_header Content-Type text/plain;
    }
}
```

### 8.7 Supervisor 配置

主控 Supervisor 配置（`shieldflow-master.conf`）管理两个进程：

```ini
# 部署路径: /etc/supervisord.d/shieldflow-master.conf (或 /etc/supervisor/conf.d/)
# 仓库模板: deploy/supervisor/zycdn-master.conf

[group:shieldflow-master]
programs=shieldflow-backend,shieldflow-grpc-server
priority=10

[program:shieldflow-backend]
command=/usr/local/shieldflow/bin/backend -config /etc/shieldflow/backend.yaml
directory=/usr/local/shieldflow
user=root
autostart=true
autorestart=true
startsecs=5
startretries=3
stopwaitsecs=15
stopsignal=TERM
stdout_logfile=/var/log/shieldflow/backend.stdout.log
stderr_logfile=/var/log/shieldflow/backend.stderr.log
stdout_logfile_maxbytes=100MB
stdout_logfile_backups=10
stderr_logfile_maxbytes=100MB
stderr_logfile_backups=10
environment=GIN_MODE="release"
priority=10

[program:shieldflow-grpc-server]
command=/usr/local/shieldflow/bin/grpc-server -config /etc/shieldflow/grpc.yaml
directory=/usr/local/shieldflow
user=root
autostart=true
autorestart=true
startsecs=5
startretries=3
stopwaitsecs=15
stopsignal=TERM
stdout_logfile=/var/log/shieldflow/grpc-server.stdout.log
stderr_logfile=/var/log/shieldflow/grpc-server.stderr.log
stdout_logfile_maxbytes=100MB
stdout_logfile_backups=10
stderr_logfile_maxbytes=100MB
stderr_logfile_backups=10
priority=20
```

---

## 9. 边缘节点部署

边缘节点是 CDN 的数据平面，负责接收用户请求、缓存内容、防护攻击、回源拉取数据。生产环境建议将边缘节点部署在离用户最近的地理位置。

### 9.1 独立部署到其他服务器

#### 步骤 1：准备服务器

```bash
# 在边缘节点服务器上执行基础初始化（参考第 3 节）
# 安装 Go（参考 3.4 节）
# 安装 Supervisor
yum install -y supervisor    # CentOS
# 或
apt-get install -y supervisor  # Ubuntu/Debian
```

#### 步骤 2：克隆代码并安装

```bash
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 执行边缘节点安装
sudo bash scripts/install-edge.sh \
  --node-id "edge-bj-01" \
  --master "主控IP:50051" \
  --region "cn-beijing" \
  --http-port 80 \
  --https-port 443
```

#### 步骤 3：配置节点认证

在主控管理后台注册该节点后，获取 License Key 和 Node Token，更新边缘节点配置：

```bash
# 编辑 edge.yaml
vi /etc/shieldflow/edge.yaml

# 修改以下字段：
# node.license_key: "从主控获取的License"
# grpc.token: "从主控获取的Token"

# 重启服务
supervisorctl restart shieldflow-edge:shieldflow-edge
```

#### 步骤 4：验证节点上线

```bash
# 在边缘节点上验证
supervisorctl status shieldflow-edge:shieldflow-edge
# 预期: RUNNING

# 健康检查
curl -s http://127.0.0.1/ping

# 连通性检查
nc -zv 主控IP 50051

# 在主控管理后台查看节点状态
# 节点管理 → 节点列表 → 确认 edge-bj-01 状态为 online
```

### 9.2 多节点部署示例

```bash
# 节点 1 - 北京
sudo bash scripts/install-edge.sh \
  --node-id "edge-bj-01" --master "10.0.1.100:50051" --region "cn-beijing"

# 节点 2 - 上海
sudo bash scripts/install-edge.sh \
  --node-id "edge-sh-01" --master "10.0.1.100:50051" --region "cn-shanghai"

# 节点 3 - 广州
sudo bash scripts/install-edge.sh \
  --node-id "edge-gz-01" --master "10.0.1.100:50051" --region "cn-guangzhou"
```

### 9.3 使用预编译二进制部署（免编译）

如在多台服务器部署，可在一台服务器编译后分发二进制文件，避免每台都安装 Go：

```bash
# 在编译服务器上
cd /root/shieldflow
go build -ldflags="-s -w" -o edge ./cmd/edge

# 分发到边缘节点
scp edge root@edge-server:/usr/local/shieldflow/bin/
scp deploy/edge.yaml root@edge-server:/etc/shieldflow/

# 在边缘节点上使用 --skip-build 安装
sudo bash scripts/install-edge.sh \
  --node-id "edge-bj-01" \
  --master "主控IP:50051" \
  --skip-build
```

---

## 10. DNS 同步配置

ShieldFlow DNS 同步服务支持自动将 CDN CNAME 记录同步到 Cloudflare、阿里云、腾讯云等 DNS 服务商。

### 10.1 Cloudflare 配置

1. 登录 Cloudflare 控制台 → My Profile → API Tokens → Create Token
2. 创建 Token，权限选择 `Zone:DNS:Edit`
3. 获取 API Token

```bash
# 编辑配置
vi /etc/shieldflow/dns-sync.yaml

# 修改 Cloudflare 配置
providers:
  cloudflare:
    enabled: true
    api_token: "your_cloudflare_api_token_here"
```

### 10.2 阿里云配置

1. 登录阿里云控制台 → AccessKey 管理 → 创建 AccessKey
2. 为 RAM 用户授权 `AliyunDNSFullAccess` 权限

```yaml
providers:
  aliyun:
    enabled: true
    access_key_id: "your_access_key_id"
    access_key_secret: "your_access_key_secret"
```

### 10.3 腾讯云配置

1. 登录腾讯云控制台 → 访问管理 → API 密钥管理 → 新建密钥
2. 为子用户授权 `QcloudDNSPodFullAccess` 权限

```yaml
providers:
  tencent:
    enabled: true
    secret_id: "your_secret_id"
    secret_key: "your_secret_key"
```

### 10.4 启动 DNS 同步服务

```bash
# 确保 supervisor 配置已部署（参考 6.6 节）
supervisorctl reread
supervisorctl update
supervisorctl start shieldflow-dns-sync:*

# 查看状态
supervisorctl status shieldflow-dns-sync:*

# 查看日志
tail -f /var/log/shieldflow/dns-sync.log
```

### 10.5 同步策略

- 默认同步间隔：**300 秒**（5 分钟），可在 `dns-sync.yaml` 的 `sync.interval` 调整
- 失败重试：**3 次**
- 同步范围：所有状态为"正常"的域名对应的 CNAME 记录

---

## 11. 日志服务器部署

日志服务器独立部署用于接收边缘节点的访问日志、攻击日志等，并批量写入 ClickHouse。适用于大规模分布式部署场景。

### 11.1 独立部署日志服务器

```bash
# 在日志服务器上执行
cd /root
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 编译 log-server
export PATH=$PATH:/usr/local/go/bin
go build -ldflags="-s -w" -o /usr/local/shieldflow/bin/log-server ./cmd/log-server

# 创建目录
mkdir -p /etc/shieldflow /var/log/shieldflow

# 部署配置
cp deploy/log-server.yaml /etc/shieldflow/

# 编辑配置
vi /etc/shieldflow/log-server.yaml
# 修改:
#   server.auth_token: 设置一个强随机 Token
#   clickhouse.host: 改为 ClickHouse 服务器 IP
```

### 11.2 配置 Supervisor

```bash
# 确定配置目录
SUP_DIR="/etc/supervisor/conf.d"   # Ubuntu/Debian
# 或 SUP_DIR="/etc/supervisord.d"   # CentOS

# 部署 Supervisor 配置
cp deploy/supervisor/zycdn-log-server.conf "${SUP_DIR}/shieldflow-log-server.conf"

# 加载并启动
supervisorctl reread
supervisorctl update
supervisorctl start shieldflow-log-server:*
```

### 11.3 配置主控和边缘节点指向日志服务器

#### 主控配置

```bash
# 在主控数据库中配置日志服务器地址
sudo -u postgres psql -d shieldflow_cdn << EOF
INSERT INTO log_server_config (mode, address, token)
VALUES ('log_server', '日志服务器IP:9529', 'your-auth-token')
ON CONFLICT (id) DO UPDATE SET address='日志服务器IP:9529', token='your-auth-token', updated_at=NOW();
EOF
```

#### 边缘节点配置

边缘节点通过 gRPC 从主控获取日志服务器地址，无需额外配置。确保主控 `log_server_config` 表中的地址和 Token 正确即可。

### 11.4 验证日志服务器

```bash
# 检查服务状态
supervisorctl status shieldflow-log-server:*

# 检查端口
ss -tlnp | grep 9529

# 检查 ClickHouse 连接
clickhouse-client -q "SELECT count() FROM shieldflow_cdn.access_logs"

# 查看日志
tail -f /var/log/shieldflow/log-server.log
```

---

## 12. SSL 证书配置

### 12.1 自签名证书（测试环境）

一键安装脚本已自动生成自签名证书。如需手动生成：

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/shieldflow/certs/shieldflow.key \
    -out /etc/shieldflow/certs/shieldflow.crt \
    -subj "/CN=shieldflow/O=ShieldFlow"

# 边缘节点证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/shieldflow/certs/edge.key \
    -out /etc/shieldflow/certs/edge.crt \
    -subj "/CN=edge-bj-01/O=ShieldFlow"
```

### 12.2 ACME 自动申请（Let's Encrypt）

ShieldFlow 内置 ACME 自动证书签发功能，通过管理后台可自动申请 Let's Encrypt 免费证书。

#### 前置条件

1. 域名已解析到服务器 IP
2. 80 端口可从公网访问（用于 HTTP-01 验证）
3. Nginx 配置了 ACME challenge 路径（默认已配置）

#### 配置 ACME

```bash
# 编辑 backend.yaml
vi /etc/shieldflow/backend.yaml

# 修改 ACME 配置
acme:
  directory: "https://acme-v02.api.letsencrypt.org/directory"
  email: "your-email@example.com"    # 改为你的邮箱

# 确保 ACME 验证目录存在
mkdir -p /var/www/acme

# 确保 Nginx 配置了 ACME 路径（默认已配置）
# location /.well-known/acme-challenge/ {
#     root /var/www/acme;
# }

# 重启后端
supervisorctl restart shieldflow-master:shieldflow-backend
```

#### 在管理后台申请证书

1. 登录管理后台 → SSL 证书 → 申请证书
2. 输入域名，选择 ACME 自动签发
3. 系统自动完成验证并签发证书
4. 证书自动部署到对应域名

### 12.3 手动导入正式证书

```bash
# 上传证书文件
cp your_domain.crt /etc/shieldflow/certs/
cp your_domain.key /etc/shieldflow/certs/

# 修改 Nginx 配置使用新证书
vi /etc/nginx/conf.d/shieldflow.conf
# 修改:
# ssl_certificate     /etc/shieldflow/certs/your_domain.crt;
# ssl_certificate_key /etc/shieldflow/certs/your_domain.key;

# 测试并重启
nginx -t && systemctl reload nginx
```

### 12.4 证书自动续期

```bash
# 添加定时任务检查并续期
crontab -e

# 添加以下内容（每天凌晨 3 点检查续期）
0 3 * * * /usr/local/shieldflow/bin/backend -acme-renew >> /var/log/shieldflow/acme-renew.log 2>&1
```

---

## 13. 防火墙配置

### 13.1 CentOS/RHEL (firewalld)

```bash
# 主控节点
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=8080/tcp      # API（可选，经 Nginx 代理）
firewall-cmd --permanent --add-port=50051/tcp     # gRPC（边缘节点需访问）
firewall-cmd --permanent --add-port=9528/tcp      # DNS 同步（内网）
firewall-cmd --permanent --add-port=9529/tcp      # 日志服务器（边缘节点需访问）
firewall-cmd --permanent --add-port=9527/tcp      # 健康检查（内网）
firewall-cmd --reload

# 边缘节点
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=9527/tcp      # 健康检查
firewall-cmd --reload
```

### 13.2 Ubuntu/Debian (ufw)

```bash
# 主控节点
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8080/tcp
ufw allow 50051/tcp
ufw allow 9528/tcp
ufw allow 9529/tcp
ufw allow 9527/tcp

# 边缘节点
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 9527/tcp

# 启用防火墙
ufw enable
ufw status verbose
```

### 13.3 iptables（通用）

```bash
# 主控节点
iptables -I INPUT -p tcp --dport 80 -j ACCEPT
iptables -I INPUT -p tcp --dport 443 -j ACCEPT
iptables -I INPUT -p tcp --dport 50051 -j ACCEPT
iptables -I INPUT -p tcp --dport 9528 -j ACCEPT
iptables -I INPUT -p tcp --dport 9529 -j ACCEPT
iptables -I INPUT -p tcp --dport 9527 -j ACCEPT

# 保存规则
iptables-save > /etc/sysconfig/iptables   # CentOS
# 或
netfilter-persistent save                   # Ubuntu/Debian
```

### 13.4 云服务器安全组

如使用阿里云/腾讯云/AWS 等云服务器，还需在控制台安全组中放行对应端口：

| 端口 | 来源 | 用途 |
|------|------|------|
| 80, 443 | 0.0.0.0/0 | 公网访问（边缘节点） |
| 50051 | 边缘节点 IP 段 | gRPC 通信 |
| 9529 | 边缘节点 IP 段 | 日志上报 |
| 8080, 9528, 9527 | 内网 | 管理接口 |

---

## 14. 服务管理

ShieldFlow 使用 Supervisor 管理所有服务进程。

### 14.1 常用命令

```bash
# 查看所有服务状态
supervisorctl status

# 查看特定服务组
supervisorctl status shieldflow-master:*
supervisorctl status shieldflow-edge:*
supervisorctl status shieldflow-dns-sync:*
supervisorctl status shieldflow-log-server:*

# 启动服务
supervisorctl start shieldflow-master:*
supervisorctl start shieldflow-edge:shieldflow-edge

# 停止服务
supervisorctl stop shieldflow-master:*
supervisorctl stop shieldflow-edge:shieldflow-edge

# 重启服务
supervisorctl restart shieldflow-master:*
supervisorctl restart shieldflow-master:shieldflow-backend    # 仅重启 backend

# 重新读取配置（修改 supervisor 配置后）
supervisorctl reread
supervisorctl update

# 查看服务日志
supervisorctl tail -f shieldflow-master:shieldflow-backend
supervisorctl tail -f shieldflow-master:shieldflow-backend stderr
```

### 14.2 服务组对照表

| 服务组 | 进程 | 说明 |
|--------|------|------|
| shieldflow-master | shieldflow-backend, shieldflow-grpc-server | 主控（API + gRPC） |
| shieldflow-edge | shieldflow-edge | 边缘节点 |
| shieldflow-dns-sync | shieldflow-dns-sync | DNS 同步 |
| shieldflow-log-server | shieldflow-log-server | 日志服务器 |

### 14.3 日志文件

| 服务 | 日志路径 |
|------|----------|
| backend | /var/log/shieldflow/backend.log<br>/var/log/shieldflow/backend.stdout.log<br>/var/log/shieldflow/backend.stderr.log |
| grpc-server | /var/log/shieldflow/grpc-server.log<br>/var/log/shieldflow/grpc-server.stdout.log<br>/var/log/shieldflow/grpc-server.stderr.log |
| edge | /var/log/shieldflow/edge.stdout.log<br>/var/log/shieldflow/edge.stderr.log |
| dns-sync | /var/log/shieldflow/dns-sync.log<br>/var/log/shieldflow/dns-sync.stdout.log<br>/var/log/shieldflow/dns-sync.stderr.log |
| log-server | /var/log/shieldflow/log-server.log<br>/var/log/shieldflow/log-server.stdout.log<br>/var/log/shieldflow/log-server.stderr.log |
| Nginx | /var/log/nginx/access.log<br>/var/log/nginx/error.log |

### 14.4 路径速查

| 类型 | 路径 |
|------|------|
| 程序二进制 | /usr/local/shieldflow/bin/ |
| 前端静态文件 | /usr/local/shieldflow/web/ |
| 配置文件 | /etc/shieldflow/ |
| SSL 证书 | /etc/shieldflow/certs/ |
| 日志文件 | /var/log/shieldflow/ |
| 缓存目录 | /var/cache/shieldflow/ |
| 备份目录 | /var/backups/shieldflow/ |
| Supervisor 配置 | /etc/supervisord.d/ 或 /etc/supervisor/conf.d/ |
| Nginx 配置 | /etc/nginx/conf.d/shieldflow.conf |

---

## 15. 升级

ShieldFlow 提供升级脚本 `scripts/upgrade.sh`，自动完成：备份当前版本 → 拉取新代码 → 编译 → 替换二进制 → 升级配置 → 执行 SQL 迁移 → 滚动重启。

### 15.1 升级流程

```bash
cd /root/shieldflow

# 拉取最新代码
git pull

# 执行升级
sudo bash scripts/upgrade.sh

# 或升级到指定版本（git tag/branch）
sudo bash scripts/upgrade.sh --version v1.1.0
```

### 15.2 升级脚本执行流程

```
1. 备份当前版本
   → 备份二进制到 /var/backups/shieldflow/upgrade-<timestamp>/bin/
   → 备份前端到 /var/backups/shieldflow/upgrade-<timestamp>/web/
   → 备份配置到 /var/backups/shieldflow/upgrade-<timestamp>/etc-shieldflow/

2. 拉取新代码
   → git fetch --all
   → git checkout <version> 或 git pull --ff-only

3. 编译后端
   → 编译 5 个组件到 /usr/local/shieldflow/bin/ (先 .new 再替换)

4. 编译前端
   → npm install && npm run build
   → 替换 /usr/local/shieldflow/web/

5. 升级配置模板（不覆盖已有配置）
   → 仅部署缺失的新配置文件

6. 执行 SQL 迁移
   → 执行 sql/ 目录下所有 .sql 文件

7. 滚动重启
   → supervisorctl reread && update
   → restart shieldflow-master, shieldflow-dns-sync, shieldflow-log-server
   → nginx -t && systemctl reload nginx
```

### 15.3 升级后验证

```bash
# 服务状态
supervisorctl status

# API 健康
curl -s http://127.0.0.1:8080/api/v1/health

# 数据库版本
sudo -u postgres psql -d shieldflow_cdn -c "SELECT value FROM system_settings WHERE key='system_version';"
```

### 15.4 回滚

```bash
# 查看备份列表
ls -la /var/backups/shieldflow/upgrade-*/

# 回滚到指定版本
BACKUP_DIR="/var/backups/shieldflow/upgrade-20250101_120000"

# 恢复二进制
cp -r "${BACKUP_DIR}/bin/"* /usr/local/shieldflow/bin/

# 恢复前端
rm -rf /usr/local/shieldflow/web
cp -r "${BACKUP_DIR}/web" /usr/local/shieldflow/web

# 恢复配置（可选）
cp -r "${BACKUP_DIR}/etc-shieldflow/"* /etc/shieldflow/

# 重启服务
supervisorctl restart shieldflow-master:*
supervisorctl restart shieldflow-dns-sync:*
supervisorctl restart shieldflow-log-server:*
systemctl reload nginx
```

### 15.5 边缘节点升级

```bash
# 在每台边缘节点上执行
cd /root/shieldflow
git pull
sudo bash scripts/upgrade.sh

# 或分发新二进制后重启
supervisorctl restart shieldflow-edge:shieldflow-edge
```

> **建议**：升级边缘节点时逐个进行，避免全部同时下线影响业务。

---

## 16. 备份与恢复

### 16.1 备份策略

| 备份项 | 备份方式 | 频率 | 保留 |
|--------|----------|------|------|
| PostgreSQL | pg_dump -Fc (自定义格式) | 每日 02:00 | 30 天 |
| ClickHouse | TabSeparated 导出 | 每日 02:00 | 30 天 |
| 配置文件 | 文件复制 | 每日 02:00 | 30 天 |
| Supervisor 配置 | 文件复制 | 每日 02:00 | 30 天 |
| Nginx 配置 | 文件复制 | 每日 02:00 | 30 天 |

### 16.2 执行备份

```bash
# 手动执行备份
sudo bash /root/shieldflow/scripts/backup.sh

# 备份到指定目录
sudo bash /root/shieldflow/scripts/backup.sh --output /mnt/backup/shieldflow

# 备份文件位置
ls -la /var/backups/shieldflow/
# 预期: shieldflow-backup-20250107_020000.tar.gz
```

### 16.3 设置自动备份

```bash
# 添加 crontab 定时任务
crontab -e

# 每天凌晨 2 点自动备份
0 2 * * * /root/shieldflow/scripts/backup.sh >> /var/log/shieldflow/backup.log 2>&1

# 保留天数设置（默认 30 天）
# 通过环境变量调整
# 在 crontab 中:
# 0 2 * * * ShieldFlow_BACKUP_RETAIN=60 /root/shieldflow/scripts/backup.sh
```

### 16.4 备份内容说明

备份脚本 `scripts/backup.sh` 打包以下内容：

```
shieldflow-backup-<timestamp>.tar.gz
├── shieldflow_cdn.dump              # PostgreSQL 备份（pg_dump 自定义格式）
├── clickhouse/                      # ClickHouse 表数据
│   ├── access_logs.tsv
│   ├── attack_logs.tsv
│   ├── ddos_logs.tsv
│   ├── layer4_logs.tsv
│   ├── ai_logs.tsv
│   └── bandwidth_stats.tsv
└── configs/                         # 配置文件
    ├── shieldflow/                  # /etc/shieldflow/ 完整目录
    ├── supervisord.d/ 或 conf.d/    # Supervisor 配置
    └── shieldflow.conf              # Nginx 配置
```

### 16.5 执行恢复

> **警告**：恢复操作会覆盖现有数据，请先确认并停止相关服务！

```bash
# 恢复（会自动停止服务 → 恢复 → 启动服务）
sudo bash /root/shieldflow/scripts/restore.sh \
  --file /var/backups/shieldflow/shieldflow-backup-20250107_020000.tar.gz

# 跳过 ClickHouse 恢复
sudo bash /root/shieldflow/scripts/restore.sh \
  --file /var/backups/shieldflow/shieldflow-backup-20250107_020000.tar.gz \
  --no-clickhouse

# 恢复到指定数据库名
sudo bash /root/shieldflow/scripts/restore.sh \
  --file /path/to/backup.tar.gz \
  --db-name shieldflow_cdn
```

恢复流程：

```
1. 停止所有 ShieldFlow 服务
2. 解包备份文件到临时目录
3. 恢复 PostgreSQL
   → 确保 database 和 user 存在
   → 终止现有连接
   → dropdb → createdb → pg_restore
4. 恢复 ClickHouse
   → CREATE DATABASE IF NOT EXISTS
   → INSERT FROM TSV 文件
5. 恢复配置文件到 /etc/shieldflow/
6. 启动 ShieldFlow 服务
7. 清理临时文件
```

### 16.6 异地备份

```bash
# 将备份同步到远程存储（如对象存储）
# 使用 rclone 同步到 S3/阿里云 OSS
rclone copy /var/backups/shieldflow/ remote:shieldflow-backups/ --progress

# 或使用 scp 同步到另一台服务器
scp /var/backups/shieldflow/shieldflow-backup-*.tar.gz user@backup-server:/backups/

# 添加到 crontab
0 3 * * * rclone copy /var/backups/shieldflow/ remote:shieldflow-backups/ --progress >> /var/log/shieldflow/rclone.log 2>&1
```

---

## 17. 常见问题（FAQ）

### Q1: 安装脚本报错 "请以 root 身份运行此脚本"

**原因**：安装脚本需要 root 权限来安装软件包、创建目录、注册系统服务。

**解决**：

```bash
sudo bash scripts/install.sh
# 或切换到 root 用户
su -
bash scripts/install.sh
```

### Q2: Go 编译失败，报模块下载错误

**原因**：Go 模块代理无法访问默认地址 `proxy.golang.org`。

**解决**：设置国内 Go 模块代理：

```bash
export GOPROXY="https://goproxy.cn,direct"
# 或
export GOPROXY="https://goproxy.io,direct"

# 永久设置
go env -w GOPROXY=https://goproxy.cn,direct
```

### Q3: npm install 失败或超时

**原因**：npm 默认源在国内访问慢。

**解决**：使用国内 npm 镜像：

```bash
npm config set registry https://registry.npmmirror.com
npm install
```

### Q4: supervisorctl status 显示服务 FATAL 状态

**排查步骤**：

```bash
# 1. 查看错误日志
supervisorctl tail shieldflow-master:shieldflow-backend stderr

# 2. 查看详细日志
cat /var/log/shieldflow/backend.stderr.log

# 3. 常见原因:
#    a. 数据库连接失败 → 检查 PostgreSQL 是否运行，密码是否正确
#    b. 配置文件路径错误 → 确认 /etc/shieldflow/backend.yaml 存在
#    c. 端口被占用 → ss -tlnp | grep 8080
#    d. 二进制文件不存在 → ls -la /usr/local/shieldflow/bin/

# 4. 修复后重启
supervisorctl start shieldflow-master:shieldflow-backend
```

### Q5: ClickHouse 初始化失败 "clickhouse-client 未安装"

**原因**：ClickHouse 未安装或未启动。

**解决**：

```bash
# 安装 ClickHouse
# CentOS
yum install -y clickhouse-server clickhouse-client
# Ubuntu/Debian
apt-get install -y clickhouse-server clickhouse-client

# 启动
systemctl enable --now clickhouse-server

# 等待就绪后执行初始化
sleep 3
clickhouse-client --multiquery < /root/shieldflow/sql/002_init_clickhouse.sql
```

### Q6: 边缘节点无法连接主控 gRPC

**排查步骤**：

```bash
# 1. 检查网络连通性（在边缘节点执行）
nc -zv 主控IP 50051
# 如失败：检查主控防火墙是否放行 50051 端口

# 2. 检查主控 gRPC 服务状态
supervisorctl status shieldflow-master:shieldflow-grpc-server
# 如非 RUNNING：查看日志并重启

# 3. 检查 edge.yaml 中 grpc.server 地址是否正确
grep 'server:' /etc/shieldflow/edge.yaml
# 确认格式为 "IP:50051"，如 "10.0.1.100:50051"

# 4. 检查 gRPC TLS 配置是否一致（主控和边缘节点）
# 如主控 tls: true，边缘节点也需 tls: true
```

### Q7: Nginx 启动失败 "cannot load certificate"

**原因**：SSL 证书文件不存在或路径错误。

**解决**：

```bash
# 1. 检查证书文件
ls -la /etc/shieldflow/certs/
# 预期: shieldflow.crt  shieldflow.key

# 2. 如不存在，生成自签名证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/shieldflow/certs/shieldflow.key \
    -out /etc/shieldflow/certs/shieldflow.crt \
    -subj "/CN=shieldflow/O=ShieldFlow"

# 3. 检查 Nginx 配置中的证书路径
grep ssl_certificate /etc/nginx/conf.d/shieldflow.conf

# 4. 测试并重启
nginx -t && systemctl restart nginx
```

### Q8: 管理后台无法访问（浏览器打不开）

**排查步骤**：

```bash
# 1. 检查 Nginx 状态
systemctl status nginx

# 2. 检查端口监听
ss -tlnp | grep -E ':80|:443'

# 3. 检查防火墙
firewall-cmd --list-ports    # CentOS
ufw status                   # Ubuntu

# 4. 检查后端 API
curl -s http://127.0.0.1:8080/api/v1/health

# 5. 检查 Nginx 配置
nginx -t

# 6. 查看错误日志
tail -f /var/log/nginx/error.log

# 7. 云服务器检查安全组是否放行 80/443
```

### Q9: PostgreSQL 密码忘记了怎么办

**解决**：

```bash
# 1. 修改 pg_hba.conf 允许信任连接（临时）
# CentOS: /var/lib/pgsql/data/pg_hba.conf
# Ubuntu: /etc/postgresql/14/main/pg_hba.conf

# 将对应行改为:
# local   all   all   trust
# host    all   all   127.0.0.1/32   trust

# 2. 重启 PostgreSQL
systemctl restart postgresql

# 3. 修改密码
sudo -u postgres psql -c "ALTER USER shieldflow WITH PASSWORD 'new_password';"

# 4. 恢复 pg_hba.conf 并重启
systemctl restart postgresql

# 5. 更新配置文件中的密码
# 修改 /etc/shieldflow/backend.yaml, grpc.yaml, dns-sync.yaml 中的 password 字段

# 6. 重启服务
supervisorctl restart shieldflow-master:*
```

### Q10: 数据库表初始化失败 "relation already exists"

**原因**：SQL 脚本使用 `CREATE TABLE IF NOT EXISTS`，重复执行会有警告但不影响功能。

**解决**：这是正常现象，可以忽略。如需彻底重建：

```bash
# 警告：以下操作会删除所有数据！
sudo -u postgres dropdb shieldflow_cdn
sudo -u postgres createdb shieldflow_cdn
sudo -u postgres psql -d shieldflow_cdn -f /root/shieldflow/sql/001_init_postgresql.sql
```

### Q11: ClickHouse 日志查询很慢

**原因**：数据量大时全表扫描慢。

**解决**：

```sql
-- 1. 确认分区裁剪生效（查询带时间条件）
SELECT count() FROM shieldflow_cdn.access_logs
WHERE domain = 'example.com'
  AND timestamp >= '2025-01-01 00:00:00'
  AND timestamp < '2025-01-08 00:00:00';

-- 2. 避免 SELECT *，只查询必要列
SELECT domain, status_code, count() as cnt
FROM shieldflow_cdn.access_logs
WHERE timestamp >= today() - 7
GROUP BY domain, status_code;

-- 3. 使用预聚合表 bandwidth_stats 查询带宽
SELECT domain, sum(traffic_bytes), max(bandwidth_bps)
FROM shieldflow_cdn.bandwidth_stats
WHERE timestamp >= today() - 30
GROUP BY domain;
```

### Q12: 如何修改默认管理员密码

```bash
# 方法一：通过管理后台
# 登录后 → 个人设置 → 修改密码

# 方法二：通过 API
curl -X POST http://127.0.0.1:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"admin123","new_password":"NewPassword@2025"}'

# 方法三：通过数据库（不推荐，仅用于重置）
# 需要使用 bcrypt 生成密码哈希
sudo -u postgres psql -d shieldflow_cdn -c \
  "UPDATE users SET password_hash='\$2a\$10\$NewBcryptHash' WHERE username='admin';"
```

---

## 18. 性能调优建议

### 18.1 PostgreSQL 调优

编辑 `postgresql.conf`（CentOS: `/var/lib/pgsql/data/`，Ubuntu: `/etc/postgresql/14/main/`）：

```ini
# 连接数（建议 = max_open_conns * 2 + 50）
max_connections = 250

# 内存（4GB 服务器示例）
shared_buffers = 1GB              # 25% 总内存
effective_cache_size = 3GB        # 75% 总内存
work_mem = 16MB
maintenance_work_mem = 256MB

# WAL
wal_buffers = 16MB
checkpoint_completion_target = 0.9
max_wal_size = 2GB

# 查询计划
random_page_cost = 1.1            # SSD 建议 1.1
effective_io_concurrency = 200    # SSD 建议 200

# 自动清理
autovacuum = on
autovacuum_max_workers = 3
```

重启生效：

```bash
systemctl restart postgresql
```

### 18.2 ClickHouse 调优

编辑 `/etc/clickhouse-server/config.xml` 或 `users.xml`：

```xml
<!-- config.xml -->
<max_concurrent_queries>100</max_concurrent_queries>
<max_connections>200</max_connections>

<!-- users.xml -->
<profiles>
  <default>
    <max_memory_usage>1000000000</max_memory_usage>      <!-- 1GB per query -->
    <max_bytes_before_external_group_by>500000000</max_bytes_before_external_group_by>
    <max_bytes_before_external_sort>500000000</max_bytes_before_external_sort>
    <use_uncompressed_cache>0</use_uncompressed_cache>
  </default>
</profiles>
```

重启生效：

```bash
systemctl restart clickhouse-server
```

### 18.3 Redis 调优

编辑 `/etc/redis.conf`（或 `/etc/redis/redis.conf`）：

```ini
maxmemory 512mb                 # 最大内存
maxmemory-policy allkeys-lru    # 淘汰策略
save ""                         # 禁用 RDB 持久化（如仅做缓存）
# 或启用 AOF
appendonly yes
appendfsync everysec
```

### 18.4 Nginx 调优

编辑 `/etc/nginx/nginx.conf`：

```nginx
worker_processes auto;           # 自动匹配 CPU 核数
worker_rlimit_nofile 65535;

events {
    worker_connections 10240;    # 每个进程最大连接数
    use epoll;                    # Linux 最优
    multi_accept on;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    keepalive_requests 100;
    server_tokens off;
    client_body_buffer_size 16k;
    client_max_body_size 50m;

    # Gzip
    gzip on;
    gzip_min_length 1k;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml image/svg+xml;
    gzip_vary on;

    # 限制连接数（防 CC）
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_conn_zone $binary_remote_addr zone=conn:10m;
}
```

### 18.5 系统内核调优

```bash
cat >> /etc/sysctl.conf << 'EOF'
# 网络缓冲区
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# TCP 连接优化
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_keepalive_time = 600
net.ipv4.ip_local_port_range = 1024 65535
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535

# 文件描述符
fs.file-max = 1048576
fs.inotify.max_user_watches = 524288

# 内存
vm.swappiness = 10
vm.overcommit_memory = 1
EOF
sysctl -p
```

### 18.6 边缘节点调优

编辑 `/etc/shieldflow/edge.yaml`：

```yaml
# 根据服务器配置调整
cache:
  max_size: "100GB"              # 增大缓存提升命中率
  ttl: "30m"                     # 适当延长 TTL
  compress: br                   # brotli 压缩率更高

ddos:
  max_connections_per_ip: 200    # 根据业务调整
  max_packets_per_sec: 10000     # 高流量节点可调高

cc:
  global_rate_limit: 5000        # 高流量节点可调高
  challenge_type: js             # JS 质询对用户友好
```

### 18.7 监控建议

```bash
# 1. 系统资源监控
# 安装 node_exporter + Prometheus + Grafana
# 关注指标: CPU、内存、磁盘 I/O、网络流量

# 2. 数据库监控
# PostgreSQL: pg_stat_activity, pg_stat_statements
sudo -u postgres psql -c "SELECT * FROM pg_stat_activity WHERE datname='shieldflow_cdn';"

# ClickHouse: system.metrics, system.events
clickhouse-client -q "SELECT * FROM system.metrics"
clickhouse-client -q "SELECT event, value FROM system.events"

# 3. 服务状态监控
# 定时检查 Supervisor 状态
echo "*/1 * * * * supervisorctl status | grep -v RUNNING | mail -s 'ShieldFlow Alert' admin@example.com" | crontab -

# 4. 日志监控
# 使用 ELK 或 Loki + Grafana 收集 /var/log/shieldflow/ 日志
```

### 18.8 容量规划参考

| 业务规模 | 边缘节点数 | 主控配置 | 数据库配置 | 日志保留 |
|----------|------------|----------|------------|----------|
| 小型 (<1k QPS) | 1-3 | 2C4G 50GB | 2C4G 50GB | 7 天 |
| 中型 (1k-10k QPS) | 3-10 | 4C8G 100GB | 4C8G 100GB | 30 天 |
| 大型 (10k-100k QPS) | 10-50 | 8C16G 200GB | 8C16G 200GB SSD | 90 天 |
| 超大型 (>100k QPS) | 50+ | 16C32G 500GB | 16C32G 500GB NVMe | 365 天 |

---

## 附录

### A. 卸载

```bash
# 卸载（保留数据库和配置）
sudo bash /root/shieldflow/scripts/uninstall.sh

# 彻底卸载（包括数据库和配置）
sudo bash /root/shieldflow/scripts/uninstall.sh --purge-data

# 保留 Nginx（如其他服务也在用）
sudo ShieldFlow_KEEP_NGINX=true bash /root/shieldflow/scripts/uninstall.sh
```

### B. 快速排障命令

```bash
# 一键检查所有服务
supervisorctl status

# 一键检查端口
ss -tlnp | grep -E '8080|50051|9528|9529|5432|9000|6379|80|443'

# 一键检查磁盘
df -h

# 一键检查内存
free -h

# 一键检查最新日志
tail -50 /var/log/shieldflow/backend.stderr.log
tail -50 /var/log/shieldflow/grpc-server.stderr.log
tail -50 /var/log/shieldflow/edge.stderr.log

# API 健康检查
curl -s http://127.0.0.1:8080/api/v1/health
```

### C. 文件清单

| 文件 | 路径 | 说明 |
|------|------|------|
| 安装脚本 | scripts/install.sh | 主控一键安装 |
| 边缘安装脚本 | scripts/install-edge.sh | 边缘节点安装 |
| 升级脚本 | scripts/upgrade.sh | 系统升级 |
| 备份脚本 | scripts/backup.sh | 数据备份 |
| 恢复脚本 | scripts/restore.sh | 数据恢复 |
| 卸载脚本 | scripts/uninstall.sh | 卸载 |
| Docker 入口 | scripts/docker-entrypoint.sh | Docker 容器入口 |
| Dockerfile | Dockerfile | 多阶段构建 |
| Docker Compose | docker-compose.yml | 全栈编排 |
| PostgreSQL SQL | sql/001_init_postgresql.sql | 24 张表初始化 |
| ClickHouse SQL | sql/002_init_clickhouse.sql | 6 张表初始化 |
| 后端配置模板 | deploy/backend.yaml | API 服务配置 |
| gRPC 配置模板 | deploy/grpc.yaml | gRPC 服务配置 |
| 边缘配置模板 | deploy/edge.yaml | 边缘节点配置 |
| DNS 同步配置 | deploy/dns-sync.yaml | DNS 同步配置 |
| 日志服务器配置 | deploy/log-server.yaml | 日志服务器配置 |
| Supervisor 配置 | deploy/supervisor/*.conf | 4 个进程组配置 |
| Nginx 配置 | deploy/nginx/zycdn.conf | 管理后台反代 |

---

> **文档版本**: 1.0
> **最后更新**: 2025-07-07
> **适用于**: ShieldFlow CDN v1.0+
> **仓库**: <https://github.com/717315051/shieldflow>
