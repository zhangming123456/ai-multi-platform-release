<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconSafe,
  IconSettings,
  IconLock,
  IconArrowUp,
  IconArrowDown,
  IconUserGroup,
  IconApps,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { usePermissionStore, type RBACRole } from '@/stores/permission'
import api from '@/utils/api'

const permStore = usePermissionStore()

const loading = ref(false)
const activeTab = ref<'list' | 'hierarchy'>('list')
const selectedRole = ref<RBACRole | null>(null)

const addVisible = ref(false)
const addSaving = ref(false)
const newRole = ref({
  name: '',
  display_name: '',
  description: '',
  role_type: 'other',
  scope: 'system',
})

const editVisible = ref(false)
const editSaving = ref(false)
const editingRole = ref<RBACRole | null>(null)
const editForm = ref({
  display_name: '',
  description: '',
  role_type: 'other',
  is_active: true,
})

const BUILTIN_COLORS: Record<string, string> = {
  super_admin: 'red',
  admin: 'orangered',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
  auditor: 'cyan',
}

const RESERVED_ROLE_NAMES = new Set([
  'super_admin',
  'admin',
  'manager',
  'operator',
  'reviewer',
  'auditor',
])

function roleColor(role: RBACRole): string {
  return BUILTIN_COLORS[role.name] || 'arcoblue'
}

function roleTypeLabel(roleType: string): string {
  return roleType === 'admin' ? '管理类型' : '普通类型'
}

function roleTypeColor(roleType: string): string {
  return roleType === 'admin' ? 'orangered' : 'gray'
}

async function loadRoles() {
  loading.value = true
  try {
    await permStore.fetchRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRoles()
})

async function createRole() {
  if (!newRole.value.name || !newRole.value.display_name) return
  if (RESERVED_ROLE_NAMES.has(newRole.value.name.trim())) {
    Message.warning(`"${newRole.value.name}" 为系统保留角色标识，不可创建`)
    return
  }
  addSaving.value = true
  try {
    await api.post('/v2/roles', newRole.value)
    Message.success('角色创建成功')
    addVisible.value = false
    newRole.value = { name: '', display_name: '', description: '', role_type: 'other', scope: 'system' }
    await loadRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    addSaving.value = false
  }
}

function openEdit(role: RBACRole) {
  editingRole.value = role
  editForm.value = {
    display_name: role.display_name,
    description: role.description || '',
    role_type: role.role_type,
    is_active: role.is_active,
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingRole.value) return
  editSaving.value = true
  try {
    await api.put(`/v2/roles/${editingRole.value.id}`, editForm.value)
    Message.success('角色已更新')
    editVisible.value = false
    await loadRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

async function deleteRole(role: RBACRole) {
  try {
    await api.delete(`/v2/roles/${role.id}`)
    Message.success('角色已删除')
    await loadRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}

const roleAncestors = ref<RBACRole[]>([])
const roleDescendants = ref<RBACRole[]>([])
const hierarchyLoading = ref(false)

async function selectRole(role: RBACRole) {
  selectedRole.value = role
  if (activeTab.value === 'hierarchy') {
    await loadRoleHierarchy(role.id)
  }
}

async function loadRoleHierarchy(roleId: string) {
  hierarchyLoading.value = true
  try {
    const [ancestorsRes, descendantsRes] = await Promise.all([
      api.get<RBACRole[]>(`/v2/roles/${roleId}/ancestors`),
      api.get<RBACRole[]>(`/v2/roles/${roleId}/descendants`),
    ])
    roleAncestors.value = ancestorsRes.data
    roleDescendants.value = descendantsRes.data
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色层级失败')
  } finally {
    hierarchyLoading.value = false
  }
}

const addParentVisible = ref(false)
const availableParents = ref<RBACRole[]>([])
const selectedParentId = ref('')

async function openAddParent(role: RBACRole) {
  selectedRole.value = role
  selectedParentId.value = ''
  availableParents.value = permStore.roles.filter(
    r => r.id !== role.id && !roleDescendants.value.some(d => d.id === r.id)
  )
  addParentVisible.value = true
}

async function addParentRole() {
  if (!selectedParentId.value || !selectedRole.value) return
  try {
    await api.post(`/v2/roles/${selectedRole.value.id}/parents`, {
      parent_role_id: selectedParentId.value,
    })
    Message.success('父角色添加成功')
    addParentVisible.value = false
    await loadRoleHierarchy(selectedRole.value.id)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '添加失败')
  }
}

async function removeParentRole(parentId: string) {
  if (!selectedRole.value) return
  try {
    await api.delete(`/v2/roles/${selectedRole.value.id}/parents/${parentId}`)
    Message.success('父角色已移除')
    await loadRoleHierarchy(selectedRole.value.id)
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '移除失败')
  }
}

const sortedRoles = computed(() => {
  return [...permStore.roles].sort((a, b) => {
    if (a.is_super_admin) return -1
    if (b.is_super_admin) return 1
    if (a.is_system && !b.is_system) return -1
    if (!a.is_system && b.is_system) return 1
    return a.display_name.localeCompare(b.display_name)
  })
})
</script>

<template>
  <div>
    <PageHeader title="角色管理 (RBAC3)" subtitle="管理系统角色及其层级继承关系，支持静态/动态职责分离">
      <template #actions>
        <a-space>
          <a-radio-group v-model="activeTab" type="button" size="small">
            <a-radio value="list">
              <template #icon><IconUserGroup /></template>
              列表视图
            </a-radio>
            <a-radio value="hierarchy">
              <template #icon><IconApps /></template>
              层级视图
            </a-radio>
          </a-radio-group>
          <a-button type="primary" @click="addVisible = true">
            <template #icon><IconPlus /></template>
            创建角色
          </a-button>
        </a-space>
      </template>
    </PageHeader>

    <a-spin :loading="loading" class="w-full">
      <div v-if="activeTab === 'list'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="role in sortedRoles"
          :key="role.id"
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden hover:border-black/[0.1] transition-all duration-200"
        >
          <div class="p-5">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-2.5">
                <a-tag :color="roleColor(role)" size="small" class="!m-0">
                  {{ role.display_name }}
                </a-tag>
                <IconSafe v-if="role.is_super_admin" :size="14" class="text-[#ff9500]" />
              </div>
              <div v-if="!role.is_system" class="flex items-center gap-1">
                <a-button type="text" size="mini" @click="openEdit(role)">
                  <template #icon><IconEdit :size="14" /></template>
                </a-button>
                <a-popconfirm content="确定要删除该角色吗？" @ok="deleteRole(role)">
                  <a-button type="text" size="mini" status="danger">
                    <template #icon><IconDelete :size="14" /></template>
                  </a-button>
                </a-popconfirm>
              </div>
              <div v-else class="text-[11px] text-[#86868B] font-medium">
                系统内置，不可编辑
              </div>
            </div>

            <p class="text-[13px] text-[#86868B] leading-relaxed mb-4">
              {{ role.description || '暂无描述' }}
            </p>

            <div class="flex flex-wrap items-center gap-2">
              <code class="text-[12px] px-2 py-1 rounded-md bg-black/[0.04] text-[#636366] font-mono">
                {{ role.name }}
              </code>
              <a-tag v-if="role.is_super_admin" size="small" color="red" class="!m-0">超级</a-tag>
              <a-tag v-else-if="role.is_system" size="small" color="gray" class="!m-0">内置</a-tag>
              <a-tag v-else size="small" color="arcoblue" class="!m-0">自定义</a-tag>
              <a-tag :color="roleTypeColor(role.role_type)" size="small" class="!m-0">
                {{ roleTypeLabel(role.role_type) }}
              </a-tag>
            </div>
          </div>

          <div
            class="px-5 py-3 bg-black/[0.01] border-t border-black/[0.04] flex items-center justify-between"
          >
            <span class="text-[12px] text-[#86868B]">配置角色权限与继承关系</span>
            <a-space size="mini">
              <a-button type="text" size="small" @click="selectRole(role); activeTab = 'hierarchy'">
                <template #icon><IconApps :size="14" /></template>
                层级
              </a-button>
              <a-button type="text" size="small">
                <template #icon><IconSettings :size="14" /></template>
                权限
              </a-button>
            </a-space>
          </div>
        </div>
      </div>

      <div v-else class="flex flex-col lg:flex-row gap-5">
        <div class="lg:w-[280px] shrink-0">
          <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-3">
            <p class="text-[12px] text-[#86868B] font-medium px-3 pb-2 pt-1">选择角色</p>
            <div
              v-for="role in sortedRoles"
              :key="role.id"
              :class="[
                'flex items-center gap-2.5 px-3 py-2.5 rounded-xl cursor-pointer transition-all duration-200 mb-1',
                selectedRole?.id === role.id
                  ? 'bg-[#007aff]/[0.08] text-[#007aff]'
                  : 'hover:bg-black/[0.03] text-[#1D1D1F]',
              ]"
              @click="selectRole(role)"
            >
              <a-tag :color="roleColor(role)" size="small" class="!m-0">
                {{ role.display_name }}
              </a-tag>
              <IconLock v-if="role.is_super_admin" :size="13" class="text-[#ff9500]" />
            </div>
          </div>
        </div>

        <div class="flex-1 min-w-0">
          <a-spin :loading="hierarchyLoading">
            <div v-if="selectedRole" class="space-y-5">
              <div
                class="bg-[#007aff]/[0.06] border border-[#007aff]/[0.15] rounded-2xl p-5 flex items-start gap-3"
              >
                <div
                  class="w-10 h-10 rounded-xl bg-[#007aff]/[0.12] flex items-center justify-center shrink-0"
                >
                  <IconUserGroup :size="18" class="text-[#007aff]" />
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-[15px] font-semibold text-[#1D1D1F]">
                      {{ selectedRole.display_name }}
                    </span>
                    <a-tag v-if="selectedRole.is_system" size="small" color="gray">内置</a-tag>
                    <a-tag :color="roleTypeColor(selectedRole.role_type)" size="small">
                      {{ roleTypeLabel(selectedRole.role_type) }}
                    </a-tag>
                  </div>
                  <p class="text-[12px] text-[#86868B] font-mono">{{ selectedRole.name }}</p>
                  <p class="text-[13px] text-[#636366] mt-2">
                    {{ selectedRole.description || '暂无描述' }}
                  </p>
                </div>
              </div>

              <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
                <div class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                  <div class="flex items-center gap-2">
                    <IconArrowUp :size="16" class="text-[#30d158]" />
                    <span class="text-[14px] font-semibold text-[#1D1D1F]">父角色（继承权限）</span>
                    <a-tag size="small" color="green" class="!m-0">{{ roleAncestors.length }}</a-tag>
                  </div>
                  <a-button v-if="!selectedRole.is_system" type="outline" size="small" @click="openAddParent(selectedRole)">
                    <template #icon><IconPlus /></template>
                    添加父角色
                  </a-button>
                </div>
                <div class="p-4">
                  <div v-if="roleAncestors.length === 0" class="text-center py-8 text-[#86868B] text-[13px]">
                    暂无父角色
                  </div>
                  <div v-else class="flex flex-wrap gap-2">
                    <div
                      v-for="ancestor in roleAncestors"
                      :key="ancestor.id"
                      class="flex items-center gap-2 px-3 py-2 rounded-xl bg-[#30d158]/[0.06] border border-[#30d158]/[0.15]"
                    >
                      <span class="text-[13px] font-medium text-[#1D1D1F]">{{ ancestor.display_name }}</span>
                      <a-button
                        v-if="!selectedRole.is_system"
                        type="text"
                        size="mini"
                        status="danger"
                        @click="removeParentRole(ancestor.id)"
                      >
                        <template #icon><IconDelete :size="12" /></template>
                      </a-button>
                    </div>
                  </div>
                </div>
              </div>

              <div class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden">
                <div class="flex items-center justify-between px-5 py-3.5 border-b border-black/[0.04] bg-black/[0.01]">
                  <div class="flex items-center gap-2">
                    <IconArrowDown :size="16" class="text-[#ff9500]" />
                    <span class="text-[14px] font-semibold text-[#1D1D1F]">子角色（被继承）</span>
                    <a-tag size="small" color="orangered" class="!m-0">{{ roleDescendants.length }}</a-tag>
                  </div>
                </div>
                <div class="p-4">
                  <div v-if="roleDescendants.length === 0" class="text-center py-8 text-[#86868B] text-[13px]">
                    暂无子角色
                  </div>
                  <div v-else class="flex flex-wrap gap-2">
                    <div
                      v-for="descendant in roleDescendants"
                      :key="descendant.id"
                      class="flex items-center gap-2 px-3 py-2 rounded-xl bg-[#ff9500]/[0.06] border border-[#ff9500]/[0.15]"
                    >
                      <span class="text-[13px] font-medium text-[#1D1D1F]">{{ descendant.display_name }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="flex items-center justify-center py-20 text-[#86868B]">
              请从左侧选择一个角色查看层级关系
            </div>
          </a-spin>
        </div>
      </div>

      <a-empty v-if="!loading && sortedRoles.length === 0" description="暂无角色数据" class="mt-20" />
    </a-spin>

    <a-modal
      v-model:visible="addVisible"
      title="创建角色"
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
        <a-form-item label="角色类型" required>
          <a-radio-group v-model="newRole.role_type">
            <a-radio value="admin">管理类型</a-radio>
            <a-radio value="other">普通类型</a-radio>
          </a-radio-group>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              {{ newRole.role_type === 'admin' ? '管理类型角色可配置数据库等高级权限' : '普通类型角色不支持数据库相关权限' }}
            </span>
          </template>
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
        <a-form-item label="角色类型">
          <a-radio-group v-model="editForm.role_type" :disabled="editingRole?.is_system">
            <a-radio value="admin">管理类型</a-radio>
            <a-radio value="other">普通类型</a-radio>
          </a-radio-group>
          <template v-if="editingRole?.is_system" #extra>
            <span class="text-[11px] text-[#86868b]">系统内置角色不可修改类型</span>
          </template>
        </a-form-item>
        <a-form-item label="状态">
          <a-switch v-model="editForm.is_active" :disabled="editingRole?.is_system" />
          <span class="ml-3 text-[13px] text-[#636366]">
            {{ editForm.is_active ? '启用' : '禁用' }}
          </span>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="addParentVisible"
      title="添加父角色"
      :width="420"
      @ok="addParentRole"
      ok-text="添加"
    >
      <a-form :model="{}" layout="vertical">
        <a-form-item label="选择父角色">
          <a-select v-model="selectedParentId" placeholder="选择要继承的父角色">
            <a-option v-for="role in availableParents" :key="role.id" :value="role.id">
              {{ role.display_name }} ({{ role.name }})
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">该角色将自动继承父角色的所有权限</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
