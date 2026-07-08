<script setup>
import { ref, reactive, onMounted, computed, h } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  PlusOutlined, ReloadOutlined, WalletOutlined,
  CheckCircleFilled, GlobalOutlined, ThunderboltOutlined,
  SafetyCertificateOutlined, ClusterOutlined, ApiOutlined,
  SafetyOutlined, RocketOutlined, CrownFilled,
} from '@ant-design/icons-vue'
import { packageApi } from '../api'
import SfPageHeader from '../components/SfPageHeader.vue'
import SfTabs from '../components/SfTabs.vue'
import SfTableCard from '../components/SfTableCard.vue'
import SfStatusBadge from '../components/SfStatusBadge.vue'

const activeTab = ref('market')
const loading = ref(false)
const balance = ref(0)

// 默认 3 档套餐(后端无数据时 fallback)
const defaultPlans = [
  {
    id: 1, name: '基础版', price: 99, period: '月',
    color: 'var(--info)',
    desc: '适合个人开发者或小型项目起步',
    features: [
      { icon: GlobalOutlined, label: '100 GB / 月 流量' },
      { icon: ApiOutlined, label: '5 个域名' },
      { icon: SafetyCertificateOutlined, label: '免费 SSL 证书' },
      { icon: SafetyOutlined, label: '基础 WAF' },
      { icon: ClusterOutlined, label: '2 个节点' },
    ],
    highlighted: false,
  },
  {
    id: 2, name: '专业版', price: 499, period: '月',
    color: 'var(--brand-primary)',
    desc: '中小型企业的标准选择',
    features: [
      { icon: GlobalOutlined, label: '1 TB / 月 流量' },
      { icon: ApiOutlined, label: '30 个域名' },
      { icon: SafetyCertificateOutlined, label: 'SSL 证书 + 自动续签' },
      { icon: SafetyOutlined, label: '完整 WAF + 语义分析' },
      { icon: ClusterOutlined, label: '10 个节点' },
      { icon: ThunderboltOutlined, label: 'HTTP/3 + Brotli' },
      { icon: RocketOutlined, label: '智能缓存预热' },
    ],
    highlighted: true,
  },
  {
    id: 3, name: '企业版', price: 1999, period: '月',
    color: '#722ed1',
    desc: '面向大型企业与高流量业务',
    features: [
      { icon: GlobalOutlined, label: '10 TB / 月 流量' },
      { icon: ApiOutlined, label: '不限域名数' },
      { icon: SafetyCertificateOutlined, label: 'SSL + 多域名通配符' },
      { icon: SafetyOutlined, label: 'AI 语义 WAF + Bot 管理' },
      { icon: ClusterOutlined, label: '不限节点' },
      { icon: ThunderboltOutlined, label: 'HTTP/3 + Brotli + 自定义回源' },
      { icon: RocketOutlined, label: '专属客户经理' },
      { icon: CrownFilled, label: '7×24 SLA 保障' },
    ],
    highlighted: false,
  },
]

const plans = ref(defaultPlans)
async function loadMarket() {
  loading.value = true
  try {
    const res = await packageApi.market({ page: 1, page_size: 100 }).catch(() => null)
    const list = res?.data?.list
    if (list && list.length) plans.value = list
  } finally {
    loading.value = false
  }
}

async function loadBalance() {
  try {
    const res = await packageApi.balance().catch(() => null)
    balance.value = res?.data?.balance || 0
  } catch {}
}

async function buy(pkg) {
  Modal.confirm({
    title: `确认购买「${pkg.name}」?`,
    content: `将扣除 ¥${pkg.price} / ${pkg.period || '月'}`,
    okText: '确认购买',
    cancelText: '取消',
    onOk: async () => {
      try {
        await packageApi.buy({ package_id: pkg.id })
        message.success('购买成功')
        loadBalance()
        activeTab.value = 'mine'
        loadMine()
      } catch (e) {
        message.error('购买失败')
      }
    },
  })
}

// 我的套餐
const mine = ref([])
async function loadMine() {
  loading.value = true
  try {
    const res = await packageApi.mine({ page: 1, page_size: 100 }).catch(() => null)
    mine.value = res?.data?.list || []
  } finally {
    loading.value = false
  }
}

const mineColumns = [
  { title: '套餐名', dataIndex: 'name', width: 140 },
  { title: '到期时间', dataIndex: 'expire_at', width: 160 },
  { title: '已用流量', dataIndex: 'used_traffic', width: 110 },
  { title: '总流量', dataIndex: 'traffic', width: 110 },
  { title: '状态', dataIndex: 'status', width: 100 },
]

// 购买记录
const orders = ref([])
const orderTotal = ref(0)
const orderQuery = reactive({ page: 1, page_size: 10 })
async function loadOrders() {
  loading.value = true
  try {
    const res = await packageApi.orders(orderQuery).catch(() => null)
    orders.value = res?.data?.list || []
    orderTotal.value = res?.data?.total || 0
  } finally {
    loading.value = false
  }
}
const orderColumns = [
  { title: '订单号', dataIndex: 'order_no', width: 200 },
  { title: '套餐', dataIndex: 'package_name', width: 120 },
  { title: '金额', dataIndex: 'amount', width: 100 },
  { title: '状态', dataIndex: 'status', width: 100 },
  { title: '时间', dataIndex: 'created_at' },
]

// 充值
const rechargeVisible = ref(false)
const rechargeForm = reactive({ amount: 100, method: 'alipay' })
async function submitRecharge() {
  if (!rechargeForm.amount) return message.error('请输入金额')
  try {
    await packageApi.recharge({ ...rechargeForm }).catch(() => null)
    message.success('充值请求已提交')
    rechargeVisible.value = false
    loadBalance()
  } catch { message.error('充值失败') }
}

function onTabChange(key) {
  if (key === 'mine' && mine.value.length === 0) loadMine()
  if (key === 'orders' && orders.value.length === 0) loadOrders()
}

onMounted(() => {
  loadBalance()
  loadMarket()
})
</script>

<template>
  <div class="packages-page">
    <SfPageHeader title="套餐管理" sub="选择适合您的 CDN 套餐">
      <template #extra>
        <div class="packages-balance">
          <span class="packages-balance__label">账户余额</span>
          <span class="packages-balance__value">¥{{ balance.toFixed(2) }}</span>
        </div>
        <a-button @click="loadBalance">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="rechargeVisible = true">
          <template #icon><WalletOutlined /></template>
          充值
        </a-button>
      </template>
    </SfPageHeader>

    <SfTabs
      :tabs="[
        { label: '套餐市场', value: 'market' },
        { label: '我的套餐', value: 'mine' },
        { label: '购买记录', value: 'orders' },
      ]"
      v-model="activeTab"
      size="large"
      @change="onTabChange"
    />

    <!-- 套餐市场: 3 列卡片网格 -->
    <div v-show="activeTab === 'market'" class="plans-grid">
      <div
        v-for="p in plans"
        :key="p.id"
        :class="['plan-card', p.highlighted && 'plan-card--featured']"
      >
        <div v-if="p.highlighted" class="plan-card__badge">
          <CrownFilled :style="{ fontSize: '12px' }" />
          推荐
        </div>

        <div class="plan-card__head" :style="{ borderTopColor: p.color }">
          <h3 class="plan-card__name">{{ p.name }}</h3>
          <p class="plan-card__desc">{{ p.desc }}</p>
        </div>

        <div class="plan-card__price">
          <span class="plan-card__currency">¥</span>
          <span class="plan-card__num">{{ p.price }}</span>
          <span class="plan-card__period">/ {{ p.period || '月' }}</span>
        </div>

        <a-button
          :type="p.highlighted ? 'primary' : 'default'"
          size="large"
          block
          class="plan-card__btn"
          @click="buy(p)"
        >
          立即购买
        </a-button>

        <div class="plan-card__divider"></div>

        <ul class="plan-card__features">
          <li v-for="(f, i) in p.features" :key="i">
            <CheckCircleFilled :style="{ color: 'var(--success)', fontSize: '14px' }" />
            <component :is="f.icon" :style="{ fontSize: '14px', color: 'var(--text-secondary)' }" />
            <span>{{ f.label }}</span>
          </li>
        </ul>
      </div>
    </div>

    <!-- 我的套餐: 表格 -->
    <SfTableCard v-show="activeTab === 'mine'" title="我的套餐" :show-search="false">
      <a-table
        :columns="mineColumns"
        :data-source="mine"
        :loading="loading"
        row-key="id"
        :pagination="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'status'">
            <SfStatusBadge :status="record.status === 'active' ? 'success' : 'neutral'" :text="record.status === 'active' ? '有效' : '已过期'" />
          </template>
        </template>
      </a-table>
    </SfTableCard>

    <!-- 购买记录 -->
    <SfTableCard v-show="activeTab === 'orders'" title="购买记录" :show-search="false">
      <a-table
        :columns="orderColumns"
        :data-source="orders"
        :loading="loading"
        row-key="id"
        :pagination="{
          current: orderQuery.page,
          pageSize: orderQuery.page_size,
          total: orderTotal,
        }"
        @change="(p) => { orderQuery.page = p.current; loadOrders() }"
      />
    </SfTableCard>

    <!-- 充值弹窗 -->
    <a-modal v-model:open="rechargeVisible" title="账户充值" @ok="submitRecharge" ok-text="确认充值">
      <a-form :model="rechargeForm" layout="vertical">
        <a-form-item label="充值金额（元）" required>
          <a-input-number v-model:value="rechargeForm.amount" :min="1" :max="100000" style="width: 100%" />
        </a-form-item>
        <a-form-item label="支付方式">
          <a-radio-group v-model:value="rechargeForm.method">
            <a-radio value="alipay">支付宝</a-radio>
            <a-radio value="wechat">微信支付</a-radio>
            <a-radio value="bank">银行转账</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.packages-page { padding-bottom: var(--sp-6); }

.packages-balance {
  display: flex; align-items: baseline; gap: var(--sp-2);
  padding: 0 var(--sp-3);
  margin-right: var(--sp-2);
  border-right: 1px solid var(--border-color);
}
.packages-balance__label { font-size: var(--fs-xs); color: var(--text-tertiary); }
.packages-balance__value {
  font-size: var(--fs-lg); font-weight: 700;
  color: var(--brand-primary);
  font-variant-numeric: tabular-nums;
}

/* 套餐卡片网格 */
.plans-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-4);
  margin-top: var(--sp-2);
}

.plan-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--r-lg);
  padding: var(--sp-6) var(--sp-5);
  position: relative;
  display: flex; flex-direction: column;
  transition: border-color var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease);
}
.plan-card:hover {
  border-color: var(--brand-primary);
  transform: translateY(-2px);
}

.plan-card__badge {
  position: absolute; top: -10px; right: var(--sp-5);
  background: var(--brand-primary);
  color: #fff;
  padding: 2px 10px;
  border-radius: var(--r-md);
  font-size: var(--fs-xs);
  font-weight: 500;
  display: inline-flex; align-items: center; gap: 4px;
  letter-spacing: 0.3px;
}

.plan-card__head {
  border-top: 3px solid;
  margin: calc(var(--sp-5) * -1) calc(var(--sp-5) * -1) 0;
  padding: var(--sp-5) var(--sp-5) var(--sp-4);
  border-radius: var(--r-lg) var(--r-lg) 0 0;
}
.plan-card__name {
  font-size: var(--fs-xl);
  font-weight: 600;
  margin: 0 0 var(--sp-2);
  color: var(--text-primary);
}
.plan-card__desc {
  font-size: var(--fs-sm);
  color: var(--text-tertiary);
  margin: 0;
  line-height: 1.5;
}

.plan-card__price {
  display: flex; align-items: baseline; gap: 4px;
  padding: var(--sp-5) 0;
}
.plan-card__currency { font-size: var(--fs-md); color: var(--text-secondary); font-weight: 500; }
.plan-card__num {
  font-size: 40px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.5px;
}
.plan-card__period { font-size: var(--fs-sm); color: var(--text-tertiary); }

.plan-card__btn {
  margin-bottom: var(--sp-4);
}

.plan-card__divider {
  height: 1px;
  background: var(--border-light);
  margin: 0 calc(var(--sp-5) * -1) var(--sp-4);
}

.plan-card__features {
  list-style: none; padding: 0; margin: 0;
  display: flex; flex-direction: column; gap: var(--sp-3);
}
.plan-card__features li {
  display: flex; align-items: center; gap: var(--sp-2);
  font-size: var(--fs-sm);
  color: var(--text-secondary);
}

/* 推荐卡: 蓝边框 + 阴影增强 */
.plan-card--featured {
  border-color: var(--brand-primary);
  box-shadow: 0 0 0 1px var(--brand-primary), var(--shadow-md);
}
.plan-card--featured:hover {
  box-shadow: 0 0 0 1px var(--brand-primary), var(--shadow-lg);
}

@media (max-width: 1024px) {
  .plans-grid { grid-template-columns: 1fr; gap: var(--sp-3); }
}
</style>