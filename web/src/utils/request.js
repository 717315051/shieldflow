import axios from 'axios'
import { message } from 'ant-design-vue'

const service = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

// 请求拦截器：注入 JWT
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('shieldflow_token')
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

// 响应拦截器：统一处理 code/message
service.interceptors.response.use(
  (response) => {
    const res = response.data
    // 文件流直接返回
    if (response.config.responseType === 'blob') {
      return response
    }
    if (res.code === 0 || res.code === 200) {
      return res
    }
    // token 失效
    if (res.code === 401 || res.code === 403) {
      message.error(res.message || '登录已失效，请重新登录')
      localStorage.removeItem('shieldflow_token')
      localStorage.removeItem('shieldflow_user')
      window.location.href = '/login'
      return Promise.reject(new Error(res.message || 'Unauthorized'))
    }
    message.error(res.message || '请求失败')
    return Promise.reject(new Error(res.message || 'Error'))
  },
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.message || error.message
    if (status === 401 || status === 403) {
      message.error('登录已失效，请重新登录')
      localStorage.removeItem('shieldflow_token')
      localStorage.removeItem('shieldflow_user')
      window.location.href = '/login'
    } else {
      message.error(msg || '网络异常')
    }
    return Promise.reject(error)
  },
)

export default service
