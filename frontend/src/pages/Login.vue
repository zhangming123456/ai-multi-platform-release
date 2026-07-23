<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconLock, IconEmail } from '@arco-design/web-vue/es/icon'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const email = ref('admin@admin.com')
const password = ref('admin123')
const loading = ref(false)

async function handleLogin() {
  if (!email.value || !password.value) {
    Message.warning('请输入邮箱和密码')
    return
  }
  loading.value = true
  try {
    await userStore.login(email.value, password.value)
    Message.success('登录成功')
    router.push('/')
  } catch (e: any) {
    const msg = e.response?.data?.detail || '登录失败，请检查邮箱和密码'
    Message.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="bg-orbs">
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
    </div>

    <div class="login-card">
      <div class="login-header">
        <div class="logo">
          <svg viewBox="0 0 24 24" width="32" height="32" fill="none">
            <rect x="3" y="3" width="8" height="8" rx="2" fill="#007AFF" />
            <rect x="13" y="3" width="8" height="8" rx="2" fill="#5856D6" opacity="0.7" />
            <rect x="3" y="13" width="8" height="8" rx="2" fill="#34C759" opacity="0.7" />
            <rect x="13" y="13" width="8" height="8" rx="2" fill="#FF9500" opacity="0.5" />
          </svg>
        </div>
        <h1>Matrix</h1>
        <p>多平台矩阵管理系统</p>
      </div>

      <a-form layout="vertical" @submit.prevent="handleLogin">
        <a-form-item field="email" label="邮箱">
          <a-input v-model="email" placeholder="请输入邮箱" size="large" allow-clear>
            <template #prefix>
              <IconEmail />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item field="password" label="密码">
          <a-input-password
            v-model="password"
            placeholder="请输入密码"
            size="large"
            allow-clear
            @keyup.enter="handleLogin"
          >
            <template #prefix>
              <IconLock />
            </template>
          </a-input-password>
        </a-form-item>
        <a-button
          type="primary"
          long
          size="large"
          :loading="loading"
          class="login-btn"
          @click="handleLogin"
        >
          登录
        </a-button>
      </a-form>

      <div class="login-tip">
        默认账号：admin@admin.com / admin123
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: #f5f5f7;
}

.bg-orbs {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
}

.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(120px);
}

.orb-1 {
  top: -200px;
  right: -100px;
  width: 700px;
  height: 700px;
  background: radial-gradient(circle, #007aff 0%, transparent 70%);
  opacity: 0.15;
}

.orb-2 {
  top: 30%;
  left: -150px;
  width: 560px;
  height: 560px;
  background: radial-gradient(circle, #5856d6 0%, transparent 70%);
  opacity: 0.1;
}

.orb-3 {
  bottom: -150px;
  right: 30%;
  width: 480px;
  height: 480px;
  background: radial-gradient(circle, #5ac8fa 0%, transparent 70%);
  opacity: 0.08;
}

.login-card {
  position: relative;
  z-index: 1;
  width: 400px;
  max-width: calc(100vw - 32px);
  padding: 40px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(40px) saturate(180%);
  -webkit-backdrop-filter: blur(40px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 24px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.08);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  display: inline-flex;
  margin-bottom: 16px;
}

.login-header h1 {
  margin: 0 0 4px;
  font-size: 26px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.5px;
}

.login-header p {
  margin: 0;
  font-size: 13px;
  color: #86868b;
}

.login-btn {
  margin-top: 8px;
  height: 44px;
  border-radius: 12px;
  font-weight: 600;
  font-size: 15px;
}

.login-tip {
  margin-top: 20px;
  text-align: center;
  font-size: 12px;
  color: #a1a1a6;
}
</style>
