<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const isSidebarCollapsed = ref(false)
const isMobileSidebarOpen = ref(false)
const isMobile = ref(false)

const siderWidth = computed(() => (isSidebarCollapsed.value ? 64 : 248))

function checkMobile() {
  isMobile.value = window.innerWidth < 768
}

function toggleSidebar() {
  if (isMobile.value) {
    isMobileSidebarOpen.value = !isMobileSidebarOpen.value
  } else {
    isSidebarCollapsed.value = !isSidebarCollapsed.value
  }
}

function closeMobileSidebar() {
  isMobileSidebarOpen.value = false
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<template>
  <div class="min-h-screen bg-[#F5F5F7] overflow-x-hidden">
    <div class="fixed inset-0 -z-10 pointer-events-none overflow-hidden">
      <div
        class="absolute -top-48 -right-32 w-[700px] h-[700px] rounded-full blur-[120px] opacity-[0.12]"
        style="background: radial-gradient(circle, #007aff 0%, transparent 70%)"
      ></div>
      <div
        class="absolute top-1/4 -left-40 w-[560px] h-[560px] rounded-full blur-[120px] opacity-[0.08]"
        style="background: radial-gradient(circle, #5856d6 0%, transparent 70%)"
      ></div>
      <div
        class="absolute -bottom-32 right-1/3 w-[480px] h-[480px] rounded-full blur-[120px] opacity-[0.06]"
        style="background: radial-gradient(circle, #5ac8fa 0%, transparent 70%)"
      ></div>
    </div>

    <div
      v-if="isMobileSidebarOpen"
      class="fixed inset-0 bg-black/20 backdrop-blur-sm z-30 md:hidden animate-fade-in"
      @click="closeMobileSidebar"
    ></div>

    <div class="flex min-h-screen">
      <aside
        :class="[
          'flex flex-col transition-all duration-350 ease-out z-50',
          isSidebarCollapsed ? 'w-16' : 'w-[248px]',
          isMobileSidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0',
          'fixed inset-y-0 left-0',
        ]"
        :style="{ width: `${siderWidth}px`, maxWidth: `70vw` }"
      >
        <div class="h-full bg-white/80 backdrop-blur-xl border-r border-black/[0.05]">
          <AppSidebar
            :collapsed="isSidebarCollapsed"
            @toggle="toggleSidebar"
            @close-mobile="closeMobileSidebar"
          />
        </div>
      </aside>

      <div
        class="flex-1 flex flex-col transition-all duration-350 ease-out"
        style="max-width: 100vw;"
        :style="{ marginLeft: isMobile ? '0px' : `${isMobileSidebarOpen ? 0 : siderWidth}px` }"
      >
        <AppHeader :collapsed="isSidebarCollapsed" @toggle-sidebar="toggleSidebar" />

        <main
          class="flex-1 px-4 md:px-6 lg:px-8 py-4 md:py-6 lg:py-7 w-full max-w-[1400px] mx-auto"
        >
          <router-view />
        </main>
      </div>
    </div>
  </div>
</template>

<style scoped>
@media (max-width: 248px) {
  main {
    padding: 8px 10px !important;
  }
}
</style>
