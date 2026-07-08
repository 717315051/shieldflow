<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import SfSidebar from '../components/SfSidebar.vue'
import SfTopbar from '../components/SfTopbar.vue'

const router = useRouter()
const userStore = useUserStore()
const collapsed = ref(false)
const theme = ref(localStorage.getItem('sf-theme') || 'light')
const mobileDrawer = ref(false)

const user = computed(() => ({ name: userStore.userInfo?.username || 'user', avatar: '' }))

// 用线性 icon 替换 emoji
const userMenus = [
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

const antdTheme = computed(() => ({
  token: {
    colorPrimary: '#165dff',
    borderRadius: 6,
    fontFamily: '-apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif',
  }
}))

// 响应式: 1024px 以下侧栏收起
const updateLayout = () => {
  if (window.innerWidth <= 1024) {
    collapsed.value = true
  } else {
    collapsed.value = false
  }
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
    <div :class="['sf-app', `sf-app--${theme}`]">
      <SfSidebar
        :menus="userMenus"
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
          :menus="userMenus"
          :collapsed="false"
          :theme="theme"
          brand="SCDN"
          version="1.0.0"
        />
      </a-drawer>

      <div class="sf-app__main">
        <SfTopbar
          :menus="userMenus"
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
}
</style>