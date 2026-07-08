<template>
  <div class="backup-page">
    <a-card title="数据备份">
      <template #extra>
        <a-button type="primary" @click="createBackup" :loading="creating">
          <template #icon><PlusOutlined /></template>
          创建备份
        </a-button>
      </template>

      <a-table :columns="columns" :dataSource="backups" :pagination="pagination" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'size'">
            {{ formatSize(record.size) }}
          </template>
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'completed' ? 'green' : 'orange'">
              {{ record.status === 'completed' ? '完成' : '进行中' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-button size="small" @click="downloadBackup(record)">下载</a-button>
            <a-popconfirm title="确定要恢复此备份吗？" @confirm="restoreBackup(record)">
              <a-button size="small" type="primary" danger style="margin-left: 8px">恢复</a-button>
            </a-popconfirm>
            <a-popconfirm title="确定删除？" @confirm="deleteBackup(record)">
              <a-button size="small" danger style="margin-left: 8px">删除</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-card title="自动备份设置" style="margin-top: 16px">
      <a-form :model="autoBackup" layout="inline">
        <a-form-item label="启用自动备份">
          <a-switch v-model:checked="autoBackup.enabled" />
        </a-form-item>
        <a-form-item label="备份频率">
          <a-select v-model:value="autoBackup.frequency" style="width: 120px">
            <a-select-option value="daily">每天</a-select-option>
            <a-select-option value="weekly">每周</a-select-option>
            <a-select-option value="monthly">每月</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="保留份数">
          <a-input-number v-model:value="autoBackup.keep_count" :min="1" :max="30" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="saveAutoBackup">保存</a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { adminSystemApi, adminBackupApi } from '@/api'

const creating = ref(false)
const backups = ref([])
const autoBackup = ref({ enabled: false, frequency: 'daily', keep_count: 7 })

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '文件名', dataIndex: 'filename', key: 'filename' },
  { title: '大小', key: 'size' },
  { title: '状态', key: 'status' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action', width: 240 },
]

const pagination = ref({ current: 1, pageSize: 20, total: 0 })

const formatSize = (bytes) => {
  if (!bytes) return '0 B'
  // 后端 size 可能是字符串（如 "8.50 MB"）或数字，统一用 parseFloat
  let n = parseFloat(bytes)
  if (isNaN(n) || n === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return n.toFixed(2) + ' ' + units[i]
}

const fetchBackups = async () => {
  try {
    const r = await adminBackupApi.list()
    backups.value = r.data?.list || []
    pagination.value.total = r.data?.total || 0
  } catch(e) { message.error('获取备份列表失败') }
}

const createBackup = async () => {
  creating.value = true
  try {
    await adminBackupApi.create()
    message.success('备份任务已创建')
    fetchBackups()
  } catch(e) { message.error('创建备份失败') }
  creating.value = false
}

const downloadBackup = (record) => {
  window.open(adminBackupApi.downloadUrl(record.id), '_blank')
}

const restoreBackup = async (record) => {
  try {
    await adminBackupApi.restore(record.id)
    message.success('恢复任务已启动')
  } catch(e) { message.error('恢复失败') }
}

const deleteBackup = async (record) => {
  try {
    await adminBackupApi.delete(record.id)
    message.success('已删除')
    fetchBackups()
  } catch(e) { message.error('删除失败') }
}

const saveAutoBackup = async () => {
  try {
    await adminSystemApi.updateSettings({ auto_backup: autoBackup.value })
    message.success('设置已保存')
  } catch(e) { message.error('保存失败') }
}

onMounted(fetchBackups)
</script>
