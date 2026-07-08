<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  UserOutlined, LockOutlined, SafetyCertificateOutlined,
  EyeOutlined, EyeInvisibleOutlined,
  BulbOutlined, BulbFilled,
  GlobalOutlined, ApiOutlined, SafetyOutlined,
} from '@ant-design/icons-vue'
import { useUserStore } from '../store/user'
import { authApi } from '../api'

const router = useRouter()
const userStore = useUserStore()

const form = reactive({
  username: '',
  password: '',
  remember: true,
  captcha: '',
})
const loading = ref(false)
const isDark = ref(localStorage.getItem('sf-theme') === 'dark')
const visiblePwd = ref(false)
const isMobile = ref(window.innerWidth <= 768)
const formRef = ref()

import { reactive } from 'vue'

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleSubmit() {
  if (loading.value) return
  try {
    await formRef.value.validate()
    loading.value = true
    await userStore.login({
      account: form.username,
      password: form.password,
      captcha_code: form.captcha || 'disabled',
      captcha_id: 'disabled',
    })
    message.success('登录成功')
    router.push('/dashboard')
  } catch (e) {
    console.error(e)
    message.error('登录失败,请检查账号密码')
  } finally {
    loading.value = false
  }
}

function onResize() { isMobile.value = window.innerWidth <= 768 }

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
  localStorage.setItem('sf-theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  onResize()
  window.addEventListener('resize', onResize)
  document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
})
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<template>
  <div :class="['login', isDark && 'login--dark']">
    <!-- 顶部条 -->
    <header class="login-top">
      <div class="login-top__brand">
        <div class="login-top__logo">
          <SafetyCertificateOutlined :style="{ fontSize: '18px', color: '#fff' }" />
        </div>
        <span class="login-top__name">ShieldFlow CDN</span>
      </div>
      <a class="login-top__theme" @click="toggleTheme">
        <component :is="isDark ? BulbFilled : BulbOutlined" :style="{ fontSize: '16px' }" />
      </a>
    </header>

    <!-- 主区: 左侧价值主张 + 右侧表单 -->
    <main class="login-main">
      <section class="login-hero">
        <h1 class="login-hero__title">
          企业级内容分发<br/>
          <span class="login-hero__accent">更快 · 更稳 · 更安全</span>
        </h1>
        <p class="login-hero__desc">
          融合智能 DNS、边缘计算与多层安全防护，为您的业务提供一站式 CDN 加速解决方案。
        </p>

        <ul class="login-hero__features">
          <li>
            <div class="login-hero__feat-icon"><GlobalOutlined :style="{ fontSize: '20px', color: 'var(--info)' }" /></div>
            <div>
              <div class="login-hero__feat-title">全球边缘节点</div>
              <div class="login-hero__feat-desc">200+ 节点，覆盖 50+ 国家与地区</div>
            </div>
          </li>
          <li>
            <div class="login-hero__feat-icon"><ApiOutlined :style="{ fontSize: '20px', color: 'var(--info)' }" /></div>
            <div>
              <div class="login-hero__feat-title">毫秒级响应</div>
              <div class="login-hero__feat-desc">智能路由 + 协议优化，端到端 &lt; 50ms</div>
            </div>
          </li>
          <li>
            <div class="login-hero__feat-icon"><SafetyOutlined :style="{ fontSize: '20px', color: 'var(--info)' }" /></div>
            <div>
              <div class="login-hero__feat-title">多层防护</div>
              <div class="login-hero__feat-desc">DDoS + WAF + Bot 管理 + CC 智能识别</div>
            </div>
          </li>
        </ul>

        <!-- 等距网格背景 -->
        <svg class="login-hero__grid" viewBox="0 0 600 400" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="none">
          <defs>
            <pattern id="dot-grid" x="0" y="0" width="24" height="24" patternUnits="userSpaceOnUse">
              <circle cx="2" cy="2" r="1" fill="currentColor" opacity="0.12" />
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#dot-grid)" />
        </svg>
      </section>

      <section class="login-form-wrap">
        <div class="login-form-card">
          <h2 class="login-form-card__title">账号登录</h2>
          <p class="login-form-card__sub">使用您的 ShieldFlow 账号登录</p>

          <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @finish="handleSubmit">
            <a-form-item name="username">
              <a-input
                v-model:value="form.username"
                size="large"
                placeholder="用户名 / 手机号 / 邮箱"
                autocomplete="username"
              >
                <template #prefix>
                  <UserOutlined :style="{ fontSize: '14px', color: 'var(--text-tertiary)' }" />
                </template>
              </a-input>
            </a-form-item>
            <a-form-item name="password">
              <a-input-password
                v-model:value="form.password"
                size="large"
                placeholder="密码"
                autocomplete="current-password"
                :type="visiblePwd ? 'text' : 'password'"
              >
                <template #prefix>
                  <LockOutlined :style="{ fontSize: '14px', color: 'var(--text-tertiary)' }" />
                </template>
              </a-input-password>
            </a-form-item>
            <div class="login-form-card__row">
              <a-checkbox v-model:checked="form.remember">记住我</a-checkbox>
              <router-link to="/forgot-password" class="login-form-card__link">忘记密码？</router-link>
            </div>
            <a-button type="primary" size="large" block :loading="loading" html-type="submit" class="login-form-card__submit">
              登录
            </a-button>
          </a-form>

          <div class="login-form-card__foot">
            还没有账号？<router-link to="/register" class="login-form-card__link">立即注册</router-link>
          </div>
        </div>
      </section>
    </main>

    <!-- 底部 -->
    <footer class="login-foot">
      © 2026 ShieldFlow CDN · hub-hupu.com · 沪 ICP 备 12345678 号
    </footer>
  </div>
</template>

<style scoped>
.login {
  min-height: 100vh;
  background: var(--bg-page);
  color: var(--text-primary);
  display: flex; flex-direction: column;
  transition: background var(--dur-med) var(--ease);
}

/* 顶部 */
.login-top {
  height: 64px;
  padding: 0 var(--sp-6);
  display: flex; align-items: center; justify-content: space-between;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-card);
}
.login-top__brand { display: flex; align-items: center; gap: var(--sp-3); }
.login-top__logo {
  width: 32px; height: 32px;
  background: var(--brand-primary);
  border-radius: var(--r-md);
  display: flex; align-items: center; justify-content: center;
}
.login-top__name {
  font-size: var(--fs-md); font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.3px;
}
.login-top__theme {
  width: 32px; height: 32px;
  border-radius: var(--r-md);
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--dur-fast) var(--ease);
}
.login-top__theme:hover { background: var(--bg-hover); color: var(--text-primary); }

/* 主区: 1:1 等宽 */
.login-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: stretch;
}

/* 左侧: 价值主张 + 等距网格 */
.login-hero {
  position: relative;
  padding: var(--sp-10) var(--sp-10);
  display: flex; flex-direction: column; justify-content: center;
  background: var(--bg-card);
  border-right: 1px solid var(--border-color);
  overflow: hidden;
}
.login-hero__title {
  font-size: 36px; font-weight: 700;
  line-height: 1.3;
  margin: 0 0 var(--sp-5);
  color: var(--text-primary);
  letter-spacing: -0.5px;
  position: relative; z-index: 1;
}
.login-hero__accent { color: var(--brand-primary); }
.login-hero__desc {
  font-size: var(--fs-md);
  color: var(--text-secondary);
  line-height: 1.7;
  margin: 0 0 var(--sp-8);
  max-width: 480px;
  position: relative; z-index: 1;
}
.login-hero__features {
  list-style: none; padding: 0; margin: 0;
  display: flex; flex-direction: column; gap: var(--sp-5);
  position: relative; z-index: 1;
}
.login-hero__features li {
  display: flex; gap: var(--sp-4); align-items: flex-start;
}
.login-hero__feat-icon {
  width: 40px; height: 40px;
  background: var(--info-soft);
  border-radius: var(--r-md);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.login-hero__feat-title { font-size: var(--fs-md); font-weight: 600; color: var(--text-primary); margin-bottom: 4px; }
.login-hero__feat-desc { font-size: var(--fs-sm); color: var(--text-tertiary); }

.login-hero__grid {
  position: absolute; inset: 0;
  width: 100%; height: 100%;
  color: var(--text-secondary);
  z-index: 0;
}

/* 右侧: 表单卡 */
.login-form-wrap {
  display: flex; align-items: center; justify-content: center;
  padding: var(--sp-10);
  background: var(--bg-page);
}
.login-form-card {
  width: 100%; max-width: 400px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--r-lg);
  padding: var(--sp-8) var(--sp-8);
  transition: border-color var(--dur-fast) var(--ease);
}
.login-form-card__title {
  font-size: var(--fs-xl);
  font-weight: 600;
  margin: 0 0 var(--sp-2);
  color: var(--text-primary);
}
.login-form-card__sub {
  font-size: var(--fs-sm);
  color: var(--text-tertiary);
  margin: 0 0 var(--sp-6);
}
.login-form-card__row {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: var(--sp-5);
}
.login-form-card__link { color: var(--brand-primary); text-decoration: none; font-size: var(--fs-sm); }
.login-form-card__link:hover { text-decoration: underline; }
.login-form-card__submit { margin-top: var(--sp-3); height: 40px; font-size: var(--fs-md); font-weight: 500; }
.login-form-card__foot {
  text-align: center;
  font-size: var(--fs-sm);
  color: var(--text-tertiary);
  margin-top: var(--sp-5);
}

.login-foot {
  padding: var(--sp-4) var(--sp-6);
  text-align: center;
  font-size: var(--fs-xs);
  color: var(--text-tertiary);
  border-top: 1px solid var(--border-color);
  background: var(--bg-card);
}

/* 响应式: 移动端隐藏左侧 */
@media (max-width: 1024px) {
  .login-main { grid-template-columns: 1fr; }
  .login-hero { display: none; }
  .login-form-wrap { padding: var(--sp-6); }
}
@media (max-width: 480px) {
  .login-hero { padding: var(--sp-6); }
  .login-form-wrap { padding: var(--sp-4); }
  .login-form-card { padding: var(--sp-6) var(--sp-5); }
  .login-hero__title { font-size: 28px; }
}
</style>