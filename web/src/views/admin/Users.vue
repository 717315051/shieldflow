<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { adminUserApi } from '../../api'

const loading = ref(false)
const dataSource = ref([])
const total = ref(0)
const query = reactive({ keyword: '', status: undefined, page: 1, page_size: 10 })

const columns = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: '用户名', dataIndex: 'username' },
  { title: '邮箱', dataIndex: 'email' },
  { title: '手机', dataIndex: 'phone' },
  { title: '角色', dataIndex: 'role' },
  { title: '余额', dataIndex: 'balance' },
  { title: '状态', dataIndex: 'status' },
  { title: '注册时间', dataIndex: 'created_at' },
  { title: '操作', key: 'action', width: 180, fixed: 'right' },
]

async function loadData() {
  loading.value = true
  try {
    const res = await adminUserApi.list(query)
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

async function toggleStatus(record) {
  const next = record.status === 'active' ? 'disabled' : 'active'
  await adminUserApi.update(record.id, { status: next })
  message.success('操作成功')
  loadData()
}

async function handleDelete(id) {
  Modal.confirm({
    title: '确认删除该用户?',
    onOk: async () => {
      await adminUserApi.delete(id)
      message.success('删除成功')
      loadData()
    },
  })
}

// 添加/编辑
const modalVisible = ref(false)
const editingId = ref(null)
const formRef = ref()
const form = reactive({
  username: '',
  email: '',
  phone: '',
  password: '',
  role: 'user',
  balance: 0,
  status: 'active',
})

const rules = {
  username: [{ required: true, message: '请输入用户名' }],
  email: [
    { required: true, message: '请输入邮箱' },
    { type: 'email', message: '邮箱格式不正确' },
  ],
  phone: [{ pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确' }],
}

function openAdd() {
  editingId.value = null
  Object.assign(form, {
    username: '', email: '', phone: '', password: '', role: 'user', balance: 0, status: 'active',
  })
  modalVisible.value = true
}

function openEdit(record) {
  editingId.value = record.id
  Object.assign(form, { ...record, password: '' })
  modalVisible.value = true
}

async function submit() {
  await formRef.value.validate()
  if (editingId.value) {
    const data = { ...form }
    if (!data.password) delete data.password
    await adminUserApi.update(editingId.value, data)
  } else {
    if (!form.password) return message.error('请输入密码')
    await adminUserApi.create({ ...form })
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
      <h2 style="margin: 0">用户管理</h2>
      <a-space>
        <a-button @click="loadData"><ReloadOutlined /> 刷新</a-button>
        <a-button type="primary" @click="openAdd"><PlusOutlined /> 添加用户</a-button>
      </a-space>
    </div>

    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline" @submit.prevent="handleSearch">
        <a-form-item label="关键词">
          <a-input v-model:value="query.keyword" allow-clear @pressEnter="handleSearch" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="query.status" allow-clear placeholder="全部" style="width: 120px">
            <a-select-option value="active">正常</a-select-option>
            <a-select-option value="disabled">禁用</a-select-option>
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
        <template v-if="column.dataIndex === 'role'">
          <a-tag :color="record.role === 'admin' ? 'red' : 'blue'">
            {{ record.role === 'admin' ? '管理员' : '普通用户' }}
          </a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'default'">
            {{ record.status === 'active' ? '正常' : '禁用' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
            <a-button type="link" size="small" @click="toggleStatus(record)">
              {{ record.status === 'active' ? '禁用' : '启用' }}
            </a-button>
            <a-button type="link" danger size="small" @click="handleDelete(record.id)">删除</a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑用户' : '添加用户'" @ok="submit">
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-form-item label="用户名" name="username">
          <a-input v-model:value="form.username" :disabled="!!editingId" />
        </a-form-item>
        <a-form-item label="邮箱" name="email">
          <a-input v-model:value="form.email" />
        </a-form-item>
        <a-form-item label="手机" name="phone">
          <a-input v-model:value="form.phone" />
        </a-form-item>
        <a-form-item label="密码" :help="editingId ? '留空表示不修改' : ''">
          <a-input-password v-model:value="form.password" />
        </a-form-item>
        <a-form-item label="角色">
          <a-select v-model:value="form.role">
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="admin">管理员</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="余额">
          <a-input-number v-model:value="form.balance" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status">
            <a-select-option value="active">正常</a-select-option>
            <a-select-option value="disabled">禁用</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
