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
import request from '@/utils/request'

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
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) { bytes /= 1024; i++ }
  return bytes.toFixed(2) + ' ' + units[i]
}

const fetchBackups = async () => {
  try {
    const r = await request.get('/admin/system/backup')
    backups.value = r.data?.data?.list || []
    pagination.value.total = r.data?.data?.total || 0
  } catch(e) { message.error('获取备份列表失败') }
}

const createBackup = async () => {
  creating.value = true
  try {
    await request.post('/admin/system/backup')
    message.success('备份任务已创建')
    fetchBackups()
  } catch(e) { message.error('创建备份失败') }
  creating.value = false
}

const downloadBackup = (record) => {
  window.open(`/api/v1/admin/system/backup/${record.id}/download`, '_blank')
}

const restoreBackup = async (record) => {
  try {
    await request.post(`/admin/system/backup/${record.id}/restore`)
    message.success('恢复任务已启动')
  } catch(e) { message.error('恢复失败') }
}

const deleteBackup = async (record) => {
  try {
    await request.delete(`/admin/system/backup/${record.id}`)
    message.success('已删除')
    fetchBackups()
  } catch(e) { message.error('删除失败') }
}

const saveAutoBackup = async () => {
  try {
    await request.put('/admin/system/settings', { auto_backup: autoBackup.value })
    message.success('设置已保存')
  } catch(e) { message.error('保存失败') }
}

onMounted(fetchBackups)
</script>
