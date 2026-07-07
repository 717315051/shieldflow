<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, GaugeChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, LineChart, BarChart, GaugeChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

import { adminDdosApi } from '../../api'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'

const activeTab = ref('dashboard')
const loading = ref(false)

// 仪表盘
const overview = reactive({
  current_gbps: 0,
  peak_gbps: 0,
  total_attacks: 0,
  active_rules: 0,
  blocked_ips: 0,
})
const trendChart = ref(null)
const trendX = ref([])
const trendData = reactive({ in: [], out: [] })

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['入流量', '清洗流量'] },
  grid: { left: 50, right: 30, bottom: 50 },
  xAxis: { type: 'category', data: trendX.value },
  yAxis: { type: 'value', name: 'Gbps' },
  series: [
    { name: '入流量', type: 'line', smooth: true, areaStyle: {}, data: trendData.in, itemStyle: { color: '#ff4d4f' } },
    { name: '清洗流量', type: 'line', smooth: true, areaStyle: {}, data: trendData.out, itemStyle: { color: '#52c41a' } },
  ],
}))

async function loadDashboard() {
  loading.value = true
  try {
    const res = await adminDdosApi.dashboard()
    Object.assign(overview, res.data?.overview || {})
    const t = res.data?.trend || []
    trendX.value = t.map((i) => i.time)
    trendData.in = t.map((i) => i.in)
    trendData.out = t.map((i) => i.out)
    nextTick(() => trendChart.value?.setOption(trendOption.value, true))
  } finally {
    loading.value = false
  }
}

// 规则
const rules = ref([])
const ruleTotal = ref(0)
const ruleQuery = reactive({ keyword: '', page: 1, page_size: 10 })

const ruleColumns = [
  { title: '名称', dataIndex: 'name' },
  { title: '类型', dataIndex: 'type' },
  { title: '阈值(Gbps)', dataIndex: 'threshold' },
  { title: '动作', dataIndex: 'action' },
  { title: '状态', dataIndex: 'status' },
  { title: '操作', key: 'action', width: 160 },
]

async function loadRules() {
  loading.value = true
  try {
    const res = await adminDdosApi.rules(ruleQuery)
    rules.value = res.data?.list || []
    ruleTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const ruleVisible = ref(false)
const editingRuleId = ref(null)
const ruleFormRef = ref()
const ruleForm = reactive({
  name: '',
  type: 'syn_flood',
  threshold: 10,
  action: 'block',
  duration: 3600,
  status: 'active',
})

const ruleRules = {
  name: [{ required: true, message: '请输入名称' }],
  threshold: [{ required: true, message: '请输入阈值' }],
}

function openAddRule() {
  editingRuleId.value = null
  Object.assign(ruleForm, { name: '', type: 'syn_flood', threshold: 10, action: 'block', duration: 3600, status: 'active' })
  ruleVisible.value = true
}

function openEditRule(record) {
  editingRuleId.value = record.id
  Object.assign(ruleForm, record)
  ruleVisible.value = true
}

async function submitRule() {
  await ruleFormRef.value.validate()
  if (editingRuleId.value) {
    await adminDdosApi.updateRule(editingRuleId.value, { ...ruleForm })
  } else {
    await adminDdosApi.createRule({ ...ruleForm })
  }
  message.success('保存成功')
  ruleVisible.value = false
  loadRules()
}

async function deleteRule(id) {
  Modal.confirm({
    title: '确认删除该规则?',
    onOk: async () => {
      await adminDdosApi.deleteRule(id)
      message.success('删除成功')
      loadRules()
    },
  })
}

// 黑白名单
const whitelist = ref([])
const blacklist = ref([])

async function loadWhitelist() {
  const res = await adminDdosApi.whitelist({ page: 1, page_size: 100 })
  whitelist.value = res.data?.list || []
}

async function loadBlacklist() {
  const res = await adminDdosApi.blacklist({ page: 1, page_size: 100 })
  blacklist.value = res.data?.list || []
}

const ipVisible = ref(false)
const ipMode = ref('whitelist')
const ipForm = reactive({ type: 'ip', value: '', note: '', expire_at: '' })

function openAddIp(mode) {
  ipMode.value = mode
  Object.assign(ipForm, { type: 'ip', value: '', note: '', expire_at: '' })
  ipVisible.value = true
}

async function submitIp() {
  if (!ipForm.value) return message.error('请输入值')
  if (ipMode.value === 'whitelist') {
    await adminDdosApi.addWhitelist({ ...ipForm })
    loadWhitelist()
  } else {
    await adminDdosApi.addBlacklist({ ...ipForm })
    loadBlacklist()
  }
  message.success('添加成功')
  ipVisible.value = false
}

async function delIp(mode, id) {
  if (mode === 'whitelist') {
    await adminDdosApi.delWhitelist(id)
    loadWhitelist()
  } else {
    await adminDdosApi.delBlacklist(id)
    loadBlacklist()
  }
  message.success('删除成功')
}

// 日志
const logs = ref([])
const logTotal = ref(0)
const logQuery = reactive({ page: 1, page_size: 10 })

const logColumns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '攻击类型', dataIndex: 'attack_type' },
  { title: '源IP', dataIndex: 'src_ip' },
  { title: '目标', dataIndex: 'target' },
  { title: '峰值(Gbps)', dataIndex: 'peak' },
  { title: '动作', dataIndex: 'action' },
  { title: '状态', dataIndex: 'status' },
]

async function loadLogs() {
  loading.value = true
  try {
    const res = await adminDdosApi.logs(logQuery)
    logs.value = res.data?.list || []
    logTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function onTabChange(key) {
  if (key === 'rules' && rules.value.length === 0) loadRules()
  if (key === 'whitelist' && whitelist.value.length === 0) loadWhitelist()
  if (key === 'blacklist' && blacklist.value.length === 0) loadBlacklist()
  if (key === 'logs' && logs.value.length === 0) loadLogs()
}

const ipColumns = (mode) => [
  { title: '类型', dataIndex: 'type' },
  { title: '值', dataIndex: 'value' },
  { title: '备注', dataIndex: 'note' },
  { title: '到期时间', dataIndex: 'expire_at' },
  { title: '操作', key: 'action', width: 80 },
]

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">DDoS 防护</h2>
      <a-button @click="loadDashboard"><ReloadOutlined /> 刷新</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="dashboard" tab="仪表盘">
        <div class="stat-card-grid">
          <a-card><a-statistic title="当前流量" :value="overview.current_gbps" suffix="Gbps" :value-style="{ color: '#ff4d4f' }" /></a-card>
          <a-card><a-statistic title="峰值流量" :value="overview.peak_gbps" suffix="Gbps" /></a-card>
          <a-card><a-statistic title="攻击次数" :value="overview.total_attacks" /></a-card>
          <a-card><a-statistic title="生效规则" :value="overview.active_rules" /></a-card>
          <a-card><a-statistic title="封锁IP" :value="overview.blocked_ips" /></a-card>
        </div>
        <a-card title="流量趋势">
          <v-chart ref="trendChart" class="chart-box" :option="trendOption" autoresize />
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="rules" tab="防护规则">
        <div class="page-toolbar">
          <a-input-search v-model:value="ruleQuery.keyword" style="width: 300px" @search="loadRules" />
          <a-button type="primary" @click="openAddRule"><PlusOutlined /> 新建规则</a-button>
        </div>
        <a-table
          :columns="ruleColumns"
          :data-source="rules"
          :loading="loading"
          row-key="id"
          :pagination="{ current: ruleQuery.page, pageSize: ruleQuery.page_size, total: ruleTotal }"
          @change="(p) => { ruleQuery.page = p.current; loadRules() }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                {{ record.status === 'active' ? '启用' : '禁用' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="openEditRule(record)">编辑</a-button>
                <a-button type="link" danger size="small" @click="deleteRule(record.id)">删除</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="whitelist" tab="白名单">
        <div class="page-toolbar">
          <span></span>
          <a-button type="primary" @click="openAddIp('whitelist')"><PlusOutlined /> 添加白名单</a-button>
        </div>
        <a-table :columns="ipColumns('whitelist')" :data-source="whitelist" row-key="id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button type="link" danger size="small" @click="delIp('whitelist', record.id)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="blacklist" tab="黑名单">
        <div class="page-toolbar">
          <span></span>
          <a-button type="primary" @click="openAddIp('blacklist')"><PlusOutlined /> 添加黑名单</a-button>
        </div>
        <a-table :columns="ipColumns('blacklist')" :data-source="blacklist" row-key="id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button type="link" danger size="small" @click="delIp('blacklist', record.id)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="logs" tab="攻击日志">
        <a-table
          :columns="logColumns"
          :data-source="logs"
          :loading="loading"
          row-key="id"
          :pagination="{ current: logQuery.page, pageSize: logQuery.page_size, total: logTotal }"
          @change="(p) => { logQuery.page = p.current; loadLogs() }"
        />
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="ruleVisible" :title="editingRuleId ? '编辑规则' : '新建规则'" @ok="submitRule">
      <a-form ref="ruleFormRef" :model="ruleForm" :rules="ruleRules" layout="vertical">
        <a-form-item label="名称" name="name"><a-input v-model:value="ruleForm.name" /></a-form-item>
        <a-form-item label="攻击类型">
          <a-select v-model:value="ruleForm.type">
            <a-select-option value="syn_flood">SYN Flood</a-select-option>
            <a-select-option value="udp_flood">UDP Flood</a-select-option>
            <a-select-option value="icmp_flood">ICMP Flood</a-select-option>
            <a-select-option value="cc">CC 攻击</a-select-option>
            <a-select-option value="dns_amplification">DNS 放大</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="阈值(Gbps)" name="threshold">
          <a-input-number v-model:value="ruleForm.threshold" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="动作">
          <a-select v-model:value="ruleForm.action">
            <a-select-option value="block">阻断</a-select-option>
            <a-select-option value="clean">清洗</a-select-option>
            <a-select-option value="limit">限速</a-select-option>
            <a-select-option value="alert">告警</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="持续时间(秒)">
          <a-input-number v-model:value="ruleForm.duration" :min="1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="ruleForm.status">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="ipVisible" :title="ipMode === 'whitelist' ? '添加白名单' : '添加黑名单'" @ok="submitIp">
      <a-form :model="ipForm" layout="vertical">
        <a-form-item label="类型">
          <a-select v-model:value="ipForm.type">
            <a-select-option value="ip">IP</a-select-option>
            <a-select-option value="cidr">CIDR</a-select-option>
            <a-select-option value="asn">ASN</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="值" required><a-input v-model:value="ipForm.value" /></a-form-item>
        <a-form-item label="备注"><a-input v-model:value="ipForm.note" /></a-form-item>
        <a-form-item label="到期时间"><a-input v-model:value="ipForm.expire_at" placeholder="留空表示永久" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
