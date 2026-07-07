import request from '../utils/request'

// ============ 认证 ============
export const authApi = {
  login: (data) => request.post('/auth/login', data),
  register: (data) => request.post('/auth/register', data),
  profile: () => request.get('/auth/profile'),
  updateProfile: (data) => request.put('/auth/profile', data),
  captcha: () => `/api/v1/auth/captcha?t=${Date.now()}`,
  forgotPassword: (email) => request.post('/auth/forgot-password', { email }),
  resetPassword: (email, code, newPassword) =>
    request.post('/auth/reset-password', { email, code, new_password: newPassword }),
}

// ============ 仪表盘 ============
export const dashboardApi = {
  overview: () => request.get('/dashboard/overview'),
  traffic: (params) => request.get('/dashboard/traffic', { params }),
  geo: () => request.get('/dashboard/geo'),
  requests: (params) => request.get('/dashboard/requests', { params }),
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

// ============ 套餐（用户端） ============
export const packageApi = {
  market: (params) => request.get('/packages/market', { params }),
  mine: (params) => request.get('/packages/mine', { params }),
  orders: (params) => request.get('/packages/orders', { params }),
  buy: (data) => request.post('/packages/buy', data),
  balance: () => request.get('/packages/balance'),
  recharge: (data) => request.post('/packages/recharge', data),
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

// ============ 管理端 - 系统 ============
export const adminSystemApi = {
  get: () => request.get('/admin/system'),
  update: (data) => request.put('/admin/system', data),
  testAlert: (data) => request.post('/admin/system/test-alert', data),
  backup: () => request.post('/admin/system/backup'),
}

// ============ 管理端 - 备份 ============
export const adminBackupApi = {
  list: (params) => request.get('/admin/backups', { params }),
  create: (data) => request.post('/admin/backups', data),
  restore: (id) => request.post(`/admin/backups/${id}/restore`),
  delete: (id) => request.delete(`/admin/backups/${id}`),
  download: (id) =>
    request.get(`/admin/backups/${id}/download`, { responseType: 'blob' }),
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
