<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined, WalletOutlined } from '@ant-design/icons-vue'
import { packageApi } from '../api'

const activeTab = ref('market')
const loading = ref(false)

// 余额
const balance = ref(0)

async function loadBalance() {
  try {
    const res = await packageApi.balance()
    balance.value = res.data?.balance || 0
  } catch {}
}

// 套餐市场
const market = ref([])
async function loadMarket() {
  loading.value = true
  try {
    const res = await packageApi.market({ page: 1, page_size: 100 })
    market.value = res.data?.list || []
  } finally {
    loading.value = false
  }
}

const marketColumns = [
  { title: '套餐名', dataIndex: 'name' },
  { title: '流量', dataIndex: 'traffic' },
  { title: '带宽', dataIndex: 'bandwidth' },
  { title: '域名数', dataIndex: 'domains' },
  { title: '价格(元/月)', dataIndex: 'price' },
  { title: '操作', key: 'action', width: 100 },
]

async function buy(pkg) {
  Modal.confirm({
    title: `确认购买「${pkg.name}」?`,
    content: `将扣除 ${pkg.price} 元`,
    onOk: async () => {
      await packageApi.buy({ package_id: pkg.id })
      message.success('购买成功')
      loadBalance()
      activeTab.value = 'mine'
      loadMine()
    },
  })
}

// 我的套餐
const mine = ref([])
async function loadMine() {
  loading.value = true
  try {
    const res = await packageApi.mine({ page: 1, page_size: 100 })
    mine.value = res.data?.list || []
  } finally {
    loading.value = false
  }
}

const mineColumns = [
  { title: '套餐名', dataIndex: 'name' },
  { title: '到期时间', dataIndex: 'expire_at' },
  { title: '已用流量', dataIndex: 'used_traffic' },
  { title: '总流量', dataIndex: 'traffic' },
  { title: '状态', dataIndex: 'status' },
]

// 购买记录
const orders = ref([])
const orderTotal = ref(0)
const orderQuery = reactive({ page: 1, page_size: 10 })

async function loadOrders() {
  loading.value = true
  try {
    const res = await packageApi.orders(orderQuery)
    orders.value = res.data?.list || []
    orderTotal.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const orderColumns = [
  { title: '订单号', dataIndex: 'order_no' },
  { title: '套餐', dataIndex: 'package_name' },
  { title: '金额', dataIndex: 'amount' },
  { title: '状态', dataIndex: 'status' },
  { title: '时间', dataIndex: 'created_at' },
]

// 充值
const rechargeVisible = ref(false)
const rechargeForm = reactive({ amount: 100, method: 'alipay' })

async function submitRecharge() {
  if (!rechargeForm.amount) return message.error('请输入金额')
  await packageApi.recharge({ ...rechargeForm })
  message.success('充值请求已提交')
  rechargeVisible.value = false
  loadBalance()
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
  <div class="page-container">
    <div class="page-toolbar">
      <h2 style="margin: 0">套餐管理</h2>
      <a-space>
        <a-statistic title="余额" :value="balance" prefix="￥" />
        <a-button type="primary" @click="rechargeVisible = true">充值</a-button>
        <a-button @click="loadBalance"><ReloadOutlined /></a-button>
      </a-space>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <a-tab-pane key="market" tab="套餐市场">
        <a-table
          :columns="marketColumns"
          :data-source="market"
          :loading="loading"
          row-key="id"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button type="primary" size="small" @click="buy(record)">购买</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="mine" tab="我的套餐">
        <a-table
          :columns="mineColumns"
          :data-source="mine"
          :loading="loading"
          row-key="id"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                {{ record.status === 'active' ? '有效' : '已过期' }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="orders" tab="购买记录">
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
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="rechargeVisible" title="账户充值" @ok="submitRecharge">
      <a-form :model="rechargeForm" layout="vertical">
        <a-form-item label="充值金额（元）" required>
          <a-input-number v-model:value="rechargeForm.amount" :min="1" style="width: 100%" />
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
