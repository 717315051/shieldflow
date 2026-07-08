<script setup>
import SfIcon from './SfIcon.vue'

defineProps({
  title: { type: String, default: '' },
  sub: { type: String, default: '' },
  showSearch: { type: Boolean, default: true },
  showRefresh: { type: Boolean, default: true },
  showAdd: { type: Boolean, default: false },
  addText: { type: String, default: '新建' },
})
const emit = defineEmits(['refresh', 'search', 'add'])
</script>

<template>
  <div class="sf-card table-card">
    <div class="table-card__head">
      <div class="table-card__title">
        <h3 v-if="title">{{ title }}</h3>
        <span v-if="sub" class="table-card__sub">{{ sub }}</span>
      </div>
      <div class="table-card__actions">
        <slot name="filters" />
        <a-input
          v-if="showSearch"
          placeholder="搜索"
          style="width: 220px"
          allow-clear
          @press-enter="emit('search', $event.target.value)"
        >
          <template #prefix>
            <SfIcon name="SearchOutlined" :size="14" tone="tertiary" />
          </template>
        </a-input>
        <a-tooltip v-if="showRefresh" title="刷新">
          <a-button shape="circle" size="small" @click="emit('refresh')">
            <template #icon><SfIcon name="ReloadOutlined" :size="14" tone="secondary" /></template>
          </a-button>
        </a-tooltip>
        <a-button v-if="showAdd" type="primary" @click="emit('add')">
          <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
          {{ addText }}
        </a-button>
        <slot name="extra" />
      </div>
    </div>
    <div class="table-card__body">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.sf-card, .table-card {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--r-lg);
  overflow: hidden;
  transition: border-color var(--dur-fast) var(--ease);
}
.table-card:hover { border-color: var(--border-color); }

.table-card__head {
  padding: var(--sp-4) var(--sp-5);
  display: flex; align-items: center; justify-content: space-between;
  gap: var(--sp-4); flex-wrap: wrap;
  border-bottom: 1px solid var(--border-light);
}
.table-card__title { display: flex; align-items: baseline; gap: var(--sp-3); flex-wrap: wrap; }
.table-card__title h3 { margin: 0; font-size: var(--fs-md); font-weight: 600; color: var(--text-primary); }
.table-card__sub { font-size: var(--fs-xs); color: var(--text-tertiary); }
.table-card__actions { display: flex; align-items: center; gap: var(--sp-2); flex-wrap: wrap; }
.table-card__body { padding: 0; }

/* 表格行 hover 动画 */
:deep(.ant-table-tbody > tr) {
  transition: background var(--dur-fast) var(--ease);
}
:deep(.ant-table-tbody > tr:hover > td) {
  background: var(--bg-hover) !important;
}
/* 表格头部 token 化 */
:deep(.ant-table-thead > tr > th) {
  background: var(--bg-page) !important;
  color: var(--text-secondary) !important;
  font-weight: 500 !important;
  font-size: var(--fs-xs) !important;
  border-bottom: 1px solid var(--border-color) !important;
}
:deep(.ant-table-tbody > tr > td) {
  border-bottom: 1px solid var(--border-light) !important;
  font-size: var(--fs-sm);
}

@media (max-width: 768px) {
  .table-card__head { padding: var(--sp-3); }
  .table-card__actions { width: 100%; }
}
</style>