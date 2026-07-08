import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '../utils/request'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('shieldflow_token') || '')
  const userInfo = ref(JSON.parse(localStorage.getItem('shieldflow_user') || 'null'))

  const isLogin = computed(() => !!token.value)
  const isAdmin = computed(() => {
    const role = userInfo.value?.role
    return role === 'admin'
  })

  async function login(payload) {
    const res = await request.post('/auth/login', payload)
    token.value = res.data.token
    localStorage.setItem('shieldflow_token', res.data.token)
    if (res.data.user) {
      userInfo.value = res.data.user
      localStorage.setItem('shieldflow_user', JSON.stringify(res.data.user))
    }
    return res.data
  }

  async function fetchProfile() {
    const res = await request.get('/auth/profile')
    userInfo.value = res.data
    localStorage.setItem('shieldflow_user', JSON.stringify(res.data))
    return res.data
  }

  async function updateProfile(data) {
    const res = await request.put('/auth/profile', data)
    userInfo.value = res.data
    localStorage.setItem('shieldflow_user', JSON.stringify(res.data))
    return res.data
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('shieldflow_token')
    localStorage.removeItem('shieldflow_user')
  }

  function setUserInfo(info) {
    userInfo.value = info
    localStorage.setItem('shieldflow_user', JSON.stringify(info))
  }

  function setToken(t) {
    token.value = t
    localStorage.setItem('shieldflow_token', t)
  }

  return {
    token,
    userInfo,
    isLogin,
    isAdmin,
    login,
    fetchProfile,
    updateProfile,
    logout,
    setUserInfo,
    setToken,
  }
})
