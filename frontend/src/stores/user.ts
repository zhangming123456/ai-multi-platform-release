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

  async function login(email: string, password: string) {
    const res = await api.post<LoginResponse>('/auth/login', { email, password })
    token.value = res.data.access_token
    userInfo.value = res.data.user
    localStorage.setItem('token', res.data.access_token)
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

  return { token, userInfo, login, logout, fetchUserInfo }
})
