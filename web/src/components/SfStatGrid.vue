<script setup>
defineProps({
  stats: { type: Array, required: true },
})
</script>

<template>
  <div class="stat-grid">
    <div v-for="(s, i) in stats" :key="i" :class="['stat-card', s.tone && `stat-card--${s.tone}`]">
      <div class="stat-card__icon">
        <component :is="s.icon" v-if="s.icon" />
        <span v-else>{{ s.iconText || '·' }}</span>
      </div>
      <div class="stat-card__body">
        <div class="stat-card__label">{{ s.label }}</div>
        <div class="stat-card__value">
          <span class="stat-card__num">{{ s.value }}</span>
          <span class="stat-card__unit" v-if="s.unit">{{ s.unit }}</span>
        </div>
        <div v-if="s.trend" :class="['stat-card__trend', s.trendUp ? 'up' : 'down']">
          <span class="arrow" v-if="s.trendUp">▲</span>
          <span class="arrow" v-else>▼</span>
          <span>较上期 {{ s.trend }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--sp-4);
  margin-bottom: var(--sp-5);
}

/* 卡片基底: 极轻阴影 + 1px 分隔线,圆角 8px */
.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--r-lg);
  padding: var(--sp-5);
  display: flex; align-items: center; gap: var(--sp-4);
  transition: border-color var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease);
}
.stat-card:hover {
  border-color: var(--border-color);
  background: var(--bg-hover);
}

.stat-card__icon {
  width: 44px; height: 44px;
  border-radius: var(--r-md);
  display: flex; align-items: center; justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
  color: var(--text-secondary);
  background: var(--bg-page);
}
.stat-card__body { flex: 1; min-width: 0; }
.stat-card__label {
  font-size: var(--fs-xs);
  color: var(--text-tertiary);
  font-weight: 400;
  margin-bottom: var(--sp-2);
}
/* 数字 26-32px 加粗 + 单位 12px 60% 灰 */
.stat-card__value { line-height: 1.1; }
.stat-card__num {
  font-size: var(--fs-xl);
  font-weight: 700;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.3px;
}
.stat-card__unit {
  font-size: var(--fs-xs);
  color: var(--text-tertiary);
  margin-left: var(--sp-1);
  font-weight: 400;
}
.stat-card__trend {
  font-size: var(--fs-xs);
  margin-top: var(--sp-2);
  display: flex; align-items: center; gap: 4px;
  color: var(--text-tertiary);
}
.stat-card__trend .arrow { font-size: 9px; }
.stat-card__trend.up { color: var(--success); }
.stat-card__trend.down { color: var(--danger); }

/* 6 色调色板: 用 token 化 tone,不再写死 hex */
.stat-card--blue   { border-left: 3px solid var(--info); }
.stat-card--blue   .stat-card__icon { background: var(--info-soft); color: var(--info); }
.stat-card--green  { border-left: 3px solid var(--success); }
.stat-card--green  .stat-card__icon { background: var(--success-soft); color: var(--success); }
.stat-card--purple { border-left: 3px solid #722ed1; }
.stat-card--purple .stat-card__icon { background: rgba(114,46,209,0.1); color: #722ed1; }
.stat-card--orange { border-left: 3px solid var(--warning); }
.stat-card--orange .stat-card__icon { background: var(--warning-soft); color: var(--warning); }
.stat-card--pink   { border-left: 3px solid #eb2f96; }
.stat-card--pink   .stat-card__icon { background: rgba(235,47,150,0.1); color: #eb2f96; }
.stat-card--white  { border-left: 3px solid var(--border-color); }
.stat-card--white  .stat-card__icon { background: var(--neutral-soft); color: var(--text-secondary); }

/* 暗色模式: tone 适配 */
:root[data-theme="dark"] .stat-card--purple { border-left-color: #b591ff; }
:root[data-theme="dark"] .stat-card--purple .stat-card__icon { background: rgba(114,46,209,0.18); color: #b591ff; }
:root[data-theme="dark"] .stat-card--pink { border-left-color: #ff85c0; }
:root[data-theme="dark"] .stat-card--pink .stat-card__icon { background: rgba(235,47,150,0.18); color: #ff85c0; }

@media (max-width: 1024px) {
  .stat-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 768px) {
  .stat-grid { grid-template-columns: 1fr 1fr; gap: var(--sp-3); margin-bottom: var(--sp-4); }
  .stat-card { padding: var(--sp-3); gap: var(--sp-3); }
  .stat-card__icon { width: 36px; height: 36px; font-size: 16px; border-radius: var(--r-sm); }
  .stat-card__num { font-size: var(--fs-lg); }
  .stat-card__label { font-size: 11px; }
}
@media (max-width: 480px) {
  .stat-grid { grid-template-columns: 1fr; }
  .stat-card__trend { display: none; }
}
</style>