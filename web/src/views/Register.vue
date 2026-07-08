<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
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
  emailCode: '',
  agreement: false,
})

// 邮箱验证码发送倒计时
const sendingCode = ref(false)
const countdown = ref(0)
let timer = null

const validatePass2 = async (_rule, value) => {
  if (value === '') {
    return Promise.reject('请再次输入密码')
  }
  if (value !== form.password) {
    return Promise.reject('两次密码不一致')
  }
  return Promise.resolve()
}

const validateEmailCode = async (_rule, value) => {
  if (!value) {
    return Promise.reject('请输入邮箱验证码')
  }
  if (!/^\d{6}$/.test(value)) {
    return Promise.reject('邮箱验证码为 6 位数字')
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
  emailCode: [{ required: true, validator: validateEmailCode, trigger: 'blur' }],
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

async function sendEmailCode() {
  if (sendingCode.value || countdown.value > 0) return
  if (!form.email) {
    message.warning('请先填写邮箱')
    return
  }
  if (!/^[\w.+-]+@[\w-]+\.[\w.-]+$/.test(form.email)) {
    message.warning('邮箱格式不正确')
    return
  }
  sendingCode.value = true
  try {
    await authApi.sendEmailCode(form.email)
    message.success('验证码已发送至邮箱，请查收')
    countdown.value = 60
    timer = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) {
        clearInterval(timer)
        timer = null
      }
    }, 1000)
  } catch (e) {
    // 拦截器已经显示错误
  } finally {
    sendingCode.value = false
  }
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
      captcha_code: form.captcha,
      captcha_id: form.captchaId,
      email_code: form.emailCode,
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

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="login-page-root">
    <!-- 装饰:4 个浮动圆点 -->
    <div class="fly bg-fly-circle1"></div>
    <div class="fly bg-fly-circle2"></div>
    <div class="fly bg-fly-circle3"></div>
    <div class="fly bg-fly-circle4"></div>

    <!-- 移动端顶部 brand -->
    <div class="login-brand login-brand--h5">
      <div class="brand-logo">SCDN</div>
    </div>

    <div class="login-center login-center--h5">
      <div class="login-row">
        <!-- PC 左侧插图区 -->
        <div class="login-left">
          <div class="login-left__art">
            <svg viewBox="0 0 600 540" xmlns="http://www.w3.org/2000/svg" class="login-svg">
              <defs>
                <linearGradient id="rg1" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stop-color="#00b42a"/>
                  <stop offset="100%" stop-color="#165dff"/>
                </linearGradient>
              </defs>
              <circle cx="300" cy="270" r="240" fill="url(#rg1)" opacity="0.12"/>
              <circle cx="300" cy="270" r="170" fill="url(#rg1)" opacity="0.10"/>
              <!-- 网盾 -->
              <path d="M300 100 L420 150 L420 290 Q420 380 300 430 Q180 380 180 290 L180 150 Z"
                    fill="url(#rg1)" opacity="0.95"/>
              <path d="M300 110 L412 157 L412 290 Q412 376 300 422 Q188 376 188 290 L188 157 Z"
                    fill="#fff" opacity="0.18"/>
              <!-- 装饰圆点 -->
              <g fill="#fff">
                <circle cx="120" cy="80" r="10"/><circle cx="480" cy="80" r="10"/>
                <circle cx="80" cy="300" r="10"/><circle cx="520" cy="300" r="10"/>
                <circle cx="120" cy="500" r="10"/><circle cx="480" cy="500" r="10"/>
              </g>
              <g stroke="#fff" stroke-width="2" opacity="0.7" stroke-dasharray="5 5" fill="none">
                <path d="M120 80 L300 270 L480 80"/>
                <path d="M80 300 L300 270 L520 300"/>
                <path d="M120 500 L300 270 L480 500"/>
              </g>
              <path d="M280 200 L320 200 L295 250 L325 250 L270 320 L290 260 L260 260 Z" fill="#fff"/>
              <text x="300" y="490" text-anchor="middle" font-family="Inter, sans-serif"
                    font-weight="700" font-size="32" fill="#fff">加入 SCDN</text>
              <text x="300" y="518" text-anchor="middle" font-family="Inter, sans-serif"
                    font-weight="400" font-size="14" fill="#fff" opacity="0.85">注册账号,开启 CDN 防护之旅</text>
            </svg>
          </div>
        </div>
        <!-- 表单 -->
        <div class="login-right">
          <div class="login-title">
            <div class="login-title__logo">SCDN</div>
            <h2 class="login-title__text">注册账号</h2>
            <p class="login-title__sub">填写以下信息创建您的 ShieldFlow 账号</p>
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
                placeholder="用户名 (3-20 字符)"
                autocomplete="username"
              />
            </a-form-item>
            <a-form-item name="email">
              <a-input v-model:value="form.email" size="large" placeholder="邮箱" autocomplete="email" />
            </a-form-item>
            <a-form-item name="emailCode">
              <div class="email-code-row">
                <a-input
                  v-model:value="form.emailCode" size="large" maxlength="6"
                  placeholder="邮箱验证码 (6位数字)" style="flex: 1"
                />
                <a-button size="large" class="send-btn"
                  :loading="sendingCode" :disabled="countdown > 0" @click="sendEmailCode"
                >{{ countdown > 0 ? `${countdown}s` : '发送验证码' }}</a-button>
              </div>
            </a-form-item>
            <a-form-item name="phone">
              <a-input v-model:value="form.phone" size="large" placeholder="手机号" autocomplete="tel" />
            </a-form-item>
            <a-form-item name="password">
              <a-input-password
                v-model:value="form.password" size="large"
                placeholder="密码 (至少 8 位)" autocomplete="new-password"
              />
            </a-form-item>
            <a-form-item name="confirmPassword">
              <a-input-password
                v-model:value="form.confirmPassword" size="large"
                placeholder="确认密码" autocomplete="new-password"
              />
            </a-form-item>
            <a-form-item name="captcha">
              <div class="captcha-row">
                <a-input v-model:value="form.captcha" size="large" placeholder="图形验证码" style="flex: 1" />
                <img :src="captchaSrc" class="captcha-img" @click="refreshCaptcha" alt="验证码" />
              </div>
            </a-form-item>
            <a-form-item name="agreement">
              <a-checkbox v-model:checked="form.agreement">
                我已阅读并同意 <a>服务条款</a> 和 <a>隐私政策</a>
              </a-checkbox>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" size="large" block :loading="loading" html-type="submit">
                立即注册
              </a-button>
            </a-form-item>
            <div class="login-signup">
              已有账号? <router-link to="/login">立即登录</router-link>
            </div>
          </a-form>
          <div class="login-copyright">© 2026 ShieldFlow · hub-hupu.com</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* === 根容器(与 Login.vue 一致的渐变) === */
.login-page-root {
  min-height: 100vh;
  width: 100vw;
  background:
    radial-gradient(1200px 800px at 20% 30%, #00b42a 0%, transparent 60%),
    radial-gradient(900px 600px at 80% 70%, #165dff 0%, transparent 60%),
    linear-gradient(135deg, #0b1e1a 0%, #0e2a35 50%, #0a1a2e 100%);
  color: #fff;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  position: relative; overflow: hidden;
  padding: 24px;
}
.login-brand--h5 {
  position: absolute; top: 0; left: 0; right: 0;
  height: 52px; display: flex; align-items: center; padding: 0 24px;
  background: rgba(0,0,0,0.18); backdrop-filter: blur(12px);
}
.brand-logo {
  font-size: 22px; font-weight: 700; letter-spacing: 3px;
  background: linear-gradient(135deg, #fff 0%, #00b42a 100%);
  -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent;
}

/* === 浮动圆点 === */
.fly {
  position: absolute; border-radius: 50%;
  filter: blur(40px); opacity: 0.6;
  animation: flyFloat 18s ease-in-out infinite alternate;
  pointer-events: none;
}
.bg-fly-circle1 { width: 220px; height: 220px; left: -50px; top: 110px;
  background: linear-gradient(135deg, #00b42a, #165dff); animation-delay: 0s; }
.bg-fly-circle2 { width: 260px; height: 260px; left: 80px; bottom: -50px;
  background: linear-gradient(135deg, #00d6b9, #165dff); animation-delay: -3s; }
.bg-fly-circle3 { width: 200px; height: 200px; right: 100px; top: -30px;
  background: linear-gradient(135deg, #722ed1, #ff85c0); animation-delay: -6s; }
.bg-fly-circle4 { width: 240px; height: 240px; right: 60px; bottom: 80px;
  background: linear-gradient(135deg, #ffd666, #00b42a); animation-delay: -9s; }
@keyframes flyFloat {
  0% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(40px, -30px) scale(1.1); }
  100% { transform: translate(-30px, 40px) scale(0.95); }
}

/* === 居中卡片 === */
.login-center { position: relative; z-index: 1; width: 100%; max-width: 1120px; padding: 24px; }
.login-center--h5 { padding: 0; }

.login-row {
  width: 100%;
  display: grid;
  grid-template-columns: 1fr 1fr;
  border-radius: 16px;
  overflow: hidden;
  background: rgba(255,255,255,0.96);
  box-shadow: 0 30px 80px rgba(0,0,0,0.45), 0 0 0 1px rgba(255,255,255,0.08);
}
.login-left {
  background: linear-gradient(135deg, #00b42a 0%, #165dff 100%);
  padding: 32px;
  display: flex; align-items: center; justify-content: center;
  min-height: 580px; position: relative; overflow: hidden;
}
.login-left::before, .login-left::after {
  content: ''; position: absolute; border-radius: 50%;
  background: rgba(255,255,255,0.08);
}
.login-left::before { width: 280px; height: 280px; left: -100px; bottom: -100px; }
.login-left::after  { width: 180px; height: 180px; right: -60px; top: -50px; }
.login-left__art { width: 100%; max-width: 480px; position: relative; }
.login-svg { width: 100%; height: auto; display: block; }

.login-right {
  padding: 40px 56px 32px;
  display: flex; flex-direction: column;
  background: #fff; min-height: 580px;
}
.login-title { margin-bottom: 24px; }
.login-title__logo {
  font-size: 28px; font-weight: 700; letter-spacing: 6px;
  background: linear-gradient(135deg, #00b42a 0%, #165dff 100%);
  -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent;
  margin-bottom: 14px;
}
.login-title__text {
  font-size: 24px; font-weight: 600; color: #1d2129; margin: 0;
}
.login-title__sub { font-size: 14px; color: #86909c; margin: 8px 0 0; }

.login-right :deep(.ant-input),
.login-right :deep(.ant-input-affix-wrapper) {
  background: #f7f8fa !important;
  border: 1px solid #e5e6eb !important;
  border-radius: 8px !important;
}
.login-right :deep(.ant-input-affix-wrapper-focused),
.login-right :deep(.ant-input:focus),
.login-right :deep(.ant-input-affix-wrapper:focus-within) {
  border-color: #00b42a !important;
  background: #fff !important;
  box-shadow: 0 0 0 3px rgba(0,180,42,0.10) !important;
}
.login-right :deep(.ant-form-item) { margin-bottom: 16px; }
.login-right :deep(.ant-form-item-label) { padding-bottom: 4px; }
.login-right :deep(.ant-btn-primary) {
  background: linear-gradient(135deg, #00b42a 0%, #165dff 100%) !important;
  border: none !important;
  height: 44px; font-size: 16px; font-weight: 500; border-radius: 8px;
  box-shadow: 0 4px 14px rgba(0,180,42,0.35) !important;
  letter-spacing: 4px;
}

.captcha-row { display: flex; gap: 8px; align-items: center; }
.email-code-row { display: flex; gap: 8px; align-items: center; }
.send-btn {
  border: 1px solid #e5e6eb !important; color: #165dff !important; background: #fff !important;
  white-space: nowrap; flex-shrink: 0;
}
.send-btn:hover { border-color: #165dff !important; background: #f7f8fa !important; }

.captcha-img {
  height: 40px; width: 120px; cursor: pointer;
  border: 1px solid #e5e6eb; border-radius: 8px;
  flex-shrink: 0;
}

.login-signup {
  text-align: center; color: #86909c; font-size: 14px;
  border-top: 1px solid #f2f3f5; padding-top: 16px; margin-top: 8px;
}
.login-signup a { color: #165dff; }
.login-copyright {
  margin-top: auto; padding-top: 16px;
  text-align: center; font-size: 12px; color: #c9cdd4;
}

@media (max-width: 992px) {
  .login-row { grid-template-columns: 1fr; border-radius: 12px; }
  .login-left { display: none; }
  .login-right { padding: 32px 32px 24px; min-height: auto; }
  .login-center { padding: 12px; }
  .login-center--h5 { padding: 52px 16px 16px; }
  .captcha-img { width: 100px; }
}
@media (max-width: 480px) {
  .login-right { padding: 24px 20px 20px; }
  .login-title__text { font-size: 20px; }
  .login-title__sub { font-size: 13px; }
  .login-right :deep(.ant-form-item) { margin-bottom: 12px; }
  .bg-fly-circle1, .bg-fly-circle3 { opacity: 0.4; }
}
</style>
