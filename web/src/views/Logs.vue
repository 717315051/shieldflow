<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { ScatterChart, EffectScatterChart } from 'echarts/charts'
import { GeoComponent, TooltipComponent, VisualMapComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, ScatterChart, EffectScatterChart, GeoComponent, TooltipComponent, VisualMapComponent])

import { logApi } from '../api'

const activeTab = ref('access')
const loading = ref(false)

const query = reactive({
  domain: '',
  ip: '',
  status: undefined,
  start_time: '',
  end_time: '',
  page: 1,
  page_size: 20,
})
const dateRange = ref(null)

const dataSource = ref([])
const total = ref(0)
const mapChart = ref(null)

const accessColumns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '域名', dataIndex: 'domain' },
  { title: '客户端IP', dataIndex: 'client_ip' },
  { title: '方法', dataIndex: 'method', width: 80 },
  { title: 'URL', dataIndex: 'url', ellipsis: true },
  { title: '状态码', dataIndex: 'status', width: 80 },
  { title: '流量', dataIndex: 'size', width: 100 },
  { title: 'UA', dataIndex: 'ua', ellipsis: true },
]

const attackColumns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '域名', dataIndex: 'domain' },
  { title: '攻击IP', dataIndex: 'client_ip' },
  { title: '攻击类型', dataIndex: 'attack_type' },
  { title: '规则', dataIndex: 'rule' },
  { title: '动作', dataIndex: 'action' },
  { title: '详情', dataIndex: 'detail', ellipsis: true },
]

const layer4Columns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '前端端口', dataIndex: 'listen_port' },
  { title: '后端地址', dataIndex: 'backend' },
  { title: '协议', dataIndex: 'protocol' },
  { title: '连接数', dataIndex: 'connections' },
  { title: '流量', dataIndex: 'traffic' },
]

const aiColumns = [
  { title: '时间', dataIndex: 'time', width: 180 },
  { title: '类型', dataIndex: 'type' },
  { title: '描述', dataIndex: 'description' },
  { title: '置信度', dataIndex: 'confidence' },
  { title: '建议', dataIndex: 'suggestion', ellipsis: true },
]

const columns = computed(() => {
  return {
    access: accessColumns,
    attack: attackColumns,
    layer4: layer4Columns,
    ai: aiColumns,
  }[activeTab.value]
})

async function loadData() {
  loading.value = true
  try {
    const fn = {
      access: logApi.access,
      attack: logApi.attack,
      layer4: logApi.layer4,
      ai: logApi.ai,
    }[activeTab.value]
    const res = await fn(query)
    dataSource.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  query.page = 1
  loadData()
}

function handleReset() {
  Object.assign(query, {
    domain: '', ip: '', status: undefined, start_time: '', end_time: '', page: 1,
  })
  loadData()
}

function onTabChange() {
  query.page = 1
  loadData()
}

function handleTableChange(p) {
  query.page = p.current
  query.page_size = p.pageSize
  loadData()
}

async function handleExport() {
  try {
    const res = await logApi.export({ ...query, type: activeTab.value })
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = `${activeTab.value}_logs.csv`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch {
    message.error('导出失败')
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">日志管理</h2>
      <a-space>
        <a-button @click="loadData"><ReloadOutlined /> 刷新</a-button>
        <a-button @click="handleExport"><DownloadOutlined /> 导出</a-button>
      </a-space>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="access" tab="访问日志" />
      <a-tab-pane key="attack" tab="攻击日志" />
      <a-tab-pane key="layer4" tab="四层日志" />
      <a-tab-pane key="ai" tab="AI日志" />
    </a-tabs>

    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline" @submit.prevent="handleSearch">
        <a-form-item label="域名">
          <a-input v-model:value="query.domain" allow-clear style="width: 180px" />
        </a-form-item>
        <a-form-item v-if="activeTab !== 'layer4'" label="IP">
          <a-input v-model:value="query.ip" allow-clear style="width: 140px" />
        </a-form-item>
        <a-form-item label="时间">
          <a-range-picker
            v-model:value="dateRange"
            show-time
            format="YYYY-MM-DD HH:mm"
          />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <a-table
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      row-key="id"
      :scroll="{ x: 1200 }"
      :pagination="{
        current: query.page,
        pageSize: query.page_size,
        total,
        showSizeChanger: true,
        showTotal: (t) => `共 ${t} 条`,
      }"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="record.status < 400 ? 'green' : 'red'">{{ record.status }}</a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'action'">
          <a-tag color="orange">{{ record.action }}</a-tag>
        </template>
      </template>
    </a-table>
  </div>
</template>
