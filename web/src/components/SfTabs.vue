<script setup>
defineProps({
  tabs: { type: Array, required: true },
  modelValue: { type: [String, Number], default: '' },
  size: { type: String, default: 'default' },
})
defineEmits(['update:modelValue', 'change'])
</script>

<template>
  <div :class="['sf-tabs', `sf-tabs--${size}`]">
    <div
      v-for="t in tabs"
      :key="t.value"
      :class="['sf-tabs__item', { active: modelValue === t.value }]"
      @click="$emit('update:modelValue', t.value); $emit('change', t.value)"
    >
      {{ t.label }}
    </div>
  </div>
</template>

<style scoped>
.sf-tabs {
  display: flex; align-items: center; gap: 0;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: var(--sp-4);
  overflow-x: auto; overflow-y: hidden;
  scrollbar-width: none;
}
.sf-tabs::-webkit-scrollbar { display: none; }
.sf-tabs__item {
  padding: var(--sp-3) var(--sp-4);
  font-size: var(--fs-sm);
  color: var(--text-secondary);
  cursor: pointer; user-select: none;
  position: relative; white-space: nowrap;
  transition: color var(--dur-fast) var(--ease);
}
.sf-tabs__item:hover { color: var(--text-primary); }
.sf-tabs__item.active { color: var(--brand-primary); font-weight: 500; }
.sf-tabs__item.active::after {
  content: ''; position: absolute; left: var(--sp-4); right: var(--sp-4); bottom: -1px;
  height: 2px;
  background: var(--brand-primary);
  border-radius: 2px 2px 0 0;
}

.sf-tabs--small .sf-tabs__item { padding: 6px var(--sp-3); font-size: var(--fs-xs); }
.sf-tabs--large .sf-tabs__item { padding: 14px var(--sp-5); font-size: var(--fs-md); }

@media (max-width: 768px) {
  .sf-tabs__item { padding: var(--sp-2) var(--sp-3); font-size: var(--fs-xs); }
}
</style>