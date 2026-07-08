<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { adminUserApi } from '../../api'
import SfPageHeader from '../../components/SfPageHeader.vue'
import SfTableCard from '../../components/SfTableCard.vue'
import SfStatusBadge from '../../components/SfStatusBadge.vue'

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
  <div class="admin-page">
    <SfPageHeader title="用户管理" sub="查看和管理所有注册用户" :show-refresh="true" @refresh="loadData">
      <template #extra>
        <a-button type="primary" @click="openAdd">
          <template #icon><PlusOutlined /></template>
          添加用户
        </a-button>
      </template>
    </SfPageHeader>

    <SfTableCard title="用户列表" :show-search="false" @refresh="loadData">
      <template #filters>
        <a-input
          v-model:value="query.keyword"
          placeholder="搜索用户名/邮箱/手机"
          allow-clear
          style="width: 220px"
          @press-enter="handleSearch"
        />
        <a-select
          v-model:value="query.status"
          allow-clear
          placeholder="状态"
          style="width: 120px"
          @change="handleSearch"
        >
          <a-select-option value="active">正常</a-select-option>
          <a-select-option value="disabled">禁用</a-select-option>
        </a-select>
        <a-button type="primary" @click="handleSearch">查询</a-button>
        <a-button @click="() => { query.keyword = ''; query.status = undefined; handleSearch() }">重置</a-button>
      </template>
      <a-table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :scroll="{ x: 1200 }"
        :pagination="{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }"
        row-key="id"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'role'">
            <SfStatusBadge :status="record.role === 'admin' ? 'info' : 'neutral'" :text="record.role === 'admin' ? '管理员' : '普通用户'" />
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <SfStatusBadge :status="record.status === 'active' ? 'success' : 'neutral'" :text="record.status === 'active' ? '正常' : '禁用'" />
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
    </SfTableCard>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑用户' : '添加用户'" @ok="submit" width="500">
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