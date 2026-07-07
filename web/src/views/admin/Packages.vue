<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { adminPackageApi } from '../../api'

const loading = ref(false)
const dataSource = ref([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined, page: 1, page_size: 10 })

const columns = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: '套餐名', dataIndex: 'name' },
  { title: '流量(GB)', dataIndex: 'traffic' },
  { title: '带宽(Mbps)', dataIndex: 'bandwidth' },
  { title: '域名数', dataIndex: 'domains' },
  { title: '价格(元/月)', dataIndex: 'price' },
  { title: '状态', dataIndex: 'status' },
  { title: '操作', key: 'action', width: 150, fixed: 'right' },
]

async function loadData() {
  loading.value = true
  try {
    const res = await adminPackageApi.list(query)
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

function handleTableChange(p) {
  query.page = p.current
  query.page_size = p.pageSize
  loadData()
}

// 添加/编辑
const modalVisible = ref(false)
const editingId = ref(null)
const formRef = ref()
const form = reactive({
  name: '',
  description: '',
  traffic: 100,
  bandwidth: 100,
  domains: 5,
  nodes: 5,
  price: 0,
  original_price: 0,
  period: 'month',
  features: '',
  status: 'active',
})

const rules = {
  name: [{ required: true, message: '请输入套餐名' }],
  traffic: [{ required: true, message: '请输入流量' }],
  bandwidth: [{ required: true, message: '请输入带宽' }],
  price: [{ required: true, message: '请输入价格' }],
}

function openAdd() {
  editingId.value = null
  Object.assign(form, {
    name: '', description: '', traffic: 100, bandwidth: 100, domains: 5, nodes: 5,
    price: 0, original_price: 0, period: 'month', features: '', status: 'active',
  })
  modalVisible.value = true
}

function openEdit(record) {
  editingId.value = record.id
  Object.assign(form, record)
  modalVisible.value = true
}

async function submit() {
  await formRef.value.validate()
  if (editingId.value) {
    await adminPackageApi.update(editingId.value, { ...form })
  } else {
    await adminPackageApi.create({ ...form })
  }
  message.success('保存成功')
  modalVisible.value = false
  loadData()
}

async function handleDelete(id) {
  Modal.confirm({
    title: '确认删除该套餐?',
    onOk: async () => {
      await adminPackageApi.delete(id)
      message.success('删除成功')
      loadData()
    },
  })
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">套餐管理</h2>
      <a-space>
        <a-button @click="loadData"><ReloadOutlined /> 刷新</a-button>
        <a-button type="primary" @click="openAdd"><PlusOutlined /> 创建套餐</a-button>
      </a-space>
    </div>

    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline" @submit.prevent="handleSearch">
        <a-form-item label="关键词">
          <a-input v-model:value="query.keyword" allow-clear @pressEnter="handleSearch" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="query.status" allow-clear placeholder="全部" style="width: 120px">
            <a-select-option value="active">上架</a-select-option>
            <a-select-option value="inactive">下架</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit">查询</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-table
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      row-key="id"
      :scroll="{ x: 1000 }"
      :pagination="{
        current: query.page,
        pageSize: query.page_size,
        total,
        showSizeChanger: true,
      }"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'default'">
            {{ record.status === 'active' ? '上架' : '下架' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
            <a-button type="link" danger size="small" @click="handleDelete(record.id)">删除</a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑套餐' : '创建套餐'" width="640px" @ok="submit">
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-form-item label="套餐名" name="name"><a-input v-model:value="form.name" /></a-form-item>
        <a-form-item label="描述"><a-input v-model:value="form.description" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="流量(GB)" name="traffic">
              <a-input-number v-model:value="form.traffic" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="带宽(Mbps)" name="bandwidth">
              <a-input-number v-model:value="form.bandwidth" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="域名数">
              <a-input-number v-model:value="form.domains" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="节点数">
              <a-input-number v-model:value="form.nodes" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="价格(元)" name="price">
              <a-input-number v-model:value="form.price" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="原价(元)">
              <a-input-number v-model:value="form.original_price" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="计费周期">
          <a-select v-model:value="form.period">
            <a-select-option value="month">月付</a-select-option>
            <a-select-option value="quarter">季付</a-select-option>
            <a-select-option value="year">年付</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="特性（逗号分隔）">
          <a-input v-model:value="form.features" placeholder="HTTP/3,WebSocket,DDoS防护" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status">
            <a-select-option value="active">上架</a-select-option>
            <a-select-option value="inactive">下架</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
