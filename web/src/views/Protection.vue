<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { protectionApi } from '../api'

const activeTab = ref('templates')
const loading = ref(false)

// 模板列表
const templates = ref([])
const templateTotal = ref(0)
const tplQuery = reactive({ keyword: '', page: 1, page_size: 10 })

const tplColumns = [
  { title: '名称', dataIndex: 'name' },
  { title: 'WAF', dataIndex: 'waf' },
  { title: 'CC防护', dataIndex: 'cc' },
  { title: '限速', dataIndex: 'rate_limit' },
  { title: 'Bot', dataIndex: 'bot' },
  { title: '描述', dataIndex: 'description' },
  { title: '操作', key: 'action', width: 160 },
]

async function loadTemplates() {
  loading.value = true
  try {
    const res = await protectionApi.templates(tplQuery)
    templates.value = res.data?.list || []
    templateTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

// 模板弹窗
const tplVisible = ref(false)
const tplFormRef = ref()
const editingId = ref(null)
const tplForm = reactive({
  name: '',
  waf: false,
  cc: false,
  cc_threshold: 60,
  rate_limit: false,
  rate_value: 1000,
  bot: false,
  description: '',
})

const tplRules = { name: [{ required: true, message: '请输入名称' }] }

function openAddTpl() {
  editingId.value = null
  Object.assign(tplForm, {
    name: '', waf: false, cc: false, cc_threshold: 60, rate_limit: false, rate_value: 1000, bot: false, description: '',
  })
  tplVisible.value = true
}

function openEditTpl(record) {
  editingId.value = record.id
  Object.assign(tplForm, record)
  tplVisible.value = true
}

async function submitTpl() {
  await tplFormRef.value.validate()
  if (editingId.value) {
    await protectionApi.updateTemplate(editingId.value, { ...tplForm })
  } else {
    await protectionApi.createTemplate({ ...tplForm })
  }
  message.success('保存成功')
  tplVisible.value = false
  loadTemplates()
}

async function deleteTpl(id) {
  Modal.confirm({
    title: '确认删除该模板?',
    onOk: async () => {
      await protectionApi.deleteTemplate(id)
      message.success('删除成功')
      loadTemplates()
    },
  })
}

// 黑白名单通用
const whitelist = ref([])
const blacklist = ref([])
const wlQuery = reactive({ keyword: '', page: 1, page_size: 10 })
const blQuery = reactive({ keyword: '', page: 1, page_size: 10 })
const wlTotal = ref(0)
const blTotal = ref(0)

const ipColumns = (delFn) => [
  { title: '类型', dataIndex: 'type' },
  { title: '值', dataIndex: 'value' },
  { title: '备注', dataIndex: 'note' },
  { title: '到期时间', dataIndex: 'expire_at' },
  { title: '操作', key: 'action', width: 80, customRender: ({ record }) => '' },
]

async function loadWhitelist() {
  const res = await protectionApi.whitelist(wlQuery)
  whitelist.value = res.data?.list || []
  wlTotal.value = res.data?.total || 0
}

async function loadBlacklist() {
  const res = await protectionApi.blacklist(blQuery)
  blacklist.value = res.data?.list || []
  blTotal.value = res.data?.total || 0
}

// 添加IP
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
    await protectionApi.addWhitelist({ ...ipForm })
    loadWhitelist()
  } else {
    await protectionApi.addBlacklist({ ...ipForm })
    loadBlacklist()
  }
  message.success('添加成功')
  ipVisible.value = false
}

async function delWhitelist(id) {
  await protectionApi.delWhitelist(id)
  message.success('删除成功')
  loadWhitelist()
}

async function delBlacklist(id) {
  await protectionApi.delBlacklist(id)
  message.success('删除成功')
  loadBlacklist()
}

function onTabChange(key) {
  if (key === 'whitelist' && whitelist.value.length === 0) loadWhitelist()
  if (key === 'blacklist' && blacklist.value.length === 0) loadBlacklist()
}

onMounted(() => {
  loadTemplates()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">防护管理</h2>
      <a-button @click="loadTemplates"><ReloadOutlined /> 刷新</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="templates" tab="策略模板">
        <div class="page-toolbar">
          <a-input-search
            v-model:value="tplQuery.keyword"
            placeholder="搜索模板"
            style="width: 300px"
            @search="loadTemplates"
          />
          <a-button type="primary" @click="openAddTpl"><PlusOutlined /> 新建模板</a-button>
        </div>
        <a-table
          :columns="tplColumns"
          :data-source="templates"
          :loading="loading"
          row-key="id"
          :pagination="{
            current: tplQuery.page,
            pageSize: tplQuery.page_size,
            total: templateTotal,
          }"
          @change="(p) => { tplQuery.page = p.current; loadTemplates() }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="['waf', 'cc', 'rate_limit', 'bot'].includes(column.dataIndex)">
              <a-tag :color="record[column.dataIndex] ? 'green' : 'default'">
                {{ record[column.dataIndex] ? '开' : '关' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-space>
                <a-button type="link" size="small" @click="openEditTpl(record)">编辑</a-button>
                <a-button type="link" danger size="small" @click="deleteTpl(record.id)">删除</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="whitelist" tab="白名单">
        <div class="page-toolbar">
          <a-input-search v-model:value="wlQuery.keyword" style="width: 300px" @search="loadWhitelist" />
          <a-button type="primary" @click="openAddIp('whitelist')"><PlusOutlined /> 添加白名单</a-button>
        </div>
        <a-table
          :columns="ipColumns()"
          :data-source="whitelist"
          row-key="id"
          :pagination="{ current: wlQuery.page, pageSize: wlQuery.page_size, total: wlTotal }"
          @change="(p) => { wlQuery.page = p.current; loadWhitelist() }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button type="link" danger size="small" @click="delWhitelist(record.id)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="blacklist" tab="黑名单">
        <div class="page-toolbar">
          <a-input-search v-model:value="blQuery.keyword" style="width: 300px" @search="loadBlacklist" />
          <a-button type="primary" @click="openAddIp('blacklist')"><PlusOutlined /> 添加黑名单</a-button>
        </div>
        <a-table
          :columns="ipColumns()"
          :data-source="blacklist"
          row-key="id"
          :pagination="{ current: blQuery.page, pageSize: blQuery.page_size, total: blTotal }"
          @change="(p) => { blQuery.page = p.current; loadBlacklist() }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button type="link" danger size="small" @click="delBlacklist(record.id)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <!-- 模板弹窗 -->
    <a-modal v-model:open="tplVisible" :title="editingId ? '编辑模板' : '新建模板'" width="600px" @ok="submitTpl">
      <a-form ref="tplFormRef" :model="tplForm" :rules="tplRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="tplForm.name" />
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="tplForm.description" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="8"><a-form-item label="WAF"><a-switch v-model:checked="tplForm.waf" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="CC防护"><a-switch v-model:checked="tplForm.cc" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="Bot防护"><a-switch v-model:checked="tplForm.bot" /></a-form-item></a-col>
        </a-row>
        <a-form-item v-if="tplForm.cc" label="CC阈值">
          <a-input-number v-model:value="tplForm.cc_threshold" :min="1" />
        </a-form-item>
        <a-form-item label="限速"><a-switch v-model:checked="tplForm.rate_limit" /></a-form-item>
        <a-form-item v-if="tplForm.rate_limit" label="限速值(req/s)">
          <a-input-number v-model:value="tplForm.rate_value" :min="1" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- IP 弹窗 -->
    <a-modal v-model:open="ipVisible" :title="ipMode === 'whitelist' ? '添加白名单' : '添加黑名单'" @ok="submitIp">
      <a-form :model="ipForm" layout="vertical">
        <a-form-item label="类型">
          <a-select v-model:value="ipForm.type">
            <a-select-option value="ip">IP</a-select-option>
            <a-select-option value="cidr">CIDR</a-select-option>
            <a-select-option value="area">地区</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="值" required>
          <a-input v-model:value="ipForm.value" placeholder="1.2.3.4 或 1.2.3.0/24" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model:value="ipForm.note" />
        </a-form-item>
        <a-form-item label="到期时间">
          <a-input v-model:value="ipForm.expire_at" placeholder="留空表示永久" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
