<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, PieChart, MapChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  ToolboxComponent,
} from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  BarChart,
  PieChart,
  MapChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  ToolboxComponent,
])

import { dashboardApi } from '../api'
import { message } from 'ant-design-vue'

const loading = ref(false)
const overview = reactive({
  total_traffic: 0,
  total_requests: 0,
  blocked_requests: 0,
  bandwidth: 0,
  domains: 0,
  nodes: 0,
})

const trafficChart = ref(null)
const geoChart = ref(null)
const trafficRange = ref('24h')

const trafficOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['流量(GB)', '请求数(万)', '拦截数'] },
  grid: { left: 50, right: 50, bottom: 60 },
  xAxis: { type: 'category', data: trafficX.value },
  yAxis: [
    { type: 'value', name: '流量' },
    { type: 'value', name: '请求数' },
  ],
  dataZoom: [{ type: 'inside' }, { type: 'slider' }],
  series: [
    {
      name: '流量(GB)',
      type: 'line',
      smooth: true,
      areaStyle: {},
      data: trafficData.value.flow,
    },
    {
      name: '请求数(万)',
      type: 'line',
      yAxisIndex: 1,
      smooth: true,
      data: trafficData.value.req,
    },
    {
      name: '拦截数',
      type: 'bar',
      yAxisIndex: 1,
      data: trafficData.value.block,
    },
  ],
}))

const geoOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} 次' },
  visualMap: {
    min: 0,
    max: 10000,
    left: 10,
    inColor: ['#e0ffff', '#006edd'],
  },
  series: [
    {
      name: '访问分布',
      type: 'map',
      map: 'china',
      roam: true,
      data: geoData.value,
    },
  ],
}))

const trafficX = ref([])
const trafficData = reactive({ flow: [], req: [], block: [] })
const geoData = ref([])

async function loadOverview() {
  try {
    const res = await dashboardApi.overview()
    Object.assign(overview, res.data)
  } catch {}
}

async function loadTraffic() {
  try {
    const res = await dashboardApi.traffic({ range: trafficRange.value })
    const d = res.data || []
    trafficX.value = d.map((i) => i.time)
    trafficData.flow = d.map((i) => i.flow)
    trafficData.req = d.map((i) => i.req)
    trafficData.block = d.map((i) => i.block)
    nextTick(() => trafficChart.value?.setOption(trafficOption.value, true))
  } catch {}
}

async function loadGeo() {
  try {
    const res = await dashboardApi.geo()
    geoData.value = res.data || []
    nextTick(() => geoChart.value?.setOption(geoOption.value, true))
  } catch {}
}

function formatNum(n) {
  if (!n) return '0'
  if (n > 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (n > 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (n > 1e3) return (n / 1e3).toFixed(2) + 'K'
  return String(n)
}

async function refresh() {
  loading.value = true
  await Promise.all([loadOverview(), loadTraffic(), loadGeo()])
  loading.value = false
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-toolbar">
      <h2 style="margin: 0">仪表盘</h2>
      <a-space>
        <a-select v-model:value="trafficRange" style="width: 120px" @change="loadTraffic">
          <a-select-option value="1h">近1小时</a-select-option>
          <a-select-option value="24h">近24小时</a-select-option>
          <a-select-option value="7d">近7天</a-select-option>
          <a-select-option value="30d">近30天</a-select-option>
        </a-select>
        <a-button @click="refresh">刷新</a-button>
      </a-space>
    </div>

    <div class="stat-card-grid">
      <a-card>
        <a-statistic title="总流量" :value="formatNum(overview.total_traffic)" suffix="GB" />
      </a-card>
      <a-card>
        <a-statistic title="总请求数" :value="formatNum(overview.total_requests)" />
      </a-card>
      <a-card>
        <a-statistic
          title="拦截数"
          :value="formatNum(overview.blocked_requests)"
          :value-style="{ color: '#ff4d4f' }"
        />
      </a-card>
      <a-card>
        <a-statistic title="当前带宽" :value="formatNum(overview.bandwidth)" suffix="Mbps" />
      </a-card>
      <a-card>
        <a-statistic title="域名数" :value="overview.domains" />
      </a-card>
      <a-card>
        <a-statistic title="节点数" :value="overview.nodes" />
      </a-card>
    </div>

    <a-row :gutter="16">
      <a-col :xs="24" :lg="16">
        <a-card title="流量趋势">
          <v-chart ref="trafficChart" class="chart-box" :option="trafficOption" autoresize />
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="8">
        <a-card title="地理分布">
          <v-chart ref="geoChart" class="chart-box" :option="geoOption" autoresize />
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>
