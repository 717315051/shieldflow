<script setup>
import { computed } from 'vue'
import SfIcon from './SfIcon.vue'

const props = defineProps({
  status: { type: String, required: true },  // success / warning / danger / info / neutral
  text: { type: String, default: '' },
})

const iconName = computed(() => {
  const map = {
    success: 'CheckCircleFilled',
    warning: 'ExclamationCircleFilled',
    danger: 'CloseCircleFilled',
    info: 'InfoCircleFilled',
    neutral: 'MinusCircleFilled',
  }
  return map[props.status] || map.neutral
})

const labelText = computed(() => {
  if (props.text) return props.text
  const map = {
    success: '成功', warning: '警告', danger: '失败', info: '信息', neutral: '禁用',
  }
  return map[props.status] || props.status
})
</script>

<template>
  <span :class="['sf-badge', `sf-badge--${status}`]">
    <SfIcon :name="iconName" :size="12" />
    {{ labelText }}
  </span>
</template>

<style scoped>
.sf-badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 8px;
  border-radius: var(--r-md);
  font-size: var(--fs-xs);
  font-weight: 500;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}
.sf-badge--success { color: var(--success); background: var(--success-soft); }
.sf-badge--warning { color: var(--warning); background: var(--warning-soft); }
.sf-badge--danger  { color: var(--danger);  background: var(--danger-soft); }
.sf-badge--info    { color: var(--info);    background: var(--info-soft); }
.sf-badge--neutral { color: var(--neutral); background: var(--neutral-soft); }
</style>