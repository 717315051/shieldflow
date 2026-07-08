<template>
  <div class="package-usage-page">
    <a-page-header
      title="套餐用量"
      sub-title="查看每个套餐实例的资源使用情况、计费详情与续费入口"
    />

    <a-tabs v-model:active-key="activeTab" :animated="true">
      <a-tab-pane v-for="(pkg, idx) in myPackages" :key="String(pkg.id || idx)" :tab="pkg.name || `套餐${idx+1}`">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-card title="套餐信息">
              <a-descriptions :column="1" bordered size="small">
                <a-descriptions-item label="当前套餐">
                  <strong>{{ pkg.name || '-' }}</strong>
                </a-descriptions-item>
                <a-descriptions-item label="实例编号">#{{ pkg.id || '-' }}</a-descriptions-item>
                <a-descriptions-item label="套餐层级">
                  <a-tag :color="pkg.type === 'l7' ? 'blue' : 'purple'">
                    {{ pkg.type === 'l7' ? '七层' : '四层' }}
                  </a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="域名数量">
                  {{ pkg.domain_count || 0 }} / {{ pkg.domain_limit || '∞' }}
                </a-descriptions-item>
                <a-descriptions-item label="流量包数量">
                  {{ pkg.traffic_count || 0 }}
                </a-descriptions-item>
                <a-descriptions-item label="到期时间">
                  {{ pkg.expire_at || '-' }}
                </a-descriptions-item>
                <a-descriptions-item label="状态">
                  <a-tag v-if="pkg.status === 'active'" color="success">生效中</a-tag>
                  <a-tag v-else color="default">{{ pkg.status || '-' }}</a-tag>
                </a-descriptions-item>
              </a-descriptions>
              <div style="margin-top: 16px">
                <a-space>
                  <a-button type="primary" @click="renew(pkg)">
                    续费套餐
                  </a-button>
                  <a-button @click="viewDetail(pkg)">查看详情</a-button>
                </a-space>
              </div>
            </a-card>
          </a-col>

          <a-col :span="16">
            <a-card title="流量使用情况" style="margin-bottom: 16px">
              <a-row :gutter="16">
                <a-col :span="6">
                  <a-statistic
                    title="已用流量"
                    :value="(pkg.used_traffic_gb || 0)"
                    :precision="2"
                    suffix="GB"
                    :value-style="{ color: '#cf1322' }"
                  />
                </a-col>
                <a-col :span="6">
                  <a-statistic
                    title="基础流量"
                    :value="(pkg.base_traffic_gb || 0)"
                    :precision="2"
                    suffix="GB"
                  />
                </a-col>
                <a-col :span="6">
                  <a-statistic
                    title="流量包"
                    :value="(pkg.bonus_traffic_gb || 0)"
                    :precision="2"
                    suffix="GB"
                  />
                </a-col>
                <a-col :span="6">
                  <a-statistic
                    title="结转流量"
                    :value="(pkg.rollover_traffic_gb || 0)"
                    :precision="2"
                    suffix="GB"
                  />
                </a-col>
              </a-row>
              <a-progress
                :percent="usagePercent(pkg)"
                :status="usagePercent(pkg) > 90 ? 'exception' : 'normal'"
                style="margin-top: 16px"
              />
            </a-card>

            <a-card title="套餐有效期与资源">
              <a-descriptions :column="2" bordered size="small">
                <a-descriptions-item label="生效时间">
                  {{ pkg.start_at || '-' }}
                </a-descriptions-item>
                <a-descriptions-item label="到期时间">
                  {{ pkg.expire_at || '-' }}
                </a-descriptions-item>
                <a-descriptions-item label="剩余天数">
                  <a-tag :color="remainingDays(pkg) > 7 ? 'green' : 'red'">
                    {{ remainingDays(pkg) }} 天
                  </a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="请求配额">
                  {{ pkg.request_quota || 0 }} 次
                </a-descriptions-item>
                <a-descriptions-item label="带宽上限">
                  {{ pkg.bandwidth_mbps || 0 }} Mbps
                </a-descriptions-item>
                <a-descriptions-item label="可用域名数">
                  {{ (pkg.domain_limit || 0) - (pkg.domain_count || 0) }}
                </a-descriptions-item>
              </a-descriptions>
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>
      <a-tab-pane v-if="myPackages.length === 0" key="empty" tab="无套餐">
        <a-empty description="您还未购买任何套餐" />
        <a-button type="primary" @click="goMarket">前往套餐市场</a-button>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { packageApi } from '../api'

const router = useRouter()
const myPackages = ref([])
const activeTab = ref('0')
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await packageApi.myPackages()
    if (res.code === 0) {
      myPackages.value = res.data?.list || res.data || []
      if (myPackages.value.length > 0) {
        activeTab.value = String(myPackages.value[0].id || 0)
      }
    } else {
      message.error(res.message || '加载套餐失败')
    }
  } catch (e) {
    message.error('加载失败：' + (e.message || e))
  } finally {
    loading.value = false
  }
}

function usagePercent(pkg) {
  const used = pkg.used_traffic_gb || 0
  const total = (pkg.base_traffic_gb || 0) + (pkg.bonus_traffic_gb || 0)
  if (total === 0) return 0
  return Math.min(100, Math.round((used / total) * 100))
}

function remainingDays(pkg) {
  if (!pkg.expire_at) return 0
  const exp = new Date(pkg.expire_at).getTime()
  const now = Date.now()
  return Math.max(0, Math.floor((exp - now) / (1000 * 60 * 60 * 24)))
}

function renew(pkg) {
  message.info('续费功能开发中: ' + pkg.name)
}

function viewDetail(pkg) {
  router.push(`/user-packages/${pkg.id}`)
}

function goMarket() {
  router.push('/packages')
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.package-usage-page {
  max-width: 1400px;
}
</style>
