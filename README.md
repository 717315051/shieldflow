<div align="center">

# 🛡️ ShieldFlow CDN（盾流）

### 企业级自建 CDN 与安全防护系统

**7 层安全防护链 · DDoS 清洗 · AI 智能防护 · 二级缓存加速 · 多云 DNS 同步**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue Version](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-passing-brightgreen)]()
[![Code Lines](https://img.shields.io/badge/Lines-~26K-orange)]()
[![Platform](https://img.shields.io/badge/Platform-Linux-888?logo=linux&logoColor=white)]()

[功能特性](#-功能特性) · [系统架构](#-系统架构) · [快速开始](#-快速开始) · [部署](#-部署方式) · [开发文档](docs/DEVELOPMENT.md) · [路线图](#-路线图)

</div>

---

ShieldFlow（盾流）是一套面向 MSP、IDC、企业用户的**自建 CDN 与边缘安全防护平台**。它在单一代码库中集成了反向代理、二级缓存、WAF、CC 防护、Bot 检测、四层 DDoS 清洗、AI 智能分析、SSL 自动签发与多云 DNS 同步，提供与商业 CDN 等价、甚至更灵活的控制能力。

- 🏗️ **主控 + 边缘** 分布式架构：主控下发配置，边缘节点执行转发与防护
- 🔐 **纵深防御**：黑白名单 → CC → 访问控制 → 区域 → Bot → 语义 WAF → 转发
- 🚀 **高性能缓存**：内存 + 磁盘二级缓存，Gzip/Brotli 压缩，HTTP/2 & HTTP/3
- 🤖 **AI 防护**：多模型适配（OpenAI / Azure / Ollama），敏感词、语义 WAF、威胁情报
- 🌐 **多云 DNS**：Cloudflare / 阿里云 / 腾讯云 DNSPod 自动 CNAME 接入
- 📊 **独立日志服务**：ClickHouse 批量写入，秒级查询，30~365 天 TTL
- 🔧 **一键运维**：install.sh / install-edge.sh / upgrade.sh / backup.sh / restore.sh

---

## ✨ 功能特性

### 🛡️ 7 层安全防护链

按纵深防御原则，请求依次经过以下 7 层中间件，任一层拦截即终止：

| 层级 | 模块 | 能力 |
|------|------|------|
| 1 | IP 黑白名单 | CIDR / 精确 / 前缀 / 正则匹配，自动封禁 |
| 2 | CC 防护 | 全局速率限制、路径级限速、JS 挑战 / 验证码 / 等候室 |
| 3 | 访问控制 | 路径密码、Basic Auth、HTTP 方法限制 |
| 4 | 区域限制 | 省份 / ASN 级黑白名单，防盗链 |
| 5 | Bot 检测 | UA 识别、搜索引擎白名单、爬虫/扫描器拦截 |
| 6 | 语义 WAF | SQL 注入、XSS、路径穿越、命令注入、XXE、CSRF 等 |
| 7 | 反向代理 | 负载均衡、健康检查、回源重试、WebSocket |

### 💥 DDoS 防护

- **四层（eBPF）**：SYN Flood / ACK Flood / UDP Flood / ICMP Flood / 连接洪水检测
- **七层（CC）**：基于速率与行为的 CC 攻击识别
- **自动封禁**：阈值触发后按规则自动拉黑，可配置封禁时长
- **黑白名单**：独立 DDoS IP 黑白名单，支持 IP / CIDR

### 🤖 AI 智能防护

- 多模型适配：OpenAI / Azure / Ollama 等
- **敏感词检测**：请求/响应内容审核
- **语义 WAF**：基于上下文语义的攻击识别（绕过传统正则）
- **威胁情报**：IP 信誉、DGA 检测、异常行为分析
- **日志分析**：AI 辅助安全事件归因与可视化

### 🚀 CDN 加速

- **二级缓存**：内存（LRU）+ 磁盘缓存，可分级配置容量
- **智能回源**：多源站负载均衡（轮询 / 加权 / IP Hash / 最少连接）
- **缓存策略**：按路径 / 后缀 / 状态码配置 TTL，支持 stale 复用
- **压缩**：Gzip + Brotli 自动协商
- **协议**：HTTP/2、HTTP/3 (QUIC)
- **刷新预热**：URL / 目录 / 全站刷新、文件预热、批量任务

### 👤 用户端

| 模块 | 说明 |
|------|------|
| 域名管理 | 接入、配置、批量创建、CNAME 校验 |
| 防护管理 | 防护模板、黑白名单（IP/URL/域名）、导入导出 |
| SSL 证书 | 上传 / ACME 自动签发 / 批量申请 |
| 日志管理 | 访问日志、攻击日志、四层日志、AI 日志、日志地图、导出 |
| 流量统计 | 带宽趋势、流量排行、缓存命中率 |
| 缓存管理 | 刷新 / 预热 / 任务跟踪 |
| 四层转发 | TCP/UDP 转发规则、负载均衡策略 |
| 套餐管理 | L7/L4 套餐、流量包、域名包、订单、余额 |

### 🛠️ 管理端

| 模块 | 说明 |
|------|------|
| 用户管理 | 创建/禁用/锁定、查看套餐 |
| 节点管理 | 注册、分组、SSH 安装、批量升级、状态监控 |
| 套餐管理 | 上下架、定价、流量包 / 域名包 |
| DDoS 防护 | 规则、黑白名单、连接日志、拦截日志 |
| 系统设置 | DNS / ACME / gRPC / 告警 / 监控 / AI 配置 |
| 数据备份 | 备份创建、恢复、版本管理 |

### 🌐 DNS 同步

自动 CNAME 校验与记录同步，支持：

- ☁️ Cloudflare
- 🅰️ 阿里云 DNS
- 🟩 腾讯云 DNSPod

### 📜 独立日志服务器

v1.2.0+ 起支持独立部署日志服务器（端口 9529）：

- 边缘节点 → gRPC 流式上报 → 日志服务器 → ClickHouse 批量写入
- 日志类型：访问日志 / 攻击日志 / DDoS 日志 / 四层日志 / AI 日志
- TTL：访问日志 30 天、攻击/DDoS 日志 90 天、AI 日志 365 天

### 🔏 SSL 证书

- ACME 自动申请（Let's Encrypt / ZeroSSL）
- DNS-01 / HTTP-01 验证
- 多 ACME 账户管理
- 证书到期自动续签

### ⚡ 批量操作

域名批量创建、防护模板批量应用、证书批量申请、节点批量升级、黑白名单批量导入导出。

---

## 🏗️ 系统架构

```
                              ┌─────────────────────────────────────────┐
                              │           用户 / 管理员浏览器              │
                              │         Vue 3 + Ant Design Vue          │
                              └────────────────┬────────────────────────┘
                                               │ HTTPS
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              主控节点 (Master)                                │
│                                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  │
│  │  Backend API │   │  gRPC Server │   │  DNS Sync    │   │  Log Server  │  │
│  │  Gin :8080   │   │  :50051      │   │  :9528       │   │  :9529       │  │
│  │  14 handlers │   │  5 services  │   │  CF/阿里/腾讯 │   │  CH 批量写入  │  │
│  └──────┬───────┘   └──────┬───────┘   └──────────────┘   └──────┬───────┘  │
│         │                  │                                       │          │
│         └────────┬─────────┴──────────────────────────────────────┘          │
│                  ▼                                                           │
│         ┌────────────┐  ┌────────────┐  ┌────────────┐                      │
│         │ PostgreSQL │  │  Redis     │  │ ClickHouse │                      │
│         │  24 表      │  │  缓存/限流  │  │  6 表日志   │                      │
│         └────────────┘  └────────────┘  └────────────┘                      │
└──────────────────────────────────────────────────────────────────────────────┘
                                  │ gRPC 下发配置 / 上报日志
                    ┌─────────────┴─────────────┐
                    ▼                           ▼
          ┌──────────────────┐        ┌──────────────────┐
          │  边缘节点 Edge-01 │  ...   │  边缘节点 Edge-N  │
          │                  │        │                  │
          │  反向代理 :80/443 │        │  反向代理 :80/443 │
          │  7层防护链        │        │  7层防护链        │
          │  二级缓存         │        │  二级缓存         │
          │  eBPF DDoS       │        │  eBPF DDoS       │
          │  Bot / WAF / CC  │        │  Bot / WAF / CC  │
          └────────┬─────────┘        └────────┬─────────┘
                   │                            │
                   └────────────┬───────────────┘
                                ▼
                        ┌──────────────┐
                        │   源站 Origin │
                        └──────────────┘
```

**数据流**：客户端请求 → 边缘节点（7 层防护 + 缓存） → 源站；日志 → gRPC 流式上报 → 日志服务器 → ClickHouse。

---

## 🧰 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端语言 | Go 1.22+ | 高性能、并发友好 |
| Web 框架 | Gin | HTTP API |
| ORM | GORM | PostgreSQL 模型映射 |
| RPC | gRPC + Protocol Buffers | 主控 ↔ 边缘通信 |
| 配置 | Viper | YAML 热加载 |
| 日志 | Zap | 结构化日志 |
| 关系数据库 | PostgreSQL 14+ | 24 张业务表 |
| 时序数据库 | ClickHouse 23+ | 6 张日志/统计表 |
| 缓存 | Redis 6+ | 限流 / 会话 / 热缓存 |
| 前端框架 | Vue 3 | Composition API |
| UI 组件 | Ant Design Vue 4 | 企业级组件库 |
| 构建工具 | Vite | 极速 HMR |
| 状态管理 | Pinia | 类型友好 |
| 图表 | ECharts + vue-echarts | 可视化 |
| 进程管理 | Supervisor | 多组件守护 |
| DDoS | eBPF + Rust | 四层清洗 |
| 容器 | Docker / Compose | 可选部署 |

---

## 📁 项目结构

```
shieldflow/
├── cmd/                        # 可执行入口
│   ├── backend/                #   API 服务 (Gin, :8080)
│   ├── grpc-server/            #   gRPC 服务 (:50051)
│   ├── edge/                   #   边缘节点 (反向代理 + WAF + CC + 缓存 + DDoS)
│   ├── dns-sync/               #   DNS 同步 (:9528)
│   ├── log-server/             #   独立日志服务器 (:9529, ClickHouse 批量写入)
│   ├── grpc-client/            #   gRPC 客户端调试工具
│   └── portal/                 #   门户入口
├── internal/
│   ├── config/                 #   配置加载 (Viper)
│   ├── models/                 #   23 个 GORM 模型
│   ├── middleware/             #   JWT / CORS / 限流 / 日志
│   ├── handlers/               #   14 个 HTTP handler
│   ├── storage/                #   PG / CH / Redis 连接
│   ├── grpc/                   #   gRPC 服务实现 (REST 模式)
│   ├── pkg/
│   │   ├── proxy/              #   反向代理 + 7 层中间件链
│   │   ├── waf/                #   WAF 语义引擎 + 托管规则
│   │   ├── cache/              #   二级缓存 (内存 + 磁盘)
│   │   ├── ddos/               #   eBPF DDoS 防护
│   │   ├── bot/                #   爬虫检测
│   │   ├── dns/                #   Cloudflare / 阿里云 / 腾讯云 Provider
│   │   ├── storage/            #   ClickHouse 批量写入 + 查询
│   │   ├── acme/               #   ACME 自动证书
│   │   └── ai/                 #   AI 智能防护
│   └── waf/                    #   CC 引擎
├── web/                        # Vue 3 + AntDesign 前端
│   └── src/
│       ├── views/              #   页面 (用户端 + 管理端)
│       ├── layouts/            #   布局 (UserLayout / AdminLayout)
│       ├── router/             #   路由
│       ├── store/              #   Pinia 状态
│       ├── api/                #   API 调用
│       └── utils/              #   工具 (request 等)
├── sql/                        # 数据库初始化
│   ├── 001_init_postgresql.sql #   PostgreSQL 24 表
│   └── 002_init_clickhouse.sql #   ClickHouse 6 表
├── proto/                      # gRPC proto 定义 (5 个服务)
├── deploy/                     # 部署配置
│   ├── *.yaml                  #   各组件配置模板
│   ├── supervisor/             #   Supervisor 配置
│   └── nginx/                  #   Nginx 反代配置
├── scripts/                    # 运维脚本
│   ├── install.sh              #   一键安装
│   ├── install-edge.sh         #   边缘节点安装
│   ├── upgrade.sh              #   升级
│   ├── backup.sh               #   备份
│   ├── restore.sh              #   恢复
│   ├── uninstall.sh            #   卸载
│   └── docker-entrypoint.sh    #   Docker 入口
├── Makefile                    # 构建入口
├── Dockerfile                  # 多阶段构建
├── docker-compose.yml          # 编排
└── docs/
    └── DEVELOPMENT.md          # 开发文档
```

---

## 🚀 快速开始

### 前置要求

- Linux 服务器（Ubuntu 18+ / Debian 10+ / CentOS 7+）
- root 权限
- PostgreSQL 14+、Redis 6+、ClickHouse 23+
- 公网 80/443 端口（边缘节点）

### 方式一：一键安装（推荐）

```bash
# 克隆仓库
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 一键安装（自动安装 Go/Node/PG/CH/Redis 并编译部署）
sudo bash scripts/install.sh

# 或通过 Makefile
make install
```

安装脚本会自动：
1. 检查系统环境
2. 安装依赖（Go、Node.js、PostgreSQL、ClickHouse、Redis、Nginx、Supervisor）
3. 编译后端 5 个组件 + 前端
4. 部署配置到 `/etc/shieldflow/`
5. 初始化数据库与 SQL
6. 注册 Supervisor 服务
7. 配置 Nginx 反代与防火墙

安装完成后访问 `https://<服务器IP>`，默认账号 `admin / admin123`（请立即修改）。

### 方式二：Docker Compose

```bash
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 可选：自定义密码
export ShieldFlow_DB_PASS=your_strong_password
export ShieldFlow_JWT_SECRET=your_jwt_secret

# 构建并启动
docker compose up -d

# 查看日志
docker compose logs -f
```

服务端口：
- `8080` Backend API
- `50051` gRPC
- `4430` Nginx 管理后台
- `80/443` 边缘节点

### 方式三：手动编译

```bash
# 后端
make build          # 编译全部 5 个组件到 /usr/local/shieldflow/bin/

# 前端
make web            # npm install + vite build → /usr/local/shieldflow/web/

# 部署配置
make deploy         # 拷贝 YAML / Supervisor / Nginx 配置

# 初始化数据库
sudo -u postgres psql -d shieldflow_cdn -f sql/001_init_postgresql.sql
clickhouse-client --multiquery < sql/002_init_clickhouse.sql
```

### 添加边缘节点

```bash
# 在主控管理端添加节点后，在边缘服务器执行：
sudo bash scripts/install-edge.sh --node-id edge-01 --master <主控IP>:50051

# 或通过 Makefile
make install-edge NODE_ID=edge-01 MASTER=<主控IP>:50051
```

---

## ⚙️ 配置说明

配置文件位于 `/etc/shieldflow/`，各组件独立配置：

| 文件 | 组件 | 默认端口 |
|------|------|----------|
| `backend.yaml` | API 服务 | 8080 |
| `grpc.yaml` | gRPC 服务 | 50051 |
| `edge.yaml` | 边缘节点 | 80/443 |
| `dns-sync.yaml` | DNS 同步 | 9528 |
| `log-server.yaml` | 日志服务器 | 9529 |

### backend.yaml 示例

```yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: production

database:
  driver: postgres
  host: 127.0.0.1
  port: 5432
  name: shieldflow_cdn
  user: shieldflow
  password: "CHANGE_ME_STRONG_PASSWORD"

redis:
  host: 127.0.0.1
  port: 6379

jwt:
  secret: "CHANGE_ME_TO_RANDOM_64_CHAR_STRING"
  expire: 24h

clickhouse:
  host: 127.0.0.1
  port: 9000
  database: shieldflow_cdn

ai:
  provider: openai
  model: gpt-4o-mini
  enabled: false

acme:
  directory: "https://acme-v02.api.letsencrypt.org/directory"
  email: "admin@example.com"
```

### 路径约定

| 用途 | 路径 |
|------|------|
| 二进制 | `/usr/local/shieldflow/bin/` |
| 前端静态 | `/usr/local/shieldflow/web/` |
| 配置 | `/etc/shieldflow/` |
| 证书 | `/etc/shieldflow/certs/` |
| 日志 | `/var/log/shieldflow/` |
| 缓存 | `/var/cache/shieldflow/` |
| 备份 | `/var/backups/shieldflow/` |

---

## 📡 API 文档概览

所有 API 前缀 `/api/v1`，JWT 鉴权（除登录/注册）。主要接口：

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 登录 |
| POST | `/auth/register` | 注册 |
| POST | `/auth/logout` | 登出 |
| GET | `/auth/captcha` | 验证码 |
| GET/PUT | `/auth/profile` | 个人资料 |
| PUT | `/auth/password` | 修改密码 |
| POST | `/auth/realname` | 实名认证 |

### 域名管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/domains` | 列表 / 创建 |
| POST | `/domains/batch` | 批量创建 |
| GET/PUT/DELETE | `/domains/:id` | 详情 / 更新 / 删除 |
| PUT | `/domains/:id/status` | 状态变更 |
| PUT | `/domains/:id/protection` | 防护配置 |
| POST | `/domains/:id/certificate` | 申请证书 |
| POST | `/domains/batch-certificate` | 批量申请证书 |

### SSL 证书

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/certificates` | 证书列表 |
| POST | `/certificates/upload` | 上传证书 |
| POST | `/certificates/apply` | ACME 申请 |
| GET/POST | `/certificates/acme-accounts` | ACME 账户 |
| GET/POST | `/certificates/dns-accounts` | DNS 账户 |

### 日志 / 流量 / 缓存 / 四层 / 防护 / 套餐

| 模块 | 主要路径 |
|------|----------|
| 日志 | `/logs/access` `/logs/attack` `/logs/layer4` `/logs/ai` `/logs/export` `/logs/map` |
| 流量 | `/traffic/stats` `/traffic/ranking` `/traffic/bandwidth` `/traffic/cache` |
| 缓存 | `/cache/file-refresh` `/cache/dir-refresh` `/cache/file-preheat` `/cache/tasks` |
| 四层 | `/layer4` CRUD |
| 防护 | `/protection/templates` `/protection/blacklists` 导入导出 |
| 套餐 | `/packages` `/user-packages` `/orders` `/balance` |
| 仪表盘 | `/dashboard/analysis` |

### 管理端（`/admin/*`，需 admin 角色）

用户、节点、节点分组、套餐、DDoS 规则/黑白名单/日志、系统设置（DNS/ACME/gRPC/告警/监控/AI/备份/版本）、数据备份恢复。

完整路由定义见 [`internal/handlers/router.go`](internal/handlers/router.go)，开发文档见 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)。

---

## 🖥️ 前端界面

ShieldFlow 前端区分**用户端**与**管理端**两套布局：

**用户端**（`/` 路径，UserLayout）：
- 📊 仪表盘 — 流量/请求/带宽/缓存命中率概览
- 🌐 域名管理 — 接入、配置、CNAME 校验
- 🔏 SSL 证书 — 上传 / ACME 自动签发
- 📜 日志管理 — 访问/攻击/四层/AI 日志 + 日志地图
- 📈 流量统计 — 带宽趋势、排行
- 🗂️ 缓存管理 — 刷新 / 预热
- 🔀 四层转发 — TCP/UDP 规则
- 🛡️ 防护管理 — 模板 / 黑白名单
- 📦 套餐管理 — 购买 / 订单 / 余额

**管理端**（`/admin` 路径，AdminLayout）：
- 👥 用户管理
- 🖧 节点管理 — 注册 / 分组 / SSH 安装 / 升级
- 📦 套餐管理
- 💥 DDoS 防护 — 规则 / 黑白名单 / 日志
- ⚙️ 系统设置 — DNS / ACME / gRPC / 告警 / AI
- 💾 数据备份

> 截图将在后续版本补充。欢迎 PR 提交你的部署截图！

---

## 🛣️ 路线图

- [x] v1.0 — 核心功能：域名/防护/证书/日志/流量/缓存/套餐
- [x] v1.1 — DDoS 防护（eBPF 四层 + 七层 CC）
- [x] v1.2 — 独立日志服务器 + ClickHouse 批量写入
- [x] v1.3 — AI 智能防护（多模型）
- [x] v1.4 — 多云 DNS 同步（Cloudflare/阿里/腾讯）
- [ ] v1.5 — WebSocket 全链路支持、Anycast 调度
- [ ] v1.6 — 多租户隔离、RBAC 细粒度权限
- [ ] v1.7 — 边缘 Serverless（边缘函数 / WASM）
- [ ] v2.0 — 分布式集群、节点自动扩缩容

---

## 🤝 贡献

欢迎提交 Issue 与 PR！请先阅读 [开发文档](docs/DEVELOPMENT.md) 了解项目结构与代码规范。

```bash
# Fork 后
git checkout -b feature/your-feature
make fmt && make vet && make test
git commit -m "feat: add your feature"
git push origin feature/your-feature
# 发起 Pull Request
```

---

## 📄 开源协议

ShieldFlow 基于 [MIT License](LICENSE) 开源。

---

## 🙏 致谢

ShieldFlow 的诞生离不开以下开源项目：

- [Go](https://go.dev/) · [Gin](https://github.com/gin-gonic/gin) · [GORM](https://gorm.io) · [gRPC](https://grpc.io)
- [Viper](https://github.com/spf13/viper) · [Zap](https://github.com/uber-go/zap)
- [Vue](https://vuejs.org/) · [Ant Design Vue](https://antdv.com/) · [Vite](https://vitejs.dev/) · [ECharts](https://echarts.apache.org/)
- [PostgreSQL](https://www.postgresql.org/) · [ClickHouse](https://clickhouse.com/) · [Redis](https://redis.io/)
- [Supervisor](http://supervisord.org/) · [Nginx](https://nginx.org/)
- [Let's Encrypt](https://letsencrypt.org/) · [Cloudflare](https://www.cloudflare.com/)

感谢所有为开源社区贡献力量的开发者。

<div align="center">

**⭐ 如果 ShieldFlow 对你有帮助，请给个 Star！**

</div>
