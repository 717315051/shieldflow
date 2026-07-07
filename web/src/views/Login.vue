<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons-vue'
import { useUserStore } from '../store/user'
import { authApi } from '../api'
import request from '../utils/request'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const loading = ref(false)
const captchaSrc = ref('')

const formRef = ref()
const form = reactive({
  username: '',
  password: '',
  captcha: '',
  captchaId: '',
})

const rules = {
  username: [{ required: true, message: '请输入用户名/邮箱/手机', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  captcha: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
}

async function refreshCaptcha() {
  try {
    const res = await request.get('/auth/captcha?t=' + Date.now())
    if (res && res.data) {
      captchaSrc.value = res.data.image
      form.captchaId = res.data.captcha_id
    }
  } catch (e) {
    console.error('获取验证码失败', e)
  }
}

async function handleSubmit() {
  try {
    await formRef.value.validate()
    loading.value = true
    await userStore.login({
      account: form.username,
      password: form.password,
      captcha: form.captcha,
      captcha_id: form.captchaId,
    })
    message.success('登录成功')
    const redirect = route.query.redirect || '/dashboard'
    router.push(redirect)
  } catch (e) {
    refreshCaptcha()
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshCaptcha()
})
</script>

<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <h1>ShieldFlow</h1>
        <p>CDN 管理后台</p>
      </div>
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        layout="vertical"
        @finish="handleSubmit"
      >
        <a-form-item name="username">
          <a-input
            v-model:value="form.username"
            size="large"
            placeholder="用户名 / 邮箱 / 手机号"
          >
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item name="password">
          <a-input-password
            v-model:value="form.password"
            size="large"
            placeholder="密码"
            @pressEnter="handleSubmit"
          >
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item name="captcha">
          <div class="captcha-row">
            <a-input
              v-model:value="form.captcha"
              size="large"
              placeholder="图形验证码"
              style="flex: 1"
            >
              <template #prefix><SafetyOutlined /></template>
            </a-input>
            <img
              :src="captchaSrc"
              class="captcha-img"
              @click="refreshCaptcha"
              alt="验证码"
            />
          </div>
        </a-form-item>
        <a-form-item>
          <a-button
            type="primary"
            size="large"
            block
            :loading="loading"
            html-type="submit"
            @click="handleSubmit"
          >
            登录
          </a-button>
        </a-form-item>
        <div class="login-footer">
          <router-link to="/forgot-password">忘记密码？</router-link>
          <router-link to="/register" style="margin-left: 16px">没有账号？立即注册</router-link>
        </div>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-box {
  width: 400px;
  max-width: 90vw;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
}
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-header h1 {
  margin: 0;
  font-size: 32px;
  color: #1890ff;
  letter-spacing: 2px;
}
.login-header p {
  margin: 8px 0 0;
  color: #999;
}
.captcha-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.captcha-img {
  height: 40px;
  width: 120px;
  cursor: pointer;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
}
.login-footer {
  text-align: center;
}
</style>
