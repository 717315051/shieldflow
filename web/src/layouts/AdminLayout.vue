<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import SfSidebar from '../components/SfSidebar.vue'
import SfTopbar from '../components/SfTopbar.vue'
import SfIcon from '../components/SfIcon.vue'

const router = useRouter()
const userStore = useUserStore()
const collapsed = ref(false)
const theme = ref(localStorage.getItem('sf-theme') || 'light')
const mobileDrawer = ref(false)

const user = computed(() => ({ name: userStore.userInfo?.username || 'admin', avatar: '' }))

const adminMenus = [
  { key: 'users', label: '用户管理', icon: 'TeamOutlined', path: '/admin/users' },
  { key: 'nodes', label: '节点管理', icon: 'ClusterOutlined', path: '/admin/nodes' },
  { key: 'packages', label: '套餐管理', icon: 'AppstoreOutlined', path: '/admin/packages' },
  { key: 'ddos', label: 'DDoS 防护', icon: 'WarningOutlined', path: '/admin/ddos' },
  { key: 'waf', label: 'WAF 管理', icon: 'SafetyOutlined', path: '/admin/waf' },
  { key: 'ai', label: 'AI 防护', icon: 'RobotOutlined', path: '/admin/ai' },
  { key: 'system', label: '系统设置', icon: 'SettingOutlined', path: '/admin/system' },
  { key: 'backup', label: '数据备份', icon: 'DatabaseOutlined', path: '/admin/backup' },
]

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
    <div :class="['sf-app sf-app--admin', `sf-app--${theme}`]">
      <div class="sf-admin-banner">
        <SfIcon name="AlertFilled" :size="14" />
        <span>管理员控制台 · 所有操作将记录审计日志</span>
      </div>

      <SfSidebar
        :menus="adminMenus"
        :collapsed="collapsed"
        :theme="theme"
        brand="Admin"
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
          :menus="adminMenus"
          :collapsed="false"
          :theme="theme"
          brand="Admin"
          version="1.0.0"
        />
      </a-drawer>

      <div class="sf-app__main">
        <SfTopbar
          :menus="adminMenus"
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
  display: flex; min-height: 100vh; flex-direction: column;
  background: var(--bg-page);
  transition: background var(--dur-med) var(--ease);
}

.sf-admin-banner {
  height: 28px;
  background: var(--warning);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  gap: var(--sp-2);
  padding: 0 var(--sp-4);
  font-size: var(--fs-xs);
  font-weight: 500;
  flex-shrink: 0;
  letter-spacing: 0.3px;
}

.sf-app__main { flex: 1; display: flex; flex-direction: column; min-width: 0; }
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