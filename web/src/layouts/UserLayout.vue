<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import SfSidebar from '../components/SfSidebar.vue'
import SfTopbar from '../components/SfTopbar.vue'
import SfIcon from '../components/SfIcon.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const collapsed = ref(false)
const theme = ref(localStorage.getItem('sf-theme') || 'light')
const mobileDrawer = ref(false)

// 合并所有菜单,根据权限过滤
const allMenus = computed(() => {
  const user = [
    { key: 'dash', label: '仪表盘', icon: 'DashboardOutlined', path: '/dashboard' },
    { key: 'domain', label: '域名管理', icon: 'GlobalOutlined',
      children: [
        { label: '域名列表', path: '/domains' },
      ]
    },
    { key: 'cert', label: 'SSL 证书', icon: 'SafetyCertificateOutlined', path: '/certificates' },
    { key: 'cache', label: '缓存管理', icon: 'ThunderboltOutlined', path: '/cache' },
    { key: 'layer4', label: '四层转发', icon: 'SwapOutlined', path: '/layer4' },
    { key: 'protection', label: '防护管理', icon: 'SafetyOutlined', path: '/protection' },
    { key: 'packages', label: '套餐管理', icon: 'AppstoreOutlined',
      children: [
        { label: '套餐列表', path: '/packages' },
        { label: '套餐用量', path: '/user-package-usage' },
      ]
    },
    { key: 'logs', label: '日志管理', icon: 'FileTextOutlined', path: '/logs' },
    { key: 'traffic', label: '流量统计', icon: 'LineChartOutlined', path: '/traffic' },
    { key: 'realname', label: '实名认证', icon: 'UserOutlined', path: '/realname' },
  ]
  if (userStore.isAdmin) {
    user.push(
      { key: '__admin_divider', label: '管理后台', divider: true },
      { key: 'admin-users', label: '用户管理', icon: 'TeamOutlined', path: '/admin/users', adminOnly: true },
      { key: 'admin-nodes', label: '节点管理', icon: 'ClusterOutlined', path: '/admin/nodes', adminOnly: true },
      { key: 'admin-packages', label: '套餐模板', icon: 'AppstoreOutlined', path: '/admin/packages', adminOnly: true },
      { key: 'admin-ddos', label: 'DDoS 防护', icon: 'WarningOutlined', path: '/admin/ddos', adminOnly: true },
      { key: 'admin-waf', label: 'WAF 管理', icon: 'SafetyOutlined', path: '/admin/waf', adminOnly: true },
      { key: 'admin-ai', label: 'AI 防护', icon: 'RobotOutlined', path: '/admin/ai', adminOnly: true },
      { key: 'admin-system', label: '系统设置', icon: 'SettingOutlined', path: '/admin/system', adminOnly: true },
      { key: 'admin-backup', label: '数据备份', icon: 'DatabaseOutlined', path: '/admin/backup', adminOnly: true },
    )
  }
  return user
})

const user = computed(() => ({ name: userStore.userInfo?.username || 'user', avatar: '' }))
const isAdminRoute = computed(() => route.path.startsWith('/admin/'))

const antdTheme = computed(() => ({
  token: {
    colorPrimary: '#165dff',
    borderRadius: 6,
    fontFamily: '-apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif',
  }
}))

const updateLayout = () => {
  if (window.innerWidth <= 1024) collapsed.value = true
  else collapsed.value = false
}

const toggleSidebar = () => {
  if (window.innerWidth <= 1024) mobileDrawer.value = !mobileDrawer.value
  else collapsed.value = !collapsed.value
}

const toggleTheme = () => {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
  localStorage.setItem('sf-theme', theme.value)
  document.documentElement.dataset.theme = theme.value
}

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

const onFullscreen = () => {}
const onRefresh = () => window.location.reload()

onMounted(() => {
  updateLayout()
  window.addEventListener('resize', updateLayout)
  document.documentElement.dataset.theme = theme.value
})
</script>

<template>
  <a-config-provider :theme="antdTheme">
    <div :class="['sf-app', `sf-app--${theme}`, isAdminRoute && 'sf-app--admin-active']">
      <!-- Admin 区顶部红色 banner -->
      <div v-if="isAdminRoute" class="sf-admin-banner">
        <SfIcon name="AlertFilled" :size="14" />
        <span>管理员控制台 · 所有操作将记录审计日志</span>
      </div>

      <SfSidebar
        :menus="allMenus"
        :collapsed="collapsed"
        :theme="theme"
        brand="SCDN"
        version="1.0.0"
        @toggle="toggleSidebar"
      />

      <a-drawer
        v-model:open="mobileDrawer"
        placement="left"
        :width="220"
        :body-style="{ padding: 0 }"
        :header-style="{ display: 'none' }"
      >
        <SfSidebar
          :menus="allMenus"
          :collapsed="false"
          :theme="theme"
          brand="SCDN"
          version="1.0.0"
        />
      </a-drawer>

      <div class="sf-app__main">
        <SfTopbar
          :menus="allMenus"
          :collapsed="collapsed"
          :user="user"
          :theme="theme"
          @toggle-sidebar="toggleSidebar"
          @toggle-theme="toggleTheme"
          @logout="handleLogout"
          @fullscreen="onFullscreen"
          @refresh="onRefresh"
        />
        <main class="sf-app__content">
          <router-view v-slot="{ Component }">
            <transition name="sf-fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </main>
      </div>
    </div>
  </a-config-provider>
</template>

<style scoped>
.sf-app {
  display: flex; min-height: 100vh;
  background: var(--bg-page);
  transition: background var(--dur-med) var(--ease);
  position: relative;
}

.sf-admin-banner {
  position: fixed; top: 0; left: 0; right: 0;
  height: 28px;
  background: var(--warning);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  gap: var(--sp-2);
  padding: 0 var(--sp-4);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.3px;
  z-index: 1001;
}
.sf-app--admin-active .sf-app__main { padding-top: 28px; }
.sf-app--admin-active :deep(.sf-topbar) { top: 28px; }

.sf-app__main { flex: 1; display: flex; flex-direction: column; min-width: 0; min-height: 100vh; }
.sf-app__content { flex: 1; padding: var(--sp-5); overflow-y: auto; }

.sf-fade-enter-active, .sf-fade-leave-active { transition: opacity var(--dur-med) var(--ease); }
.sf-fade-enter-from, .sf-fade-leave-to { opacity: 0; }

@media (max-width: 1024px) {
  :deep(.sf-sidebar:not(.sf-sidebar--collapsed)) { display: none; }
}
@media (max-width: 768px) {
  .sf-app__content { padding: var(--sp-3); }
  .sf-admin-banner { font-size: 11px; padding: 0 var(--sp-3); }
}
</style>