<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SfIcon from './SfIcon.vue'

const props = defineProps({
  menus: { type: Array, default: () => [] },
  collapsed: { type: Boolean, default: false },
  user: { type: Object, default: () => ({ name: '', avatar: '' }) },
  theme: { type: String, default: 'light' },
})
const emit = defineEmits(['toggle-sidebar', 'toggle-theme', 'logout', 'fullscreen', 'refresh'])
const route = useRoute()
const router = useRouter()
const searchValue = ref('')
const userMenuOpen = ref(false)
const notifOpen = ref(false)

const breadcrumbs = computed(() => {
  const list = [{ label: '首页', path: '/dashboard' }]
  for (const m of (props.menus || [])) {
    if (m.children) {
      const c = m.children.find(c => c.path === route.path)
      if (c) { list.push({ label: m.label, path: '' }); list.push({ label: c.label, path: '' }); return list }
    } else if (m.path === route.path) {
      list.push({ label: m.label, path: '' }); return list
    }
  }
  return list
})

const onSearch = (v) => {
  if (!v) return
}

const toggleFullscreen = () => {
  if (document.fullscreenElement) document.exitFullscreen()
  else document.documentElement.requestFullscreen()
  emit('fullscreen')
}
</script>

<template>
  <header :class="['sf-topbar', `sf-topbar--${theme}`]">
    <!-- 左侧: 折叠 + 面包屑 -->
    <div class="sf-topbar__left">
      <a-tooltip :title="collapsed ? '展开侧栏' : '折叠侧栏'">
        <a-button shape="circle" size="small" @click="emit('toggle-sidebar')">
          <template #icon>
            <SfIcon :name="collapsed ? 'MenuOutlined' : 'CloseOutlined'" :size="14" tone="secondary" />
          </template>
        </a-button>
      </a-tooltip>
      <nav class="sf-topbar__crumbs">
        <template v-for="(c, i) in breadcrumbs" :key="i">
          <span v-if="c.path" class="sf-topbar__crumb sf-topbar__crumb--link" @click="router.push(c.path)">
            {{ c.label }}
          </span>
          <span v-else class="sf-topbar__crumb">{{ c.label }}</span>
          <span v-if="i < breadcrumbs.length - 1" class="sf-topbar__crumb-sep">
            <SfIcon name="RightOutlined" :size="10" tone="tertiary" />
          </span>
        </template>
      </nav>
    </div>

    <!-- 中间: 搜索 -->
    <div class="sf-topbar__center">
      <a-input-search
        v-model:value="searchValue"
        placeholder="搜索菜单、域名、节点…"
        style="max-width: 320px; width: 100%"
        @search="onSearch"
        allow-clear
      >
        <template #prefix>
          <SfIcon name="SearchOutlined" :size="14" tone="tertiary" />
        </template>
      </a-input-search>
    </div>

    <!-- 右侧: 工具栏 -->
    <div class="sf-topbar__right">
      <a-tooltip title="刷新">
        <a-button shape="circle" size="small" @click="emit('refresh')">
          <template #icon><SfIcon name="ReloadOutlined" :size="14" tone="secondary" /></template>
        </a-button>
      </a-tooltip>
      <a-tooltip title="全屏">
        <a-button shape="circle" size="small" @click="toggleFullscreen">
          <template #icon><SfIcon name="ExpandOutlined" :size="14" tone="secondary" /></template>
        </a-button>
      </a-tooltip>
      <a-tooltip :title="theme === 'dark' ? '切换浅色' : '切换深色'">
        <a-button shape="circle" size="small" @click="emit('toggle-theme')">
          <template #icon>
            <SfIcon :name="theme === 'dark' ? 'BulbFilled' : 'BulbOutlined'" :size="14" tone="secondary" />
          </template>
        </a-button>
      </a-tooltip>

      <a-popover v-model:open="notifOpen" trigger="click" placement="bottomRight" :arrow="false">
        <a-badge :count="3" :offset="[-4, 4]">
          <a-button shape="circle" size="small">
            <template #icon><SfIcon name="BellOutlined" :size="14" tone="secondary" /></template>
          </a-button>
        </a-badge>
        <template #content>
          <div class="sf-topbar__notif">
            <div class="sf-topbar__notif-title">通知 (3)</div>
            <div class="sf-topbar__notif-item">
              <div class="t">域名 a.example.com 证书即将过期</div>
              <div class="d">5 分钟前</div>
            </div>
            <div class="sf-topbar__notif-item">
              <div class="t">节点 node-02 出现高负载</div>
              <div class="d">1 小时前</div>
            </div>
            <div class="sf-topbar__notif-item">
              <div class="t">套餐续费成功</div>
              <div class="d">3 小时前</div>
            </div>
          </div>
        </template>
      </a-popover>

      <a-dropdown trigger="click">
        <div class="sf-topbar__user">
          <div class="sf-topbar__avatar">{{ (user.name || 'U').charAt(0).toUpperCase() }}</div>
          <span class="sf-topbar__uname">{{ user.name || '用户' }}</span>
          <SfIcon name="DownOutlined" :size="10" tone="tertiary" />
        </div>
        <template #overlay>
          <a-menu>
            <a-menu-item @click="router.push('/realname')">
              <template #icon><SfIcon name="UserOutlined" :size="14" tone="secondary" /></template>
              个人中心
            </a-menu-item>
            <a-menu-item @click="router.push('/user-package-usage')">
              <template #icon><SfIcon name="DatabaseOutlined" :size="14" tone="secondary" /></template>
              套餐用量
            </a-menu-item>
            <a-menu-divider />
            <a-menu-item @click="$emit('logout')">
              <template #icon><SfIcon name="LogoutOutlined" :size="14" tone="danger" /></template>
              退出登录
            </a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>
  </header>
</template>

<style scoped>
.sf-topbar {
  height: 56px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
  display: flex; align-items: center;
  padding: 0 var(--sp-5);
  gap: var(--sp-4);
  position: sticky; top: 0; z-index: 100;
  transition: background var(--dur-med) var(--ease), border-color var(--dur-med) var(--ease);
}

.sf-topbar__left { display: flex; align-items: center; gap: var(--sp-3); flex-shrink: 0; min-width: 0; }
.sf-topbar__center { flex: 1; display: flex; justify-content: center; min-width: 0; }
.sf-topbar__right { display: flex; align-items: center; gap: var(--sp-2); flex-shrink: 0; }

.sf-topbar__crumbs { display: flex; align-items: center; gap: var(--sp-2); font-size: var(--fs-sm); }
.sf-topbar__crumb { color: var(--text-primary); font-weight: 500; }
.sf-topbar__crumb--link { color: var(--text-secondary); cursor: pointer; transition: color var(--dur-fast) var(--ease); }
.sf-topbar__crumb--link:hover { color: var(--brand-primary); }
.sf-topbar__crumb-sep { display: inline-flex; align-items: center; }

.sf-topbar__user {
  display: flex; align-items: center; gap: var(--sp-2);
  padding: 4px var(--sp-2) 4px 4px;
  border-radius: var(--r-md);
  cursor: pointer;
  color: var(--text-primary);
  transition: background var(--dur-fast) var(--ease);
}
.sf-topbar__user:hover { background: var(--bg-hover); }
.sf-topbar__avatar {
  width: 28px; height: 28px;
  border-radius: 50%;
  background: var(--brand-primary);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-size: var(--fs-xs);
  font-weight: 600;
}
.sf-topbar__uname { font-size: var(--fs-sm); }

.sf-topbar__notif { width: 280px; }
.sf-topbar__notif-title { font-size: var(--fs-sm); font-weight: 600; padding: 4px 0 var(--sp-2); border-bottom: 1px solid var(--border-color); margin-bottom: var(--sp-2); color: var(--text-primary); }
.sf-topbar__notif-item { padding: var(--sp-2); border-radius: var(--r-sm); cursor: pointer; transition: background var(--dur-fast) var(--ease); }
.sf-topbar__notif-item:hover { background: var(--bg-hover); }
.sf-topbar__notif-item .t { font-size: var(--fs-sm); color: var(--text-primary); }
.sf-topbar__notif-item .d { font-size: var(--fs-xs); color: var(--text-tertiary); margin-top: 2px; }

@media (max-width: 1024px) {
  .sf-topbar__center { display: none; }
}
@media (max-width: 768px) {
  .sf-topbar { padding: 0 var(--sp-3); gap: var(--sp-2); }
  .sf-topbar__crumbs { display: none; }
  .sf-topbar__uname { display: none; }
}
</style>