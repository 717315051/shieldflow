<script setup>
defineProps({
  title: { type: String, default: '' },
  sub: { type: String, default: '' },
  showRefresh: { type: Boolean, default: false },
})
const emit = defineEmits(['refresh'])
</script>

<template>
  <div class="sf-page-header">
    <div class="sf-page-header__left">
      <h2 class="sf-page-header__title">{{ title }}</h2>
      <span v-if="sub" class="sf-page-header__sub">{{ sub }}</span>
    </div>
    <div class="sf-page-header__right">
      <slot name="extra" />
      <a-tooltip v-if="showRefresh" title="刷新">
        <a-button shape="circle" size="small" @click="emit('refresh')">
          <template #icon><span style="font-size:14px;line-height:1">↻</span></template>
        </a-button>
      </a-tooltip>
      <a-dropdown trigger="click">
        <a-tooltip title="更多">
          <a-button shape="circle" size="small">
            <template #icon><span style="font-size:14px;line-height:1">⋯</span></template>
          </a-button>
        </a-tooltip>
        <template #overlay>
          <a-menu>
            <a-menu-item key="fullscreen">全屏</a-menu-item>
            <a-menu-item key="export">导出</a-menu-item>
            <a-menu-item key="help">帮助</a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>
  </div>
</template>

<style scoped>
.sf-page-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: var(--sp-5);
  gap: var(--sp-4); flex-wrap: wrap;
}
.sf-page-header__left { display: flex; align-items: baseline; gap: var(--sp-3); flex-wrap: wrap; }
.sf-page-header__title {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--text-primary);
  position: relative; padding-left: 10px;
}
.sf-page-header__title::before {
  content: ''; position: absolute; left: 0; top: 50%;
  transform: translateY(-50%);
  width: 3px; height: 16px;
  background: var(--brand-primary);
  border-radius: 2px;
}
.sf-page-header__sub {
  font-size: var(--fs-xs);
  color: var(--text-tertiary);
}
.sf-page-header__right { display: flex; align-items: center; gap: var(--sp-2); }
@media (max-width: 768px) {
  .sf-page-header { margin-bottom: var(--sp-3); }
  .sf-page-header__title { font-size: var(--fs-md); }
}
</style>