# 更新日志

所有 notable 的变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [1.0.0] - 2026-07-07

### ✨ 新增

- **后端 API Server** (Go + Gin)
  - 用户认证（JWT + 图片验证码 + 邮箱验证码 + 实名认证）
  - 域名管理（单/批量添加、配置、CNAME、状态管理）
  - 节点管理（注册、安装、升级、SSH远程安装、分组）
  - 套餐管理（七层/四层套餐、流量包、域名包、余额、订单）
  - SSL证书（上传、ACME自动申请、Let's Encrypt/ZeroSSL）
  - 日志管理（访问/攻击/四层/AI日志、导出CSV/JSON、地理地图）
  - 流量统计（总流量/请求数/带宽/缓存命中率、Top排行）
  - 缓存管理（文件刷新、目录刷新、文件预热）
  - 四层转发（TCP/UDP、负载均衡）
  - 防护管理（策略模板、黑白名单、CSV导入导出）
  - DDoS防护（eBPF四层规则、自动封禁、黑白名单、日志）
  - 系统设置（全局/DNS/ACME/gRPC/告警/监控/AI/备份/版本）
  - 仪表盘（总流量/请求数/拦截数/带宽图/地理分布）

- **gRPC Server**
  - 配置下发（域名/DDoS/全局配置推送）
  - 日志收集（访问/攻击/DDoS/四层/AI日志流式上报）
  - 节点管理（注册、状态同步、心跳）
  - 缓存管理（刷新、预热、统计）
  - 授权验证（License 校验）

- **Edge 边缘节点**
  - 反向代理（多源站负载均衡、健康检查、故障转移）
  - 7层防护链（黑白名单 → CC → 访问控制 → 区域限制 → Bot检测 → 语义WAF → 转发）
  - CC防护（8种质询类型：JS/无感/滑块/验证码/重定向/等待室等）
  - WAF语义引擎（SQL注入/XSS/路径穿越/命令注入/LDAP注入检测）
  - 缓存系统（内存+磁盘二级缓存、热点缓存、智能过期）
  - DDoS防护（eBPF四层 + 七层CC，自动封禁，黑白名单）
  - Bot检测（搜索引擎识别、爬虫拦截、空UA拦截）
  - 动态压缩（Gzip/Brotli自动协商）
  - HTTP/2 & HTTP/3 支持

- **DNS 同步组件**
  - Cloudflare DNS API v4 集成
  - 阿里云 DNS API 集成（HMAC-SHA1签名）
  - 腾讯云 DNSPod API 集成（TC3-HMAC-SHA256 v3签名）
  - 定时自动同步 CNAME 记录
  - DNS 状态检查
  - 批量管理 + 错误重试

- **独立日志服务器** (v1.2.0+)
  - HTTP REST 接口接收日志（兼容 gRPC 语义）
  - ClickHouse 批量写入（channel缓冲 + 批量flush + 重试）
  - 6种日志类型（访问/攻击/DDoS/四层/四层拦截/AI）
  - 查询 API（分页/筛选/导出/统计/排行/地理聚合）
  - 健康检查和监控指标

- **前端 Vue3 + Ant Design Vue**
  - 23 个完整页面（用户端 15 + 管理端 8）
  - 登录/注册/仪表盘/域名/证书/日志/流量/缓存/四层/防护/套餐
  - 管理端：用户/节点/套餐/DDoS/系统设置/备份
  - ECharts 数据可视化
  - 响应式布局
  - 路由守卫 + 权限控制

- **数据库**
  - PostgreSQL 24 张表（完整关系模型、JSONB配置、GIN索引、触发器）
  - ClickHouse 6 张表（MergeTree分区、TTL过期、IPv4类型）

- **gRPC Proto 定义**
  - 5 个服务（Config/Log/Node/Cache/Auth）
  - 完整消息体（DomainConfig/DDoSConfig/CCProtection 等）

- **部署工具**
  - 一键安装脚本（install.sh）
  - 边缘节点安装脚本（install-edge.sh）
  - 升级/备份/恢复/卸载脚本
  - Docker + docker-compose 支持
  - Supervisor 配置（4个进程组）
  - Nginx 反向代理配置
  - Makefile 构建系统
  - 5 个 YAML 配置模板

### 🛡️ 安全特性

- JWT 认证 + 角色权限（user/admin）
- 图片验证码 + 邮箱验证码
- 实名认证（真实姓名 + 身份证号）
- API 限流中间件
- CORS 跨域控制
- 操作审计日志
- DDoS 自动封禁
- WAF 托管规则集
- ACME 自动证书管理

### 📊 性能特性

- 二级缓存（内存 + 磁盘）
- ClickHouse 批量写入（1000条/批，5秒flush）
- Gzip/Brotli 动态压缩
- HTTP/2 & HTTP/3
- 连接池管理
- Redis 热点缓存

### 🏗️ 架构特性

- 主控 / 边缘节点 分离部署
- 日志服务器可独立部署
- DNS 同步可独立部署
- 水平扩展支持
- 多区域节点支持
- 配置 gRPC 下发 + 实时生效
