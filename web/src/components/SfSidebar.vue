<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SfIcon from './SfIcon.vue'

const props = defineProps({
  menus: { type: Array, required: true },
  collapsed: { type: Boolean, default: false },
  brand: { type: String, default: 'SCDN' },
  version: { type: String, default: '' },
  theme: { type: String, default: 'light' },
})
const emit = defineEmits(['toggle'])
const route = useRoute()
const router = useRouter()
const openKeys = ref(new Set())

const isActive = (m) => {
  if (m.path && route.path === m.path) return true
  if (m.children) return m.children.some(c => route.path.startsWith(c.path || ''))
  return false
}
const hasActiveChild = (m) => m.children && m.children.some(c => route.path.startsWith(c.path || ''))

const toggleGroup = (key) => {
  if (openKeys.value.has(key)) openKeys.value.delete(key)
  else openKeys.value.add(key)
  openKeys.value = new Set(openKeys.value)
}

const go = (m) => {
  if (m.path) router.push(m.path)
  else if (m.children) toggleGroup(m.key)
}

const onChildClick = (m, c) => {
  if (c.path) router.push(c.path)
}

const themeClass = computed(() => `sidebar--${props.theme}`)
</script>

<template>
  <aside :class="['sf-sidebar', themeClass, collapsed && 'sf-sidebar--collapsed']">
    <!-- Brand -->
    <div class="sf-sidebar__brand" @click="emit('toggle')">
      <div class="sf-sidebar__logo">
        <SfIcon name="ThunderboltOutlined" :size="20" tone="brand" />
      </div>
      <span v-if="!collapsed" class="sf-sidebar__brand-text">{{ brand }}</span>
    </div>

    <!-- Menu -->
    <nav class="sf-sidebar__nav">
      <template v-for="(m, idx) in menus" :key="m.key">
        <!-- 分隔符 (管理后台分组标题) -->
        <div v-if="m.divider" class="sf-sidebar__divider">
          <span v-if="!collapsed">{{ m.label }}</span>
          <span v-else class="sf-sidebar__divider-dot"></span>
        </div>
        <div
          v-else
          :class="['sf-sidebar__group', { active: isActive(m), open: openKeys.has(m.key) || hasActiveChild(m) }]"
        >
          <div class="sf-sidebar__group-title" @click="go(m)">
            <SfIcon v-if="m.icon" :name="m.icon" :size="16" :tone="isActive(m) ? 'brand' : 'secondary'" class="sf-sidebar__icon" />
            <span v-if="!collapsed" class="sf-sidebar__label">{{ m.label }}</span>
            <SfIcon v-if="!collapsed && m.children" :name="(openKeys.has(m.key) || hasActiveChild(m)) ? 'DownOutlined' : 'RightOutlined'" :size="10" tone="tertiary" class="sf-sidebar__arrow" />
          </div>
          <div v-if="!collapsed && m.children && (openKeys.has(m.key) || hasActiveChild(m))" class="sf-sidebar__children">
            <div
              v-for="c in m.children"
              :key="c.path"
              :class="['sf-sidebar__child', { active: route.path === c.path }]"
              @click="onChildClick(m, c)"
            >
              {{ c.label }}
            </div>
          </div>
        </div>
      </template>
    </nav>

    <!-- Version -->
    <div v-if="!collapsed && version" class="sf-sidebar__version">
      <div class="sf-sidebar__version-line">v{{ version }}</div>
      <div class="sf-sidebar__version-sub">ShieldFlow CDN</div>
    </div>
  </aside>
</template>

<style scoped>
.sf-sidebar {
  width: 220px; height: 100%;
  background: var(--bg-card);
  border-right: 1px solid var(--border-color);
  display: flex; flex-direction: column;
  transition: width var(--dur-med) var(--ease), background var(--dur-med) var(--ease), border-color var(--dur-med) var(--ease);
  flex-shrink: 0;
}
.sf-sidebar--collapsed { width: 56px; }

.sf-sidebar--dark { background: var(--bg-card); }

.sf-sidebar__brand {
  height: 56px;
  display: flex; align-items: center; gap: var(--sp-3);
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  cursor: pointer;
  user-select: none;
}
.sf-sidebar__logo {
  width: 28px; height: 28px;
  background: var(--brand-primary);
  border-radius: var(--r-sm);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.sf-sidebar__logo :deep(svg) { color: #fff !important; }
.sf-sidebar__brand-text {
  font-size: var(--fs-md);
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.5px;
}

.sf-sidebar__nav {
  flex: 1; overflow-y: auto; padding: var(--sp-2) 0;
}

/* 分组分隔符 (用于管理后台分组) */
.sf-sidebar__divider {
  padding: var(--sp-4) var(--sp-4) var(--sp-2);
  font-size: var(--fs-xs);
  color: var(--text-tertiary);
  font-weight: 500;
  letter-spacing: 0.5px;
  display: flex; align-items: center; gap: var(--sp-2);
}
.sf-sidebar__divider::before {
  content: '';
  width: 16px; height: 1px;
  background: var(--border-color);
}
.sf-sidebar__divider-dot {
  width: 4px; height: 4px;
  background: var(--text-tertiary);
  border-radius: 50%;
  margin: 0 auto;
}
.sf-sidebar__nav::-webkit-scrollbar { width: 4px; }
.sf-sidebar__nav::-webkit-scrollbar-thumb { background: var(--border-color); border-radius: 2px; }

.sf-sidebar__group-title {
  display: flex; align-items: center; gap: var(--sp-3);
  padding: 8px var(--sp-4);
  margin: 1px var(--sp-2);
  border-radius: var(--r-md);
  cursor: pointer;
  font-size: var(--fs-sm);
  color: var(--text-secondary);
  transition: all var(--dur-fast) var(--ease);
}
.sf-sidebar__group-title:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.sf-sidebar__icon { flex-shrink: 0; }
.sf-sidebar__label { flex: 1; }
.sf-sidebar__arrow { flex-shrink: 0; }

.sf-sidebar__group.active .sf-sidebar__group-title {
  color: var(--brand-primary);
  background: var(--brand-primary-soft);
  font-weight: 500;
}

.sf-sidebar__children {
  padding: 2px 0;
}
.sf-sidebar__child {
  padding: 6px var(--sp-4) 6px 44px;
  font-size: var(--fs-sm);
  color: var(--text-secondary);
  cursor: pointer;
  position: relative;
  margin: 1px var(--sp-2);
  border-radius: var(--r-md);
  transition: all var(--dur-fast) var(--ease);
}
.sf-sidebar__child:hover { color: var(--brand-primary); background: var(--bg-hover); }
.sf-sidebar__child.active {
  color: var(--brand-primary);
  background: var(--brand-primary-soft);
  font-weight: 500;
}

.sf-sidebar__version {
  padding: var(--sp-3) var(--sp-4);
  border-top: 1px solid var(--border-color);
  background: var(--bg-page);
  flex-shrink: 0;
}
.sf-sidebar__version-line { font-size: var(--fs-xs); font-weight: 600; color: var(--text-secondary); }
.sf-sidebar__version-sub { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }

@media (max-width: 1024px) {
  .sf-sidebar { position: fixed; left: 0; top: 0; bottom: 0; z-index: 1000; box-shadow: 2px 0 8px rgba(0,0,0,0.06); }
  .sf-sidebar--collapsed { transform: translateX(-100%); width: 220px; }
}
</style>