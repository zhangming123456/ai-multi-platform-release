import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { UserInfo } from '@/types'
import api from '@/utils/api'

interface LoginResponse {
  access_token: string
  token_type: string
  user: UserInfo
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<UserInfo | null>(null)

  async function login(username: string, password: string) {
    const res = await api.post<LoginResponse>('/auth/login', { username, password })
    token.value = res.data.access_token
    userInfo.value = res.data.user
    localStorage.setItem('token', res.data.access_token)
    await fetchUserInfo()
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
  }

  async function fetchUserInfo() {
    const res = await api.get<UserInfo>('/auth/me')
    userInfo.value = res.data
  }

  function hasPermission(key: string, mode: 'read' | 'write' = 'read'): boolean {
    if (!userInfo.value?.permissions) return true
    const access = userInfo.value.permissions[key]
    if (!access) return false
    return mode === 'read' ? access.read : access.write
  }

  return { token, userInfo, login, logout, fetchUserInfo, hasPermission }
})
