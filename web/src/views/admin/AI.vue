<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { aiApi } from '../../api'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'

use([CanvasRenderer, PieChart, LineChart, BarChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

const activeTab = ref('dashboard')
const loading = ref(false)

// ============ 仪表盘 ============
const overview = reactive({
  today_calls: 0,
  today_tokens: 0,
  total_calls: 0,
  total_cost: 0,
})
const funcPieChart = ref(null)
const funcPieData = ref([])

const funcPieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { bottom: 0 },
  series: [
    {
      name: '功能调用',
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false, position: 'center' },
      emphasis: { label: { show: true, fontSize: 18, fontWeight: 'bold' } },
      labelLine: { show: false },
      data: funcPieData.value,
    },
  ],
}))

async function loadDashboard() {
  loading.value = true
  try {
    const res = await aiApi.dashboard()
    const d = res.data || {}
    Object.assign(overview, {
      today_calls: d.today_calls ?? 0,
      today_tokens: d.today_tokens ?? 0,
      total_calls: d.total_calls ?? 0,
      total_cost: d.total_cost ?? 0,
    })
    funcPieData.value = (d.func_distribution || []).map((f) => ({ name: f.name, value: f.value }))
    nextTick(() => funcPieChart.value?.setOption(funcPieOption.value, true))
  } finally {
    loading.value = false
  }
}

// ============ Token 统计 ============
const tokenChart = ref(null)
const tokenX = ref([])
const tokenSeries = ref([])
const tokenRange = ref('7d')

const tokenOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: tokenSeries.value.map((s) => s.name), bottom: 0 },
  grid: { left: 60, right: 30, bottom: 60, top: 30 },
  xAxis: { type: 'category', data: tokenX.value, boundaryGap: false },
  yAxis: { type: 'value', name: 'Token' },
  series: tokenSeries.value.map((s) => ({
    name: s.name,
    type: 'line',
    smooth: true,
    areaStyle: {},
    data: s.data,
  })),
}))

async function loadTokenStats() {
  loading.value = true
  try {
    const res = await aiApi.tokenStats({ range: tokenRange.value })
    const d = res.data || {}
    tokenX.value = d.dates || []
    tokenSeries.value = (d.series || []).map((s) => ({ name: s.name, data: s.data }))
    nextTick(() => tokenChart.value?.setOption(tokenOption.value, true))
  } finally {
    loading.value = false
  }
}

// ============ 成本分析 ============
const costChart = ref(null)
const costX = ref([])
const costData = ref([])
const costRange = ref('7d')

const costOption = computed(() => ({
  tooltip: { trigger: 'axis', formatter: '{b}<br/>{a}: ¥{c}' },
  grid: { left: 60, right: 30, bottom: 40, top: 30 },
  xAxis: { type: 'category', data: costX.value },
  yAxis: { type: 'value', name: '成本(¥)' },
  series: [
    {
      name: '成本',
      type: 'bar',
      data: costData.value,
      itemStyle: { color: '#1890ff', borderRadius: [4, 4, 0, 0] },
    },
  ],
}))

async function loadCostAnalysis() {
  loading.value = true
  try {
    const res = await aiApi.costAnalysis({ range: costRange.value })
    const d = res.data || {}
    costX.value = (d.models || []).map((m) => m.name)
    costData.value = (d.models || []).map((m) => m.cost)
    nextTick(() => costChart.value?.setOption(costOption.value, true))
  } finally {
    loading.value = false
  }
}

// ============ 模型管理 ============
const models = ref([])
const modelTotal = ref(0)
const modelQuery = reactive({ page: 1, page_size: 10 })

const modelColumns = [
  { title: '模型名称', dataIndex: 'name' },
  { title: 'Provider', dataIndex: 'provider' },
  { title: '模型标识', dataIndex: 'model_id', ellipsis: true },
  { title: '单价(¥/千Token)', dataIndex: 'price' },
  { title: '状态', dataIndex: 'status' },
  { title: '操作', key: 'action', width: 160 },
]

async function loadModels() {
  loading.value = true
  try {
    const res = await aiApi.models(modelQuery)
    models.value = res.data?.list || res.data || []
    modelTotal.value = res.data?.total ?? models.value.length
  } finally {
    loading.value = false
  }
}

const modelVisible = ref(false)
const editingModelId = ref(null)
const modelFormRef = ref()
const modelForm = reactive({
  name: '',
  provider: 'openai',
  model_id: '',
  api_key: '',
  api_base: '',
  price: 0,
  status: 'active',
})

const modelRules = {
  name: [{ required: true, message: '请输入模型名称' }],
  model_id: [{ required: true, message: '请输入模型标识' }],
}

function openAddModel() {
  editingModelId.value = null
  Object.assign(modelForm, {
    name: '',
    provider: 'openai',
    model_id: '',
    api_key: '',
    api_base: '',
    price: 0,
    status: 'active',
  })
  modelVisible.value = true
}

function openEditModel(record) {
  editingModelId.value = record.id
  Object.assign(modelForm, record)
  modelVisible.value = true
}

async function submitModel() {
  await modelFormRef.value.validate()
  if (editingModelId.value) {
    await aiApi.updateModel(editingModelId.value, { ...modelForm })
  } else {
    await aiApi.createModel({ ...modelForm })
  }
  message.success('保存成功')
  modelVisible.value = false
  loadModels()
}

async function deleteModel(id) {
  Modal.confirm({
    title: '确认删除该模型?',
    onOk: async () => {
      await aiApi.deleteModel(id)
      message.success('删除成功')
      loadModels()
    },
  })
}

function onTabChange(key) {
  if (key === 'token' && tokenX.value.length === 0) loadTokenStats()
  if (key === 'cost' && costX.value.length === 0) loadCostAnalysis()
  if (key === 'models' && models.value.length === 0) loadModels()
}

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">AI 防护管理</h2>
      <a-button @click="activeTab === 'dashboard' ? loadDashboard() : activeTab === 'token' ? loadTokenStats() : activeTab === 'cost' ? loadCostAnalysis() : loadModels()">
        <ReloadOutlined /> 刷新
      </a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <!-- 仪表盘 -->
      <a-tab-pane key="dashboard" tab="仪表盘">
        <div class="stat-card-grid">
          <a-card><a-statistic title="今日调用" :value="overview.today_calls" :value-style="{ color: '#1890ff' }" /></a-card>
          <a-card><a-statistic title="今日Token" :value="overview.today_tokens" /></a-card>
          <a-card><a-statistic title="累计调用" :value="overview.total_calls" /></a-card>
          <a-card><a-statistic title="累计成本" :value="overview.total_cost" prefix="¥" :precision="2" :value-style="{ color: '#fa8c16' }" /></a-card>
        </div>
        <a-card title="功能调用占比">
          <v-chart ref="funcPieChart" class="chart-box" :option="funcPieOption" autoresize />
        </a-card>
      </a-tab-pane>

      <!-- Token 统计 -->
      <a-tab-pane key="token" tab="Token统计">
        <div class="page-toolbar">
          <span></span>
          <a-radio-group v-model:value="tokenRange" @change="loadTokenStats">
            <a-radio-button value="1d">今日</a-radio-button>
            <a-radio-button value="7d">近7天</a-radio-button>
            <a-radio-button value="30d">近30天</a-radio-button>
          </a-radio-group>
        </div>
        <a-card title="按模型 Token 使用趋势">
          <v-chart ref="tokenChart" class="chart-box" :option="tokenOption" autoresize />
        </a-card>
      </a-tab-pane>

      <!-- 成本分析 -->
      <a-tab-pane key="cost" tab="成本分析">
        <div class="page-toolbar">
          <span></span>
          <a-radio-group v-model:value="costRange" @change="loadCostAnalysis">
            <a-radio-button value="7d">近7天</a-radio-button>
            <a-radio-button value="30d">近30天</a-radio-button>
            <a-radio-button value="90d">近90天</a-radio-button>
          </a-radio-group>
        </div>
        <a-card title="按模型成本">
          <v-chart ref="costChart" class="chart-box" :option="costOption" autoresize />
        </a-card>
      </a-tab-pane>

      <!-- 模型管理 -->
      <a-tab-pane key="models" tab="模型管理">
        <div class="page-toolbar">
          <span></span>
          <a-button type="primary" @click="openAddModel"><PlusOutlined /> 添加模型</a-button>
        </div>
        <a-table
          :columns="modelColumns"
          :data-source="models"
          :loading="loading"
          row-key="id"
          :pagination="{ current: modelQuery.page, pageSize: modelQuery.page_size, total: modelTotal, showSizeChanger: true }"
          @change="(p) => { modelQuery.page = p.current; modelQuery.page_size = p.pageSize; loadModels() }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                {{ record.status === 'active' ? '启用' : '禁用' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="openEditModel(record)">编辑</a-button>
                <a-button type="link" danger size="small" @click="deleteModel(record.id)">删除</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="modelVisible" :title="editingModelId ? '编辑模型' : '添加模型'" @ok="submitModel" width="600px">
      <a-form ref="modelFormRef" :model="modelForm" :rules="modelRules" layout="vertical">
        <a-form-item label="模型名称" name="name"><a-input v-model:value="modelForm.name" placeholder="如：GPT-4o" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="Provider">
              <a-select v-model:value="modelForm.provider">
                <a-select-option value="openai">OpenAI</a-select-option>
                <a-select-option value="anthropic">Anthropic</a-select-option>
                <a-select-option value="google">Google</a-select-option>
                <a-select-option value="deepseek">DeepSeek</a-select-option>
                <a-select-option value="zhipu">智谱AI</a-select-option>
                <a-select-option value="custom">自定义</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态">
              <a-select v-model:value="modelForm.status">
                <a-select-option value="active">启用</a-select-option>
                <a-select-option value="inactive">禁用</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="模型标识" name="model_id">
          <a-input v-model:value="modelForm.model_id" placeholder="如：gpt-4o" />
        </a-form-item>
        <a-form-item label="API Key">
          <a-input-password v-model:value="modelForm.api_key" placeholder="sk-..." />
        </a-form-item>
        <a-form-item label="API Base URL">
          <a-input v-model:value="modelForm.api_base" placeholder="https://api.openai.com/v1" />
        </a-form-item>
        <a-form-item label="单价（¥/千Token）">
          <a-input-number v-model:value="modelForm.price" :min="0" :step="0.001" :precision="3" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
