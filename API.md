# ShieldFlow CDN API 文档

## 认证接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/register | 注册 |
| POST | /api/v1/auth/logout | 退出 |
| GET | /api/v1/auth/captcha | 获取验证码 |
| GET | /api/v1/auth/profile | 获取用户信息 |
| PUT | /api/v1/auth/profile | 更新用户信息 |
| PUT | /api/v1/auth/password | 修改密码 |
| POST | /api/v1/auth/forgot-password | 忘记密码 |
| POST | /api/v1/auth/reset-password | 重置密码 |
| POST | /api/v1/auth/realname | 实名认证 |
| POST | /api/v1/auth/verify-code | 发送验证码 |

## 域名管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/domains | 列表 |
| POST | /api/v1/domains | 创建 |
| POST | /api/v1/domains/batch | 批量创建 |
| GET | /api/v1/domains/:id | 详情 |
| PUT | /api/v1/domains/:id | 更新 |
| DELETE | /api/v1/domains/:id | 删除 |
| PUT | /api/v1/domains/:id/status | 更新状态 |
| PUT | /api/v1/domains/:id/basic | 基础设置 |
| PUT | /api/v1/domains/:id/protection | 防护设置 |
| PUT | /api/v1/domains/:id/custom-pages | 自定义页面 |
| PUT | /api/v1/domains/:id/package | 套餐设置 |
| POST | /api/v1/domains/:id/certificate | 申请证书 |

## SSL证书

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/certificates | 列表 |
| POST | /api/v1/certificates/upload | 上传 |
| POST | /api/v1/certificates/apply | 申请 |
| GET | /api/v1/certificates/:id | 详情 |
| DELETE | /api/v1/certificates/:id | 删除 |
| GET | /api/v1/acme-accounts | ACME账户 |
| POST | /api/v1/acme-accounts | 创建ACME |
| GET | /api/v1/dns-accounts | DNS账户 |

## 日志管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/logs/access | 访问日志 |
| GET | /api/v1/logs/attack | 攻击日志 |
| GET | /api/v1/logs/layer4 | 四层日志 |
| GET | /api/v1/logs/ai | AI日志 |
| GET | /api/v1/logs/map | 地图 |
| POST | /api/v1/logs/export | 导出 |

## 流量统计

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/traffic/stats | 统计 |
| GET | /api/v1/traffic/ranking | 排行 |
| GET | /api/v1/traffic/bandwidth | 带宽 |
| GET | /api/v1/traffic/cache-hit | 缓存命中 |

## 缓存管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/cache/file-refresh | 文件刷新 |
| POST | /api/v1/cache/dir-refresh | 目录刷新 |
| POST | /api/v1/cache/file-preheat | 文件预热 |
| GET | /api/v1/cache/tasks | 任务列表 |

## 四层转发

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/layer4 | 列表 |
| POST | /api/v1/layer4 | 创建 |
| PUT | /api/v1/layer4/:id | 更新 |
| DELETE | /api/v1/layer4/:id | 删除 |
| PUT | /api/v1/layer4/:id/status | 状态 |

## 防护管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/protection/templates | 模板 |
| POST | /api/v1/protection/templates | 创建 |
| GET | /api/v1/protection/blacklists | 黑名单 |
| POST | /api/v1/protection/blacklists | 添加 |
| GET | /api/v1/protection/whitelist | 白名单 |
| POST | /api/v1/protection/whitelist | 添加 |

## 套餐管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/packages | 列表 |
| GET | /api/v1/packages/traffic | 流量包 |
| GET | /api/v1/packages/domain | 域名包 |
| POST | /api/v1/packages/:id/purchase | 购买 |
| GET | /api/v1/orders | 订单 |
| GET | /api/v1/balance | 余额 |
| POST | /api/v1/balance/recharge | 充值 |

## 管理端

### 用户管理
| GET | /api/v1/admin/users | 列表 |
| POST | /api/v1/admin/users | 创建 |
| PUT | /api/v1/admin/users/:id | 更新 |
| DELETE | /api/v1/admin/users/:id | 删除 |
| PUT | /api/v1/admin/users/:id/status | 状态 |

### 节点管理
| GET | /api/v1/admin/nodes | 列表 |
| POST | /api/v1/admin/nodes | 创建 |
| PUT | /api/v1/admin/nodes/:id | 更新 |
| DELETE | /api/v1/admin/nodes/:id | 删除 |
| POST | /api/v1/admin/nodes/:id/install | 安装 |
| POST | /api/v1/admin/nodes/:id/upgrade | 升级 |
| GET | /api/v1/admin/node-groups | 分组 |

### DDoS防护
| GET | /api/v1/admin/ddos/dashboard | 仪表盘 |
| GET | /api/v1/admin/ddos/rules | 规则 |
| POST | /api/v1/admin/ddos/rules | 创建 |
| GET | /api/v1/admin/ddos/blacklist | 黑名单 |
| GET | /api/v1/admin/ddos/whitelist | 白名单 |
| GET | /api/v1/admin/ddos/logs | 日志 |

### WAF管理
| GET | /api/v1/admin/waf/dashboard | 仪表盘 |
| GET | /api/v1/admin/waf/config | 配置 |
| PUT | /api/v1/admin/waf/config | 更新 |
| GET | /api/v1/admin/waf/logs | 日志 |
| GET | /api/v1/admin/waf/analysis | 分析 |

### AI管理
| GET | /api/v1/admin/ai/dashboard | 仪表盘 |
| GET | /api/v1/admin/ai/models | 模型 |
| POST | /api/v1/admin/ai/models | 创建 |
| PUT | /api/v1/admin/ai/models/:id | 更新 |
| DELETE | /api/v1/admin/ai/models/:id | 删除 |
| GET | /api/v1/admin/ai/token-stats | Token统计 |
| GET | /api/v1/admin/ai/cost-analysis | 成本分析 |

### 系统设置
| GET | /api/v1/admin/system/settings | 全局 |
| PUT | /api/v1/admin/system/settings | 更新 |
| GET | /api/v1/admin/system/dns | DNS |
| PUT | /api/v1/admin/system/dns | 更新 |
| GET | /api/v1/admin/system/acme | ACME |
| PUT | /api/v1/admin/system/acme | 更新 |
| GET | /api/v1/admin/system/grpc | gRPC |
| PUT | /api/v1/admin/system/grpc | 更新 |
| GET | /api/v1/admin/system/alert | 告警 |
| PUT | /api/v1/admin/system/alert | 更新 |
| GET | /api/v1/admin/system/ai | AI |
| PUT | /api/v1/admin/system/ai | 更新 |
| GET | /api/v1/admin/system/backup | 备份列表 |
| POST | /api/v1/admin/system/backup | 创建备份 |
| POST | /api/v1/admin/system/backup/:id/restore | 恢复 |
| GET | /api/v1/admin/system/version | 版本 |
| POST | /api/v1/admin/system/upgrade | 升级 |
