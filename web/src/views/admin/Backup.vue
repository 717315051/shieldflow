<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import SfPageHeader from '../../components/SfPageHeader.vue'
import SfTableCard from '../../components/SfTableCard.vue'
import SfStatusBadge from '../../components/SfStatusBadge.vue'
import SfIcon from '../../components/SfIcon.vue'
import { adminSystemApi, adminBackupApi } from '@/api'

const creating = ref(false)
const backups = ref([])
const autoBackup = ref({ enabled: false, frequency: 'daily', keep_count: 7 })

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '文件名', dataIndex: 'filename', key: 'filename' },
  { title: '大小', key: 'size', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 200 },
  { title: '操作', key: 'action', width: 280 },
]

const pagination = ref({ current: 1, pageSize: 20, total: 0 })

const formatSize = (bytes) => {
  if (!bytes) return '0 B'
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

<template>
  <div class="page-container">
    <SfPageHeader title="数据备份" sub="一键备份与恢复、自动备份策略" :show-refresh="true" @refresh="fetchBackups">
      <template #extra>
        <a-button type="primary" :loading="creating" @click="createBackup">
          <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
          创建备份
        </a-button>
      </template>
    </SfPageHeader>

    <SfTableCard title="备份列表" sub="历史备份文件,可下载/恢复/删除" :show-search="false" :show-refresh="false">
      <a-table
        :columns="columns"
        :data-source="backups"
        :pagination="pagination"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'size'">
            {{ formatSize(record.size) }}
          </template>
          <template v-if="column.key === 'status'">
            <SfStatusBadge
              :status="record.status === 'completed' ? 'success' : (record.status === 'failed' ? 'danger' : 'warning')"
              :text="record.status === 'completed' ? '完成' : (record.status === 'failed' ? '失败' : '进行中')"
            />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="downloadBackup(record)">
                <template #icon><SfIcon name="DownloadOutlined" :size="14" /></template>
                下载
              </a-button>
              <a-popconfirm title="确定要恢复此备份吗?" @confirm="restoreBackup(record)">
                <a-button type="link" size="small" style="color: var(--brand-primary)">
                  <template #icon><SfIcon name="ReloadOutlined" :size="14" /></template>
                  恢复
                </a-button>
              </a-popconfirm>
              <a-popconfirm title="确定删除?" @confirm="deleteBackup(record)">
                <a-button type="link" danger size="small">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </SfTableCard>

    <SfPageHeader title="自动备份设置" sub="启用自动备份及保留策略" :show-refresh="false" style="margin-top: 24px" />

    <SfTableCard title="自动备份策略" :show-search="false" :show-refresh="false">
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
          <a-button type="primary" @click="saveAutoBackup">
            <template #icon><SfIcon name="SaveOutlined" :size="14" /></template>
            保存
          </a-button>
        </a-form-item>
      </a-form>
    </SfTableCard>
  </div>
</template>