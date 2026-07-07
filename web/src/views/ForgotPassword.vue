<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { MailOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons-vue'
import { authApi } from '../api'

const router = useRouter()
const loading = ref(false)
const current = ref(0)

const emailFormRef = ref()
const emailForm = reactive({ email: '' })
const emailRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
}

const resetFormRef = ref()
const resetForm = reactive({
  code: '',
  password: '',
  confirmPassword: '',
})

const validatePass2 = async (_rule, value) => {
  if (value === '') {
    return Promise.reject('请再次输入密码')
  }
  if (value !== resetForm.password) {
    return Promise.reject('两次密码不一致')
  }
  return Promise.resolve()
}

const resetRules = {
  code: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '至少 8 位', trigger: 'blur' },
  ],
  confirmPassword: [{ required: true, validator: validatePass2, trigger: 'blur' }],
}

async function handleSendCode() {
  await emailFormRef.value.validate()
  loading.value = true
  try {
    await authApi.forgotPassword(emailForm.email)
    message.success('验证码已发送至邮箱')
    current.value = 1
  } finally {
    loading.value = false
  }
}

async function handleReset() {
  await resetFormRef.value.validate()
  loading.value = true
  try {
    await authApi.resetPassword(emailForm.email, resetForm.code, resetForm.password)
    message.success('密码重置成功，请登录')
    router.push('/login')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="forgot-container">
    <div class="forgot-box">
      <div class="forgot-header">
        <h1>ShieldFlow</h1>
        <p>找回密码</p>
      </div>

      <a-steps :current="current" class="forgot-steps">
        <a-step title="验证邮箱" />
        <a-step title="重置密码" />
      </a-steps>

      <div v-if="current === 0">
        <a-form
          ref="emailFormRef"
          :model="emailForm"
          :rules="emailRules"
          layout="vertical"
        >
          <a-form-item label="注册邮箱" name="email">
            <a-input
              v-model:value="emailForm.email"
              size="large"
              placeholder="请输入注册邮箱"
            >
              <template #prefix><MailOutlined /></template>
            </a-input>
          </a-form-item>
          <a-form-item>
            <a-button
              type="primary"
              size="large"
              block
              :loading="loading"
              @click="handleSendCode"
            >
              发送验证码
            </a-button>
          </a-form-item>
        </a-form>
      </div>

      <div v-else>
        <a-form
          ref="resetFormRef"
          :model="resetForm"
          :rules="resetRules"
          layout="vertical"
        >
          <a-form-item label="验证码" name="code">
            <a-input
              v-model:value="resetForm.code"
              size="large"
              placeholder="请输入邮箱收到的验证码"
            >
              <template #prefix><SafetyOutlined /></template>
            </a-input>
          </a-form-item>
          <a-form-item label="新密码" name="password">
            <a-input-password
              v-model:value="resetForm.password"
              size="large"
              placeholder="新密码（至少8位）"
            >
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>
          <a-form-item label="确认密码" name="confirmPassword">
            <a-input-password
              v-model:value="resetForm.confirmPassword"
              size="large"
              placeholder="请再次输入新密码"
            >
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>
          <a-form-item>
            <a-space direction="vertical" style="width: 100%">
              <a-button
                type="primary"
                size="large"
                block
                :loading="loading"
                @click="handleReset"
              >
                重置密码
              </a-button>
              <a-button block size="large" @click="current = 0">
                返回上一步
              </a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </div>

      <div class="forgot-footer">
        <router-link to="/login">返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.forgot-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.forgot-box {
  width: 420px;
  max-width: 90vw;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
}
.forgot-header {
  text-align: center;
  margin-bottom: 24px;
}
.forgot-header h1 {
  margin: 0;
  font-size: 32px;
  color: #1890ff;
  letter-spacing: 2px;
}
.forgot-header p {
  margin: 8px 0 0;
  color: #999;
}
.forgot-steps {
  margin-bottom: 32px;
}
.forgot-footer {
  text-align: center;
  margin-top: 16px;
}
</style>
