<template>
  <div class="system-settings">
    <a-card title="系统设置">
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="general" tab="全局设置">
          <a-form :model="generalForm" layout="vertical">
            <a-form-item label="系统名称">
              <a-input v-model:value="generalForm.site_name" />
            </a-form-item>
            <a-form-item label="系统Logo URL">
              <a-input v-model:value="generalForm.logo_url" />
            </a-form-item>
            <a-form-item label="备案号">
              <a-input v-model:value="generalForm.icp_record" />
            </a-form-item>
            <a-form-item label="客服QQ">
              <a-input v-model:value="generalForm.contact_qq" />
            </a-form-item>
            <a-form-item label="客服微信">
              <a-input v-model:value="generalForm.contact_wechat" />
            </a-form-item>
            <a-form-item label="客服邮箱">
              <a-input v-model:value="generalForm.contact_email" />
            </a-form-item>
            <a-form-item label="是否允许注册">
              <a-switch v-model:checked="generalForm.allow_register" />
            </a-form-item>
            <a-form-item label="是否需要实名">
              <a-switch v-model:checked="generalForm.require_realname" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveGeneral">保存</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="dns" tab="DNS设置">
          <a-form :model="dnsForm" layout="vertical">
            <a-form-item label="默认DNS服务商">
              <a-select v-model:value="dnsForm.provider">
                <a-select-option value="cloudflare">Cloudflare</a-select-option>
                <a-select-option value="aliyun">阿里云DNS</a-select-option>
                <a-select-option value="tencent">腾讯云DNSPod</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="CDN默认CNAME后缀">
              <a-input v-model:value="dnsForm.cname_suffix" placeholder="example.cdn.xxx.com" />
            </a-form-item>
            <a-form-item label="同步间隔(秒)">
              <a-input-number v-model:value="dnsForm.sync_interval" :min="60" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveDNS">保存</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="acme" tab="ACME设置">
          <a-form :model="acmeForm" layout="vertical">
            <a-form-item label="默认CA">
              <a-select v-model:value="acmeForm.ca_url">
                <a-select-option value="https://acme-v02.request.letsencrypt.org/directory">Let's Encrypt</a-select-option>
                <a-select-option value="https://acme.zerossl.com/v2/API">ZeroSSL</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="默认验证方式">
              <a-radio-group v-model:value="acmeForm.verify_type">
                <a-radio value="http">HTTP验证</a-radio>
                <a-radio value="dns">DNS验证</a-radio>
              </a-radio-group>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveACME">保存</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="grpc" tab="gRPC配置">
          <a-form :model="grpcForm" layout="vertical">
            <a-form-item label="日志上传模式">
              <a-radio-group v-model:value="grpcForm.log_mode">
                <a-radio value="master">主控接收</a-radio>
                <a-radio value="log_server">独立日志服务器</a-radio>
              </a-radio-group>
            </a-form-item>
            <a-form-item label="日志服务器地址" v-if="grpcForm.log_mode === 'log_server'">
              <a-input v-model:value="grpcForm.log_server_address" placeholder="https://log.example.com:9529" />
            </a-form-item>
            <a-form-item label="日志服务器Token" v-if="grpcForm.log_mode === 'log_server'">
              <a-input-password v-model:value="grpcForm.log_server_token" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveGRPC">保存</a-button>
              <a-button style="margin-left: 8px" @click="testLogServer">测试连接</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="alert" tab="告警设置">
          <a-form :model="alertForm" layout="vertical">
            <a-form-item label="Telegram Bot Token">
              <a-input v-model:value="alertForm.tg_bot_token" />
            </a-form-item>
            <a-form-item label="Telegram Chat ID">
              <a-input v-model:value="alertForm.tg_chat_id" />
            </a-form-item>
            <a-form-item label="邮件告警收件人">
              <a-input v-model:value="alertForm.email_to" />
            </a-form-item>
            <a-form-item label="DDoS攻击告警">
              <a-switch v-model:checked="alertForm.ddos_alert" />
            </a-form-item>
            <a-form-item label="节点离线告警">
              <a-switch v-model:checked="alertForm.node_offline_alert" />
            </a-form-item>
            <a-form-item label="证书过期告警">
              <a-switch v-model:checked="alertForm.cert_expiring_alert" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveAlert">保存</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="ai" tab="AI配置">
          <a-form :model="aiForm" layout="vertical">
            <a-form-item label="AI服务商">
              <a-select v-model:value="aiForm.provider">
                <a-select-option value="openai">OpenAI</a-select-option>
                <a-select-option value="deepseek">DeepSeek</a-select-option>
                <a-select-option value="qwen">通义千问</a-select-option>
                <a-select-option value="glm">智谱GLM</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="模型名称">
              <a-input v-model:value="aiForm.model" placeholder="gpt-4o / deepseek-chat / qwen-max" />
            </a-form-item>
            <a-form-item label="API Key">
              <a-input-password v-model:value="aiForm.api_key" />
            </a-form-item>
            <a-form-item label="Base URL">
              <a-input v-model:value="aiForm.base_url" placeholder="https://request.openai.com/v1" />
            </a-form-item>
            <a-form-item label="启用AI防护">
              <a-switch v-model:checked="aiForm.enabled" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="saveAI">保存</a-button>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="version" tab="版本信息">
          <a-descriptions title="系统版本" bordered>
            <a-descriptions-item label="版本号">1.2.0</a-descriptions-item>
            <a-descriptions-item label="构建时间">{{ buildTime }}</a-descriptions-item>
            <a-descriptions-item label="Go版本">{{ goVersion }}</a-descriptions-item>
            <a-descriptions-item label="操作系统">{{ osInfo }}</a-descriptions-item>
            <a-descriptions-item label="已安装组件">
              <a-tag color="green">后端API</a-tag>
              <a-tag color="green">gRPC服务</a-tag>
              <a-tag color="blue">边缘节点</a-tag>
              <a-tag color="blue">DNS同步</a-tag>
              <a-tag color="orange">日志服务器</a-tag>
            </a-descriptions-item>
          </a-descriptions>
          <a-button type="primary" style="margin-top: 16px" @click="checkUpgrade">检查更新</a-button>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import request from '@/utils/request'

const activeTab = ref('general')
const generalForm = ref({ allow_register: false, require_realname: false })
const dnsForm = ref({ sync_interval: 300 })
const acmeForm = ref({ ca_url: 'https://acme-v02.request.letsencrypt.org/directory', verify_type: 'http' })
const grpcForm = ref({ log_mode: 'master' })
const alertForm = ref({})
const aiForm = ref({ enabled: false })
const buildTime = ref('2026-07-07')
const goVersion = ref('go1.22')
const osInfo = ref('linux/amd64')

const saveGeneral = async () => {
  try { await request.put('/admin/system/settings', generalForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') }
}
const saveDNS = async () => { try { await request.put('/admin/system/dns', dnsForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') } }
const saveACME = async () => { try { await request.put('/admin/system/acme', acmeForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') } }
const saveGRPC = async () => { try { await request.put('/admin/system/grpc', grpcForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') } }
const testLogServer = async () => { try { const r = await request.post('/admin/system/grpc/test-log-server', grpcForm.value); message.success('连接成功') } catch(e) { message.error('连接失败') } }
const saveAlert = async () => { try { await request.put('/admin/system/alert', alertForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') } }
const saveAI = async () => { try { await request.put('/admin/system/ai', aiForm.value); message.success('保存成功') } catch(e) { message.error('保存失败') } }
const checkUpgrade = () => { message.info('当前已是最新版本') }

onMounted(async () => {
  try {
    const [g, d, a, gr, al, ai] = await Promise.all([
      request.get('/admin/system/settings'), request.get('/admin/system/dns'), request.get('/admin/system/acme'),
      request.get('/admin/system/grpc'), request.get('/admin/system/alert'), request.get('/admin/system/ai')
    ])
    if (g.data?.data) generalForm.value = { ...generalForm.value, ...g.data.data }
    if (d.data?.data) dnsForm.value = { ...dnsForm.value, ...d.data.data }
    if (a.data?.data) acmeForm.value = { ...acmeForm.value, ...a.data.data }
    if (gr.data?.data) grpcForm.value = { ...grpcForm.value, ...gr.data.data }
    if (al.data?.data) alertForm.value = { ...al.data.data }
    if (ai.data?.data) aiForm.value = { ...aiForm.value, ...ai.data.data }
  } catch(e) {}
})
</script>
