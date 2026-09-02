<template>
  <div class="page-main">
    <PageHeader
      :title="pageTitle"
      :subtitle="isEdit ? '修改权限字典的名称、Key 和描述' : '添加新的权限字典'"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="goBack">
          <template #icon><IconLeft :size="13" /></template>
          返回列表
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <div class="max-w-[560px]">
          <a-card :bordered="false" class="!rounded-xl">
            <a-form :model="form" layout="vertical" class="!max-w-[480px]">
              <a-form-item label="权限类型">
                <a-radio-group v-model="form.type" type="button" :disabled="isEdit">
                  <a-radio value="page">页面权限</a-radio>
                  <a-radio value="action">操作权限</a-radio>
                </a-radio-group>
              </a-form-item>

              <a-form-item label="name（Key 第1段）" required>
                <a-input v-model="form.keyName" placeholder="模块名，如 dashboard、content" />
              </a-form-item>

              <a-form-item v-if="form.type === 'action'" label="operation（Key 第2段）" required>
                <a-input v-model="form.operation" placeholder="操作名，如 create、update、delete" />
              </a-form-item>

              <a-form-item v-if="form.type === 'action'" label="后缀模式（Key 第3段）">
                <a-select v-model="form.mode">
                  <a-option value="read">read</a-option>
                  <a-option value="write">write</a-option>
                </a-select>
              </a-form-item>

              <a-form-item label="最终合成 Key">
                <div class="flex items-center gap-2">
                  <a-tag
                    size="medium"
                    :color="form.type === 'page' ? 'blue' : 'arcoblue'"
                    class="!m-0 font-mono text-[13px]"
                  >
                    {{ computedKey || '填写 name 后自动生成' }}
                  </a-tag>
                  <span class="text-[11px] text-[#86868B]">自动生成</span>
                </div>
              </a-form-item>

              <a-form-item label="显示名称" required>
                <a-input
                  v-model="form.displayName"
                  placeholder="权限的中文显示名称，如 仪表盘、创建内容"
                />
              </a-form-item>

              <a-form-item label="描述">
                <a-textarea
                  v-model="form.description"
                  placeholder="权限的描述说明"
                  :auto-size="{ minRows: 2, maxRows: 4 }"
                />
              </a-form-item>

              <a-form-item>
                <a-space>
                  <a-button type="primary" :loading="saving" @click="handleSave">
                    {{ isEdit ? '保存' : '创建' }}
                  </a-button>
                  <a-button @click="goBack">取消</a-button>
                </a-space>
              </a-form-item>
            </a-form>
          </a-card>
        </div>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { IconLeft } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api, { getApiErrorDetail } from '@/utils/api'

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

const router = useRouter()
const route = useRoute()

const isEdit = computed(() => !!route.params.resourceId)
const resourceId = computed(() => route.params.resourceId as string | undefined)
const pageTitle = computed(() => (isEdit.value ? '编辑权限字典' : '新增权限字典'))

const loading = ref(false)
const saving = ref(false)
const existingItem = ref<PermissionItem | null>(null)

const form = ref({
  type: 'page' as PermType,
  keyName: '',
  operation: '',
  mode: 'write' as PermMode,
  displayName: '',
  description: '',
})

const computedKey = computed(() => {
  if (!form.value.keyName.trim()) return ''
  if (form.value.type === 'page') {
    return _buildPageKey(form.value.keyName.trim())
  }
  if (!form.value.operation.trim()) return ''
  return _buildActionKey(form.value.keyName.trim(), form.value.operation.trim(), form.value.mode)
})

async function loadPermission() {
  if (!isEdit.value || !resourceId.value) return
  loading.value = true
  try {
    const res = await api.get<PermissionItem[]>('/v2/permissions')
    const permissions = Array.isArray(res.data) ? res.data : []
    const item = permissions.find((p) => p.resource.id === resourceId.value)
    if (!item) {
      Message.error('权限字典不存在')
      router.push({ name: 'RBACPermissionEnumManage' })
      return
    }
    existingItem.value = item
    const parsed = _parseKey(item.resource.key)
    form.value = {
      type: _isPageKey(item.resource.key) ? 'page' : 'action',
      keyName: parsed.keyName,
      operation: _isPageKey(item.resource.key) ? '' : parsed.operation,
      mode: _isPageKey(item.resource.key) ? 'write' : parsed.mode,
      displayName: item.resource.name,
      description: item.resource.description || '',
    }
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载权限字典失败')
    router.push({ name: 'RBACPermissionEnumManage' })
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!form.value.keyName.trim()) {
    Message.warning('name 不能为空')
    return
  }
  if (form.value.type === 'action' && !form.value.operation.trim()) {
    Message.warning('operation 不能为空')
    return
  }
  if (!form.value.displayName.trim()) {
    Message.warning('显示名称不能为空')
    return
  }
  saving.value = true
  try {
    const key = computedKey.value

    if (isEdit.value && resourceId.value) {
      await api.put(`/v2/resources/${resourceId.value}`, {
        name: form.value.displayName.trim(),
        key,
        description: form.value.description.trim() || null,
      })
      Message.success('权限字典已更新')
    } else {
      const res = await api.post<{ id: string }>('/v2/resources', {
        key,
        name: form.value.displayName.trim(),
        description: form.value.description.trim() || null,
      })

      const operation = form.value.type === 'page' ? 'read' : form.value.operation.trim()
      await api.post('/v2/permissions', {
        resource_id: res.data.id,
        operation,
        key,
      })

      Message.success('权限字典已创建')
    }

    router.push({ name: 'RBACPermissionEnumManage' })
  } catch (e) {
    Message.error(getApiErrorDetail(e) || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push({ name: 'RBACPermissionEnumManage' })
}

onMounted(loadPermission)
</script>
