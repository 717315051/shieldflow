<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import SfPageHeader from '../components/SfPageHeader.vue'
import SfTableCard from '../components/SfTableCard.vue'
import SfStatusBadge from '../components/SfStatusBadge.vue'
import SfIcon from '../components/SfIcon.vue'
import { domainApi } from '../api'

const router = useRouter()
const loading = ref(false)
const dataSource = ref([])
const total = ref(0)
const selectedRowKeys = ref([])

const query = reactive({
  keyword: '', status: undefined, page: 1, page_size: 10,
})

const columns = [
  { title: '域名', dataIndex: 'domain', key: 'domain' },
  { title: 'CNAME', dataIndex: 'cname', key: 'cname' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 110 },
  { title: 'HTTPS', dataIndex: 'https', key: 'https', width: 110 },
  { title: '源站类型', dataIndex: 'origin_type', key: 'origin_type', width: 110 },
  { title: '今日流量', dataIndex: 'traffic', key: 'traffic', width: 120 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 180, fixed: 'right' },
]

const statusMap = {
  active: { text: '运行中', tone: 'success' },
  pending: { text: '待配置', tone: 'warning' },
  stopped: { text: '已停止', tone: 'neutral' },
  error: { text: '异常', tone: 'danger' },
}

async function loadData() {
  loading.value = true
  try {
    const res = await domainApi.list(query)
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
  query.keyword = ''
  query.status = undefined
  handleSearch()
}

function handleTableChange(pagination) {
  query.page = pagination.current
  query.page_size = pagination.pageSize
  loadData()
}

function goDetail(id) {
  router.push(`/domains/${id}`)
}

async function handleDelete(id) {
  Modal.confirm({
    title: '确认删除该域名?',
    onOk: async () => {
      await domainApi.delete(id)
      message.success('删除成功')
      loadData()
    },
  })
}

const addVisible = ref(false)
const addMode = ref('single')
const addLoading = ref(false)
const addFormRef = ref()
const addForm = reactive({
  domain: '', domains: '',
  origin_type: 'ip', origin_value: '', origin_port: 80, https: false,
})

const addRules = {
  origin_type: [{ required: true, message: '请选择源站类型' }],
  origin_value: [{ required: true, message: '请输入源站' }],
}

function openAdd() {
  addMode.value = 'single'
  Object.assign(addForm, {
    domain: '', domains: '',
    origin_type: 'ip', origin_value: '', origin_port: 80, https: false,
  })
  addVisible.value = true
}

async function submitAdd() {
  try {
    await addFormRef.value.validate()
    addLoading.value = true
    if (addMode.value === 'single') {
      await domainApi.create({
        domain: addForm.domain,
        origin_type: addForm.origin_type,
        origin_value: addForm.origin_value,
        origin_port: addForm.origin_port,
        https: addForm.https,
      })
    } else {
      const domains = addForm.domains.split('\n').map((s) => s.trim()).filter(Boolean)
      await domainApi.batchCreate({
        domains,
        origin_type: addForm.origin_type,
        origin_value: addForm.origin_value,
        origin_port: addForm.origin_port,
        https: addForm.https,
      })
    }
    message.success('添加成功')
    addVisible.value = false
    loadData()
  } finally {
    addLoading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="page-container">
    <SfPageHeader title="域名管理" sub="接入域名、CNAME、源站配置" :show-refresh="true" @refresh="loadData">
      <template #extra>
        <a-button type="primary" @click="openAdd">
          <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
          添加域名
        </a-button>
      </template>
    </SfPageHeader>

    <SfTableCard title="域名列表" sub="所有接入域名及其运行状态" :show-search="false" :show-refresh="false">
      <template #filters>
        <a-input
          v-model:value="query.keyword"
          placeholder="搜索域名"
          style="width: 220px"
          allow-clear
          @press-enter="handleSearch"
        />
        <a-select v-model:value="query.status" placeholder="状态" allow-clear style="width: 140px">
          <a-select-option value="active">运行中</a-select-option>
          <a-select-option value="pending">待配置</a-select-option>
          <a-select-option value="stopped">已停止</a-select-option>
          <a-select-option value="error">异常</a-select-option>
        </a-select>
        <a-button type="primary" @click="handleSearch">
          <template #icon><SfIcon name="SearchOutlined" :size="14" /></template>
          查询
        </a-button>
        <a-button @click="handleReset">重置</a-button>
      </template>

      <a-table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :row-key="(r) => r.id"
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
              :status="statusMap[record.status]?.tone || 'neutral'"
              :text="statusMap[record.status]?.text || record.status"
            />
          </template>
          <template v-else-if="column.key === 'https'">
            <SfStatusBadge
              :status="record.https ? 'success' : 'neutral'"
              :text="record.https ? '已开启' : '未开启'"
            />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="goDetail(record.id)">
                <template #icon><SfIcon name="SettingOutlined" :size="14" /></template>
                配置
              </a-button>
              <a-button type="link" danger size="small" @click="handleDelete(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </SfTableCard>

    <a-modal v-model:open="addVisible" title="添加域名" width="640px" :confirm-loading="addLoading" @ok="submitAdd">
      <a-form ref="addFormRef" :model="addForm" :rules="addRules" layout="vertical">
        <a-form-item label="添加方式">
          <a-radio-group v-model:value="addMode">
            <a-radio value="single">单个添加</a-radio>
            <a-radio value="batch">批量添加</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="addMode === 'single'" label="域名" name="domain" :rules="[{ required: true, message: '请输入域名' }]">
          <a-input v-model:value="addForm.domain" placeholder="example.com" />
        </a-form-item>
        <a-form-item v-else label="域名列表(每行一个)" name="domains" :rules="[{ required: true, message: '请输入域名' }]">
          <a-textarea v-model:value="addForm.domains" :rows="5" placeholder="example.com&#10;www.example.com" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="源站类型" name="origin_type">
              <a-select v-model:value="addForm.origin_type">
                <a-select-option value="ip">IP</a-select-option>
                <a-select-option value="domain">域名</a-select-option>
                <a-select-option value="oss">对象存储</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="10">
            <a-form-item label="源站地址" name="origin_value">
              <a-input v-model:value="addForm.origin_value" placeholder="源站 IP 或域名" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="端口" name="origin_port">
              <a-input-number v-model:value="addForm.origin_port" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="HTTPS">
          <a-switch v-model:checked="addForm.https" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>