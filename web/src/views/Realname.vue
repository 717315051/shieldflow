<template>
  <div class="realname-page">
    <a-page-header
      title="实名认证"
      sub-title="为保障账户安全并满足合规要求，绑定域名前需要先完成实名认证。"
    />

    <a-row :gutter="24">
      <a-col :span="14">
        <a-card title="认证信息">
          <a-alert
            v-if="userInfo?.verified"
            message="已完成实名认证"
            type="success"
            show-icon
            style="margin-bottom: 16px"
          />
          <a-alert
            v-else
            message="尚未完成实名认证"
            type="warning"
            show-icon
            style="margin-bottom: 16px"
          />

          <a-form
            layout="vertical"
            :model="form"
            :rules="rules"
            ref="formRef"
            @finish="submit"
            :disabled="userInfo?.verified"
          >
            <a-form-item label="真实姓名" name="real_name" required>
              <a-input
                v-model:value="form.real_name"
                placeholder="请输入真实姓名"
                :maxlength="50"
                allow-clear
              />
            </a-form-item>
            <a-form-item label="身份证号" name="id_card" required>
              <a-input
                v-model:value="form.id_card"
                placeholder="请输入 18 位身份证号"
                :maxlength="18"
                allow-clear
              />
            </a-form-item>
            <a-form-item>
              <a-space>
                <a-button type="primary" html-type="submit" :loading="loading" :disabled="userInfo?.verified">
                  {{ userInfo?.verified ? '已认证' : '提交认证' }}
                </a-button>
                <a-button @click="reset">重置</a-button>
              </a-space>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <a-col :span="10">
        <a-card title="认证状态">
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="用户名">
              {{ userInfo?.username || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="真实姓名">
              {{ userInfo?.real_name || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="认证状态">
              <a-tag v-if="userInfo?.verified" color="success">已认证</a-tag>
              <a-tag v-else color="warning">未认证</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="认证时间">
              {{ userInfo?.updated_at || '-' }}
            </a-descriptions-item>
          </a-descriptions>
        </a-card>

        <a-card title="说明" style="margin-top: 16px">
          <p>• 实名信息仅用于身份核验。</p>
          <p>• 通过认证后才能添加域名、配置 CDN 加速。</p>
          <p>• 身份证号需为 18 位合法身份证号。</p>
          <p>• 认证信息会脱敏处理后展示。</p>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { useUserStore } from '../store/user'
import { authApi } from '../api'

const userStore = useUserStore()
const userInfo = ref(userStore.userInfo)
const loading = ref(false)
const formRef = ref()

const form = reactive({
  real_name: userInfo.value?.real_name || '',
  id_card: userInfo.value?.id_card || '',
})

const rules = {
  real_name: [
    { required: true, message: '请输入真实姓名' },
    { min: 2, max: 50, message: '姓名长度 2-50 个字符' },
  ],
  id_card: [
    { required: true, message: '请输入身份证号' },
    { pattern: /^\d{17}[\dX]$/, message: '身份证号必须为 18 位（末位可为 X）' },
  ],
}

async function loadProfile() {
  try {
    const res = await authApi.profile()
    if (res.code === 0) {
      userInfo.value = res.data
      userStore.setUserInfo(res.data)
    }
  } catch (e) {
    console.error('load profile failed', e)
  }
}

async function submit() {
  loading.value = true
  try {
    const res = await authApi.realname({
      real_name: form.real_name,
      id_card: form.id_card,
    })
    if (res.code === 0) {
      message.success('认证提交成功')
      await loadProfile()
    } else {
      message.error(res.message || '认证失败')
    }
  } catch (e) {
    message.error('认证请求失败：' + (e.message || e))
  } finally {
    loading.value = false
  }
}

function reset() {
  form.real_name = userInfo.value?.real_name || ''
  form.id_card = userInfo.value?.id_card || ''
}

onMounted(() => {
  loadProfile()
})
</script>

<style scoped>
.realname-page {
  max-width: 1200px;
}
</style>
