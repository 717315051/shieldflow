<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { cacheApi } from '../api'

const activeTab = ref('file')
const loading = ref(false)

// 文件刷新
const fileForm = reactive({ urls: '' })
const dirForm = reactive({ urls: '' })
const prefetchForm = reactive({ urls: '' })
const submitting = ref(false)

async function refreshFile() {
  const urls = fileForm.urls.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!urls.length) return message.warning('请输入URL')
  submitting.value = true
  try {
    await cacheApi.refreshFile({ urls })
    message.success('已提交刷新任务')
    fileForm.urls = ''
    loadTasks()
  } finally {
    submitting.value = false
  }
}

async function refreshDir() {
  const urls = dirForm.urls.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!urls.length) return message.warning('请输入目录URL')
  submitting.value = true
  try {
    await cacheApi.refreshDir({ urls })
    message.success('已提交目录刷新任务')
    dirForm.urls = ''
    loadTasks()
  } finally {
    submitting.value = false
  }
}

async function prefetch() {
  const urls = prefetchForm.urls.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!urls.length) return message.warning('请输入URL')
  submitting.value = true
  try {
    await cacheApi.prefetch({ urls })
    message.success('已提交预读任务')
    prefetchForm.urls = ''
    loadTasks()
  } finally {
    submitting.value = false
  }
}

// 任务列表
const tasks = ref([])
const taskTotal = ref(0)
const taskQuery = reactive({ type: undefined, status: undefined, page: 1, page_size: 10 })

const taskColumns = [
  { title: '任务ID', dataIndex: 'id', width: 80 },
  { title: '类型', dataIndex: 'type' },
  { title: 'URL', dataIndex: 'url', ellipsis: true },
  { title: '状态', dataIndex: 'status' },
  { title: '进度', dataIndex: 'progress' },
  { title: '创建时间', dataIndex: 'created_at' },
  { title: '操作', key: 'action', width: 80 },
]

async function loadTasks() {
  loading.value = true
  try {
    const res = await cacheApi.tasks(taskQuery)
    tasks.value = res.data?.list || []
    taskTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

async function deleteTask(id) {
  Modal.confirm({
    title: '确认删除该任务?',
    onOk: async () => {
      await cacheApi.deleteTask(id)
      message.success('删除成功')
      loadTasks()
    },
  })
}

function handleTableChange(p) {
  taskQuery.page = p.current
  taskQuery.page_size = p.pageSize
  loadTasks()
}

onMounted(() => {
  loadTasks()
})
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">缓存管理</h2>
      <a-button @click="loadTasks"><ReloadOutlined /> 刷新</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="file" tab="文件刷新">
        <a-card>
          <a-form layout="vertical">
            <a-form-item label="URL列表（每行一个）">
              <a-textarea v-model:value="fileForm.urls" :rows="8" placeholder="https://example.com/a.js" />
            </a-form-item>
            <a-button type="primary" :loading="submitting" @click="refreshFile">提交刷新</a-button>
          </a-form>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="dir" tab="目录刷新">
        <a-card>
          <a-form layout="vertical">
            <a-form-item label="目录URL（每行一个）">
              <a-textarea v-model:value="dirForm.urls" :rows="8" placeholder="https://example.com/static/" />
            </a-form-item>
            <a-button type="primary" :loading="submitting" @click="refreshDir">提交刷新</a-button>
          </a-form>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="prefetch" tab="文件预读">
        <a-card>
          <a-form layout="vertical">
            <a-form-item label="URL列表（每行一个）">
              <a-textarea v-model:value="prefetchForm.urls" :rows="8" placeholder="https://example.com/a.js" />
            </a-form-item>
            <a-button type="primary" :loading="submitting" @click="prefetch">提交预读</a-button>
          </a-form>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="tasks" tab="任务列表">
        <div class="page-toolbar">
          <a-space>
            <a-select v-model:value="taskQuery.type" placeholder="类型" allow-clear style="width: 120px" @change="loadTasks">
              <a-select-option value="file">文件刷新</a-select-option>
              <a-select-option value="dir">目录刷新</a-select-option>
              <a-select-option value="prefetch">预读</a-select-option>
            </a-select>
            <a-select v-model:value="taskQuery.status" placeholder="状态" allow-clear style="width: 120px" @change="loadTasks">
              <a-select-option value="pending">等待中</a-select-option>
              <a-select-option value="processing">处理中</a-select-option>
              <a-select-option value="done">已完成</a-select-option>
              <a-select-option value="failed">失败</a-select-option>
            </a-select>
          </a-space>
        </div>
        <a-table
          :columns="taskColumns"
          :data-source="tasks"
          :loading="loading"
          row-key="id"
          :pagination="{
            current: taskQuery.page,
            pageSize: taskQuery.page_size,
            total: taskTotal,
          }"
          @change="handleTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="{ done: 'green', processing: 'blue', pending: 'orange', failed: 'red' }[record.status]">
                {{ record.status }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-button type="link" danger size="small" @click="deleteTask(record.id)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>
