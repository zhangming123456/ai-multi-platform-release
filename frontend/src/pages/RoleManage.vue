<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete, IconSafe, IconSettings } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

interface RoleDef {
  name: string
  display_name: string
  description: string | null
  is_builtin: boolean
}

const router = useRouter()
const loading = ref(false)
const roles = ref<RoleDef[]>([])

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  operator: 'blue',
  reviewer: 'green',
}

function roleColor(name: string): string {
  return BUILTIN_COLORS[name] || 'arcoblue'
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get<RoleDef[]>('/roles')
    roles.value = res.data
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色列表失败')
  } finally {
    loading.value = false
  }
})

const addVisible = ref(false)
const addSaving = ref(false)
const newRole = ref({ name: '', display_name: '', description: '' })

async function createRole() {
  if (!newRole.value.name || !newRole.value.display_name) return
  addSaving.value = true
  try {
    const res = await api.post<RoleDef>('/roles', newRole.value)
    roles.value.push(res.data)
    Message.success('角色创建成功')
    addVisible.value = false
    newRole.value = { name: '', display_name: '', description: '' }
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    addSaving.value = false
  }
}

const editVisible = ref(false)
const editSaving = ref(false)
const editingRole = ref<RoleDef | null>(null)
const editForm = ref({ display_name: '', description: '' })

function openEdit(role: RoleDef) {
  editingRole.value = role
  editForm.value = { display_name: role.display_name, description: role.description || '' }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingRole.value) return
  editSaving.value = true
  try {
    const res = await api.put<RoleDef>(`/roles/${editingRole.value.name}`, editForm.value)
    const idx = roles.value.findIndex((r) => r.name === editingRole.value!.name)
    if (idx !== -1) roles.value[idx] = res.data
    Message.success('角色已更新')
    editVisible.value = false
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

async function deleteRole(role: RoleDef) {
  try {
    await api.delete(`/roles/${role.name}`)
    roles.value = roles.value.filter((r) => r.name !== role.name)
    Message.success('角色已删除')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}

function goToPermissions(roleName: string) {
  router.push({ path: '/settings/permissions', query: { role: roleName } })
}
</script>

<template>
  <div>
    <PageHeader title="角色管理" subtitle="管理系统内置角色与自定义角色，管理员可创建和管理自定义角色">
      <template #actions>
        <a-button type="primary" @click="addVisible = true">
          <template #icon><IconPlus /></template>
          创建角色
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" class="w-full">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="role in roles"
          :key="role.name"
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden hover:border-black/[0.1] transition-all duration-200"
        >
          <div class="p-5">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-2.5">
                <a-tag :color="roleColor(role.name)" size="small" class="!m-0">
                  {{ role.display_name }}
                </a-tag>
                <IconSafe v-if="role.name === 'admin'" :size="14" class="text-[#86868B]" />
              </div>
              <div v-if="!role.is_builtin" class="flex items-center gap-1">
                <a-button type="text" size="mini" @click="openEdit(role)">
                  <template #icon><IconEdit :size="14" /></template>
                </a-button>
                <a-popconfirm
                  content="确定要删除该角色吗？该操作不可逆。"
                  @ok="deleteRole(role)"
                >
                  <a-button type="text" size="mini" status="danger">
                    <template #icon><IconDelete :size="14" /></template>
                  </a-button>
                </a-popconfirm>
              </div>
            </div>

            <p class="text-[13px] text-[#86868B] leading-relaxed mb-4">
              {{ role.description || '暂无描述' }}
            </p>

            <div class="flex items-center gap-2">
              <code class="text-[12px] px-2 py-1 rounded-md bg-black/[0.04] text-[#636366] font-mono">
                {{ role.name }}
              </code>
              <a-tag v-if="role.is_builtin" size="small" color="gray" class="!m-0">内置</a-tag>
              <a-tag v-else size="small" color="arcoblue" class="!m-0">自定义</a-tag>
            </div>
          </div>

          <div
            class="px-5 py-3 bg-black/[0.01] border-t border-black/[0.04] flex items-center justify-between"
          >
            <span class="text-[12px] text-[#86868B]">配置该角色的页面和操作权限</span>
            <a-button type="text" size="small" @click="goToPermissions(role.name)">
              <template #icon><IconSettings :size="14" /></template>
              权限配置
            </a-button>
          </div>
        </div>
      </div>

      <a-empty v-if="!loading && roles.length === 0" description="暂无角色数据" class="mt-20" />
    </a-spin>

    <a-modal
      v-model:visible="addVisible"
      title="创建自定义角色"
      :width="480"
      @ok="createRole"
      :ok-loading="addSaving"
      ok-text="创建"
    >
      <a-form :model="newRole" layout="vertical">
        <a-form-item label="角色标识" required>
          <a-input v-model="newRole.name" placeholder="英文标识，如 editor、viewer" />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">仅支持英文、数字和下划线，创建后不可修改</span>
          </template>
        </a-form-item>
        <a-form-item label="显示名称" required>
          <a-input v-model="newRole.display_name" placeholder="角色中文名称" />
        </a-form-item>
        <a-form-item label="描述（选填）">
          <a-textarea v-model="newRole.description" placeholder="角色职责说明" :auto-size="{ minRows: 2, maxRows: 4 }" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="editVisible"
      title="编辑角色"
      :width="480"
      @ok="saveEdit"
      :ok-loading="editSaving"
      ok-text="保存"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="显示名称">
          <a-input v-model="editForm.display_name" placeholder="角色中文名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model="editForm.description" placeholder="角色职责说明" :auto-size="{ minRows: 2, maxRows: 4 }" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
