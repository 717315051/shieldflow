<script setup>
import * as Icons from '@ant-design/icons-vue'
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },   // e.g. 'DashboardOutlined'
  size: { type: [Number, String], default: 16 },  // 16 / 20 / 24
  tone: { type: String, default: 'secondary' },   // primary / secondary / tertiary / brand / success / warning / danger
})

const iconComponent = computed(() => {
  return Icons[props.name] || Icons.QuestionCircleOutlined
})

const styleObj = computed(() => ({
  fontSize: typeof props.size === 'number' ? `${props.size}px` : props.size,
  color: `var(--text-${props.tone})`,
  lineHeight: 1,
  verticalAlign: 'middle',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
}))
</script>

<template>
  <component :is="iconComponent" :style="styleObj" />
</template>