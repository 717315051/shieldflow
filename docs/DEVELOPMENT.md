# ShieldFlow 开发文档

本文档面向希望参与 ShieldFlow 开发的工程师，涵盖开发环境、项目结构、数据库设计、API 接口、gRPC 服务、前端开发、代码规范与贡献流程。

> 仓库：https://github.com/717315051/shieldflow
> Go module：`github.com/shieldflow/shieldflow`

---

## 目录

- [开发环境要求](#开发环境要求)
- [项目目录结构详解](#项目目录结构详解)
- [数据库设计文档](#数据库设计文档)
- [API 接口文档](#api-接口文档)
- [gRPC 服务定义](#grpc-服务定义)
- [前端开发指南](#前端开发指南)
- [代码规范](#代码规范)
- [测试指南](#测试指南)
- [贡献指南](#贡献指南)

---

## 开发环境要求

### 必需

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.22+（go.mod 声明 1.25.0） | 后端编译 |
| Node.js | 18+ | 前端构建 |
| npm | 随 Node 安装 | 依赖管理 |
| PostgreSQL | 14+ | 关系型业务数据 |
| ClickHouse | 23+ | 日志与统计 |
| Redis | 6+ | 缓存 / 限流 / 会话 |
| Git | 任意 | 版本控制 |
| protoc + protoc-gen-go | 任意 | 生成 gRPC 代码（仅修改 proto 时需要） |

### 可选

| 工具 | 用途 |
|------|------|
| Docker + Compose | 容器化部署 / 一键起依赖 |
| Supervisor | 进程管理（生产） |
| Nginx | 反向代理（生产） |
| golangci-lint | 静态检查 |
| clang | eBPF/Rust 编译 |

### 环境初始化

```bash
# 克隆
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# Go 依赖
export GOPROXY=https://goproxy.cn,direct   # 国内推荐
go mod download

# 前端依赖
cd web && npm install && cd ..

# 起依赖（Docker 方式最快）
docker compose up -d postgres clickhouse redis

# 初始化数据库
sudo -u postgres psql -c "CREATE DATABASE shieldflow_cdn;"
sudo -u postgres psql -d shieldflow_cdn -f sql/001_init_postgresql.sql
clickhouse-client --multiquery < sql/002_init_clickhouse.sql

# 复制配置
mkdir -p /etc/shieldflow/certs
cp deploy/backend.yaml /etc/shieldflow/
cp deploy/grpc.yaml    /etc/shieldflow/
# 按需修改密码 / JWT secret
```

### 开发运行

```bash
# 后端 API（开发模式，热修改）
go run ./cmd/backend

# gRPC 服务
go run ./cmd/grpc-server

# 边缘节点
go run ./cmd/edge

# DNS 同步
go run ./cmd/dns-sync

# 日志服务器
go run ./cmd/log-server

# 前端开发服务器（HMR）
cd web && npm run dev
```

---

## 项目目录结构详解

ShieldFlow 采用 **单仓多组件** 架构：一个 Go module 下包含 5 个可执行入口，共享 `internal/` 下的业务包。

### `cmd/` — 可执行入口

每个子目录对应一个独立二进制，`main.go` 仅做初始化与启动：

| 目录 | 组件 | 端口 | 职责 |
|------|------|------|------|
| `cmd/backend/` | API 服务 | 8080 | Gin HTTP API，对接前端，CRUD 全部业务 |
| `cmd/grpc-server/` | gRPC 服务 | 50051 | 主控 gRPC，下发配置 / 接收日志 / 节点管理 |
| `cmd/edge/` | 边缘节点 | 80/443 | 反向代理 + 7 层防护 + 缓存 + DDoS |
| `cmd/dns-sync/` | DNS 同步 | 9528 | Cloudflare/阿里云/腾讯云 CNAME 同步 |
| `cmd/log-server/` | 日志服务器 | 9529 | 接收边缘日志，ClickHouse 批量写入 |
| `cmd/grpc-client/` | gRPC 客户端 | — | 调试工具 |
| `cmd/portal/` | 门户 | — | 门户入口 |

### `internal/` — 共享业务包

#### `internal/config/`
基于 Viper 的配置加载，支持 YAML 文件、环境变量、命令行参数。`Config` 结构体聚合了 server / database / redis / jwt / clickhouse / grpc / ai / acme / log 等所有配置段。提供 `IsProduction()` 等辅助方法。

#### `internal/models/`
23 个 GORM 模型，与 PostgreSQL 表一一对应：
`User` `Node` `NodeGroup` `Domain` `Package` `UserPackage` `Certificate` `AcmeAccount` `DNSAccount` `BlacklistEntry` `ProtectionTemplate` `Layer4Forward` `CacheTask` `SystemSetting` `DDoSRule` `DDoSBlacklistEntry` `LogServerConfig` `AIConfigModel` `Order` `OperationLog` `TrafficPackage` `DomainPackage` `CertificateRequest`。

所有模型带 `gorm.DeletedAt` 实现软删除，密码/密钥字段打 `json:"-"` 防止泄露。

#### `internal/middleware/`
Gin HTTP 中间件：
- `JWTMiddleware` — JWT 解析与注入用户上下文
- `CORSMiddleware` — 跨域
- `LoggerMiddleware` — 访问日志
- `RateLimitMiddleware` — 基于 Redis 的限流
- `AdminOnly()` — 管理员鉴权

#### `internal/handlers/`
14 个 HTTP handler，按业务模块组织：

| 文件 | 模块 |
|------|------|
| `auth.go` | 登录 / 注册 / 验证码 / 实名 |
| `user.go` | 管理端用户管理 |
| `domain.go` | 域名 CRUD + 配置 + 批量 |
| `node.go` | 节点 / 节点分组 / SSH 安装 |
| `package.go` | 套餐 / 订单 / 余额 |
| `certificate.go` | 证书上传 / ACME / 账户 |
| `log.go` | 日志查询 / 导出 / 地图 |
| `traffic.go` | 流量统计 / 排行 / 带宽 |
| `cache.go` | 缓存刷新 / 预热 / 任务 |
| `layer4.go` | 四层转发 CRUD |
| `protection.go` | 防护模板 / 黑白名单 |
| `ddos.go` | DDoS 规则 / 黑白名单 / 日志 |
| `system.go` | 系统设置 / 备份 / 升级 |
| `dashboard.go` | 仪表盘聚合 |
| `router.go` | **路由注册总入口** |

`router.go` 的 `SetupRouter()` 是所有路由的注册点，依赖通过 `gin.Context` 注入（db / ch / rdb / config）。

#### `internal/storage/`
- `postgres.go` — GORM 连接 PostgreSQL
- `clickhouse.go` — ClickHouse 连接
- `redis.go` — Redis 客户端

#### `internal/grpc/`
gRPC 服务端实现（REST 模式，未生成桩代码，直接实现接口）：
- `server.go` — gRPC server 启动
- `handler.go` — 5 个 service 的方法实现
- `client.go` — 边缘节点侧 gRPC 客户端
- `types.go` — 本地类型定义

### `internal/pkg/` — 功能引擎

| 包 | 职责 |
|----|------|
| `pkg/proxy/` | 反向代理 + 7 层中间件链。`reverse_proxy.go` 负责回源；`middleware.go` 实现 `MiddlewareChain`，按顺序执行黑白名单 → CC → 访问控制 → 区域 → Bot → WAF → 缓存 → 转发。`LayerSwitch` 控制每层开关。 |
| `pkg/waf/` | WAF 语义引擎。`waf.go` 核心检测逻辑；`managed_rules.go` 托管规则集（SQL 注入 / XSS / 路径穿越 / 命令注入 / XXE 等）。 |
| `pkg/cache/` | 二级缓存。内存层 LRU + 磁盘层，支持 TTL、stale 复用、分片缓存。 |
| `pkg/ddos/` | eBPF 四层 DDoS 防护。`Guard` 结构维护阈值、自动封禁、黑白名单。 |
| `pkg/bot/` | 爬虫检测引擎。UA 识别、搜索引擎白名单、扫描器拦截。 |
| `pkg/dns/` | 多云 DNS Provider。`provider.go` 定义接口；`cloudflare.go` / `aliyun.go` / `tencent.go` 实现；`manager.go` 调度。 |
| `pkg/storage/` | ClickHouse 批量写入与查询。`clickhouse_writer.go` 批量 buffer；`clickhouse_queries.go` 查询封装；`models.go` 日志结构体。 |
| `pkg/acme/` | ACME 自动证书申请（Let's Encrypt / ZeroSSL），DNS-01 / HTTP-01。 |
| `pkg/ai/` | AI 智能防护。多模型适配，敏感词 / 语义 WAF / 威胁情报。 |

### `internal/waf/`
CC 防护引擎（`cc.go`），基于速率与行为的 CC 攻击识别，支持 JS 挑战 / 验证码 / 等候室。

### `web/` — 前端

Vue 3 + Ant Design Vue 4 + Vite + Pinia。详见 [前端开发指南](#前端开发指南)。

### `sql/` — 数据库初始化

- `001_init_postgresql.sql` — PostgreSQL 24 表 + 索引 + 触发器 + 默认数据
- `002_init_clickhouse.sql` — ClickHouse 6 表（MergeTree，分区 + TTL）

### `proto/` — gRPC 定义

`zycdn.proto`（1548 行），定义 5 个 service。详见 [gRPC 服务定义](#grpc-服务定义)。

### `deploy/` — 部署配置

- `backend.yaml` / `grpc.yaml` / `edge.yaml` / `dns-sync.yaml` / `log-server.yaml` — 各组件配置模板
- `supervisor/*.conf` — Supervisor 进程配置
- `nginx/shieldflow.conf` — Nginx 反代配置

### `scripts/` — 运维脚本

| 脚本 | 功能 |
|------|------|
| `install.sh` | 主控一键安装 |
| `install-edge.sh` | 边缘节点安装 |
| `upgrade.sh` | 升级 |
| `backup.sh` | 数据备份 |
| `restore.sh` | 数据恢复 |
| `uninstall.sh` | 卸载 |
| `docker-entrypoint.sh` | Docker 容器入口 |

---

## 数据库设计文档

ShieldFlow 使用 **PostgreSQL**（关系型业务数据）+ **ClickHouse**（日志与统计）双库架构。

### PostgreSQL — 24 张表

| # | 表名 | 用途 |
|---|------|------|
| 1 | `users` | 用户表，含普通用户与管理员，bcrypt 密码哈希 |
| 2 | `node_groups` | 节点逻辑分组 |
| 3 | `nodes` | CDN 边缘节点，含 IP/区域/规格/心跳/gRPC 地址 |
| 4 | `packages` | 套餐定义（L7 七层 / L4 四层），流量/带宽/域名限额 |
| 5 | `user_packages` | 用户已购套餐实例，记录已用流量/域名数/到期 |
| 6 | `domains` | 加速域名，含源站/HTTPS/缓存/防护等 JSONB 配置 |
| 7 | `certificates` | SSL 证书 PEM 存储 |
| 8 | `acme_accounts` | ACME 自动签发账户（Let's Encrypt 等） |
| 9 | `dns_accounts` | 第三方 DNS 服务商账户（CF/阿里/腾讯） |
| 10 | `blacklists` | 黑白名单（IP/URL/域名，支持 exact/prefix/suffix/regex/cidr） |
| 11 | `protection_templates` | 防护模板（WAF/CC/Bot 配置 JSON） |
| 12 | `layer4_forwards` | TCP/UDP 四层转发规则，负载均衡策略 |
| 13 | `cache_tasks` | 缓存刷新/预热任务（file_refresh/dir_refresh/file_preheat） |
| 14 | `system_settings` | 全局键值配置 |
| 15 | `ddos_rules` | DDoS 防护规则（global/node/domain 作用域） |
| 16 | `ddos_blacklist` | DDoS IP 黑名单 |
| 17 | `ddos_whitelist` | DDoS IP 白名单（独立表加速查询） |
| 18 | `log_server_config` | 日志服务器/主节点通信配置 |
| 19 | `ai_config` | AI 大模型配置（provider/model/api_key） |
| 20 | `traffic_packages` | 可单独购买的流量补充包 |
| 21 | `domain_packages` | 可单独购买的域名额度补充包 |
| 22 | `orders` | 购买记录（package/traffic/domain） |
| 23 | `operation_logs` | 操作审计日志 |
| 24 | `balances` | 用户账户余额（可用 + 冻结） |

**设计要点**：
- 所有业务表使用 `BIGSERIAL` 主键
- `domains` 表大量使用 `JSONB` 存储灵活配置（源站/HTTPS/缓存/防护），并建 GIN 索引加速查询
- `updated_at` 通过触发器 `update_updated_at_column()` 自动维护
- 软删除：GORM 层使用 `gorm.DeletedAt`，SQL 层不强制
- 初始化脚本插入默认管理员（`admin`，密码哈希需重置）、默认节点分组、默认系统设置

### ClickHouse — 6 张表

所有表引擎 `MergeTree()`，按时间分区，配 TTL 自动清理。

| # | 表名 | 分区 | TTL | ORDER BY | 用途 |
|---|------|------|-----|----------|------|
| 1 | `access_logs` | 按天 | 30 天 | (domain, timestamp, client_ip) | 七层 HTTP/HTTPS 访问日志 |
| 2 | `attack_logs` | 按月 | 90 天 | (domain, timestamp, client_ip) | WAF/CC/Bot 安全事件 |
| 3 | `ddos_logs` | 按月 | 90 天 | (domain, timestamp, client_ip) | DDoS 防护事件 |
| 4 | `layer4_logs` | 按天 | 30 天 | (domain, timestamp, src_ip) | 四层转发事件 |
| 5 | `ai_logs` | 按月 | 365 天 | (domain, timestamp) | AI 分析调用记录 |
| 6 | `bandwidth_stats` | 按月 | 365 天 | (domain, timestamp) | 带宽/流量/请求数预聚合 |

**设计要点**：
- 访问日志/四层日志高频写入，TTL 30 天控制体积
- 攻击/DDoS 日志保留 90 天用于回溯
- AI 日志与带宽统计保留 1 年用于长期分析
- `index_granularity = 8192` 平衡查询与压缩
- 写入由 `internal/pkg/storage/clickhouse_writer.go` 批量 buffer 后 flush

---

## API 接口文档

所有 API 前缀 `/api/v1`，返回 JSON。除 `/auth/login`、`/auth/register`、`/auth/captcha` 外均需 JWT（`Authorization: Bearer <token>`）。管理端接口需 admin 角色。

路由注册见 `internal/handlers/router.go`。

### 认证模块 `/auth`

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/auth/login` | 公开 | 登录，返回 JWT |
| POST | `/auth/register` | 公开 | 注册 |
| POST | `/auth/logout` | 用户 | 登出 |
| GET | `/auth/captcha` | 公开 | 图形验证码 |
| POST | `/auth/verify-code` | 公开 | 校验验证码 |
| GET | `/auth/profile` | 用户 | 获取个人资料 |
| PUT | `/auth/profile` | 用户 | 更新资料 |
| PUT | `/auth/password` | 用户 | 修改密码 |
| POST | `/auth/realname` | 用户 | 实名认证 |

### 域名管理 `/domains`

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/domains` | 用户 | 域名列表（分页） |
| POST | `/domains` | 用户 | 创建域名 |
| POST | `/domains/batch` | 用户 | 批量创建 |
| GET | `/domains/:id` | 用户 | 域名详情 |
| PUT | `/domains/:id` | 用户 | 更新域名 |
| DELETE | `/domains/:id` | 用户 | 删除域名 |
| PUT | `/domains/:id/status` | 用户 | 变更状态 |
| PUT | `/domains/:id/package` | 用户 | 变更套餐 |
| GET | `/domains/:id/config` | 用户 | 获取完整配置 |
| PUT | `/domains/:id/basic` | 用户 | 保存基础配置 |
| PUT | `/domains/:id/protection` | 用户 | 保存防护配置 |
| PUT | `/domains/:id/custom-pages` | 用户 | 保存自定义错误页 |
| POST | `/domains/:id/certificate` | 用户 | 申请证书 |
| POST | `/domains/batch-certificate` | 用户 | 批量申请证书 |

### SSL 证书 `/certificates`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/certificates` | 证书列表 |
| POST | `/certificates/upload` | 上传证书 |
| GET | `/certificates/:id` | 证书详情 |
| DELETE | `/certificates/:id` | 删除证书 |
| GET | `/certificates/:id/download` | 下载证书 |
| GET | `/certificates/requests` | 申请记录列表 |
| POST | `/certificates/apply` | ACME 申请 |
| GET | `/certificates/requests/:id` | 申请详情 |
| GET | `/certificates/requests/:id/log` | 申请日志 |
| DELETE | `/certificates/requests/:id` | 删除申请 |
| GET | `/certificates/acme-accounts` | ACME 账户列表 |
| POST | `/certificates/acme-accounts` | 创建 ACME 账户 |
| GET | `/certificates/dns-accounts` | DNS 账户列表 |
| POST | `/certificates/dns-accounts` | 创建 DNS 账户 |

### 日志管理 `/logs`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/logs/access` | 访问日志（ClickHouse） |
| GET | `/logs/attack` | 攻击日志 |
| GET | `/logs/layer4` | 四层日志 |
| GET | `/logs/layer4-intercept` | 四层拦截日志 |
| GET | `/logs/ai` | AI 日志 |
| POST | `/logs/export` | 导出日志（blob） |
| GET | `/logs/map` | 日志地图（地理分布） |

### 流量统计 `/traffic`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/traffic/stats` | 流量统计 |
| GET | `/traffic/ranking` | 流量排行 |
| GET | `/traffic/bandwidth` | 带宽趋势 |
| GET | `/traffic/cache` | 缓存命中率 |

### 缓存管理 `/cache`

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/cache/file-refresh` | 文件刷新 |
| POST | `/cache/dir-refresh` | 目录刷新 |
| POST | `/cache/file-preheat` | 文件预热 |
| GET | `/cache/tasks` | 任务列表 |
| GET | `/cache/tasks/:id` | 任务详情 |
| POST | `/cache/tasks/:id/cancel` | 取消任务 |

### 四层转发 `/layer4`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/layer4` | 列表 |
| POST | `/layer4` | 创建 |
| PUT | `/layer4/:id` | 更新 |
| DELETE | `/layer4/:id` | 删除 |
| PUT | `/layer4/:id/status` | 状态变更 |

### 防护管理 `/protection`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/protection/templates` | 模板列表 |
| POST | `/protection/templates` | 创建模板 |
| PUT | `/protection/templates/:id` | 更新模板 |
| DELETE | `/protection/templates/:id` | 删除模板 |
| POST | `/protection/templates/:id/apply` | 应用模板到域名 |
| GET | `/protection/templates/system` | 系统模板列表 |
| POST | `/protection/templates/system/:id/apply` | 应用系统模板 |
| GET | `/protection/blacklists` | 黑白名单列表 |
| POST | `/protection/blacklists` | 添加名单 |
| DELETE | `/protection/blacklists/:id` | 删除名单 |
| POST | `/protection/blacklists/import` | 批量导入 |
| GET | `/protection/blacklists/export` | 批量导出 |

### 套餐与订单 `/packages` `/user-packages` `/orders` `/balance`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/packages` | 套餐市场 |
| GET | `/packages/traffic` | 流量包 |
| GET | `/packages/domain` | 域名包 |
| POST | `/packages/:id/purchase` | 购买套餐 |
| POST | `/packages/traffic/:id/purchase` | 购买流量包 |
| POST | `/packages/domain/:id/purchase` | 购买域名包 |
| GET | `/user-packages` | 我的套餐 |
| GET | `/user-packages/:id` | 套餐详情 |
| POST | `/user-packages/:id/renew` | 续费 |
| GET | `/orders` | 订单列表 |
| GET | `/orders/:id` | 订单详情 |
| GET | `/balance` | 余额 |
| POST | `/balance/recharge` | 充值 |

### 仪表盘 `/dashboard`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/dashboard/analysis` | 综合分析数据 |

### 管理端 `/admin`（需 admin 角色）

**用户管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/users` | 用户列表 |
| POST | `/admin/users` | 创建用户 |
| PUT | `/admin/users/:id` | 更新用户 |
| DELETE | `/admin/users/:id` | 删除用户 |
| PUT | `/admin/users/:id/status` | 变更状态 |
| GET | `/admin/users/:id/packages` | 用户套餐 |

**节点管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/nodes` | 节点列表 |
| POST | `/admin/nodes` | 添加节点 |
| GET | `/admin/nodes/:id` | 节点详情 |
| PUT | `/admin/nodes/:id` | 更新节点 |
| DELETE | `/admin/nodes/:id` | 删除节点 |
| POST | `/admin/nodes/:id/install` | 安装命令 |
| POST | `/admin/nodes/:id/ssh-install` | SSH 远程安装 |
| POST | `/admin/nodes/:id/upgrade` | 升级节点 |
| POST | `/admin/nodes/batch-upgrade` | 批量升级 |
| GET | `/admin/nodes/:id/status` | 节点状态 |
| GET/POST/PUT/DELETE | `/admin/node-groups[/:id]` | 节点分组 CRUD |

**套餐管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/admin/packages` | 列表/创建 |
| PUT/DELETE | `/admin/packages/:id` | 更新/删除 |
| POST | `/admin/packages/traffic` | 创建流量包 |
| POST | `/admin/packages/domain` | 创建域名包 |

**DDoS 防护**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/ddos/dashboard` | DDoS 仪表盘 |
| GET/POST/PUT/DELETE | `/admin/ddos/rules[/:id]` | 规则 CRUD |
| GET/POST/DELETE | `/admin/ddos/blacklist[/:id]` | 黑名单 |
| GET/POST/DELETE | `/admin/ddos/whitelist[/:id]` | 白名单 |
| GET | `/admin/ddos/logs` | 连接日志 |
| GET | `/admin/ddos/intercept-logs` | 拦截日志 |

**系统设置**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/PUT | `/admin/system/settings` | 系统设置 |
| GET/PUT | `/admin/system/dns` | DNS 配置 |
| GET/PUT | `/admin/system/acme` | ACME 配置 |
| GET/PUT | `/admin/system/grpc` | gRPC 配置 |
| POST | `/admin/system/grpc/test-log-server` | 测试日志服务器 |
| GET/PUT | `/admin/system/alert` | 告警配置 |
| GET/PUT | `/admin/system/monitor` | 监控配置 |
| GET/PUT | `/admin/system/ai` | AI 配置 |
| GET/POST | `/admin/system/backup` | 备份列表/创建 |
| POST | `/admin/system/backup/:id/restore` | 恢复备份 |
| GET | `/admin/system/version` | 版本信息 |
| POST | `/admin/system/upgrade` | 系统升级 |

### 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { },
  "timestamp": 1690000000
}
```

错误时 `code` 非 0，`message` 为错误描述。

---

## gRPC 服务定义

Proto 文件：`proto/zycdn.proto`（1548 行），package `shieldflow`。

共定义 **5 个 service**，覆盖主控 ↔ 边缘节点的全部通信。

### 1. ConfigService — 配置下发

主控 → 边缘，下发各类配置并同步状态。

| 方法 | 请求 | 响应 | 说明 |
|------|------|------|------|
| `PushDomainConfig` | `PushDomainConfigRequest{domain_id, DomainConfig}` | `PushResponse` | 下发域名配置 |
| `PushDDoSConfig` | `PushDDoSConfigRequest{DDoSConfig}` | `PushResponse` | 下发 DDoS 防护配置 |
| `PushGlobalConfig` | `PushGlobalConfigRequest{GlobalConfig}` | `PushResponse` | 下发全局配置 |
| `SyncNodeStatus` | `SyncNodeStatusRequest{node_id}` | `NodeStatus` | 同步节点状态 |
| `Heartbeat` | `HeartbeatRequest{node_id, NodeMetrics}` | `HeartbeatResponse` | 节点心跳，返回 `NodeAction` 指令 |

`NodeAction` 指令类型（`ActionType` 枚举）：
`RELOAD_CONFIG` `PURGE_CACHE` `RESTART_SERVICE` `UPDATE_CERTIFICATE` `ENABLE_MAINTENANCE` `DISABLE_MAINTENANCE`

### 2. LogService — 日志上报

边缘 → 日志服务器，全部为 **客户端流式**（`stream` 请求）。

| 方法 | 请求流 | 响应 | 说明 |
|------|--------|------|------|
| `ReportAccessLogs` | `stream AccessLogEntry` | `BatchResponse` | 访问日志 |
| `ReportAttackLogs` | `stream AttackLogEntry` | `BatchResponse` | 攻击日志 |
| `ReportDDoSLogs` | `stream DDoSLogEntry` | `BatchResponse` | DDoS 日志 |
| `ReportLayer4Logs` | `stream Layer4LogEntry` | `BatchResponse` | 四层日志 |
| `ReportAILogs` | `stream AILogEntry` | `BatchResponse` | AI 日志 |

`BatchResponse` 返回成功/拒绝计数与错误详情列表。

`AttackType` 枚举覆盖：SQL 注入、XSS、CC、路径穿越、命令注入、文件包含、XXE、CSRF、Bot、扫描器、自定义规则、速率限制、区域封锁、IP 黑名单。

`DDoSAttackType` 枚举覆盖：SYN/ACK/UDP/ICMP Flood、Slowloris、连接洪水、端口扫描、放大攻击、零窗口。

`Action` 枚举：`ALLOW` `BLOCK` `CHALLENGE` `CAPTCHA` `RATE_LIMIT` `LOG_ONLY` `REDIRECT` `JS_CHALLENGE`。

### 3. NodeService — 节点管理

| 方法 | 请求 | 响应 | 说明 |
|------|------|------|------|
| `RegisterNode` | `NodeInfo` | `RegisterResponse` | 节点注册，返回 auth_token + 初始配置 |
| `UpdateNodeStatus` | `UpdateNodeStatusRequest{node_id, NodeStatus}` | `UpdateResponse` | 更新状态 |
| `GetNodeConfig` | `GetNodeConfigRequest{node_id}` | `NodeConfig` | 拉取节点配置 |

`NodeInfo` 包含：node_id / hostname / ip / role / status / version / region / zone / isp / 规格 / 证书信息。

`NodeRole` 枚举：`EDGE` `ORIGIN_SHIELD` `LOG_SERVER` `CONTROL`。

`NodeState` 枚举：`ONLINE` `OFFLINE` `DRAINING` `MAINTENANCE` `OVERLOADED` `DEGRADED`。

`NodeMetrics` 涵盖 CPU / 内存 / 磁盘 / 网络 / 连接 / 业务指标 / 四层指标 / 防护指标。

### 4. CacheService — 缓存管理

| 方法 | 请求 | 响应 | 说明 |
|------|------|------|------|
| `PurgeCache` | `PurgeCacheRequest{domain_id, urls, type, directories, tags}` | `PurgeResponse` | 刷新缓存（URL/目录/全站/标签/正则） |
| `PreheatCache` | `PreheatCacheRequest{domain_id, urls, concurrency, target_nodes}` | `PreheatResponse` | 预热缓存 |
| `GetCacheStats` | `GetCacheStatsRequest{domain_id}` | `CacheStats` | 缓存统计 |

`PurgeType` 枚举：`URL` `DIRECTORY` `FULL` `TAG` `REGEX`。

`CacheStats` 包含：缓存空间/容量/对象数/命中率/状态码分布/按节点统计。

### 5. AuthService — 授权验证

| 方法 | 请求 | 响应 | 说明 |
|------|------|------|------|
| `VerifyLicense` | `VerifyLicenseRequest{license_key, node_id, hardware_fingerprint}` | `VerifyResponse` | 验证 License |

`LicenseType` 枚举：`TRIAL` `STANDARD` `PROFESSIONAL` `ENTERPRISE` `ULTIMATE`。

### 关键配置消息体

- **DomainConfig** — 域名完整配置：源站 `OriginConfig`、HTTPS `HTTPSConfig`、缓存 `CacheConfig`、防护 `ProtectionConfig`、自定义头/页、超时、HTTP/2&3、限速、重写规则
- **ProtectionConfig** — 7 层防护：`IPFilter` `CCProtection` `DynamicProtection` `AccessControl` `GeoRestriction` `BotDetection` `SemanticWAF`，防护模式 `OBSERVE/PROTECT/STRICT/DISABLED`
- **OriginConfig** — 源站组、主备、回源协议、超时、重试、保活、负载均衡策略、健康检查

### 生成 Go 代码

```bash
# 需先安装 protoc + protoc-gen-go + protoc-gen-go-grpc
make proto
# 等价于
protoc --go_out=. --go-grpc_out=. --proto_path=proto proto/shieldflow.proto
```

> 当前 `internal/grpc/` 采用 REST 模式实现，未依赖生成代码；如需切换到生成桩代码模式，运行 `make proto` 后将生成代码引入 `internal/grpc/proto/`。

---

## 前端开发指南

### 技术栈

- Vue 3（Composition API）
- Ant Design Vue 4
- Vite 8
- Pinia 3
- Vue Router 4
- Axios
- ECharts 6 + vue-echarts
- dayjs

### 目录结构

```
web/
├── src/
│   ├── api/
│   │   └── index.js          # 全部 API 调用，按模块分组导出
│   ├── assets/               # 静态资源
│   ├── components/           # 公共组件
│   ├── layouts/
│   │   ├── UserLayout.vue    # 用户端布局（侧边栏 + 顶栏）
│   │   └── AdminLayout.vue   # 管理端布局
│   ├── router/
│   │   └── index.js          # 路由定义 + 鉴权守卫
│   ├── store/
│   │   └── user.js           # Pinia 用户状态（token/role/isLogin）
│   ├── utils/
│   │   └── request.js        # Axios 实例（拦截器、错误处理）
│   ├── views/                # 页面
│   │   ├── Login.vue
│   │   ├── Register.vue
│   │   ├── Dashboard.vue
│   │   ├── Domains.vue
│   │   ├── DomainDetail.vue
│   │   ├── Certificates.vue
│   │   ├── Logs.vue
│   │   ├── Traffic.vue
│   │   ├── Cache.vue
│   │   ├── Layer4.vue
│   │   ├── Protection.vue
│   │   ├── Packages.vue
│   │   └── admin/            # 管理端页面
│   │       ├── Users.vue
│   │       ├── Nodes.vue
│   │       ├── Packages.vue
│   │       ├── DDoS.vue
│   │       ├── System.vue
│   │       └── Backup.vue
│   ├── App.vue
│   ├── main.js               # 入口
│   └── style.css
├── vite.config.js
├── package.json
└── index.html
```

### 路由

路由定义在 `src/router/index.js`，使用 `createWebHistory` 模式。两套布局：

**用户端**（`/`，`UserLayout`）：

| 路径 | 组件 | 说明 |
|------|------|------|
| `/dashboard` | Dashboard | 仪表盘 |
| `/domains` | Domains | 域名列表 |
| `/domains/:id` | DomainDetail | 域名配置 |
| `/certificates` | Certificates | SSL 证书 |
| `/logs` | Logs | 日志管理 |
| `/traffic` | Traffic | 流量统计 |
| `/cache` | Cache | 缓存管理 |
| `/layer4` | Layer4 | 四层转发 |
| `/protection` | Protection | 防护管理 |
| `/packages` | Packages | 套餐管理 |

**管理端**（`/admin`，`AdminLayout`，需 admin）：

| 路径 | 组件 | 说明 |
|------|------|------|
| `/admin/users` | admin/Users | 用户管理 |
| `/admin/nodes` | admin/Nodes | 节点管理 |
| `/admin/packages` | admin/Packages | 套餐管理 |
| `/admin/ddos` | admin/DDoS | DDoS 防护 |
| `/admin/system` | admin/System | 系统设置 |
| `/admin/backup` | admin/Backup | 数据备份 |

**路由守卫**：
- `meta.public` 路由（login/register）免登录
- 未登录访问受保护路由 → 跳转 `/login?redirect=...`
- `meta.admin` 路由需 admin 角色，否则提示「无权限访问管理端」

### 状态管理

`src/store/user.js`（Pinia）维护：
- `token` — JWT，持久化到 localStorage
- `userInfo` — 用户信息
- `isLogin` / `isAdmin` — 计算属性

### API 调用

`src/utils/request.js` 创建 Axios 实例：
- baseURL `/api/v1`
- 请求拦截器：自动注入 `Authorization: Bearer <token>`
- 响应拦截器：统一错误处理，401 自动跳转登录

`src/api/index.js` 按模块导出 API 对象：
- `authApi` `dashboardApi` `domainApi` `certApi` `logApi` `trafficApi` `cacheApi` `layer4Api` `protectionApi` `packageApi`
- `adminUserApi` `adminNodeApi` `adminPackageApi` `adminDdosApi` `adminSystemApi` `adminBackupApi`

使用示例：
```js
import { domainApi } from '@/api'

const { data } = await domainApi.list({ page: 1, size: 20 })
await domainApi.create({ domain_name: 'example.com', ... })
```

### 开发与构建

```bash
cd web
npm install         # 安装依赖
npm run dev         # 开发服务器（HMR，默认 5173）
npm run build       # 生产构建 → dist/
npm run preview     # 预览构建产物
```

`vite.config.js` 中配置了代理，开发时 API 请求转发到后端 `http://localhost:8080`。

### 添加新页面

1. 在 `src/views/` 创建 `.vue` 组件
2. 在 `src/router/index.js` 注册路由（注意布局与 meta）
3. 如需调接口，在 `src/api/index.js` 添加 API 方法
4. 管理端页面放 `src/views/admin/` 并加 `meta: { admin: true }`

---

## 代码规范

### Go

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 与 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` / `go vet`，推荐 `golangci-lint`
- 包名小写单数，文件名小写下划线
- 导出标识符需有注释，以标识符名开头
- 错误必须处理，不得忽略 `_ = err`
- 业务表用 GORM 模型 + 软删除；日志用 ClickHouse
- 配置通过 Viper 注入，禁止硬编码
- 中间件按层级顺序注册，保持 7 层防护链顺序

```bash
make fmt     # go fmt
make vet     # go vet
make lint    # golangci-lint（如已安装）
```

### Vue / JS

- 使用 Composition API（`<script setup>`）
- 组件名 PascalCase，文件名与组件名一致
- API 调用统一走 `src/api/`，不在组件内直接写 axios
- 全局状态走 Pinia，不滥用 provide/inject
- 样式优先使用 Ant Design Vue 组件，自定义样式加 `scoped`

### 通用

- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)：
  - `feat:` 新功能
  - `fix:` 修复
  - `docs:` 文档
  - `refactor:` 重构
  - `test:` 测试
  - `chore:` 杂项
- 中文注释与文档 welcome，但标识符与 Git 提交信息使用英文

---

## 测试指南

### 运行测试

```bash
make test          # go test ./... -v
make test-race     # 竞态检测
```

### 测试范围

- `internal/handlers/` — HTTP handler 单元测试（建议用 `httptest`）
- `internal/pkg/waf/` — WAF 规则匹配测试
- `internal/pkg/cache/` — 缓存读写与淘汰测试
- `internal/pkg/dns/` — DNS Provider 测试（mock HTTP）
- `internal/grpc/` — gRPC handler 测试

### 手动验证

```bash
# 健康检查
curl http://localhost:8080/ping
curl http://localhost:8080/api/v1/health

# 登录获取 token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 使用 token 调接口
curl http://localhost:8080/api/v1/domains \
  -H "Authorization: Bearer <token>"
```

### 数据库测试

建议使用独立的测试数据库，避免污染开发数据：

```bash
sudo -u postgres createdb shieldflow_test
sudo -u postgres psql -d shieldflow_test -f sql/001_init_postgresql.sql
```

---

## 贡献指南

### 贡献流程

1. **Fork** 仓库到你的 GitHub 账号
2. **克隆** 你的 fork 到本地
3. **创建分支**：`git checkout -b feature/your-feature`（修复用 `fix/`，文档用 `docs/`）
4. **开发**：遵循代码规范，必要时补充测试
5. **本地验证**：
   ```bash
   make fmt && make vet && make test
   cd web && npm run build && cd ..
   ```
6. **提交**：使用 Conventional Commits 格式
   ```bash
   git commit -m "feat(cache): add stale-while-revalidate support"
   ```
7. **推送**：`git push origin feature/your-feature`
8. **发起 Pull Request** 到 `main` 分支，描述动机、改动、测试情况

### PR 要求

- 一个 PR 只做一件事，保持小而聚焦
- 标题使用 Conventional Commits 格式
- 描述清楚动机（为什么做）、改动（做了什么）、影响（影响哪些模块）
- 新功能需补充测试
- 不破坏现有 API 与数据库兼容（如需迁移，提供 SQL 变更脚本）
- 通过 `make fmt && make vet && make test`

### 分支模型

- `main` — 稳定主线，始终可构建可部署
- `feature/*` — 新功能开发
- `fix/*` — Bug 修复
- `release/*` — 发布准备（可选）

### 发布流程

1. 更新版本号（Makefile `VERSION`、`system_settings`）
2. 打 tag：`git tag v1.x.0 && git push --tags`
3. GitHub Release 附 Changelog
4. 升级脚本 `scripts/upgrade.sh` 支持新版本

### 反馈

- Bug 与功能建议：[提交 Issue](https://github.com/717315051/shieldflow/issues)
- 安全漏洞：请勿公开 Issue，私信维护者
- 讨论：欢迎在 Issue / PR 中讨论设计

---

<div align="center">

感谢你考虑为 ShieldFlow 贡献代码！⭐

</div>
