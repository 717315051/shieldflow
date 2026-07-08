<script setup>
defineProps({
  title: { type: String, default: '' },
  sub: { type: String, default: '' },
  padding: { type: Boolean, default: true },
})
</script>

<template>
  <div class="sf-card chart-card">
    <div class="chart-card__head" v-if="title || $slots.extra">
      <div>
        <h3 v-if="title">{{ title }}</h3>
        <span v-if="sub" class="chart-card__sub">{{ sub }}</span>
      </div>
      <div class="chart-card__extra"><slot name="extra" /></div>
    </div>
    <div :class="['chart-card__body', { 'no-pad': !padding }]">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.sf-card, .chart-card {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--r-lg);
  overflow: hidden;
  transition: border-color var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease);
}
.sf-card:hover, .chart-card:hover { border-color: var(--border-color); }

.chart-card__head {
  padding: var(--sp-4) var(--sp-5);
  border-bottom: 1px solid var(--border-light);
  display: flex; align-items: flex-start; justify-content: space-between; gap: var(--sp-4);
}
.chart-card__head h3 { margin: 0; font-size: var(--fs-md); font-weight: 600; color: var(--text-primary); }
.chart-card__sub { font-size: var(--fs-xs); color: var(--text-tertiary); margin-top: 4px; display: block; }
.chart-card__extra { flex-shrink: 0; }
.chart-card__body { padding: var(--sp-4) var(--sp-5); }
.chart-card__body.no-pad { padding: 0; }

@media (max-width: 768px) {
  .chart-card__head { padding: var(--sp-3); }
  .chart-card__body { padding: var(--sp-3); }
}
</style>