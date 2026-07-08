import request from '../utils/request'

// ============ 认证 ============
export const authApi = {
  login: (data) => request.post('/auth/login', data),
  register: (data) => request.post('/auth/register', data),
  logout: () => request.post('/auth/logout'),
  profile: () => request.get('/auth/profile'),
  updateProfile: (data) => request.put('/auth/profile', data),
  changePassword: (data) => request.put('/auth/password', data),
  captcha: () => `/api/v1/auth/captcha?t=${Date.now()}`,
  sendEmailCode: (email) => request.post('/auth/send-email-code', { email }),
  verifyCode: (data) => request.post('/auth/verify-code', data),
  forgotPassword: (email) => request.post('/auth/forgot-password', { email }),
  resetPassword: (email, code, newPassword) =>
    request.post('/auth/reset-password', { email, code, new_password: newPassword }),
  realname: (data) => request.post('/auth/realname', data),
}

// ============ 仪表盘 ============
// 后端只有一个聚合端点: GET /api/v1/dashboard/analysis
// 前端从同一份响应里切出 overview / traffic / geo 三块
export const dashboardApi = {
  overview: () => request.get('/dashboard/analysis'),
  traffic: (params) => request.get('/dashboard/analysis', { params }),
  geo: () => request.get('/dashboard/analysis'),
}

// ============ 域名管理 ============
export const domainApi = {
  list: (params) => request.get('/domains', { params }),
  create: (data) => request.post('/domains', data),
  batchCreate: (data) => request.post('/domains/batch', data),
  detail: (id) => request.get(`/domains/${id}`),
  update: (id, data) => request.put(`/domains/${id}`, data),
  delete: (id) => request.delete(`/domains/${id}`),
  updateConfig: (id, data) => request.put(`/domains/${id}/config`, data),
  updateProtection: (id, data) => request.put(`/domains/${id}/protection`, data),
  customPages: (id) => request.get(`/domains/${id}/pages`),
  updateCustomPages: (id, data) => request.put(`/domains/${id}/pages`, data),
}

// ============ SSL 证书 ============
export const certApi = {
  list: (params) => request.get('/certificates', { params }),
  upload: (data) => request.post('/certificates/upload', data),
  apply: (data) => request.post('/certificates/apply', data),
  delete: (id) => request.delete(`/certificates/${id}`),
  acmeList: (params) => request.get('/acme-accounts', { params }),
  acmeCreate: (data) => request.post('/acme-accounts', data),
  acmeDelete: (id) => request.delete(`/acme-accounts/${id}`),
  dnsList: (params) => request.get('/dns-accounts', { params }),
  dnsCreate: (data) => request.post('/dns-accounts', data),
  dnsDelete: (id) => request.delete(`/dns-accounts/${id}`),
}

// ============ 日志 ============
export const logApi = {
  access: (params) => request.get('/logs/access', { params }),
  attack: (params) => request.get('/logs/attack', { params }),
  layer4: (params) => request.get('/logs/layer4', { params }),
  ai: (params) => request.get('/logs/ai', { params }),
  export: (params) =>
    request.get('/logs/export', { params, responseType: 'blob' }),
  map: (params) => request.get('/logs/map', { params }),
}

// ============ 流量统计 ============
export const trafficApi = {
  stats: (params) => request.get('/traffic/stats', { params }),
  ranking: (params) => request.get('/traffic/ranking', { params }),
  bandwidth: (params) => request.get('/traffic/bandwidth', { params }),
  cacheHit: (params) => request.get('/traffic/cache-hit', { params }),
}

// ============ 缓存管理 ============
export const cacheApi = {
  refreshFile: (data) => request.post('/cache/refresh-file', data),
  refreshDir: (data) => request.post('/cache/refresh-dir', data),
  prefetch: (data) => request.post('/cache/prefetch', data),
  tasks: (params) => request.get('/cache/tasks', { params }),
  deleteTask: (id) => request.delete(`/cache/tasks/${id}`),
}

// ============ 四层转发 ============
export const layer4Api = {
  list: (params) => request.get('/layer4', { params }),
  create: (data) => request.post('/layer4', data),
  update: (id, data) => request.put(`/layer4/${id}`, data),
  delete: (id) => request.delete(`/layer4/${id}`),
}

// ============ 防护管理 ============
export const protectionApi = {
  templates: (params) => request.get('/protection/templates', { params }),
  createTemplate: (data) => request.post('/protection/templates', data),
  updateTemplate: (id, data) =>
    request.put(`/protection/templates/${id}`, data),
  deleteTemplate: (id) => request.delete(`/protection/templates/${id}`),
  whitelist: (params) => request.get('/protection/whitelist', { params }),
  addWhitelist: (data) => request.post('/protection/whitelist', data),
  delWhitelist: (id) => request.delete(`/protection/whitelist/${id}`),
  blacklist: (params) => request.get('/protection/blacklist', { params }),
  addBlacklist: (data) => request.post('/protection/blacklist', data),
  delBlacklist: (id) => request.delete(`/protection/blacklist/${id}`),
}

// ============ 套餐（用户端，按 API.md 文档）============
// 文档路径：
//   GET    /api/v1/packages            套餐列表（l7/l4）
//   GET    /api/v1/packages/traffic    流量包列表
//   GET    /api/v1/packages/domain     域名包列表
//   GET    /api/v1/packages/market     套餐市场（聚合 3 类 + 余额）
//   GET    /api/v1/user-packages       我的套餐
//   GET    /api/v1/user-packages/:id   套餐详情
//   POST   /api/v1/user-packages/:id/renew  续费
//   GET    /api/v1/orders              订单列表
//   GET    /api/v1/orders/:id          订单详情
//   GET    /api/v1/balance             余额
//   POST   /api/v1/balance/recharge    充值
export const packageApi = {
  // 套餐市场
  list: (params) => request.get('/packages', { params }),
  traffic: (params) => request.get('/packages/traffic', { params }),
  domain: (params) => request.get('/packages/domain', { params }),
  market: (params) => request.get('/packages/market', { params }),
  // 购买
  purchase: (id, data) => request.post(`/packages/${id}/purchase`, data || {}),
  purchaseTraffic: (id, data) => request.post(`/packages/traffic/${id}/purchase`, data || {}),
  purchaseDomain: (id, data) => request.post(`/packages/domain/${id}/purchase`, data || {}),
  // 我的套餐
  myPackages: (params) => request.get('/user-packages', { params }),
  packageDetail: (id) => request.get(`/user-packages/${id}`),
  renew: (id, data) => request.post(`/user-packages/${id}/renew`, data || {}),
  // 订单
  orders: (params) => request.get('/orders', { params }),
  orderDetail: (id) => request.get(`/orders/${id}`),
  // 余额
  balance: () => request.get('/balance'),
  recharge: (data) => request.post('/balance/recharge', data),
}

// ============ 管理端 - 用户 ============
export const adminUserApi = {
  list: (params) => request.get('/admin/users', { params }),
  create: (data) => request.post('/admin/users', data),
  update: (id, data) => request.put(`/admin/users/${id}`, data),
  delete: (id) => request.delete(`/admin/users/${id}`),
}

// ============ 管理端 - 节点 ============
export const adminNodeApi = {
  list: (params) => request.get('/admin/nodes', { params }),
  create: (data) => request.post('/admin/nodes', data),
  update: (id, data) => request.put(`/admin/nodes/${id}`, data),
  delete: (id) => request.delete(`/admin/nodes/${id}`),
  install: (id) => request.post(`/admin/nodes/${id}/install`),
  upgrade: (id, data) => request.post(`/admin/nodes/${id}/upgrade`, data),
  groups: (params) => request.get('/admin/node-groups', { params }),
  createGroup: (data) => request.post('/admin/node-groups', data),
  deleteGroup: (id) => request.delete(`/admin/node-groups/${id}`),
}

// ============ 管理端 - 套餐 ============
export const adminPackageApi = {
  list: (params) => request.get('/admin/packages', { params }),
  create: (data) => request.post('/admin/packages', data),
  update: (id, data) => request.put(`/admin/packages/${id}`, data),
  delete: (id) => request.delete(`/admin/packages/${id}`),
}

// ============ 管理端 - DDoS ============
export const adminDdosApi = {
  dashboard: () => request.get('/admin/ddos/dashboard'),
  rules: (params) => request.get('/admin/ddos/rules', { params }),
  createRule: (data) => request.post('/admin/ddos/rules', data),
  updateRule: (id, data) => request.put(`/admin/ddos/rules/${id}`, data),
  deleteRule: (id) => request.delete(`/admin/ddos/rules/${id}`),
  whitelist: (params) => request.get('/admin/ddos/whitelist', { params }),
  addWhitelist: (data) => request.post('/admin/ddos/whitelist', data),
  delWhitelist: (id) => request.delete(`/admin/ddos/whitelist/${id}`),
  blacklist: (params) => request.get('/admin/ddos/blacklist', { params }),
  addBlacklist: (data) => request.post('/admin/ddos/blacklist', data),
  delBlacklist: (id) => request.delete(`/admin/ddos/blacklist/${id}`),
  logs: (params) => request.get('/admin/ddos/logs', { params }),
}

// ============ 管理端 - 系统设置 ============
// 后端路由：
//   GET    /api/v1/admin/system/settings
//   PUT    /api/v1/admin/system/settings
//   GET    /api/v1/admin/system/dns
//   PUT    /api/v1/admin/system/dns
//   GET    /api/v1/admin/system/acme
//   PUT    /api/v1/admin/system/acme
//   GET    /api/v1/admin/system/grpc
//   PUT    /api/v1/admin/system/grpc
//   POST   /api/v1/admin/system/grpc/test-log-server
//   GET    /api/v1/admin/system/alert
//   PUT    /api/v1/admin/system/alert
//   GET    /api/v1/admin/system/ai
//   PUT    /api/v1/admin/system/ai
export const adminSystemApi = {
  // 通用设置
  getSettings: () => request.get('/admin/system/settings'),
  updateSettings: (data) => request.put('/admin/system/settings', data),
  // DNS
  getDNS: () => request.get('/admin/system/dns'),
  updateDNS: (data) => request.put('/admin/system/dns', data),
  // ACME
  getACME: () => request.get('/admin/system/acme'),
  updateACME: (data) => request.put('/admin/system/acme', data),
  // GRPC + 测试
  getGRPC: () => request.get('/admin/system/grpc'),
  updateGRPC: (data) => request.put('/admin/system/grpc', data),
  testLogServer: (data) => request.post('/admin/system/grpc/test-log-server', data),
  // 告警
  getAlert: () => request.get('/admin/system/alert'),
  updateAlert: (data) => request.put('/admin/system/alert', data),
  // AI
  getAI: () => request.get('/admin/system/ai'),
  updateAI: (data) => request.put('/admin/system/ai', data),
  // 版本/升级
  getVersion: () => request.get('/admin/system/version'),
  upgrade: (data) => request.post('/admin/system/upgrade', data),
}

// ============ 管理端 - 备份 ============
// 后端路由：
//   GET    /api/v1/admin/system/backup
//   POST   /api/v1/admin/system/backup
//   POST   /api/v1/admin/system/backup/:id/restore
//   GET    /api/v1/admin/system/backup/:id/download  (待后端补)
export const adminBackupApi = {
  list: (params) => request.get('/admin/system/backup', { params }),
  create: () => request.post('/admin/system/backup'),
  restore: (id) => request.post(`/admin/system/backup/${id}/restore`),
  delete: (id) => request.delete(`/admin/system/backup/${id}`),
  downloadUrl: (id) => `/api/v1/admin/system/backup/${id}/download`,
}

// ============ 管理端 - WAF ============
export const wafApi = {
  dashboard: () => request.get('/admin/waf/dashboard'),
  config: () => request.get('/admin/waf/config'),
  updateConfig: (data) => request.put('/admin/waf/config', data),
  logs: (params) => request.get('/admin/waf/logs', { params }),
  analysis: (params) => request.get('/admin/waf/analysis', { params }),
}

// ============ 管理端 - AI ============
export const aiApi = {
  dashboard: () => request.get('/admin/ai/dashboard'),
  tokenStats: (params) => request.get('/admin/ai/token-stats', { params }),
  costAnalysis: (params) => request.get('/admin/ai/cost-analysis', { params }),
  models: (params) => request.get('/admin/ai/models', { params }),
  createModel: (data) => request.post('/admin/ai/models', data),
  updateModel: (id, data) => request.put(`/admin/ai/models/${id}`, data),
  deleteModel: (id) => request.delete(`/admin/ai/models/${id}`),
}

export default {
  authApi,
  dashboardApi,
  domainApi,
  certApi,
  logApi,
  trafficApi,
  cacheApi,
  layer4Api,
  protectionApi,
  packageApi,
  adminUserApi,
  adminNodeApi,
  adminPackageApi,
  adminDdosApi,
  adminSystemApi,
  adminBackupApi,
  wafApi,
  aiApi,
}
