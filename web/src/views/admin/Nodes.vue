<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import SfPageHeader from '../../components/SfPageHeader.vue'
import SfTableCard from '../../components/SfTableCard.vue'
import SfStatusBadge from '../../components/SfStatusBadge.vue'
import SfIcon from '../../components/SfIcon.vue'
import { adminNodeApi } from '../../api'

const activeTab = ref('nodes')
const loading = ref(false)

const dataSource = ref([])
const total = ref(0)
const query = reactive({ keyword: '', group_id: undefined, status: undefined, page: 1, page_size: 10 })

const columns = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: '名称', dataIndex: 'name' },
  { title: 'IP', dataIndex: 'ip' },
  { title: '地区', dataIndex: 'region' },
  { title: '分组', dataIndex: 'group_name' },
  { title: '版本', dataIndex: 'version' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '操作', key: 'action', width: 280, fixed: 'right' },
]

async function loadData() {
  loading.value = true
  try {
    const res = await adminNodeApi.list(query)
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

const modalVisible = ref(false)
const editingId = ref(null)
const formRef = ref()
const form = reactive({
  name: '', ip: '', region: '', group_id: undefined,
  ssh_port: 22, ssh_user: 'root', ssh_password: '', listen_port: 443,
})

const rules = {
  name: [{ required: true, message: '请输入名称' }],
  ip: [{ required: true, message: '请输入IP' }],
}

function openAdd() {
  editingId.value = null
  Object.assign(form, {
    name: '', ip: '', region: '', group_id: undefined,
    ssh_port: 22, ssh_user: 'root', ssh_password: '', listen_port: 443,
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
    await adminNodeApi.update(editingId.value, { ...form })
  } else {
    await adminNodeApi.create({ ...form })
  }
  message.success('保存成功')
  modalVisible.value = false
  loadData()
}

async function handleDelete(id) {
  Modal.confirm({
    title: '确认删除该节点?',
    onOk: async () => {
      await adminNodeApi.delete(id)
      message.success('删除成功')
      loadData()
    },
  })
}

async function install(id) {
  Modal.confirm({
    title: '确认安装节点程序?',
    content: '将远程连接节点并安装 ShieldFlow 节点程序',
    onOk: async () => {
      const hide = message.loading('安装中...', 0)
      try {
        await adminNodeApi.install(id)
        message.success('安装成功')
      } finally {
        hide()
      }
      loadData()
    },
  })
}

const upgradeVisible = ref(false)
const upgradeId = ref(null)
const upgradeForm = reactive({ version: '' })

function openUpgrade(id) {
  upgradeId.value = id
  upgradeForm.version = ''
  upgradeVisible.value = true
}

async function submitUpgrade() {
  if (!upgradeForm.version) return message.error('请输入版本号')
  await adminNodeApi.upgrade(upgradeId.value, { ...upgradeForm })
  message.success('升级请求已提交')
  upgradeVisible.value = false
  loadData()
}

const groups = ref([])
const groupVisible = ref(false)
const groupForm = reactive({ name: '', description: '' })

async function loadGroups() {
  const res = await adminNodeApi.groups({ page: 1, page_size: 100 })
  groups.value = res.data?.list || []
}

async function submitGroup() {
  if (!groupForm.name) return message.error('请输入名称')
  await adminNodeApi.createGroup({ ...groupForm })
  message.success('创建成功')
  groupVisible.value = false
  loadGroups()
}

async function deleteGroup(id) {
  await adminNodeApi.deleteGroup(id)
  message.success('删除成功')
  loadGroups()
}

function onTabChange(key) {
  if (key === 'groups' && groups.value.length === 0) loadGroups()
}

onMounted(() => {
  loadData()
  loadGroups()
})
</script>

<template>
  <div class="page-container">
    <SfPageHeader title="节点管理" sub="管理边缘节点、分组、安装升级" :show-refresh="true" @refresh="loadData">
      <template #extra>
        <a-button type="primary" @click="openAdd">
          <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
          添加节点
        </a-button>
      </template>
    </SfPageHeader>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="nodes" tab="节点列表">
        <SfTableCard title="节点列表" sub="全部边缘节点及其运行状态" :show-search="false" :show-refresh="false" @refresh="loadData">
          <template #filters>
            <a-input
              v-model:value="query.keyword"
              placeholder="搜索节点名称/IP"
              style="width: 200px"
              allow-clear
              @press-enter="handleSearch"
            />
            <a-select
              v-model:value="query.group_id"
              placeholder="分组"
              allow-clear
              style="width: 140px"
            >
              <a-select-option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</a-select-option>
            </a-select>
            <a-select
              v-model:value="query.status"
              placeholder="状态"
              allow-clear
              style="width: 120px"
            >
              <a-select-option value="online">在线</a-select-option>
              <a-select-option value="offline">离线</a-select-option>
            </a-select>
            <a-button type="primary" @click="handleSearch">
              <template #icon><SfIcon name="SearchOutlined" :size="14" /></template>
              查询
            </a-button>
          </template>

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
              <template v-if="column.key === 'status'">
                <SfStatusBadge
                  :status="record.status === 'online' ? 'success' : 'neutral'"
                  :text="record.status === 'online' ? '在线' : '离线'"
                />
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="install(record.id)">
                    <template #icon><SfIcon name="CloudUploadOutlined" :size="14" /></template>
                    安装
                  </a-button>
                  <a-button type="link" size="small" @click="openUpgrade(record.id)">
                    <template #icon><SfIcon name="ArrowUpOutlined" :size="14" /></template>
                    升级
                  </a-button>
                  <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
                  <a-button type="link" danger size="small" @click="handleDelete(record.id)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </SfTableCard>
      </a-tab-pane>

      <a-tab-pane key="groups" tab="节点分组" force-render>
        <SfPageHeader title="节点分组" sub="按区域/运营商等维度组织节点" :show-refresh="true" @refresh="loadGroups">
          <template #extra>
            <a-button type="primary" @click="groupVisible = true">
              <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
              新建分组
            </a-button>
          </template>
        </SfPageHeader>

        <SfTableCard title="分组列表" :show-search="false" :show-refresh="false">
          <a-table
            :columns="[
              { title: '名称', dataIndex: 'name' },
              { title: '描述', dataIndex: 'description' },
              { title: '节点数', dataIndex: 'node_count', width: 120 },
              { title: '操作', key: 'action', width: 120 },
            ]"
            :data-source="groups"
            row-key="id"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" danger size="small" @click="deleteGroup(record.id)">删除</a-button>
              </template>
            </template>
          </a-table>
        </SfTableCard>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑节点' : '添加节点'" width="600px" @ok="submit">
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="名称" name="name"><a-input v-model:value="form.name" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="IP" name="ip"><a-input v-model:value="form.ip" /></a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="地区"><a-input v-model:value="form.region" placeholder="如:北京" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="分组">
              <a-select v-model:value="form.group_id" allow-clear>
                <a-select-option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-divider>SSH 信息</a-divider>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="SSH端口"><a-input-number v-model:value="form.ssh_port" style="width: 100%" /></a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="SSH用户"><a-input v-model:value="form.ssh_user" /></a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="SSH密码"><a-input-password v-model:value="form.ssh_password" /></a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="监听端口"><a-input-number v-model:value="form.listen_port" style="width: 100%" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="upgradeVisible" title="节点升级" @ok="submitUpgrade">
      <a-form :model="upgradeForm" layout="vertical">
        <a-form-item label="目标版本" required>
          <a-input v-model:value="upgradeForm.version" placeholder="如 v1.2.0" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="groupVisible" title="新建分组" @ok="submitGroup">
      <a-form :model="groupForm" layout="vertical">
        <a-form-item label="名称" required><a-input v-model:value="groupForm.name" /></a-form-item>
        <a-form-item label="描述"><a-input v-model:value="groupForm.description" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>