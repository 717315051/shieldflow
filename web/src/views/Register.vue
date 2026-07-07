<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  MailOutlined,
  PhoneOutlined,
  SafetyOutlined,
} from '@ant-design/icons-vue'
import { authApi } from '../api'

const router = useRouter()
const loading = ref(false)
const captchaSrc = ref('')

const formRef = ref()
const form = reactive({
  username: '',
  email: '',
  phone: '',
  password: '',
  confirmPassword: '',
  captcha: '',
  captchaId: '',
  agreement: false,
})

const validatePass2 = async (_rule, value) => {
  if (value === '') {
    return Promise.reject('请再次输入密码')
  }
  if (value !== form.password) {
    return Promise.reject('两次密码不一致')
  }
  return Promise.resolve()
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '3-20 个字符', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '至少 8 位', trigger: 'blur' },
  ],
  confirmPassword: [{ required: true, validator: validatePass2, trigger: 'blur' }],
  captcha: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
  agreement: [
    {
      validator: (_r, v) =>
        v ? Promise.resolve() : Promise.reject('请阅读并同意服务条款'),
      trigger: 'change',
    },
  ],
}

function refreshCaptcha() {
  form.captchaId = String(Date.now())
  captchaSrc.value = authApi.captcha() + '&id=' + form.captchaId
}

async function handleSubmit() {
  try {
    await formRef.value.validate()
    loading.value = true
    await authApi.register({
      username: form.username,
      email: form.email,
      phone: form.phone,
      password: form.password,
      captcha: form.captcha,
      captcha_id: form.captchaId,
    })
    message.success('注册成功，请登录')
    router.push('/login')
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
  <div class="register-container">
    <div class="register-box">
      <div class="register-header">
        <h1>ShieldFlow</h1>
        <p>新用户注册</p>
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
            placeholder="用户名"
          >
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item name="email">
          <a-input v-model:value="form.email" size="large" placeholder="邮箱">
            <template #prefix><MailOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item name="phone">
          <a-input v-model:value="form.phone" size="large" placeholder="手机号">
            <template #prefix><PhoneOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item name="password">
          <a-input-password
            v-model:value="form.password"
            size="large"
            placeholder="密码（至少8位）"
          >
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item name="confirmPassword">
          <a-input-password
            v-model:value="form.confirmPassword"
            size="large"
            placeholder="确认密码"
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
        <a-form-item name="agreement">
          <a-checkbox v-model:checked="form.agreement">
            我已阅读并同意 <a>服务条款</a> 和 <a>隐私政策</a>
          </a-checkbox>
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
            注册
          </a-button>
        </a-form-item>
        <div class="register-footer">
          已有账号？<router-link to="/login">立即登录</router-link>
        </div>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.register-box {
  width: 420px;
  max-width: 90vw;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
}
.register-header {
  text-align: center;
  margin-bottom: 24px;
}
.register-header h1 {
  margin: 0;
  font-size: 32px;
  color: #1890ff;
}
.register-header p {
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
.register-footer {
  text-align: center;
}
</style>
