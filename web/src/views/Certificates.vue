<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import SfPageHeader from '../components/SfPageHeader.vue'
import SfTableCard from '../components/SfTableCard.vue'
import SfStatusBadge from '../components/SfStatusBadge.vue'
import SfIcon from '../components/SfIcon.vue'
import { certApi } from '../api'

const activeTab = ref('certs')
const loading = ref(false)

// 证书列表
const certs = ref([])
const certTotal = ref(0)
const certQuery = reactive({ keyword: '', page: 1, page_size: 10 })

const certColumns = [
  { title: '域名', dataIndex: 'domain' },
  { title: '颁发者', dataIndex: 'issuer', width: 200 },
  { title: '有效期', dataIndex: 'not_after', width: 200 },
  { title: '状态', dataIndex: 'status', width: 110, key: 'status' },
  { title: '操作', key: 'action', width: 120 },
]

async function loadCerts() {
  loading.value = true
  try {
    const res = await certApi.list(certQuery)
    certs.value = res.data?.list || []
    certTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

// 上传证书
const uploadVisible = ref(false)
const uploadForm = reactive({ name: '', domain: '', cert: '', key: '' })

async function submitUpload() {
  if (!uploadForm.cert || !uploadForm.key) {
    message.error('请粘贴证书和私钥')
    return
  }
  await certApi.upload({ ...uploadForm })
  message.success('上传成功')
  uploadVisible.value = false
  loadCerts()
}

async function deleteCert(id) {
  Modal.confirm({
    title: '确认删除该证书?',
    onOk: async () => {
      await certApi.delete(id)
      message.success('删除成功')
      loadCerts()
    },
  })
}

// ACME 账户
const acmeList = ref([])
const acmeVisible = ref(false)
const acmeForm = reactive({ email: '', directory: 'https://acme-v02.api.letsencrypt.org/directory', key_type: 'EC256' })

async function loadAcme() {
  const res = await certApi.acmeList({ page: 1, page_size: 100 })
  acmeList.value = res.data?.list || []
}

async function submitAcme() {
  if (!acmeForm.email) return message.error('请输入邮箱')
  await certApi.acmeCreate({ ...acmeForm })
  message.success('创建成功')
  acmeVisible.value = false
  loadAcme()
}

async function deleteAcme(id) {
  await certApi.acmeDelete(id)
  message.success('删除成功')
  loadAcme()
}

// DNS 账户
const dnsList = ref([])
const dnsVisible = ref(false)
const dnsForm = reactive({ provider: 'aliyun', name: '', access_key: '', secret_key: '' })

async function loadDns() {
  const res = await certApi.dnsList({ page: 1, page_size: 100 })
  dnsList.value = res.data?.list || []
}

async function submitDns() {
  if (!dnsForm.name) return message.error('请输入名称')
  await certApi.dnsCreate({ ...dnsForm })
  message.success('创建成功')
  dnsVisible.value = false
  loadDns()
}

async function deleteDns(id) {
  await certApi.dnsDelete(id)
  message.success('删除成功')
  loadDns()
}

function onTabChange(key) {
  if (key === 'acme' && acmeList.value.length === 0) loadAcme()
  if (key === 'dns' && dnsList.value.length === 0) loadDns()
}

onMounted(() => {
  loadCerts()
})
</script>

<template>
  <div class="page-container">
    <SfPageHeader title="SSL 证书" sub="证书管理、ACME 账户与 DNS 账户" :show-refresh="true" @refresh="loadCerts" />

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="certs" tab="证书列表">
        <SfTableCard title="证书列表" sub="所有 SSL/TLS 证书" :show-search="false" :show-refresh="false">
          <template #filters>
            <a-input
              v-model:value="certQuery.keyword"
              placeholder="搜索域名"
              style="width: 220px"
              allow-clear
              @press-enter="loadCerts"
            />
            <a-button type="primary" @click="uploadVisible = true">
              <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
              申请/上传证书
            </a-button>
          </template>

          <a-table
            :columns="certColumns"
            :data-source="certs"
            :loading="loading"
            row-key="id"
            :pagination="{
              current: certQuery.page,
              pageSize: certQuery.page_size,
              total: certTotal,
              showSizeChanger: true,
              showTotal: (t) => `共 ${t} 条`,
            }"
            @change="(p) => { certQuery.page = p.current; loadCerts() }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <SfStatusBadge
                  :status="record.status === 'valid' ? 'success' : (record.status === 'expiring' ? 'warning' : 'danger')"
                  :text="record.status === 'valid' ? '有效' : (record.status === 'expiring' ? '即将过期' : '已过期')"
                />
              </template>
              <template v-else-if="column.key === 'action'">
                <a-button type="link" danger size="small" @click="deleteCert(record.id)">删除</a-button>
              </template>
            </template>
          </a-table>
        </SfTableCard>
      </a-tab-pane>

      <a-tab-pane key="acme" tab="ACME 账户">
        <SfPageHeader title="ACME 账户" sub="用于自动签发证书的 ACME 账户" :show-refresh="true" @refresh="loadAcme">
          <template #extra>
            <a-button type="primary" @click="acmeVisible = true">
              <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
              添加 ACME 账户
            </a-button>
          </template>
        </SfPageHeader>
        <SfTableCard title="ACME 账户列表" :show-search="false" :show-refresh="false">
          <a-table
            :columns="[
              { title: '邮箱', dataIndex: 'email' },
              { title: '目录', dataIndex: 'directory', ellipsis: true },
              { title: '密钥类型', dataIndex: 'key_type', width: 120 },
              { title: '操作', key: 'action', width: 120 },
            ]"
            :data-source="acmeList"
            row-key="id"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" danger size="small" @click="deleteAcme(record.id)">删除</a-button>
              </template>
            </template>
          </a-table>
        </SfTableCard>
      </a-tab-pane>

      <a-tab-pane key="dns" tab="DNS 账户">
        <SfPageHeader title="DNS 账户" sub="用于 DNS 验证的 DNS 服务商账户" :show-refresh="true" @refresh="loadDns">
          <template #extra>
            <a-button type="primary" @click="dnsVisible = true">
              <template #icon><SfIcon name="PlusOutlined" :size="14" /></template>
              添加 DNS 账户
            </a-button>
          </template>
        </SfPageHeader>
        <SfTableCard title="DNS 账户列表" :show-search="false" :show-refresh="false">
          <a-table
            :columns="[
              { title: '名称', dataIndex: 'name' },
              { title: '服务商', dataIndex: 'provider', width: 140 },
              { title: '操作', key: 'action', width: 120 },
            ]"
            :data-source="dnsList"
            row-key="id"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" danger size="small" @click="deleteDns(record.id)">删除</a-button>
              </template>
            </template>
          </a-table>
        </SfTableCard>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="uploadVisible" title="上传/申请证书" width="640px" @ok="submitUpload">
      <a-form :model="uploadForm" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="uploadForm.name" />
        </a-form-item>
        <a-form-item label="域名" required>
          <a-input v-model:value="uploadForm.domain" placeholder="*.example.com" />
        </a-form-item>
        <a-form-item label="证书内容(PEM)" required>
          <a-textarea v-model:value="uploadForm.cert" :rows="6" placeholder="-----BEGIN CERTIFICATE-----" />
        </a-form-item>
        <a-form-item label="私钥内容(PEM)" required>
          <a-textarea v-model:value="uploadForm.key" :rows="6" placeholder="-----BEGIN PRIVATE KEY-----" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="acmeVisible" title="添加 ACME 账户" @ok="submitAcme">
      <a-form :model="acmeForm" layout="vertical">
        <a-form-item label="邮箱" required>
          <a-input v-model:value="acmeForm.email" />
        </a-form-item>
        <a-form-item label="ACME 目录">
          <a-input v-model:value="acmeForm.directory" />
        </a-form-item>
        <a-form-item label="密钥类型">
          <a-select v-model:value="acmeForm.key_type">
            <a-select-option value="EC256">EC256</a-select-option>
            <a-select-option value="EC384">EC384</a-select-option>
            <a-select-option value="RSA2048">RSA2048</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="dnsVisible" title="添加 DNS 账户" @ok="submitDns">
      <a-form :model="dnsForm" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="dnsForm.name" />
        </a-form-item>
        <a-form-item label="服务商">
          <a-select v-model:value="dnsForm.provider">
            <a-select-option value="aliyun">阿里云</a-select-option>
            <a-select-option value="tencent">腾讯云</a-select-option>
            <a-select-option value="cloudflare">Cloudflare</a-select-option>
            <a-select-option value="huawei">华为云</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="Access Key">
          <a-input v-model:value="dnsForm.access_key" />
        </a-form-item>
        <a-form-item label="Secret Key">
          <a-input-password v-model:value="dnsForm.secret_key" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>