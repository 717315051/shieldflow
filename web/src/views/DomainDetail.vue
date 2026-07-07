<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { domainApi } from '../api'

const route = useRoute()
const router = useRouter()
const id = route.params.id
const loading = ref(false)
const activeKey = ref('basic')
const domain = ref({})
const formRef = ref()

const basicForm = reactive({
  origin_type: 'ip',
  origin_value: '',
  origin_port: 80,
  https: false,
  ssl_cert_id: undefined,
  cache_ttl: 3600,
  cache_ignore_query: false,
  advanced: '',
  headers: [],
})

const protectionForm = reactive({
  waf_enabled: false,
  cc_enabled: false,
  cc_threshold: 60,
  cc_period: 60,
  anti_leech: false,
  leech_domains: '',
  bot_protection: false,
  geo_block: [],
  ip_whitelist: '',
  ip_blacklist: '',
})

const pagesForm = reactive({
  block_page: '',
  page_404: '',
  page_502: '',
  page_503: '',
  custom_headers: '',
})

async function loadDetail() {
  loading.value = true
  try {
    const res = await domainApi.detail(id)
    domain.value = res.data || {}
    const cfg = res.data?.config || {}
    Object.assign(basicForm, {
      origin_type: cfg.origin_type || 'ip',
      origin_value: cfg.origin_value || '',
      origin_port: cfg.origin_port || 80,
      https: cfg.https || false,
      ssl_cert_id: cfg.ssl_cert_id,
      cache_ttl: cfg.cache_ttl || 3600,
      cache_ignore_query: cfg.cache_ignore_query || false,
      advanced: cfg.advanced || '',
      headers: cfg.headers || [],
    })
    const prot = res.data?.protection || {}
    Object.assign(protectionForm, prot)
    const pages = res.data?.pages || {}
    Object.assign(pagesForm, pages)
  } finally {
    loading.value = false
  }
}

async function saveBasic() {
  await formRef.value?.validate()
  await domainApi.updateConfig(id, { ...basicForm })
  message.success('保存成功')
}

async function saveProtection() {
  await domainApi.updateProtection(id, { ...protectionForm })
  message.success('保存成功')
}

async function savePages() {
  await domainApi.updateCustomPages(id, { ...pagesForm })
  message.success('保存成功')
}

onMounted(() => {
  loadDetail()
})
</script>

<template>
  <div class="page-container">
    <a-page-header
      :title="domain.domain || '域名配置'"
      :sub-title="domain.cname"
      @back="router.back()"
    />
    <a-spin :spinning="loading">
      <a-tabs v-model:activeKey="activeKey">
        <!-- 基础配置 -->
        <a-tab-pane key="basic" tab="基础配置">
          <a-form
            ref="formRef"
            :model="basicForm"
            layout="vertical"
            style="max-width: 720px"
          >
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="源站类型" name="origin_type">
                  <a-select v-model:value="basicForm.origin_type">
                    <a-select-option value="ip">IP</a-select-option>
                    <a-select-option value="domain">域名</a-select-option>
                    <a-select-option value="oss">对象存储</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="10">
                <a-form-item
                  label="源站地址"
                  name="origin_value"
                  :rules="[{ required: true, message: '请输入源站地址' }]"
                >
                  <a-input v-model:value="basicForm.origin_value" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="源站端口" name="origin_port">
                  <a-input-number v-model:value="basicForm.origin_port" style="width: 100%" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="HTTPS">
              <a-switch v-model:checked="basicForm.https" />
            </a-form-item>
            <a-form-item v-if="basicForm.https" label="SSL证书">
              <a-select v-model:value="basicForm.ssl_cert_id" placeholder="选择证书" allow-clear>
                <a-select-option :value="1">通配符证书 *.example.com</a-select-option>
              </a-select>
            </a-form-item>
            <a-divider>缓存配置</a-divider>
            <a-form-item label="缓存时间（秒）">
              <a-input-number v-model:value="basicForm.cache_ttl" :min="0" style="width: 200px" />
            </a-form-item>
            <a-form-item label="忽略查询参数缓存">
              <a-switch v-model:checked="basicForm.cache_ignore_query" />
            </a-form-item>
            <a-divider>高级配置</a-divider>
            <a-form-item label="高级配置（JSON）">
              <a-textarea
                v-model:value="basicForm.advanced"
                :rows="6"
                placeholder='{"websocket": false, "http2": true}'
              />
            </a-form-item>
            <a-form-item label="自定义 Header">
              <div v-for="(h, i) in basicForm.headers" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px">
                <a-input v-model:value="h.key" placeholder="Header 名" />
                <a-input v-model:value="h.value" placeholder="Header 值" />
                <a-button danger @click="basicForm.headers.splice(i, 1)">删除</a-button>
              </div>
              <a-button @click="basicForm.headers.push({ key: '', value: '' })">添加 Header</a-button>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveBasic">保存基础配置</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- 防护配置 -->
        <a-tab-pane key="protection" tab="防护配置">
          <a-form :model="protectionForm" layout="vertical" style="max-width: 720px">
            <a-form-item label="WAF 防护">
              <a-switch v-model:checked="protectionForm.waf_enabled" />
            </a-form-item>
            <a-form-item label="CC 防护">
              <a-switch v-model:checked="protectionForm.cc_enabled" />
            </a-form-item>
            <a-row v-if="protectionForm.cc_enabled" :gutter="16">
              <a-col :span="12">
                <a-form-item label="CC 阈值（次）">
                  <a-input-number v-model:value="protectionForm.cc_threshold" :min="1" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="统计周期（秒）">
                  <a-input-number v-model:value="protectionForm.cc_period" :min="1" style="width: 100%" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="防盗链">
              <a-switch v-model:checked="protectionForm.anti_leech" />
            </a-form-item>
            <a-form-item v-if="protectionForm.anti_leech" label="允许的来域名（逗号分隔）">
              <a-input v-model:value="protectionForm.leech_domains" placeholder="example.com,www.example.com" />
            </a-form-item>
            <a-form-item label="Bot 防护">
              <a-switch v-model:checked="protectionForm.bot_protection" />
            </a-form-item>
            <a-form-item label="IP 白名单（每行一个）">
              <a-textarea v-model:value="protectionForm.ip_whitelist" :rows="4" />
            </a-form-item>
            <a-form-item label="IP 黑名单（每行一个）">
              <a-textarea v-model:value="protectionForm.ip_blacklist" :rows="4" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveProtection">保存防护配置</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <!-- 自定义页面 -->
        <a-tab-pane key="pages" tab="自定义页面">
          <a-form :model="pagesForm" layout="vertical" style="max-width: 720px">
            <a-form-item label="拦截页面 HTML">
              <a-textarea v-model:value="pagesForm.block_page" :rows="5" placeholder="可包含 HTML" />
            </a-form-item>
            <a-form-item label="404 页面 HTML">
              <a-textarea v-model:value="pagesForm.page_404" :rows="5" />
            </a-form-item>
            <a-form-item label="502 页面 HTML">
              <a-textarea v-model:value="pagesForm.page_502" :rows="5" />
            </a-form-item>
            <a-form-item label="503 页面 HTML">
              <a-textarea v-model:value="pagesForm.page_503" :rows="5" />
            </a-form-item>
            <a-form-item label="自定义响应头（JSON）">
              <a-textarea v-model:value="pagesForm.custom_headers" :rows="4" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="savePages">保存自定义页面</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>
      </a-tabs>
    </a-spin>
  </div>
</template>
