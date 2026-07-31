<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconExclamationCircle,
  IconLink,
  IconUser,
} from '@arco-design/web-vue/es/icon'
import { Message, Modal } from '@arco-design/web-vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { formatDateTime } from '@/utils/time'
import api from '@/utils/api'

interface Role {
  id: string
  name: string
  display_name: string
  description: string | null
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface Association {
  id: string
  role_id: string
  role_name: string
  role_display_name: string
  association_type: string
}

interface Constraint {
  id: string
  name: string
  description: string | null
  constraint_type: 'mutual_exclusive' | 'prerequisite' | 'cardinality'
  config: Record<string, any>
  is_active: boolean
  created_at: string
  roles: Association[]
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
  let idx = 0
  for (const r of roles.value) {
    map[r.id] = BUILTIN_COLORS[r.name] || CUSTOM_COLORS[idx++ % CUSTOM_COLORS.length]
  }
  return map
})

function roleColor(roleId: string): string {
  return roleColorMap.value[roleId] || 'arcoblue'
}

async function fetchRoles() {
  try {
    const res = await api.get<Role[]>('/v2/roles')
    roles.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载角色列表失败')
  }
}

async function fetchConstraints() {
  loading.value = true
  try {
    const res = await api.get<Constraint[]>('/v2/constraints')
    constraints.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载约束失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchRoles()
  await fetchConstraints()
})

function typeLabel(type: string): string {
  const map: Record<string, string> = {
    mutual_exclusive: '互斥角色',
    prerequisite: '先决角色',
    cardinality: '基数约束',
  }
  return map[type] || type
}

function typeColor(type: string): string {
  const map: Record<string, string> = {
    mutual_exclusive: 'red',
    prerequisite: 'arcoblue',
    cardinality: 'purple',
  }
  return map[type] || 'gray'
}

function formatConfig(constraint: Constraint): string {
  const { config, constraint_type } = constraint
  if (constraint_type === 'mutual_exclusive') {
    return `作用域：${config?.scope === 'dynamic' ? '动态' : '静态'}`
  }
  if (constraint_type === 'prerequisite') {
    return `先决要求：${config?.require_all ? '拥有全部先决角色' : '至少拥有一个先决角色'}`
  }
  if (constraint_type === 'cardinality') {
    return `最大用户数：${config?.max_users ?? '未设置'}`
  }
  return ''
}

function subjectRoles(constraint: Constraint): Association[] {
  return constraint.roles.filter((r) => r.association_type === 'subject')
}

function prerequisiteRoles(constraint: Constraint): Association[] {
  return constraint.roles.filter((r) => r.association_type === 'prerequisite')
}

const modalVisible = ref(false)
const modalSaving = ref(false)
const isEdit = computed(() => !!editingConstraint.value)

const editingConstraint = ref<Constraint | null>(null)
const form = ref({
  name: '',
  description: '',
  constraint_type: 'mutual_exclusive' as Constraint['constraint_type'],
  is_active: true,
  scope: 'static',
  require_all: true,
  max_users: 1,
  subject_role_ids: [] as string[],
  prerequisite_role_ids: [] as string[],
})

const availableRoles = computed(() => roles.value.filter((r) => !r.is_super_admin))

const roleOptions = computed(() =>
  availableRoles.value.map((r) => ({ value: r.id, label: r.display_name })),
)

function resetForm() {
  form.value = {
    name: '',
    description: '',
    constraint_type: 'mutual_exclusive',
    is_active: true,
    scope: 'static',
    require_all: true,
    max_users: 1,
    subject_role_ids: [],
    prerequisite_role_ids: [],
  }
}

function openAdd() {
  editingConstraint.value = null
  resetForm()
  modalVisible.value = true
}

function openEdit(constraint: Constraint) {
  editingConstraint.value = constraint
  form.value = {
    name: constraint.name,
    description: constraint.description || '',
    constraint_type: constraint.constraint_type,
    is_active: constraint.is_active,
    scope: constraint.config?.scope || 'static',
    require_all: constraint.config?.require_all ?? true,
    max_users: constraint.config?.max_users ?? 1,
    subject_role_ids: subjectRoles(constraint).map((r) => r.role_id),
    prerequisite_role_ids: prerequisiteRoles(constraint).map((r) => r.role_id),
  }
  modalVisible.value = true
}

function buildPayload() {
  const associations: { role_id: string; association_type: string }[] = []
  for (const roleId of form.value.subject_role_ids) {
    associations.push({ role_id: roleId, association_type: 'subject' })
  }
  if (form.value.constraint_type === 'prerequisite') {
    for (const roleId of form.value.prerequisite_role_ids) {
      associations.push({ role_id: roleId, association_type: 'prerequisite' })
    }
  }
  const config: Record<string, any> = {}
  if (form.value.constraint_type === 'mutual_exclusive') {
    config.scope = form.value.scope
  } else if (form.value.constraint_type === 'prerequisite') {
    config.require_all = form.value.require_all
  } else if (form.value.constraint_type === 'cardinality') {
    config.max_users = Number(form.value.max_users)
  }
  return {
    name: form.value.name,
    description: form.value.description || null,
    constraint_type: form.value.constraint_type,
    is_active: form.value.is_active,
    config,
    role_associations: associations,
  }
}

function validateForm(): boolean {
  if (!form.value.name.trim()) {
    Message.warning('请输入约束名称')
    return false
  }
  if (form.value.subject_role_ids.length === 0) {
    Message.warning('请至少选择一个作用角色')
    return false
  }
  if (form.value.constraint_type === 'mutual_exclusive' && form.value.subject_role_ids.length < 2) {
    Message.warning('互斥约束至少需要两个作用角色')
    return false
  }
  if (
    form.value.constraint_type === 'prerequisite' &&
    form.value.prerequisite_role_ids.length === 0
  ) {
    Message.warning('请至少选择一个先决角色')
    return false
  }
  return true
}

async function saveConstraint() {
  if (!validateForm()) return
  modalSaving.value = true
  try {
    const payload = buildPayload()
    if (isEdit.value && editingConstraint.value) {
      await api.put(`/v2/constraints/${editingConstraint.value.id}`, payload)
      Message.success('约束已更新')
    } else {
      await api.post('/v2/constraints', payload)
      Message.success('约束已创建')
    }
    modalVisible.value = false
    await fetchConstraints()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '保存失败')
  } finally {
    modalSaving.value = false
  }
}

async function toggleActive(constraint: Constraint, next: boolean) {
  try {
    await api.put(`/v2/constraints/${constraint.id}`, { is_active: next })
    constraint.is_active = next
    Message.success('状态已更新')
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  }
}

function removeConstraint(constraint: Constraint) {
  Modal.warning({
    title: '删除约束',
    content: `确定要删除约束「${constraint.name}」吗？该操作不可逆。`,
    hideCancel: false,
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.delete(`/v2/constraints/${constraint.id}`)
        constraints.value = constraints.value.filter((c) => c.id !== constraint.id)
        Message.success('约束已删除')
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '删除失败')
      }
    },
  })
}

function onSubjectRoleIdsChange(value: unknown) {
  form.value.subject_role_ids = Array.isArray(value) ? value.map(String) : []
}

function onPrerequisiteRoleIdsChange(value: unknown) {
  form.value.prerequisite_role_ids = Array.isArray(value) ? value.map(String) : []
}
</script>

<template>
  <div class="page-main">
    <PageHeader title="约束管理" subtitle="管理角色互斥、先决条件与成员基数约束">
      <template #actions>
        <a-button
          v-perm="'constraints:manage:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="openAdd"
        >
          <template #icon><IconPlus :size="13" /></template>
          创建约束
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <div
        v-if="constraints.length > 0"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
      >
        <div
          v-for="constraint in constraints"
          :key="constraint.id"
          class="constraint-card"
          :class="{ 'opacity-70': !constraint.is_active }"
        >
          <div class="p-5">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <a-tag
                  :color="typeColor(constraint.constraint_type)"
                  size="small"
                  class="!m-0 shrink-0"
                >
                  {{ typeLabel(constraint.constraint_type) }}
                </a-tag>
                <span class="text-[14px] font-semibold text-[#1D1D1F] truncate">
                  {{ constraint.name }}
                </span>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <a-button
                  v-perm="'constraints:manage:write'"
                  type="text"
                  size="mini"
                  @click="openEdit(constraint)"
                >
                  <template #icon><IconEdit :size="14" /></template>
                </a-button>
                <a-button
                  v-perm="'constraints:manage:write'"
                  type="text"
                  size="mini"
                  status="danger"
                  @click="removeConstraint(constraint)"
                >
                  <template #icon><IconDelete :size="14" /></template>
                </a-button>
              </div>
            </div>

            <p class="text-[13px] text-[#86868B] leading-relaxed mb-4 min-h-[20px]">
              {{ constraint.description || '暂无描述' }}
            </p>

            <div class="space-y-3">
              <div class="flex items-start gap-2">
                <IconExclamationCircle :size="14" class="text-[#86868B] mt-0.5 shrink-0" />
                <span class="text-[12px] text-[#86868B]">{{ formatConfig(constraint) }}</span>
              </div>

              <div class="flex items-start gap-2">
                <IconLink :size="14" class="text-[#86868B] mt-0.5 shrink-0" />
                <div class="flex flex-wrap gap-1.5">
                  <a-tag
                    v-for="assoc in subjectRoles(constraint)"
                    :key="assoc.id"
                    :color="roleColor(assoc.role_id)"
                    size="small"
                    class="!m-0"
                  >
                    {{ assoc.role_display_name || assoc.role_name }}
                  </a-tag>
                </div>
              </div>

              <div v-if="prerequisiteRoles(constraint).length > 0" class="flex items-start gap-2">
                <IconUser :size="14" class="text-[#86868B] mt-0.5 shrink-0" />
                <div class="flex flex-wrap gap-1.5">
                  <a-tag
                    v-for="assoc in prerequisiteRoles(constraint)"
                    :key="assoc.id"
                    :color="roleColor(assoc.role_id)"
                    size="small"
                    class="!m-0"
                  >
                    先决：{{ assoc.role_display_name || assoc.role_name }}
                  </a-tag>
                </div>
              </div>
            </div>
          </div>

          <div class="constraint-footer">
            <div class="flex items-center justify-between">
              <span class="text-[12px] text-[#86868B]">
                创建于 {{ formatDateTime(constraint.created_at) }}
              </span>
              <a-switch
                v-perm="'constraints:manage:write'"
                :model-value="constraint.is_active"
                size="small"
                @change="(v: boolean | string | number) => toggleActive(constraint, Boolean(v))"
              />
              <a-tag size="small" :color="constraint.is_active ? 'green' : 'gray'" class="!m-0">
                {{ constraint.is_active ? '启用' : '停用' }}
              </a-tag>
            </div>
          </div>
        </div>
      </div>

      <a-empty v-else description="暂无约束配置" class="mt-20" />
    </a-spin>

    <a-modal
      v-model:visible="modalVisible"
      :title="isEdit ? '编辑约束' : '创建约束'"
      :width="520"
      :ok-loading="modalSaving"
      :ok-text="isEdit ? '保存' : '创建'"
      @ok="saveConstraint"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="约束名称" required>
          <a-input v-model="form.name" placeholder="如：运营与审核互斥" />
        </a-form-item>
        <a-form-item label="描述（选填）">
          <a-textarea
            v-model="form.description"
            placeholder="约束说明"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>
        <a-form-item label="约束类型" required>
          <a-select
            v-model="form.constraint_type"
            :disabled="isEdit"
            :options="[
              { value: 'mutual_exclusive', label: '互斥角色' },
              { value: 'prerequisite', label: '先决角色' },
              { value: 'cardinality', label: '基数约束' },
            ]"
          />
        </a-form-item>

        <a-form-item v-if="form.constraint_type === 'mutual_exclusive'" label="作用域">
          <a-radio-group v-model="form.scope">
            <a-radio value="static">静态（同一用户不能同时拥有）</a-radio>
            <a-radio value="dynamic">动态（同一会话不能同时激活）</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="form.constraint_type === 'prerequisite'" label="先决要求">
          <a-radio-group v-model="form.require_all">
            <a-radio :value="true">必须拥有全部先决角色</a-radio>
            <a-radio :value="false">至少拥有一个先决角色</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="form.constraint_type === 'cardinality'" label="最大用户数">
          <a-input-number v-model="form.max_users" :min="1" placeholder="最多可分配的用户数" />
        </a-form-item>

        <a-form-item label="作用角色" required>
          <a-select
            :model-value="form.subject_role_ids"
            placeholder="选择受此约束影响的角色"
            multiple
            :options="roleOptions"
            @change="onSubjectRoleIdsChange"
          />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              {{
                form.constraint_type === 'mutual_exclusive'
                  ? '至少选择两个角色'
                  : '至少选择一个角色'
              }}
            </span>
          </template>
        </a-form-item>

        <a-form-item v-if="form.constraint_type === 'prerequisite'" label="先决角色" required>
          <a-select
            :model-value="form.prerequisite_role_ids"
            placeholder="选择必须先拥有的角色"
            multiple
            :options="roleOptions"
            @change="onPrerequisiteRoleIdsChange"
          />
        </a-form-item>

        <a-form-item label="启用状态">
          <a-switch v-model="form.is_active" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.constraint-card {
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
.constraint-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 36px -18px rgba(0, 0, 0, 0.18);
  border-color: rgba(0, 0, 0, 0.1);
}
.constraint-footer {
  padding: 12px 20px;
  background: rgba(0, 0, 0, 0.01);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}
</style>
