import axios from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types'
import router from '@/router'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 20000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => {
    const body = res.data as ApiResponse
    if (body.code !== 0) {
      ElMessage.error(body.message || '请求失败')
      return Promise.reject(new Error(body.message))
    }
    return res
  },
  (err) => {
    const status = err.response?.status
    const msg = err.response?.data?.message || err.message || '网络错误'
    if (status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (router.currentRoute.value.path !== '/login') {
        router.push({ path: '/login', query: { redirect: router.currentRoute.value.fullPath } })
      }
    }
    ElMessage.error(msg)
    return Promise.reject(err)
  },
)

export async function request<T>(fn: () => Promise<{ data: ApiResponse<T> }>): Promise<T> {
  const res = await fn()
  return res.data.data
}

export default http
