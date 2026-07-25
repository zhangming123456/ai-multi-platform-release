<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconSafe,
  IconSettings,
  IconLink,
  IconExclamationCircle,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

interface RoleRef {
  id: string
  name: string
  display_name: string
}

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
  created_at: string
  updated_at: string
  parent_roles: RoleRef[]
  child_roles: RoleRef[]
}

interface Constraint {
  id: string
  name: string
  constraint_type: string
  is_active: boolean
  roles: {
    role_id: string
    role_name: string
    role_display_name: string
    association_type: string
  }[]
}

const loading = ref(false)
const roles = ref<Role[]>([])
const constraints = ref<Constraint[]>([])

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
}

const CUSTOM_COLORS = ['arcoblue', 'purple', 'cyan', 'orange', 'pink', 'gold', 'lime', 'magenta']

const roleColorMap = computed(() => {
  const map: Record<string, string> = {}
  let customIdx = 0
  for (const r of roles.value) {
    if (BUILTIN_COLORS[r.name]) {
      map[r.name] = BUILTIN_COLORS[r.name]
    } else {
      map[r.name] = CUSTOM_COLORS[customIdx % CUSTOM_COLORS.length]
      customIdx++
    }
  }
  return map
})

function roleColor(role: Role): string {
  return roleColorMap.value[role.name] || 'arcoblue'
}

function roleColorById(roleId: string): string {
  const role = roles.value.find((r) => r.id === roleId)
  return role ? roleColor(role) : 'arcoblue'
}

const sortedRoles = computed(() => {
  return [...roles.value].sort((a, b) => {
    if (a.is_super_admin) return -1
    if (b.is_super_admin) return 1
    const builtinOrder = ['manager', 'operator', 'reviewer']
    const aIdx = builtinOrder.indexOf(a.name)
    const bIdx = builtinOrder.indexOf(b.name)
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
    if (aIdx !== -1) return -1
    if (bIdx !== -1) return 1
    return 0
  })
})

async function fetchRoles() {
  loading.value = true
  try {
    const res = await api.get<Role[]>('/v2/roles')
    roles.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchConstraints() {
  try {
    const res = await api.get<Constraint[]>('/v2/constraints')
    constraints.value = Array.isArray(res.data) ? res.data : []
  } catch {}
}

onMounted(async () => {
  await Promise.all([fetchRoles(), fetchConstraints()])
})

const addVisible = ref(false)
const addSaving = ref(false)
const newRole = ref({
  name: '',
  display_name: '',
  description: '',
  role_type: 'other' as 'admin' | 'other',
  parent_role_ids: [] as string[],
})

const roleTypeOptions = [
  { value: 'admin', label: '管理类型' },
  { value: 'other', label: '普通类型' },
]

function openAdd() {
  newRole.value = {
    name: '',
    display_name: '',
    description: '',
    role_type: 'other',
    parent_role_ids: [],
  }
  addVisible.value = true
}

async function createRole() {
  if (!newRole.value.name.trim() || !newRole.value.display_name.trim()) {
    Message.warning('请填写角色标识和显示名称')
    return
  }
  addSaving.value = true
  try {
    await api.post<Role>('/v2/roles', newRole.value)
    Message.success('角色创建成功')
    addVisible.value = false
    await fetchRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    addSaving.value = false
  }
}

const editVisible = ref(false)
const editSaving = ref(false)
const editingRole = ref<Role | null>(null)
const editForm = ref({
  display_name: '',
  description: '',
  role_type: 'other' as 'admin' | 'other',
})

function openEdit(role: Role) {
  editingRole.value = role
  editForm.value = {
    display_name: role.display_name,
    description: role.description || '',
    role_type: (role.role_type as 'admin' | 'other') || 'other',
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingRole.value) return
  editSaving.value = true
  try {
    await api.put<Role>(`/v2/roles/${editingRole.value.id}`, editForm.value)
    Message.success('角色已更新')
    editVisible.value = false
    await fetchRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    editSaving.value = false
  }
}

function deleteRole(role: Role) {
  Modal.warning({
    title: '删除角色',
    content: `确定要删除角色「${role.display_name}」吗？该操作不可逆，已分配的权限和继承关系将被清理。`,
    hideCancel: false,
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.delete(`/v2/roles/${role.id}`)
        Message.success('角色已删除')
        await fetchRoles()
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '删除失败')
      }
    },
  })
}

const parentsVisible = ref(false)
const parentsRole = ref<Role | null>(null)
const parentsSaving = ref(false)
const selectedParents = ref<string[]>([])

function openParents(role: Role) {
  parentsRole.value = role
  selectedParents.value = role.parent_roles.map((p) => p.id)
  parentsVisible.value = true
}

async function saveParents() {
  if (!parentsRole.value) return
  parentsSaving.value = true
  try {
    const currentIds = new Set(parentsRole.value.parent_roles.map((p) => p.id))
    const nextIds = new Set(selectedParents.value)

    const toAdd = selectedParents.value.filter((id) => !currentIds.has(id))
    const toRemove = parentsRole.value.parent_roles.filter((p) => !nextIds.has(p.id))

    for (const parentId of toAdd) {
      await api.post(`/v2/roles/${parentsRole.value.id}/parents`, { id: parentId })
    }
    for (const parent of toRemove) {
      await api.delete(`/v2/roles/${parentsRole.value.id}/parents/${parent.id}`)
    }

    Message.success('父角色已更新')
    parentsVisible.value = false
    await fetchRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    parentsSaving.value = false
  }
}

function availableParentOptions(role: Role) {
  return roles.value.filter(
    (r) => r.id !== role.id && !r.is_super_admin,
  )
}

function constraintTypeLabel(type: string): string {
  const map: Record<string, string> = {
    mutual_exclusive: '互斥角色',
    prerequisite: '先决角色',
    cardinality: '基数约束',
  }
  return map[type] || type
}

function constraintTypeColor(type: string): string {
  const map: Record<string, string> = {
    mutual_exclusive: 'red',
    prerequisite: 'arcoblue',
    cardinality: 'purple',
  }
  return map[type] || 'gray'
}

function constraintsForRole(roleId: string): Constraint[] {
  return constraints.value.filter(
    (c) => c.is_active && c.roles.some((r) => r.role_id === roleId),
  )
}

function formatConstraintRoles(constraint: Constraint, roleId: string): string {
  const others = constraint.roles
    .filter((r) => r.role_id !== roleId)
    .map((r) => r.role_display_name || r.role_name)
  return others.length > 0 ? others.join('、') : '—'
}
</script>

<template>
  <div class="page-main">
    <PageHeader title="角色设置" subtitle="管理系统角色、自定义角色与角色继承关系">
      <template #actions>
        <a-button type="primary" @click="openAdd">
          <template #icon><IconPlus /></template>
          创建角色
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="role in sortedRoles"
          :key="role.id"
          class="role-card"
        >
          <div class="p-5">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <a-tag :color="roleColor(role)" size="small" class="!m-0">
                  {{ role.display_name }}
                </a-tag>
                <IconSafe v-if="role.is_super_admin" :size="14" class="text-[#ff9500] shrink-0" />
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <a-button type="text" size="mini" :disabled="role.is_super_admin" @click="openEdit(role)">
                  <template #icon><IconEdit :size="14" /></template>
                </a-button>
                <a-button
                  type="text"
                  size="mini"
                  status="danger"
                  :disabled="role.is_builtin"
                  @click="deleteRole(role)"
                >
                  <template #icon><IconDelete :size="14" /></template>
                </a-button>
              </div>
            </div>

            <p class="text-[13px] text-[#86868B] leading-relaxed mb-4">
              {{ role.description || '暂无描述' }}
            </p>

            <div class="flex flex-wrap items-center gap-2 mb-4">
              <code class="role-code">{{ role.name }}</code>
              <a-tag v-if="role.is_super_admin" size="small" color="orangered" class="!m-0">超级</a-tag>
              <a-tag v-else-if="role.is_builtin" size="small" color="gray" class="!m-0">内置</a-tag>
              <a-tag v-else size="small" color="arcoblue" class="!m-0">自定义</a-tag>
              <a-tag
                v-if="role.role_type === 'admin'"
                size="small"
                color="orangered"
                class="!m-0"
              >管理类型</a-tag>
              <a-tag v-else size="small" color="gray" class="!m-0">普通类型</a-tag>
            </div>

            <div class="space-y-2">
              <div v-if="role.parent_roles.length > 0" class="flex items-start gap-2">
                <span class="text-[11px] text-[#86868B] shrink-0 mt-0.5">继承自</span>
                <div class="flex flex-wrap gap-1.5">
                  <a-tag
                    v-for="parent in role.parent_roles"
                    :key="parent.id"
                    size="small"
                    :color="roleColorById(parent.id)"
                    class="!m-0"
                  >
                    {{ parent.display_name }}
                  </a-tag>
                </div>
              </div>

              <div v-if="role.child_roles.length > 0" class="flex items-start gap-2">
                <span class="text-[11px] text-[#86868B] shrink-0 mt-0.5">被继承</span>
                <div class="flex flex-wrap gap-1.5">
                  <a-tag
                    v-for="child in role.child_roles"
                    :key="child.id"
                    size="small"
                    :color="roleColorById(child.id)"
                    class="!m-0"
                  >
                    {{ child.display_name }}
                  </a-tag>
                </div>
              </div>

              <div v-if="constraintsForRole(role.id).length > 0" class="pt-2 mt-2 border-t border-black/[0.04]">
                <div class="flex items-center gap-1.5 mb-1.5">
                  <IconExclamationCircle :size="12" class="text-[#ff9500]" />
                  <span class="text-[11px] text-[#86868B]">相关约束</span>
                </div>
                <div class="flex flex-col gap-1">
                  <div
                    v-for="c in constraintsForRole(role.id)"
                    :key="c.id"
                    class="flex items-center gap-2"
                  >
                    <a-tag size="small" :color="constraintTypeColor(c.constraint_type)" class="!m-0">
                      {{ constraintTypeLabel(c.constraint_type) }}
                    </a-tag>
                    <span class="text-[11px] text-[#86868B] truncate">
                      {{ formatConstraintRoles(c, role.id) }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="role-footer">
            <div class="flex items-center justify-between">
              <span class="text-[12px] text-[#86868B]">
                {{ role.is_super_admin ? '超级管理员拥有所有权限' : '配置继承关系与权限' }}
              </span>
              <a-space :size="6">
                <a-button
                  type="text"
                  size="small"
                  :disabled="role.is_super_admin"
                  @click="openParents(role)"
                >
                  <template #icon><IconLink :size="14" /></template>
                  继承
                </a-button>
                <a-button type="text" size="small" @click="$router.push(`/rbac/permissions?role=${role.id}`)">
                  <template #icon><IconSettings :size="14" /></template>
                  权限
                </a-button>
              </a-space>
            </div>
          </div>
        </div>
      </div>

      <a-empty v-if="!loading && roles.length === 0" description="暂无角色数据" class="mt-20" />
    </a-spin>

    <a-modal
      v-model:visible="addVisible"
      title="创建自定义角色"
      :width="480"
      :ok-loading="addSaving"
      @ok="createRole"
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
          <a-textarea
            v-model="newRole.description"
            placeholder="角色职责说明"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>
        <a-form-item label="角色类型" required>
          <a-radio-group v-model="newRole.role_type" :options="roleTypeOptions" />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              {{ newRole.role_type === 'admin' ? '管理类型角色可配置数据库等高级权限' : '普通类型角色不支持数据库相关权限' }}
            </span>
          </template>
        </a-form-item>
        <a-form-item label="父角色（可选）">
          <a-select
            v-model="newRole.parent_role_ids"
            placeholder="选择父角色以继承其权限"
            multiple
            :options="roles.filter((r) => !r.is_super_admin).map((r) => ({ value: r.id, label: r.display_name }))"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="editVisible"
      title="编辑角色"
      :width="480"
      :ok-loading="editSaving"
      @ok="saveEdit"
      ok-text="保存"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="显示名称" required>
          <a-input v-model="editForm.display_name" placeholder="角色中文名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea
            v-model="editForm.description"
            placeholder="角色职责说明"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>
        <a-form-item label="角色类型" required>
          <a-radio-group v-model="editForm.role_type" :options="roleTypeOptions" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="parentsVisible"
      title="配置父角色"
      :width="480"
      :ok-loading="parentsSaving"
      @ok="saveParents"
      ok-text="保存"
    >
      <a-form v-if="parentsRole" :model="{ selectedParents }" layout="vertical">
        <a-form-item label="选择父角色">
          <a-select
            v-model="selectedParents"
            placeholder="选择父角色以继承其权限"
            multiple
            :options="availableParentOptions(parentsRole).map((r) => ({ value: r.id, label: r.display_name }))"
          />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">子角色将自动继承所选父角色的权限</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.role-card {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 20px;
  overflow: hidden;
  transition:
    transform 0.22s cubic-bezier(0.25, 0.1, 0.25, 1),
    box-shadow 0.22s,
    border-color 0.22s;
}
.role-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 36px -18px rgba(0, 0, 0, 0.18);
  border-color: rgba(0, 0, 0, 0.1);
}
.role-code {
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.04);
  color: #636366;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}
.role-footer {
  padding: 12px 20px;
  background: rgba(0, 0, 0, 0.01);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}
</style>
