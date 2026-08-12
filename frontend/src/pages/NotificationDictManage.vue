<template>
  <div class="page-main">
    <PageHeader
      title="通知消息字典管理"
      subtitle="管理通知消息中字段名和枚举值的显示名称，支持新增、编辑、删除。"
    >
      <template #actions>
        <a-button type="text" size="mini" class="!text-[#007AFF] !px-0 !h-auto" @click="openCreate">
          <template #icon><IconPlus :size="13" /></template>
          新增字典
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <div class="space-y-5">
          <div
            v-for="group in groups"
            :key="group.group_key"
            class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-5"
          >
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2">
                <h3 class="text-[15px] font-semibold text-[#1D1D1F] m-0">
                  {{ groupDisplayName(group.group_key) }}
                </h3>
                <a-tag size="small" color="gray" class="!m-0">{{ group.group_key }}</a-tag>
                <span class="text-[12px] text-[#86868B] font-medium">{{ group.items.length }}</span>
              </div>
            </div>

            <div v-if="group.items.length === 0" class="py-6">
              <a-empty description="暂无字典条目" />
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div
                v-for="item in group.items"
                :key="item.id"
                class="flex items-center justify-between px-4 py-3 rounded-xl border border-black/[0.04] bg-black/[0.01] hover:bg-black/[0.02] transition-colors"
                :class="{ 'opacity-50': !item.is_active }"
              >
                <div class="min-w-0 mr-3 flex-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <p class="text-[13px] font-medium text-[#1D1D1F] m-0 truncate">
                      {{ item.dict_value }}
                    </p>
                    <a-tag
                      v-if="item.is_private"
                      size="small"
                      color="orangered"
                      class="!m-0 shrink-0"
                    >
                      隐私
                    </a-tag>
                  </div>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5 font-mono">
                    {{ item.dict_key }}
                  </p>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <a-tooltip content="编辑">
                    <a-button size="mini" type="text" @click="openEdit(item)">
                      <template #icon><IconEdit :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="删除">
                    <a-button size="mini" type="text" status="danger" @click="handleDelete(item)">
                      <template #icon><IconDelete :size="13" /></template>
                    </a-button>
                  </a-tooltip>
                </div>
              </div>
            </div>
          </div>

          <div v-if="groups.length === 0 && !loading" class="py-12">
            <a-empty description="暂无字典数据" />
          </div>
        </div>
      </a-spin>
    </div>

    <a-modal
      v-model:visible="modalVisible"
      :title="editingId ? '编辑字典' : '新增字典'"
      :width="480"
      @ok="handleSave"
      @cancel="modalVisible = false"
    >
      <a-form :model="form" layout="vertical" class="mt-2">
        <a-form-item label="类别">
          <a-select v-model="form.category" placeholder="选择类别" :disabled="!!editingId">
            <a-option value="field">字段（field）</a-option>
            <a-option value="enum">枚举（enum）</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="分组">
          <a-auto-complete
            v-model="form.group_key"
            :data="existingGroupKeys"
            placeholder="选择或输入分组，如 user_fields"
            :disabled="!!editingId"
            allow-clear
            :filter-option="true"
          />
        </a-form-item>
        <a-form-item label="Key（原始值）">
          <a-input
            v-model="form.dict_key"
            placeholder="如 nickname、review_submit"
            :disabled="!!editingId"
          />
        </a-form-item>
        <a-form-item label="显示名称">
          <a-input v-model="form.dict_value" placeholder="如 昵称、审核提交" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model="form.sort_order" :min="0" :max="999" style="width: 100%" />
        </a-form-item>
        <a-form-item label="状态">
          <a-switch v-model="form.is_active">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </a-switch>
        </a-form-item>
        <a-form-item label="隐私保护">
          <a-switch v-model="form.is_private">
            <template #checked>隐藏原始值</template>
            <template #unchecked>显示原始值</template>
          </a-switch>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              开启后通知消息中该字段值将显示为 ***，保护敏感信息
            </span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'

interface DictItem {
  id: string
  category: string
  group_key: string
  dict_key: string
  dict_value: string
  is_active: boolean
  is_private: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

interface DictGroup {
  group_key: string
  items: DictItem[]
}

const loading = ref(false)
const groups = ref<DictGroup[]>([])
const modalVisible = ref(false)
const editingId = ref<string | null>(null)

const form = ref({
  category: 'field' as string,
  group_key: '',
  dict_key: '',
  dict_value: '',
  sort_order: 0,
  is_active: true,
  is_private: false,
})

const GROUP_DISPLAY: Record<string, string> = {
  user_fields: '用户字段',
  notification_type: '通知类型',
  user_status: '用户状态',
}

function groupDisplayName(key: string): string {
  return GROUP_DISPLAY[key] || key
}

const existingGroupKeys = computed(() => {
  const keys = new Set(groups.value.map((g) => g.group_key))
  return Array.from(keys).map((k) => (GROUP_DISPLAY[k] ? `${k}（${GROUP_DISPLAY[k]}）` : k))
})

async function fetchGroups() {
  loading.value = true
  try {
    const res = await api.get<DictGroup[]>('/notification-dict/groups')
    groups.value = res.data || []
  } catch {
    Message.error('加载字典数据失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = {
    category: 'field',
    group_key: '',
    dict_key: '',
    dict_value: '',
    sort_order: 0,
    is_active: true,
    is_private: false,
  }
  modalVisible.value = true
}

function openEdit(item: DictItem) {
  editingId.value = item.id
  form.value = {
    category: item.category,
    group_key: item.group_key,
    dict_key: item.dict_key,
    dict_value: item.dict_value,
    sort_order: item.sort_order,
    is_active: item.is_active,
    is_private: item.is_private,
  }
  modalVisible.value = true
}

async function handleSave() {
  const rawGroupKey = form.value.group_key.replace(/[（(].*[）)]$/, '').trim()
  if (!rawGroupKey || !form.value.dict_key || !form.value.dict_value) {
    Message.warning('请填写完整信息')
    return
  }

  try {
    if (editingId.value) {
      await api.put(`/notification-dict/${editingId.value}`, {
        dict_value: form.value.dict_value,
        is_active: form.value.is_active,
        is_private: form.value.is_private,
        sort_order: form.value.sort_order,
      })
      Message.success('更新成功')
    } else {
      await api.post('/notification-dict/', {
        ...form.value,
        group_key: rawGroupKey,
      })
      Message.success('创建成功')
    }
    modalVisible.value = false
    await fetchGroups()
  } catch (err: any) {
    Message.error(err?.response?.data?.detail || '操作失败')
  }
}

async function handleDelete(item: DictItem) {
  try {
    await api.delete(`/notification-dict/${item.id}`)
    Message.success('删除成功')
    await fetchGroups()
  } catch (err: any) {
    Message.error(err?.response?.data?.detail || '删除失败')
  }
}

onMounted(() => {
  fetchGroups()
})
</script>
