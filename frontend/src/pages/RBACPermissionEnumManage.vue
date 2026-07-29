<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconSafe,
  IconEdit,
  IconDelete,
  IconPlus,
  IconEye,
  IconEyeInvisible,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

interface ResourceRef {
  id: string
  key: string
  name: string
  description?: string | null
}

interface PermissionItem {
  id: string
  key: string
  operation: string
  is_active: boolean
  resource: ResourceRef
}

type PermType = 'page' | 'action'
type PermMode = 'read' | 'write'

function _isPageKey(key: string): boolean {
  const parts = key.split(':')
  return parts.length === 2 && (parts[1] === 'read' || parts[1] === 'write')
}

function _parseKey(key: string): { keyName: string; operation: string; mode: PermMode } {
  const parts = key.split(':')
  if (_isPageKey(key)) {
    return { keyName: parts[0], operation: parts[1], mode: parts[1] as PermMode }
  }
  return { keyName: parts[0], operation: parts[1] || '', mode: (parts[2] || 'write') as PermMode }
}

function _buildPageKey(keyName: string): string {
  return `${keyName}:read`
}

function _buildActionKey(keyName: string, operation: string, mode: PermMode): string {
  return `${keyName}:${operation}:${mode}`
}

const loading = ref(false)
const saving = ref(false)
const permissions = ref<PermissionItem[]>([])

const showCreateDialog = ref(false)
const creating = ref(false)
const createForm = ref({
  type: 'page' as PermType,
  keyName: '',
  operation: '',
  mode: 'write' as PermMode,
  displayName: '',
  description: '',
})

const createComputedKey = computed(() => {
  if (!createForm.value.keyName.trim()) return ''
  if (createForm.value.type === 'page') {
    return _buildPageKey(createForm.value.keyName.trim())
  }
  if (!createForm.value.operation.trim()) return ''
  return _buildActionKey(createForm.value.keyName.trim(), createForm.value.operation.trim(), createForm.value.mode)
})

const editingId = ref<string | null>(null)
const editingKeyName = ref('')
const editingOperation = ref('')
const editingMode = ref<PermMode>('write')
const editingOriginalKey = ref('')
const editingDisplayName = ref('')
const editingDescription = ref('')

const editingType = computed<PermType>(() =>
  _isPageKey(editingOriginalKey.value) ? 'page' : 'action'
)

const editingComputedKey = computed(() => {
  if (!editingKeyName.value.trim()) return ''
  if (editingType.value === 'page') {
    return _buildPageKey(editingKeyName.value.trim())
  }
  if (!editingOperation.value.trim()) return ''
  return _buildActionKey(editingKeyName.value.trim(), editingOperation.value.trim(), editingMode.value)
})

const pagePermissions = computed(() =>
  permissions.value.filter((p) => _isPageKey(p.key))
)

const actionPermissions = computed(() =>
  permissions.value.filter((p) => !_isPageKey(p.key))
)

async function fetchPermissions() {
  loading.value = true
  try {
    const res = await api.get<PermissionItem[]>('/v2/permissions')
    permissions.value = Array.isArray(res.data) ? res.data : []
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载权限列表失败')
  } finally {
    loading.value = false
  }
}

function startEdit(item: PermissionItem) {
  editingId.value = item.resource.id
  const parsed = _parseKey(item.resource.key)
  editingKeyName.value = parsed.keyName
  editingOperation.value = parsed.operation
  editingMode.value = parsed.mode
  editingOriginalKey.value = item.resource.key
  editingDisplayName.value = item.resource.name
  editingDescription.value = item.resource.description || ''
}

function cancelEdit() {
  editingId.value = null
  editingKeyName.value = ''
  editingOperation.value = ''
  editingMode.value = 'write'
  editingOriginalKey.value = ''
  editingDisplayName.value = ''
  editingDescription.value = ''
}

async function saveEdit(resourceId: string) {
  if (!editingKeyName.value.trim()) {
    Message.warning('name 不能为空')
    return
  }
  if (editingType.value === 'action' && !editingOperation.value.trim()) {
    Message.warning('operation 不能为空')
    return
  }
  if (!editingDisplayName.value.trim()) {
    Message.warning('显示名称不能为空')
    return
  }
  saving.value = true
  try {
    await api.put(`/v2/resources/${resourceId}`, {
      name: editingDisplayName.value.trim(),
      key: editingComputedKey.value,
      description: editingDescription.value.trim() || null,
    })
    Message.success('权限定义已更新')
    await fetchPermissions()
    cancelEdit()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '更新失败')
  } finally {
    saving.value = false
  }
}

async function toggleActive(permissionId: string, currentActive: boolean) {
  saving.value = true
  try {
    await api.put(`/v2/permissions/${permissionId}`, { is_active: !currentActive })
    Message.success(currentActive ? '权限已禁用' : '权限已启用')
    await fetchPermissions()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '操作失败')
  } finally {
    saving.value = false
  }
}

async function deletePermission(item: PermissionItem) {
  Modal.warning({
    title: '确认删除',
    content: `确定要删除权限 "${item.resource.name}" (${item.key}) 吗？此操作不可恢复。`,
    hideCancel: false,
    onOk: async () => {
      saving.value = true
      try {
        await api.delete(`/v2/resources/${item.resource.id}`)
        Message.success('权限已删除')
        await fetchPermissions()
      } catch (e: any) {
        Message.error(e.response?.data?.detail || '删除失败')
      } finally {
        saving.value = false
      }
    },
  })
}

async function handleCreate() {
  if (!createForm.value.keyName.trim()) {
    Message.warning('name 不能为空')
    return
  }
  if (createForm.value.type === 'action' && !createForm.value.operation.trim()) {
    Message.warning('operation 不能为空')
    return
  }
  if (!createForm.value.displayName.trim()) {
    Message.warning('显示名称不能为空')
    return
  }
  creating.value = true
  try {
    const key = createComputedKey.value
    const res = await api.post<{ id: string }>('/v2/resources', {
      key,
      name: createForm.value.displayName.trim(),
      description: createForm.value.description.trim() || null,
    })

    const resourceId = res.data.id
    const operation = createForm.value.type === 'page' ? 'read' : createForm.value.operation.trim()

    await api.post('/v2/permissions', {
      resource_id: resourceId,
      operation,
      key,
    })

    Message.success('权限定义已创建')
    showCreateDialog.value = false
    resetCreateForm()
    await fetchPermissions()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    creating.value = false
  }
}

function resetCreateForm() {
  createForm.value = { type: 'page', keyName: '', operation: '', mode: 'write', displayName: '', description: '' }
}

onMounted(fetchPermissions)
</script>

<template>
  <div class="page-main">
    <PageHeader title="权限定义管理" subtitle="编辑权限枚举的名称、Key 和描述，或自定义添加新的权限定义。仅超级管理员可操作。">
      <template #actions>
        <a-button type="primary" @click="showCreateDialog = true">
          <template #icon><IconPlus /></template>
          新增权限定义
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full">
      <div class="space-y-4">
        <div
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
        >
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">页面权限</h3>
              <span class="text-[12px] text-[#86868B] font-medium">{{ pagePermissions.length }}</span>
            </div>
          </div>

          <div v-if="pagePermissions.length === 0" class="py-8">
            <a-empty description="暂无页面权限定义" />
          </div>

          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="item in pagePermissions"
              :key="item.id"
              class="flex items-start justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
              :class="{ 'opacity-50': !item.is_active }"
            >
              <div class="min-w-0 mr-3 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <template v-if="editingId === item.resource.id">
                    <a-input
                      v-model="editingKeyName"
                      size="mini"
                      placeholder="name"
                      class="!w-[100px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <span class="text-[12px] text-[#86868B] font-mono">:read</span>
                    <a-input
                      v-model="editingDisplayName"
                      size="mini"
                      placeholder="显示名称"
                      class="!w-[100px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <a-input
                      v-model="editingDescription"
                      size="mini"
                      placeholder="描述"
                      class="!w-[120px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <a-button
                      size="mini"
                      type="primary"
                      :loading="saving"
                      @click="saveEdit(item.resource.id)"
                    >
                      确定
                    </a-button>
                    <a-button size="mini" @click="cancelEdit">取消</a-button>
                  </template>
                  <template v-else>
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                      {{ item.resource.name }}
                    </p>
                    <a-tag size="small" color="blue" class="!m-0 shrink-0 font-mono">
                      {{ item.key }}
                    </a-tag>
                    <a-tag v-if="!item.is_active" size="small" color="gray" class="!m-0 shrink-0">
                      已禁用
                    </a-tag>
                  </template>
                </div>
                <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                  <template v-if="editingId === item.resource.id && editingComputedKey">
                    &#123;{{ editingComputedKey }}&#125;
                  </template>
                  <template v-else>
                    &#123;{{ _parseKey(item.resource.key).keyName }}&#125;:read
                  </template>
                </p>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  @click="startEdit(item)"
                >
                  <template #icon><IconEdit :size="13" /></template>
                </a-button>
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  :status="item.is_active ? 'normal' : 'warning'"
                  @click="toggleActive(item.id, item.is_active)"
                >
                  <template #icon>
                    <component :is="item.is_active ? IconEye : IconEyeInvisible" :size="13" />
                  </template>
                </a-button>
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  status="danger"
                  @click="deletePermission(item)"
                >
                  <template #icon><IconDelete :size="13" /></template>
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <div
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
        >
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">操作权限</h3>
              <span class="text-[12px] text-[#86868B] font-medium">{{ actionPermissions.length }}</span>
            </div>
          </div>

          <div v-if="actionPermissions.length === 0" class="py-8">
            <a-empty description="暂无操作权限定义" />
          </div>

          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="item in actionPermissions"
              :key="item.id"
              class="flex items-start justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
              :class="{ 'opacity-50': !item.is_active }"
            >
              <div class="min-w-0 mr-3 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <template v-if="editingId === item.resource.id">
                    <a-input
                      v-model="editingKeyName"
                      size="mini"
                      placeholder="name"
                      class="!w-[80px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <span class="text-[11px] text-[#86868B] font-mono">:</span>
                    <a-input
                      v-model="editingOperation"
                      size="mini"
                      placeholder="operation"
                      class="!w-[80px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <span class="text-[11px] text-[#86868B] font-mono">:</span>
                    <a-select
                      v-model="editingMode"
                      size="mini"
                      class="!w-[72px]"
                    >
                      <a-option value="read">read</a-option>
                      <a-option value="write">write</a-option>
                    </a-select>
                    <a-input
                      v-model="editingDisplayName"
                      size="mini"
                      placeholder="显示名称"
                      class="!w-[100px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <a-input
                      v-model="editingDescription"
                      size="mini"
                      placeholder="描述"
                      class="!w-[100px]"
                      @keyup.enter="saveEdit(item.resource.id)"
                      @keyup.escape="cancelEdit"
                    />
                    <a-button
                      size="mini"
                      type="primary"
                      :loading="saving"
                      @click="saveEdit(item.resource.id)"
                    >
                      确定
                    </a-button>
                    <a-button size="mini" @click="cancelEdit">取消</a-button>
                  </template>
                  <template v-else>
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                      {{ item.resource.name }}
                    </p>
                    <a-tag size="small" color="arcoblue" class="!m-0 shrink-0 font-mono">
                      {{ item.key }}
                    </a-tag>
                    <a-tag v-if="!item.is_active" size="small" color="gray" class="!m-0 shrink-0">
                      已禁用
                    </a-tag>
                  </template>
                </div>
                <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                  <template v-if="editingId === item.resource.id && editingComputedKey">
                    &#123;{{ editingComputedKey }}&#125;
                  </template>
                  <template v-else>
                    &#123;{{ _parseKey(item.resource.key).keyName }}&#125;:&#123;{{ _parseKey(item.resource.key).operation }}&#125;:{{ _parseKey(item.resource.key).mode }}
                  </template>
                </p>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  @click="startEdit(item)"
                >
                  <template #icon><IconEdit :size="13" /></template>
                </a-button>
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  :status="item.is_active ? 'normal' : 'warning'"
                  @click="toggleActive(item.id, item.is_active)"
                >
                  <template #icon>
                    <component :is="item.is_active ? IconEye : IconEyeInvisible" :size="13" />
                  </template>
                </a-button>
                <a-button
                  v-if="editingId !== item.resource.id"
                  size="mini"
                  type="text"
                  status="danger"
                  @click="deletePermission(item)"
                >
                  <template #icon><IconDelete :size="13" /></template>
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <div
          class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
        >
          <div class="flex items-center gap-2 mb-4">
            <IconSafe :size="18" class="text-[#86868B]" />
            <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">权限 Key 格式说明</h3>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="p-3 rounded-xl bg-[#007AFF]/[0.04] border border-[#007AFF]/[0.1]">
              <p class="text-[12px] font-semibold text-[#1D1D1F] m-0 mb-1">页面权限 (page)</p>
              <code class="text-[12px] text-[#007AFF]">&#123;name&#125;:read</code>
              <p class="text-[11px] text-[#86868B] m-0 mt-1">
                2段格式，name 不能含 read/write。结尾固定为 read。根据 key 格式自动推断类型。
              </p>
            </div>
            <div class="p-3 rounded-xl bg-[#34C759]/[0.04] border border-[#34C759]/[0.1]">
              <p class="text-[12px] font-semibold text-[#1D1D1F] m-0 mb-1">操作权限 (action)</p>
              <code class="text-[12px] text-[#34C759]">&#123;name&#125;:&#123;operation&#125;:read | write</code>
              <p class="text-[11px] text-[#86868B] m-0 mt-1">
                3段格式，name 和 operation 不能含 read/write。结尾为 read 或 write。根据 key 格式自动推断类型。
              </p>
            </div>
          </div>
        </div>
      </div>
    </a-spin>

    <a-modal
      v-model:visible="showCreateDialog"
      title="新增权限定义"
      :width="480"
      :footer="false"
      @cancel="resetCreateForm"
    >
      <a-form layout="vertical" class="mt-2">
        <a-form-item label="权限类型">
          <a-radio-group v-model="createForm.type" type="button">
            <a-radio value="page">页面权限</a-radio>
            <a-radio value="action">操作权限</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item label="name（Key 第1段）" required>
          <a-input
            v-model="createForm.keyName"
            placeholder="模块名，如 dashboard、content"
          />
        </a-form-item>

        <a-form-item v-if="createForm.type === 'action'" label="operation（Key 第2段）" required>
          <a-input
            v-model="createForm.operation"
            placeholder="操作名，如 create、update、delete"
          />
        </a-form-item>

        <a-form-item v-if="createForm.type === 'action'" label="后缀模式（Key 第3段）">
          <a-select v-model="createForm.mode">
            <a-option value="read">read</a-option>
            <a-option value="write">write</a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="最终合成 Key">
          <div class="flex items-center gap-2">
            <a-tag size="medium" :color="createForm.type === 'page' ? 'blue' : 'arcoblue'" class="!m-0 font-mono text-[13px]">
              {{ createComputedKey || '填写 name 后自动生成' }}
            </a-tag>
            <span class="text-[11px] text-[#86868B]">自动生成</span>
          </div>
        </a-form-item>

        <a-form-item label="显示名称" required>
          <a-input
            v-model="createForm.displayName"
            placeholder="权限的中文显示名称，如 仪表盘、创建内容"
          />
        </a-form-item>

        <a-form-item label="描述">
          <a-textarea
            v-model="createForm.description"
            placeholder="权限的描述说明"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>

        <div class="flex justify-end gap-3 mt-4">
          <a-button @click="showCreateDialog = false; resetCreateForm()">
            取消
          </a-button>
          <a-button type="primary" :loading="creating" @click="handleCreate">
            创建
          </a-button>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>
