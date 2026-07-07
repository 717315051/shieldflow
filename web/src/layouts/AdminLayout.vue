<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '../store/user'
import {
  UserOutlined,
  ClusterOutlined,
  CrownOutlined,
  ThunderboltOutlined,
  SafetyOutlined,
  RobotOutlined,
  SettingOutlined,
  DatabaseOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons-vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const collapsed = ref(false)

const selectedKeys = computed(() => [route.path.split('/')[2] || 'users'])

const menus = [
  { key: 'users', icon: UserOutlined, label: '用户管理', path: '/admin/users' },
  { key: 'nodes', icon: ClusterOutlined, label: '节点管理', path: '/admin/nodes' },
  { key: 'packages', icon: CrownOutlined, label: '套餐管理', path: '/admin/packages' },
  { key: 'ddos', icon: ThunderboltOutlined, label: 'DDoS防护', path: '/admin/ddos' },
  { key: 'waf', icon: SafetyOutlined, label: 'WAF管理', path: '/admin/waf' },
  { key: 'ai', icon: RobotOutlined, label: 'AI防护', path: '/admin/ai' },
  { key: 'system', icon: SettingOutlined, label: '系统设置', path: '/admin/system' },
  { key: 'backup', icon: DatabaseOutlined, label: '数据备份', path: '/admin/backup' },
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
    <a-layout-sider v-model:collapsed="collapsed" collapsible theme="dark">
      <div class="logo">
        <span v-if="!collapsed">ShieldFlow 管理端</span>
        <span v-else>AD</span>
      </div>
      <a-menu
        theme="dark"
        mode="inline"
        :selected-keys="selectedKeys"
        @click="(e) => go(menus.find((m) => m.key === e.key)?.path || '/admin/users')"
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
          <a-button type="link" @click="go('/dashboard')">返回用户端</a-button>
          <a-dropdown>
            <a class="ant-dropdown-link" @click.prevent>
              <UserOutlined /> {{ userStore.userInfo?.username || '管理员' }}
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
