<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { wafApi } from '../../api'
import { ReloadOutlined } from '@ant-design/icons-vue'

use([CanvasRenderer, PieChart, LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

const activeTab = ref('dashboard')
const loading = ref(false)

// ============ 仪表盘 ============
const overview = reactive({
  today_blocked: 0,
  total_blocked: 0,
  active_rules: 0,
  blocked_ips: 0,
})
const pieChart = ref(null)
const pieData = ref([])
const topIps = ref([])

const pieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { bottom: 0 },
  series: [
    {
      name: '威胁类型',
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false, position: 'center' },
      emphasis: { label: { show: true, fontSize: 18, fontWeight: 'bold' } },
      labelLine: { show: false },
      data: pieData.value,
    },
  ],
}))

const topIpColumns = [
  { title: '排名', key: 'rank', width: 70, customRender: ({ index }) => index + 1 },
  { title: 'IP', dataIndex: 'ip' },
  { title: '攻击次数', dataIndex: 'count', sorter: (a, b) => a.count - b.count },
  { title: '主要威胁', dataIndex: 'threat_type' },
  { title: '最近攻击时间', dataIndex: 'last_time' },
]

async function loadDashboard() {
  loading.value = true
  try {
    const res = await wafApi.dashboard()
    const d = res.data || {}
    Object.assign(overview, {
      today_blocked: d.today_blocked ?? 0,
      total_blocked: d.total_blocked ?? 0,
      active_rules: d.active_rules ?? 0,
      blocked_ips: d.blocked_ips ?? 0,
    })
    pieData.value = (d.threat_types || []).map((t) => ({ name: t.name, value: t.value }))
    topIps.value = d.top_ips || []
    nextTick(() => pieChart.value?.setOption(pieOption.value, true))
  } finally {
    loading.value = false
  }
}

// ============ 配置 ============
const configForm = reactive({
  enabled: false,
  mode: 'block',
  detection_level: 'medium',
  sql_injection: true,
  xss: true,
  path_traversal: true,
  command_injection: true,
  cc_protection: true,
  scanner_block: true,
  rate_limit: 100,
  block_duration: 3600,
  log_retention: 30,
  whitelist: '',
})
const configLoading = ref(false)

async function loadConfig() {
  configLoading.value = true
  try {
    const res = await wafApi.config()
    Object.assign(configForm, res.data || {})
  } finally {
    configLoading.value = false
  }
}

async function saveConfig() {
  await wafApi.updateConfig({ ...configForm })
  message.success('配置已保存')
}

// ============ 日志 ============
const logs = ref([])
const logTotal = ref(0)
const logQuery = reactive({
  domain: '',
  ip: '',
  threat_type: undefined,
  start_time: undefined,
  end_time: undefined,
  page: 1,
  page_size: 10,
})

const logColumns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '域名', dataIndex: 'domain' },
  { title: '客户端IP', dataIndex: 'ip' },
  { title: '威胁类型', dataIndex: 'threat_type' },
  { title: '规则', dataIndex: 'rule' },
  { title: '匹配内容', dataIndex: 'match', ellipsis: true },
  { title: '动作', dataIndex: 'action' },
  { title: '状态', dataIndex: 'status' },
]

async function loadLogs() {
  loading.value = true
  try {
    const res = await wafApi.logs(logQuery)
    logs.value = res.data?.list || []
    logTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function resetLogQuery() {
  Object.assign(logQuery, {
    domain: '',
    ip: '',
    threat_type: undefined,
    start_time: undefined,
    end_time: undefined,
    page: 1,
  })
  loadLogs()
}

// ============ 攻击分析 ============
const trendChart = ref(null)
const trendX = ref([])
const trendSeries = ref([])
const analysisRange = ref('7d')

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: trendSeries.value.map((s) => s.name), bottom: 0 },
  grid: { left: 50, right: 30, bottom: 60, top: 30 },
  xAxis: { type: 'category', data: trendX.value, boundaryGap: false },
  yAxis: { type: 'value', name: '拦截数' },
  series: trendSeries.value.map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    data: s.data,
  })),
}))

async function loadAnalysis() {
  loading.value = true
  try {
    const res = await wafApi.analysis({ range: analysisRange.value })
    const d = res.data || {}
    trendX.value = d.dates || []
    trendSeries.value = (d.series || []).map((s) => ({ name: s.name, data: s.data }))
    nextTick(() => trendChart.value?.setOption(trendOption.value, true))
  } finally {
    loading.value = false
  }
}

function onTabChange(key) {
  if (key === 'config') loadConfig()
  if (key === 'logs') loadLogs()
  if (key === 'analysis') loadAnalysis()
}

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">WAF 管理</h2>
      <a-button @click="activeTab === 'dashboard' ? loadDashboard() : (activeTab === 'config' ? loadConfig() : activeTab === 'logs' ? loadLogs() : loadAnalysis())">
        <ReloadOutlined /> 刷新
      </a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <!-- 仪表盘 -->
      <a-tab-pane key="dashboard" tab="仪表盘">
        <div class="stat-card-grid">
          <a-card><a-statistic title="今日拦截" :value="overview.today_blocked" :value-style="{ color: '#ff4d4f' }" /></a-card>
          <a-card><a-statistic title="累计拦截" :value="overview.total_blocked" /></a-card>
          <a-card><a-statistic title="生效规则" :value="overview.active_rules" /></a-card>
          <a-card><a-statistic title="封锁IP" :value="overview.blocked_ips" /></a-card>
        </div>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card title="威胁类型分布">
              <v-chart ref="pieChart" class="chart-box" :option="pieOption" autoresize />
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="Top 10 攻击 IP">
              <a-table
                :columns="topIpColumns"
                :data-source="topIps"
                row-key="ip"
                :loading="loading"
                :pagination="false"
                size="small"
              />
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 配置 -->
      <a-tab-pane key="config" tab="配置">
        <a-form :model="configForm" layout="vertical" style="max-width: 720px" :loading="configLoading">
          <a-form-item label="启用 WAF">
            <a-switch v-model:checked="configForm.enabled" />
          </a-form-item>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="拦截模式">
                <a-select v-model:value="configForm.mode">
                  <a-select-option value="block">阻断</a-select-option>
                  <a-select-option value="monitor">仅观察</a-select-option>
                  <a-select-option value="captcha">人机验证</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="检测级别">
                <a-select v-model:value="configForm.detection_level">
                  <a-select-option value="low">低</a-select-option>
                  <a-select-option value="medium">中</a-select-option>
                  <a-select-option value="high">高</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="封锁时长（秒）">
                <a-input-number v-model:value="configForm.block_duration" :min="0" style="width: 100%" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-divider>防护规则</a-divider>
          <a-form-item label="SQL 注入防护"><a-switch v-model:checked="configForm.sql_injection" /></a-form-item>
          <a-form-item label="XSS 跨站脚本防护"><a-switch v-model:checked="configForm.xss" /></a-form-item>
          <a-form-item label="路径遍历防护"><a-switch v-model:checked="configForm.path_traversal" /></a-form-item>
          <a-form-item label="命令注入防护"><a-switch v-model:checked="configForm.command_injection" /></a-form-item>
          <a-form-item label="CC 攻击防护"><a-switch v-model:checked="configForm.cc_protection" /></a-form-item>
          <a-form-item label="扫描器拦截"><a-switch v-model:checked="configForm.scanner_block" /></a-form-item>
          <a-divider>其他</a-divider>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="速率限制（次/秒）">
                <a-input-number v-model:value="configForm.rate_limit" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="日志保留（天）">
                <a-input-number v-model:value="configForm.log_retention" :min="1" style="width: 100%" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="全局白名单（每行一个IP/CIDR）">
            <a-textarea v-model:value="configForm.whitelist" :rows="4" placeholder="192.168.1.1&#10;10.0.0.0/8" />
          </a-form-item>
          <a-form-item>
            <a-button type="primary" @click="saveConfig">保存配置</a-button>
          </a-form-item>
        </a-form>
      </a-tab-pane>

      <!-- 日志 -->
      <a-tab-pane key="logs" tab="拦截日志">
        <div class="page-toolbar">
          <div class="left">
            <a-input v-model:value="logQuery.domain" placeholder="域名" allow-clear style="width: 180px" />
            <a-input v-model:value="logQuery.ip" placeholder="客户端IP" allow-clear style="width: 140px" />
            <a-select
              v-model:value="logQuery.threat_type"
              placeholder="威胁类型"
              allow-clear
              style="width: 150px"
            >
              <a-select-option value="sql_injection">SQL注入</a-select-option>
              <a-select-option value="xss">XSS</a-select-option>
              <a-select-option value="path_traversal">路径遍历</a-select-option>
              <a-select-option value="command_injection">命令注入</a-select-option>
              <a-select-option value="cc">CC攻击</a-select-option>
              <a-select-option value="scanner">扫描器</a-select-option>
            </a-select>
            <a-date-picker v-model:value="logQuery.start_time" placeholder="开始时间" show-time />
            <a-date-picker v-model:value="logQuery.end_time" placeholder="结束时间" show-time />
          </div>
          <div class="right">
            <a-button @click="resetLogQuery">重置</a-button>
            <a-button type="primary" @click="logQuery.page = 1; loadLogs()">查询</a-button>
          </div>
        </div>
        <a-table
          :columns="logColumns"
          :data-source="logs"
          :loading="loading"
          row-key="id"
          :pagination="{ current: logQuery.page, pageSize: logQuery.page_size, total: logTotal, showSizeChanger: true }"
          @change="(p) => { logQuery.page = p.current; logQuery.page_size = p.pageSize; loadLogs() }"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'action'">
              <a-tag :color="record.action === 'block' ? 'red' : record.action === 'captcha' ? 'orange' : 'blue'">
                {{ record.action === 'block' ? '阻断' : record.action === 'captcha' ? '人机验证' : '观察' }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 攻击分析 -->
      <a-tab-pane key="analysis" tab="攻击分析">
        <div class="page-toolbar">
          <span></span>
          <a-radio-group v-model:value="analysisRange" @change="loadAnalysis">
            <a-radio-button value="1d">今日</a-radio-button>
            <a-radio-button value="7d">近7天</a-radio-button>
            <a-radio-button value="30d">近30天</a-radio-button>
          </a-radio-group>
        </div>
        <a-card title="威胁类型趋势">
          <v-chart ref="trendChart" class="chart-box" :option="trendOption" autoresize />
        </a-card>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>
