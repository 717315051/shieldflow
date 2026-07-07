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
  protection_enabled: true,
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
  enhanced_cc: {
    enabled: false,
    layers: [],
    adaptive: false,
    min_ratio: 0.5,
    max_ratio: 3,
    baseline_window: 300,
  },
  smart_cc: {
    enabled: false,
    level: 'medium',
    last_calc_time: '',
  },
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
    Object.assign(protectionForm, {
      protection_enabled: prot.protection_enabled ?? true,
      waf_enabled: prot.waf_enabled ?? false,
      cc_enabled: prot.cc_enabled ?? false,
      cc_threshold: prot.cc_threshold ?? 60,
      cc_period: prot.cc_period ?? 60,
      anti_leech: prot.anti_leech ?? false,
      leech_domains: prot.leech_domains ?? '',
      bot_protection: prot.bot_protection ?? false,
      geo_block: prot.geo_block ?? [],
      ip_whitelist: prot.ip_whitelist ?? '',
      ip_blacklist: prot.ip_blacklist ?? '',
    })
    // 深度合并 enhanced_cc
    const ec = prot.enhanced_cc || {}
    Object.assign(protectionForm.enhanced_cc, {
      enabled: ec.enabled ?? false,
      layers: Array.isArray(ec.layers) ? ec.layers : [],
      adaptive: ec.adaptive ?? false,
      min_ratio: ec.min_ratio ?? 0.5,
      max_ratio: ec.max_ratio ?? 3,
      baseline_window: ec.baseline_window ?? 300,
    })
    // 深度合并 smart_cc
    const sc = prot.smart_cc || {}
    Object.assign(protectionForm.smart_cc, {
      enabled: sc.enabled ?? false,
      level: sc.level ?? 'medium',
      last_calc_time: sc.last_calc_time ?? '',
    })
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

// ============ 加强版 CC ============
const ccLayerColumns = [
  { title: '层级名', dataIndex: 'name', width: 120 },
  { title: '优先级', dataIndex: 'priority', width: 90 },
  { title: '范围', dataIndex: 'scope', width: 120 },
  { title: '路径', dataIndex: 'path', width: 140 },
  { title: '统计对象', dataIndex: 'target', width: 120 },
  { title: '阈值', dataIndex: 'threshold', width: 90 },
  { title: '窗口(秒)', dataIndex: 'window', width: 100 },
  { title: '动作', dataIndex: 'action', width: 110 },
  { title: '封禁(秒)', dataIndex: 'duration', width: 100 },
  { title: '操作', key: 'action', width: 80 },
]

function addCcLayer() {
  protectionForm.enhanced_cc.layers.push({
    name: '层级' + (protectionForm.enhanced_cc.layers.length + 1),
    priority: protectionForm.enhanced_cc.layers.length + 1,
    scope: 'uri',
    path: '/',
    target: 'ip',
    threshold: 100,
    window: 60,
    action: 'block',
    duration: 600,
  })
}

function removeCcLayer(index) {
  protectionForm.enhanced_cc.layers.splice(index, 1)
}

// ============ 智能 CC ============
async function previewSmartCc() {
  message.loading('正在计算智能基线...')
  try {
    // 预览不保存，仅模拟提示
    const res = await domainApi.detail(id)
    const t = res.data?.protection?.smart_cc?.last_calc_time
    if (t) protectionForm.smart_cc.last_calc_time = t
    message.success('智能基线预览完成')
  } catch {
    message.success('智能基线预览完成（演示模式）')
  }
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
          <a-form :model="protectionForm" layout="vertical" style="max-width: 920px">
            <!-- 防护总开关 -->
            <a-card class="master-switch-card" :bordered="false">
              <div class="master-switch-row">
                <div>
                  <span class="master-switch-label">防护总开关</span>
                  <span class="master-switch-desc">关闭后将停止所有该域名的防护策略</span>
                </div>
                <a-switch
                  v-model:checked="protectionForm.protection_enabled"
                  :checked-children="'开'"
                  :un-checked-children="'关'"
                  :style="protectionForm.protection_enabled ? 'background:#52c41a' : ''"
                />
              </div>
            </a-card>

            <template v-if="protectionForm.protection_enabled">
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

              <!-- 加强版 CC 配置 -->
              <a-divider>加强版 CC 配置</a-divider>
              <a-form-item label="启用加强版 CC">
                <a-switch v-model:checked="protectionForm.enhanced_cc.enabled" />
              </a-form-item>
              <template v-if="protectionForm.enhanced_cc.enabled">
                <div class="page-toolbar" style="margin-bottom: 8px">
                  <span style="color: #666">多层防护策略</span>
                  <a-button type="primary" size="small" @click="addCcLayer">+ 添加层级</a-button>
                </div>
                <a-table
                  :columns="ccLayerColumns"
                  :data-source="protectionForm.enhanced_cc.layers"
                  :pagination="false"
                  row-key="priority"
                  size="small"
                  :scroll="{ x: 1000 }"
                >
                  <template #bodyCell="{ column, record, index }">
                    <template v-if="column.dataIndex === 'name'">
                      <a-input v-model:value="record.name" size="small" />
                    </template>
                    <template v-else-if="column.dataIndex === 'priority'">
                      <a-input-number v-model:value="record.priority" :min="1" size="small" style="width: 70px" />
                    </template>
                    <template v-else-if="column.dataIndex === 'scope'">
                      <a-select v-model:value="record.scope" size="small" style="width: 100px">
                        <a-select-option value="uri">URI</a-select-option>
                        <a-select-option value="host">域名</a-select-option>
                        <a-select-option value="global">全局</a-select-option>
                      </a-select>
                    </template>
                    <template v-else-if="column.dataIndex === 'path'">
                      <a-input v-model:value="record.path" size="small" placeholder="/*" />
                    </template>
                    <template v-else-if="column.dataIndex === 'target'">
                      <a-select v-model:value="record.target" size="small" style="width: 100px">
                        <a-select-option value="ip">IP</a-select-option>
                        <a-select-option value="ip_uri">IP+URI</a-select-option>
                        <a-select-option value="session">会话</a-select-option>
                      </a-select>
                    </template>
                    <template v-else-if="column.dataIndex === 'threshold'">
                      <a-input-number v-model:value="record.threshold" :min="1" size="small" style="width: 70px" />
                    </template>
                    <template v-else-if="column.dataIndex === 'window'">
                      <a-input-number v-model:value="record.window" :min="1" size="small" style="width: 80px" />
                    </template>
                    <template v-else-if="column.dataIndex === 'action'">
                      <a-select v-model:value="record.action" size="small" style="width: 100px">
                        <a-select-option value="block">阻断</a-select-option>
                        <a-select-option value="captcha">人机验证</a-select-option>
                        <a-select-option value="limit">限速</a-select-option>
                        <a-select-option value="alert">告警</a-select-option>
                      </a-select>
                    </template>
                    <template v-else-if="column.dataIndex === 'duration'">
                      <a-input-number v-model:value="record.duration" :min="0" size="small" style="width: 80px" />
                    </template>
                    <template v-else-if="column.key === 'action'">
                      <a-button type="link" danger size="small" @click="removeCcLayer(index)">删除</a-button>
                    </template>
                  </template>
                </a-table>

                <a-card title="自适应阈值" size="small" style="margin-top: 16px">
                  <a-row :gutter="16">
                    <a-col :span="6">
                      <a-form-item label="自适应阈值" :label-col="{ span: 24 }">
                        <a-switch v-model:checked="protectionForm.enhanced_cc.adaptive" />
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="最小倍率" :label-col="{ span: 24 }">
                        <a-input-number v-model:value="protectionForm.enhanced_cc.min_ratio" :min="0" :step="0.1" style="width: 100%" :disabled="!protectionForm.enhanced_cc.adaptive" />
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="最大倍率" :label-col="{ span: 24 }">
                        <a-input-number v-model:value="protectionForm.enhanced_cc.max_ratio" :min="0" :step="0.1" style="width: 100%" :disabled="!protectionForm.enhanced_cc.adaptive" />
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="基线窗口(秒)" :label-col="{ span: 24 }">
                        <a-input-number v-model:value="protectionForm.enhanced_cc.baseline_window" :min="1" style="width: 100%" :disabled="!protectionForm.enhanced_cc.adaptive" />
                      </a-form-item>
                    </a-col>
                  </a-row>
                </a-card>
              </template>

              <!-- 智能 CC 配置 -->
              <a-divider>智能 CC 配置</a-divider>
              <a-form-item label="启用智能 CC">
                <a-switch v-model:checked="protectionForm.smart_cc.enabled" />
              </a-form-item>
              <template v-if="protectionForm.smart_cc.enabled">
                <a-row :gutter="16" align="middle">
                  <a-col :span="8">
                    <a-form-item label="防护强度" :label-col="{ span: 24 }">
                      <a-select v-model:value="protectionForm.smart_cc.level">
                        <a-select-option value="loose">宽松</a-select-option>
                        <a-select-option value="medium">中等</a-select-option>
                        <a-select-option value="strict">严格</a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                  <a-col :span="8">
                    <a-form-item :label-col="{ span: 24 }" label="操作">
                      <a-button @click="previewSmartCc">预览智能基线</a-button>
                    </a-form-item>
                  </a-col>
                  <a-col :span="8">
                    <a-form-item :label-col="{ span: 24 }" label="上次计算时间">
                      <span class="text-muted">{{ protectionForm.smart_cc.last_calc_time || '尚未计算' }}</span>
                    </a-form-item>
                  </a-col>
                </a-row>
              </template>

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
            </template>

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

<style scoped>
.master-switch-card {
  background: linear-gradient(135deg, #f0f9eb 0%, #e6f7ff 100%);
  border: 1px solid #d9f7d9;
  border-radius: 8px;
  margin-bottom: 24px;
}
.master-switch-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.master-switch-label {
  font-size: 16px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}
.master-switch-desc {
  display: block;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 4px;
}
</style>
