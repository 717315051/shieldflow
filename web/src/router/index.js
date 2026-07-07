import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../store/user'
import { message } from 'ant-design-vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/Register.vue'),
    meta: { public: true },
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('../views/ForgotPassword.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../layouts/UserLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'domains',
        name: 'Domains',
        component: () => import('../views/Domains.vue'),
        meta: { title: '域名管理' },
      },
      {
        path: 'domains/:id',
        name: 'DomainDetail',
        component: () => import('../views/DomainDetail.vue'),
        meta: { title: '域名配置' },
      },
      {
        path: 'certificates',
        name: 'Certificates',
        component: () => import('../views/Certificates.vue'),
        meta: { title: 'SSL证书' },
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('../views/Logs.vue'),
        meta: { title: '日志管理' },
      },
      {
        path: 'traffic',
        name: 'Traffic',
        component: () => import('../views/Traffic.vue'),
        meta: { title: '流量统计' },
      },
      {
        path: 'cache',
        name: 'Cache',
        component: () => import('../views/Cache.vue'),
        meta: { title: '缓存管理' },
      },
      {
        path: 'layer4',
        name: 'Layer4',
        component: () => import('../views/Layer4.vue'),
        meta: { title: '四层转发' },
      },
      {
        path: 'protection',
        name: 'Protection',
        component: () => import('../views/Protection.vue'),
        meta: { title: '防护管理' },
      },
      {
        path: 'packages',
        name: 'Packages',
        component: () => import('../views/Packages.vue'),
        meta: { title: '套餐管理' },
      },
    ],
  },
  {
    path: '/admin',
    component: () => import('../layouts/AdminLayout.vue'),
    redirect: '/admin/users',
    meta: { admin: true },
    children: [
      {
        path: 'users',
        name: 'AdminUsers',
        component: () => import('../views/admin/Users.vue'),
        meta: { title: '用户管理', admin: true },
      },
      {
        path: 'nodes',
        name: 'AdminNodes',
        component: () => import('../views/admin/Nodes.vue'),
        meta: { title: '节点管理', admin: true },
      },
      {
        path: 'packages',
        name: 'AdminPackages',
        component: () => import('../views/admin/Packages.vue'),
        meta: { title: '套餐管理', admin: true },
      },
      {
        path: 'ddos',
        name: 'AdminDDoS',
        component: () => import('../views/admin/DDoS.vue'),
        meta: { title: 'DDoS防护', admin: true },
      },
      {
        path: 'waf',
        name: 'AdminWAF',
        component: () => import('../views/admin/WAF.vue'),
        meta: { title: 'WAF管理', admin: true },
      },
      {
        path: 'ai',
        name: 'AdminAI',
        component: () => import('../views/admin/AI.vue'),
        meta: { title: 'AI防护', admin: true },
      },
      {
        path: 'system',
        name: 'AdminSystem',
        component: () => import('../views/admin/System.vue'),
        meta: { title: '系统设置', admin: true },
      },
      {
        path: 'backup',
        name: 'AdminBackup',
        component: () => import('../views/admin/Backup.vue'),
        meta: { title: '数据备份', admin: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()
  if (to.meta.public) {
    return next()
  }
  if (!userStore.isLogin) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }
  if (to.meta.admin && !userStore.isAdmin) {
    message.error('无权限访问管理端')
    return next({ path: '/dashboard' })
  }
  next()
})

export default router
