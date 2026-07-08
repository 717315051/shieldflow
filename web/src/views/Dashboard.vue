<script setup>
import { ref, onMounted, computed, h } from 'vue'
import { message } from 'ant-design-vue'
import {
  GlobalOutlined, BarChartOutlined, ApiOutlined, SafetyOutlined,
  DatabaseOutlined, ClusterOutlined,
} from '@ant-design/icons-vue'
import { dashboardApi } from '../api/index'
import SfStatGrid from '../components/SfStatGrid.vue'
import SfPageHeader from '../components/SfPageHeader.vue'
import SfChartCard from '../components/SfChartCard.vue'
import SfTableCard from '../components/SfTableCard.vue'
import SfTabs from '../components/SfTabs.vue'
import SfStatusBadge from '../components/SfStatusBadge.vue'
import SfIcon from '../components/SfIcon.vue'

const overview = ref({
  totalDomains: 12, totalTraffic: 184.6, totalRequests: 2384912,
  attackCount: 1842, cacheHitRate: 0.872, activeNodes: 8,
})

const trendTab = ref('china')
const chartType = ref('flow')

const statCards = computed(() => [
  { label: '域名总数', value: overview.value.totalDomains, unit: '个', icon: h(GlobalOutlined), tone: 'blue' },
  { label: '总流量', value: formatNumber(overview.value.totalTraffic), unit: 'GB', icon: h(BarChartOutlined), tone: 'green', trend: '12.5%', trendUp: true },
  { label: '总请求数', value: formatNumber(overview.value.totalRequests), unit: '次', icon: h(ApiOutlined), tone: 'purple', trend: '8.2%', trendUp: true },
  { label: 'DDoS 拦截', value: formatNumber(overview.value.attackCount), unit: '次', icon: h(SafetyOutlined), tone: 'orange', trend: '24.1%', trendUp: true },
  { label: '缓存命中率', value: ((overview.value.cacheHitRate || 0) * 100).toFixed(1), unit: '%', icon: h(DatabaseOutlined), tone: 'pink', trend: '3.6%', trendUp: false },
  { label: '在线节点', value: overview.value.activeNodes, unit: '台', icon: h(ClusterOutlined), tone: 'white' },
])

const trendTabs = [
  { label: '中国', value: 'china' },
  { label: '世界', value: 'world' },
  { label: '访问统计', value: 'visit' },
  { label: '攻击统计', value: 'attack' },
]
const chartTabs = [
  { label: '流量', value: 'flow' },
  { label: '请求', value: 'req' },
  { label: '带宽', value: 'bw' },
]

const recentAttacks = ref([
  { time: '2 分钟前', type: 'SYN Flood', src: '192.168.1.100', target: 'a.example.com', status: '已拦截', statusType: 'success' },
  { time: '25 分钟前', type: 'CC Attack', src: '203.0.113.42', target: 'b.example.com', status: '已拦截', statusType: 'success' },
  { time: '1 小时前', type: 'UDP Flood', src: '198.51.100.7', target: 'c.example.com', status: '已清洗', statusType: 'info' },
  { time: '3 小时前', type: 'HTTP Flood', src: '203.0.113.99', target: 'a.example.com', status: '已拦截', statusType: 'success' },
  { time: '昨天 09:17', type: 'DNS Query', src: '192.0.2.55', target: 'd.example.com', status: '已放行', statusType: 'neutral' },
])
const recentLogs = ref([
  { time: '14:32:01', level: 'INFO', module: '缓存', message: '已刷新 23 个 URL' },
  { time: '14:30:22', level: 'WARN', module: '节点', message: 'node-02 CPU 使用率 85%' },
  { time: '14:28:00', level: 'INFO', module: '证书', message: 'example.com 证书续签成功' },
  { time: '14:25:11', level: 'ERROR', module: 'API', message: '/v1/cdn/purge 接口超时' },
  { time: '14:20:33', level: 'INFO', module: '用户', message: '新用户注册: zhangsan' },
])

const chartData = ref({ xAxis: [], series: [] })

const refresh = async () => {
  try {
    const r = await dashboardApi.overview().catch(() => null)
    if (r?.data) overview.value = r.data
    const xs = Array.from({ length: 24 }, (_, i) => `${i.toString().padStart(2, '0')}:00`)
    chartData.value = {
      xAxis: xs,
      series: [{
        name: chartType.value,
        data: xs.map(() => Math.floor(Math.random() * 5000) + 1000),
        area: true, smooth: true,
      }],
    }
    message.success('已刷新')
  } catch (e) { console.error(e) }
}

onMounted(refresh)

function formatNumber(n) {
  if (n == null) return '0'
  if (n >= 1e8) return (n / 1e8).toFixed(1)
  if (n >= 1e4) return (n / 1e4).toFixed(1)
  return n.toLocaleString()
}

const pathD = computed(() => {
  const s = chartData.value.series[0]
  if (!s?.data?.length) return ''
  const max = Math.max(...s.data), min = Math.min(...s.data)
  const w = 800, h = 220, pad = 20
  const stepX = (w - pad * 2) / (s.data.length - 1)
  return s.data.map((v, i) => {
    const x = pad + i * stepX
    const y = pad + (1 - (v - min) / (max - min || 1)) * (h - pad * 2)
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
})
const areaD = computed(() => {
  const line = pathD.value
  if (!line) return ''
  return `${line} L780,220 L20,220 Z`
})

const logLevelType = (l) => {
  if (l === 'INFO') return 'info'
  if (l === 'WARN') return 'warning'
  if (l === 'ERROR') return 'danger'
  return 'neutral'
}
</script>

<template>
  <div class="dashboard">
    <SfPageHeader title="数据总览" sub="实时业务监控" :show-refresh="true" @refresh="refresh" />

    <SfStatGrid :stats="statCards" />

    <div class="dashboard__row">
      <SfChartCard title="流量趋势" sub="最近 24 小时" class="dashboard__row-main">
        <template #extra>
          <SfTabs v-model="chartType" :tabs="chartTabs" size="small" />
        </template>
        <div class="dashboard__chart">
          <svg viewBox="0 0 800 240" preserveAspectRatio="none" style="width:100%;height:240px">
            <defs>
              <linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#165dff" stop-opacity="0.18" />
                <stop offset="100%" stop-color="#165dff" stop-opacity="0" />
              </linearGradient>
            </defs>
            <line v-for="i in 4" :key="i" x1="20" :y1="20 + (i-1) * 60" x2="780" :y2="20 + (i-1) * 60" stroke="var(--border-light)" stroke-dasharray="3,3" />
            <path :d="areaD" fill="url(#chartGrad)" />
            <path :d="pathD" fill="none" stroke="var(--brand-primary)" stroke-width="2" />
            <text v-for="(x, i) in chartData.xAxis" v-show="i % 4 === 0" :key="i" :x="20 + i * (760 / (chartData.xAxis.length - 1))" y="234" font-size="10" fill="var(--text-tertiary)" text-anchor="middle">{{ x }}</text>
          </svg>
        </div>
      </SfChartCard>

      <SfChartCard title="访问来源" sub="地区分布" class="dashboard__row-side">
        <template #extra>
          <SfTabs v-model="trendTab" :tabs="trendTabs" size="small" />
        </template>
        <div class="dashboard__map">
          <svg viewBox="0 0 360 180" style="width:100%;height:auto">
            <g fill="var(--border-color)">
              <template v-for="row in 9" :key="`r${row}`">
                <template v-for="col in 18" :key="`c${col}`">
                  <circle :cx="10 + col * 19" :cy="10 + row * 19" r="1.5" />
                </template>
              </template>
            </g>
            <g>
              <g>
                <circle cx="240" cy="80" r="6" fill="var(--warning)" opacity="0.4">
                  <animate attributeName="r" values="6;10;6" dur="2s" repeatCount="indefinite" />
                </circle>
                <circle cx="240" cy="80" r="3" fill="var(--warning)" />
                <text x="252" y="84" font-size="10" fill="var(--text-secondary)">北京</text>
              </g>
              <g>
                <circle cx="270" cy="95" r="5" fill="var(--info)" opacity="0.4">
                  <animate attributeName="r" values="5;9;5" dur="2.5s" repeatCount="indefinite" />
                </circle>
                <circle cx="270" cy="95" r="2.5" fill="var(--info)" />
                <text x="282" y="99" font-size="10" fill="var(--text-secondary)">上海</text>
              </g>
              <g>
                <circle cx="100" cy="60" r="5" fill="var(--success)" opacity="0.4">
                  <animate attributeName="r" values="5;9;5" dur="3s" repeatCount="indefinite" />
                </circle>
                <circle cx="100" cy="60" r="2.5" fill="var(--success)" />
                <text x="112" y="64" font-size="10" fill="var(--text-secondary)">纽约</text>
              </g>
              <g>
                <circle cx="80" cy="50" r="4" fill="#722ed1" opacity="0.4">
                  <animate attributeName="r" values="4;8;4" dur="2.8s" repeatCount="indefinite" />
                </circle>
                <circle cx="80" cy="50" r="2" fill="#722ed1" />
                <text x="92" y="54" font-size="10" fill="var(--text-secondary)">伦敦</text>
              </g>
            </g>
          </svg>
          <div class="dashboard__map-legend">
            <span><i style="background:var(--warning)"></i>华北</span>
            <span><i style="background:var(--info)"></i>华东</span>
            <span><i style="background:var(--success)"></i>美洲</span>
            <span><i style="background:#722ed1"></i>欧洲</span>
          </div>
        </div>
      </SfChartCard>
    </div>

    <div class="dashboard__row dashboard__row--table">
      <SfTableCard title="最近攻击事件" :show-search="false" :show-refresh="false">
        <a-table
          :data-source="recentAttacks"
          :columns="[
            { title: '时间', dataIndex: 'time', width: 120 },
            { title: '类型', dataIndex: 'type', width: 110 },
            { title: '来源 IP', dataIndex: 'src', width: 140 },
            { title: '目标域名', dataIndex: 'target' },
            { title: '状态', dataIndex: 'status', width: 110 },
          ]"
          :pagination="false"
          size="middle"
          row-key="time"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <SfStatusBadge :status="record.statusType" :text="record.status" />
            </template>
            <template v-else-if="column.dataIndex === 'src'">
              <a href="javascript:;" style="color: var(--brand-primary)">{{ record.src }}</a>
            </template>
          </template>
        </a-table>
      </SfTableCard>

      <SfTableCard title="系统日志" :show-search="false" :show-refresh="false">
        <a-table
          :data-source="recentLogs"
          :columns="[
            { title: '时间', dataIndex: 'time', width: 100 },
            { title: '级别', dataIndex: 'level', width: 80 },
            { title: '模块', dataIndex: 'module', width: 80 },
            { title: '消息', dataIndex: 'message' },
          ]"
          :pagination="false"
          size="middle"
          row-key="time"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'level'">
              <SfStatusBadge :status="logLevelType(record.level)" :text="record.level" />
            </template>
          </template>
        </a-table>
      </SfTableCard>
    </div>
  </div>
</template>

<style scoped>
.dashboard__row {
  display: grid; grid-template-columns: 2fr 1fr; gap: var(--sp-4);
  margin-bottom: var(--sp-4);
}
.dashboard__row--table { grid-template-columns: 1fr 1fr; }

.dashboard__chart { padding-top: var(--sp-2); }
.dashboard__map { padding-top: var(--sp-2); }
.dashboard__map-legend {
  display: flex; flex-wrap: wrap; gap: var(--sp-3);
  padding: var(--sp-2) 4px 0; font-size: var(--fs-xs); color: var(--text-secondary);
}
.dashboard__map-legend span { display: inline-flex; align-items: center; gap: 4px; }
.dashboard__map-legend i {
  display: inline-block; width: 8px; height: 8px; border-radius: 50%;
}

@media (max-width: 1024px) {
  .dashboard__row, .dashboard__row--table { grid-template-columns: 1fr; }
}
</style>