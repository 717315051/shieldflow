<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, PieChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent, DataZoomComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, LineChart, BarChart, PieChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, DataZoomComponent])

import { trafficApi } from '../api'

const activeTab = ref('stats')
const loading = ref(false)
const range = ref('24h')
const bandwidthChart = ref(null)
const cacheHitChart = ref(null)

const statsData = reactive({
  total_traffic: 0,
  total_requests: 0,
  avg_bandwidth: 0,
  peak_bandwidth: 0,
  hit_rate: 0,
  miss_rate: 0,
})

const rankingData = ref([])
const rankingColumns = [
  { title: '排名', dataIndex: 'rank', width: 80 },
  { title: '域名', dataIndex: 'domain' },
  { title: '流量(GB)', dataIndex: 'traffic' },
  { title: '请求数', dataIndex: 'requests' },
  { title: '占比', dataIndex: 'percent' },
]

const bwX = ref([])
const bwSeries = reactive({ up: [], down: [] })
const cacheX = ref([])
const cacheSeries = reactive({ hit: [], miss: [] })

const bandwidthOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['上行', '下行'] },
  grid: { left: 50, right: 30, bottom: 60 },
  xAxis: { type: 'category', data: bwX.value },
  yAxis: { type: 'value', name: 'Mbps' },
  dataZoom: [{ type: 'inside' }, { type: 'slider' }],
  series: [
    { name: '上行', type: 'line', smooth: true, areaStyle: {}, data: bwSeries.up },
    { name: '下行', type: 'line', smooth: true, areaStyle: {}, data: bwSeries.down },
  ],
}))

const cacheHitOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['命中', '未命中'] },
  grid: { left: 50, right: 30, bottom: 60 },
  xAxis: { type: 'category', data: cacheX.value },
  yAxis: { type: 'value', name: '次数' },
  series: [
    { name: '命中', type: 'bar', stack: 'total', data: cacheSeries.hit, itemStyle: { color: '#52c41a' } },
    { name: '未命中', type: 'bar', stack: 'total', data: cacheSeries.miss, itemStyle: { color: '#ff4d4f' } },
  ],
}))

async function loadStats() {
  try {
    const res = await trafficApi.stats({ range: range.value })
    Object.assign(statsData, res.data || {})
  } catch {}
}

async function loadRanking() {
  try {
    const res = await trafficApi.ranking({ range: range.value })
    rankingData.value = (res.data || []).map((d, i) => ({ ...d, rank: i + 1 }))
  } catch {}
}

async function loadBandwidth() {
  try {
    const res = await trafficApi.bandwidth({ range: range.value })
    const d = res.data || []
    bwX.value = d.map((i) => i.time)
    bwSeries.up = d.map((i) => i.up)
    bwSeries.down = d.map((i) => i.down)
    nextTick(() => bandwidthChart.value?.setOption(bandwidthOption.value, true))
  } catch {}
}

async function loadCacheHit() {
  try {
    const res = await trafficApi.cacheHit({ range: range.value })
    const d = res.data || []
    cacheX.value = d.map((i) => i.time)
    cacheSeries.hit = d.map((i) => i.hit)
    cacheSeries.miss = d.map((i) => i.miss)
    nextTick(() => cacheHitChart.value?.setOption(cacheHitOption.value, true))
  } catch {}
}

async function refresh() {
  loading.value = true
  await Promise.all([loadStats(), loadRanking(), loadBandwidth(), loadCacheHit()])
  loading.value = false
}

function formatNum(n) {
  if (!n) return '0'
  if (n > 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (n > 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (n > 1e3) return (n / 1e3).toFixed(2) + 'K'
  return String(n)
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-toolbar">
      <h2 style="margin: 0">流量统计</h2>
      <a-space>
        <a-select v-model:value="range" style="width: 120px" @change="refresh">
          <a-select-option value="1h">近1小时</a-select-option>
          <a-select-option value="24h">近24小时</a-select-option>
          <a-select-option value="7d">近7天</a-select-option>
          <a-select-option value="30d">近30天</a-select-option>
        </a-select>
        <a-button @click="refresh">刷新</a-button>
      </a-space>
    </div>

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="stats" tab="总体统计">
        <div class="stat-card-grid">
          <a-card><a-statistic title="总流量" :value="formatNum(statsData.total_traffic)" suffix="GB" /></a-card>
          <a-card><a-statistic title="总请求数" :value="formatNum(statsData.total_requests)" /></a-card>
          <a-card><a-statistic title="平均带宽" :value="formatNum(statsData.avg_bandwidth)" suffix="Mbps" /></a-card>
          <a-card><a-statistic title="峰值带宽" :value="formatNum(statsData.peak_bandwidth)" suffix="Mbps" /></a-card>
          <a-card><a-statistic title="缓存命中率" :value="statsData.hit_rate" suffix="%" :value-style="{ color: '#52c41a' }" /></a-card>
          <a-card><a-statistic title="未命中率" :value="statsData.miss_rate" suffix="%" :value-style="{ color: '#ff4d4f' }" /></a-card>
        </div>
        <a-card title="带宽趋势" style="margin-top: 16px">
          <v-chart ref="bandwidthChart" class="chart-box" :option="bandwidthOption" autoresize />
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="ranking" tab="域名排行">
        <a-table :columns="rankingColumns" :data-source="rankingData" row-key="domain" :pagination="false" />
      </a-tab-pane>

      <a-tab-pane key="cache" tab="缓存命中率">
        <v-chart ref="cacheHitChart" class="chart-box" :option="cacheHitOption" autoresize />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>
