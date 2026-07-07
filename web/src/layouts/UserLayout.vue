<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '../store/user'
import {
  DashboardOutlined,
  GlobalOutlined,
  SafetyCertificateOutlined,
  FileTextOutlined,
  BarChartOutlined,
  CloudOutlined,
  ApartmentOutlined,
  SafetyOutlined,
  CrownOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons-vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const collapsed = ref(false)

const selectedKeys = computed(() => {
  const path = route.path
  if (path.startsWith('/domains')) return ['domains']
  return [path.split('/')[1] || 'dashboard']
})

const openKeys = ref(['sub1'])

const menus = [
  { key: 'dashboard', icon: DashboardOutlined, label: '仪表盘', path: '/dashboard' },
  { key: 'domains', icon: GlobalOutlined, label: '域名管理', path: '/domains' },
  { key: 'certificates', icon: SafetyCertificateOutlined, label: 'SSL证书', path: '/certificates' },
  { key: 'logs', icon: FileTextOutlined, label: '日志管理', path: '/logs' },
  { key: 'traffic', icon: BarChartOutlined, label: '流量统计', path: '/traffic' },
  { key: 'cache', icon: CloudOutlined, label: '缓存管理', path: '/cache' },
  { key: 'layer4', icon: ApartmentOutlined, label: '四层转发', path: '/layer4' },
  { key: 'protection', icon: SafetyOutlined, label: '防护管理', path: '/protection' },
  { key: 'packages', icon: CrownOutlined, label: '套餐管理', path: '/packages' },
]

function go(path) {
  router.push(path)
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <a-layout style="min-height: 100vh">
    <a-layout-sider v-model:collapsed="collapsed" collapsible>
      <div class="logo">
        <span v-if="!collapsed">ShieldFlow 控制台</span>
        <span v-else>ZY</span>
      </div>
      <a-menu
        theme="dark"
        mode="inline"
        :selected-keys="selectedKeys"
        @click="(e) => go(menus.find((m) => m.key === e.key)?.path || '/dashboard')"
      >
        <a-menu-item v-for="m in menus" :key="m.key">
          <component :is="m.icon" />
          <span>{{ m.label }}</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="header">
        <component
          :is="collapsed ? MenuUnfoldOutlined : MenuFoldOutlined"
          class="trigger"
          @click="collapsed = !collapsed"
        />
        <div class="header-right">
          <a-button v-if="userStore.isAdmin" type="link" @click="go('/admin/users')">
            <SettingOutlined /> 管理后台
          </a-button>
          <a-dropdown>
            <a class="ant-dropdown-link" @click.prevent>
              <UserOutlined /> {{ userStore.userInfo?.username || '用户' }}
            </a>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="handleLogout">
                  <LogoutOutlined /> 退出登录
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.logo {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.08);
}
.header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}
.trigger {
  font-size: 18px;
  cursor: pointer;
  padding: 0 12px;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.content {
  margin: 16px;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  min-height: 360px;
  overflow-x: hidden;
}
</style>
