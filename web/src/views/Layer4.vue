<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { layer4Api } from '../api'

const loading = ref(false)
const dataSource = ref([])
const total = ref(0)
const query = reactive({ keyword: '', protocol: undefined, page: 1, page_size: 10 })

const columns = [
  { title: '名称', dataIndex: 'name' },
  { title: '监听端口', dataIndex: 'listen_port' },
  { title: '后端地址', dataIndex: 'backend' },
  { title: '后端端口', dataIndex: 'backend_port' },
  { title: '协议', dataIndex: 'protocol' },
  { title: '状态', dataIndex: 'status' },
  { title: '操作', key: 'action', width: 150 },
]

async function loadData() {
  loading.value = true
  try {
    const res = await layer4Api.list(query)
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

async function handleDelete(id) {
  Modal.confirm({
    title: '确认删除该转发规则?',
    onOk: async () => {
      await layer4Api.delete(id)
      message.success('删除成功')
      loadData()
    },
  })
}

// 添加/编辑
const modalVisible = ref(false)
const modalTitle = ref('添加转发')
const editingId = ref(null)
const formRef = ref()
const form = reactive({
  name: '',
  listen_port: undefined,
  backend: '',
  backend_port: undefined,
  protocol: 'tcp',
  scheduler: 'round_robin',
})

const rules = {
  name: [{ required: true, message: '请输入名称' }],
  listen_port: [{ required: true, message: '请输入监听端口' }],
  backend: [{ required: true, message: '请输入后端地址' }],
  backend_port: [{ required: true, message: '请输入后端端口' }],
}

function openAdd() {
  modalTitle.value = '添加转发'
  editingId.value = null
  Object.assign(form, {
    name: '', listen_port: undefined, backend: '', backend_port: undefined, protocol: 'tcp', scheduler: 'round_robin',
  })
  modalVisible.value = true
}

function openEdit(record) {
  modalTitle.value = '编辑转发'
  editingId.value = record.id
  Object.assign(form, record)
  modalVisible.value = true
}

async function submit() {
  await formRef.value.validate()
  if (editingId.value) {
    await layer4Api.update(editingId.value, { ...form })
  } else {
    await layer4Api.create({ ...form })
  }
  message.success('保存成功')
  modalVisible.value = false
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">四层转发</h2>
      <a-space>
        <a-button @click="loadData"><ReloadOutlined /> 刷新</a-button>
        <a-button type="primary" @click="openAdd"><PlusOutlined /> 添加转发</a-button>
      </a-space>
    </div>

    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline" @submit.prevent="handleSearch">
        <a-form-item label="关键词">
          <a-input v-model:value="query.keyword" allow-clear @pressEnter="handleSearch" />
        </a-form-item>
        <a-form-item label="协议">
          <a-select v-model:value="query.protocol" allow-clear placeholder="全部" style="width: 100px">
            <a-select-option value="tcp">TCP</a-select-option>
            <a-select-option value="udp">UDP</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit">查询</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <a-table
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      row-key="id"
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
            {{ record.status === 'active' ? '运行中' : '已停用' }}
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

    <a-modal v-model:open="modalVisible" :title="modalTitle" @ok="submit">
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item label="监听端口" name="listen_port">
          <a-input-number v-model:value="form.listen_port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="后端地址" name="backend">
          <a-input v-model:value="form.backend" placeholder="1.2.3.4 或 example.com" />
        </a-form-item>
        <a-form-item label="后端端口" name="backend_port">
          <a-input-number v-model:value="form.backend_port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="协议">
          <a-select v-model:value="form.protocol">
            <a-select-option value="tcp">TCP</a-select-option>
            <a-select-option value="udp">UDP</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="调度算法">
          <a-select v-model:value="form.scheduler">
            <a-select-option value="round_robin">轮询</a-select-option>
            <a-select-option value="least_conn">最小连接</a-select-option>
            <a-select-option value="source_hash">源地址哈希</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
