<template>
  <div class="page-main">
    <PageHeader title="创作设置" subtitle="风格模板与 AI Prompt 可视化配置">
      <template #actions>
        <a-button type="primary" size="small" @click="openCreate">
          <template #icon><IconPlus /></template>
          新建模板
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1">
      <a-tabs v-model:active-key="activeTab" class="settings-tabs">
        <a-tab-pane key="style" title="风格模板">
          <a-alert type="info" class="mb-4">
            <template #icon><IconInfoCircle /></template>
            风格模板按平台配置，生成文案时自动注入 Prompt，覆盖内置平台风格映射。
          </a-alert>
          <a-spin :loading="styleLoading">
            <a-table
              :columns="styleColumns"
              :data="styleTemplates"
              :bordered="false"
              :hoverable="true"
              :pagination="false"
              row-key="id"
            >
              <template #platform="{ record }">
                <PlatformIcon :platform="record.platform as any" size="sm" />
              </template>
              <template #keywords="{ record }">
                <a-space v-if="record.keywords.length" wrap :size="4">
                  <a-tag v-for="k in record.keywords" :key="k" color="arcoblue" size="small">
                    {{ k }}
                  </a-tag>
                </a-space>
                <span v-else class="text-[#86909c]">—</span>
              </template>
              <template #status="{ record }">
                <StatusBadge :status="record.status" />
              </template>
              <template #actions="{ record }">
                <a-button type="text" size="small" title="编辑" @click="openEditStyle(record)">
                  <template #icon><IconEdit /></template>
                </a-button>
                <a-popconfirm content="确定删除该风格模板吗？" @ok="removeStyle(record)">
                  <a-button type="text" size="small" status="danger" title="删除">
                    <template #icon><IconDelete /></template>
                  </a-button>
                </a-popconfirm>
              </template>
            </a-table>
          </a-spin>
        </a-tab-pane>

        <a-tab-pane key="prompt" title="Prompt 模板">
          <a-alert type="info" class="mb-4">
            <template #icon><IconInfoCircle /></template>
            每个平台可配置一个默认 Prompt（角色 + 强制规则），生成时优先使用。未配置时使用内置默认
            Prompt。
          </a-alert>
          <a-spin :loading="promptLoading">
            <a-table
              :columns="promptColumns"
              :data="promptTemplates"
              :bordered="false"
              :hoverable="true"
              :pagination="false"
              row-key="id"
            >
              <template #platform="{ record }">
                <PlatformIcon :platform="record.platform as any" size="sm" />
              </template>
              <template #is_default="{ record }">
                <a-tag v-if="record.is_default" color="green" size="small">默认</a-tag>
                <span v-else class="text-[#c9cdd4]">—</span>
              </template>
              <template #rules="{ record }">
                <span class="prompt-preview" :title="record.rules || ''">
                  {{ record.rules || '—' }}
                </span>
              </template>
              <template #status="{ record }">
                <StatusBadge :status="record.status" />
              </template>
              <template #actions="{ record }">
                <a-button type="text" size="small" title="编辑" @click="openEditPrompt(record)">
                  <template #icon><IconEdit /></template>
                </a-button>
                <a-popconfirm content="确定删除该 Prompt 模板吗？" @ok="removePrompt(record)">
                  <a-button type="text" size="small" status="danger" title="删除">
                    <template #icon><IconDelete /></template>
                  </a-button>
                </a-popconfirm>
              </template>
            </a-table>
          </a-spin>
        </a-tab-pane>
      </a-tabs>
    </div>

    <a-modal
      v-model:visible="modalVisible"
      :title="modalMode === 'create' ? '新建模板' : '编辑模板'"
      :width="640"
      @ok="submitModal"
      :ok-loading="modalSaving"
    >
      <template v-if="modalType === 'style'">
        <a-form :model="styleForm" layout="vertical" class="!mb-0">
          <a-form-item label="模板名称" required>
            <a-input v-model="styleForm.name" placeholder="如：小红书种草风" />
          </a-form-item>
          <a-form-item label="目标平台" required>
            <a-select v-model="styleForm.platform">
              <a-option v-for="p in platformOptions" :key="p.value" :value="p.value">{{
                p.label
              }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="风格描述">
            <a-textarea
              v-model="styleForm.description"
              :auto-size="{ minRows: 3, maxRows: 6 }"
              placeholder="描述该平台的内容风格，如：活泼、种草、使用emoji、段落短小、口语化"
            />
          </a-form-item>
          <a-form-item label="推荐关键词">
            <a-select
              v-model="styleForm.keywords"
              mode="tags"
              allow-create
              :retain-input-value="true"
              placeholder="输入后回车添加"
            />
          </a-form-item>
        </a-form>
      </template>
      <template v-else>
        <a-form :model="promptForm" layout="vertical" class="!mb-0">
          <a-form-item label="模板名称" required>
            <a-input v-model="promptForm.name" placeholder="如：小红书默认 Prompt" />
          </a-form-item>
          <a-form-item label="目标平台" required>
            <a-select v-model="promptForm.platform">
              <a-option v-for="p in platformOptions" :key="p.value" :value="p.value">{{
                p.label
              }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="系统角色（Role）">
            <a-textarea
              v-model="promptForm.role"
              :auto-size="{ minRows: 3, maxRows: 8 }"
              placeholder="AI 的角色设定，如：你是专业新媒体文案专家…"
            />
          </a-form-item>
          <a-form-item label="强制规则（Rules）">
            <a-textarea
              v-model="promptForm.rules"
              :auto-size="{ minRows: 4, maxRows: 10 }"
              placeholder="每行一条规则，如：仅输出纯净 JSON，禁止多余内容"
            />
          </a-form-item>
          <a-form-item label="设为默认">
            <a-switch v-model="promptForm.is_default" />
            <span class="text-[12px] text-[#86909c] ml-2">生成时优先使用该平台默认模板</span>
          </a-form-item>
        </a-form>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconEdit, IconDelete, IconInfoCircle } from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import PlatformIcon from '@/components/shared/PlatformIcon.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import api from '@/utils/api'

interface StyleTemplate {
  id: string
  name: string
  platform: string
  description: string
  keywords: string[]
  status: string
}

interface PromptTemplate {
  id: string
  name: string
  platform: string
  role: string
  rules: string
  output_format: string
  is_default: boolean
  status: string
}

const activeTab = ref('style')
const styleLoading = ref(false)
const promptLoading = ref(false)
const styleTemplates = ref<StyleTemplate[]>([])
const promptTemplates = ref<PromptTemplate[]>([])

const platformOptions = [
  { value: 'wechat_mp', label: '公众号' },
  { value: 'xiaohongshu', label: '小红书' },
  { value: 'douyin', label: '抖音' },
  { value: 'wechat_video', label: '视频号' },
  { value: 'wechat_moments', label: '朋友圈' },
  { value: 'weibo', label: '微博' },
]

const styleColumns = [
  { title: '平台', dataIndex: 'platform', slotName: 'platform', width: 90 },
  { title: '模板名称', dataIndex: 'name' },
  { title: '风格描述', dataIndex: 'description' },
  { title: '推荐关键词', dataIndex: 'keywords', slotName: 'keywords', width: 220 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 100 },
]

const promptColumns = [
  { title: '平台', dataIndex: 'platform', slotName: 'platform', width: 90 },
  { title: '模板名称', dataIndex: 'name' },
  { title: '强制规则', dataIndex: 'rules', slotName: 'rules' },
  { title: '默认', dataIndex: 'is_default', slotName: 'is_default', width: 80 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 90 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 100 },
]

const modalVisible = ref(false)
const modalSaving = ref(false)
const modalMode = ref<'create' | 'edit'>('create')
const modalType = ref<'style' | 'prompt'>('style')

const styleForm = reactive({
  id: '',
  name: '',
  platform: 'xiaohongshu',
  description: '',
  keywords: [] as string[],
})

const promptForm = reactive({
  id: '',
  name: '',
  platform: 'xiaohongshu',
  role: '',
  rules: '',
  output_format: '',
  is_default: false,
})

onMounted(() => {
  fetchStyleTemplates()
  fetchPromptTemplates()
})

async function fetchStyleTemplates() {
  styleLoading.value = true
  try {
    const res = await api.get<StyleTemplate[]>('/style-templates/')
    styleTemplates.value = Array.isArray(res.data) ? res.data : []
  } catch {
    Message.error('加载风格模板失败')
  } finally {
    styleLoading.value = false
  }
}

async function fetchPromptTemplates() {
  promptLoading.value = true
  try {
    const res = await api.get<PromptTemplate[]>('/prompt-templates/')
    promptTemplates.value = Array.isArray(res.data) ? res.data : []
  } catch {
    Message.error('加载 Prompt 模板失败')
  } finally {
    promptLoading.value = false
  }
}

function openCreate() {
  modalType.value = activeTab.value === 'style' ? 'style' : 'prompt'
  modalMode.value = 'create'
  resetForms()
  modalVisible.value = true
}

function openEditStyle(record: StyleTemplate) {
  modalType.value = 'style'
  modalMode.value = 'edit'
  Object.assign(styleForm, {
    id: record.id,
    name: record.name,
    platform: record.platform,
    description: record.description || '',
    keywords: [...(record.keywords || [])],
  })
  modalVisible.value = true
}

function openEditPrompt(record: PromptTemplate) {
  modalType.value = 'prompt'
  modalMode.value = 'edit'
  Object.assign(promptForm, {
    id: record.id,
    name: record.name,
    platform: record.platform,
    role: record.role || '',
    rules: record.rules || '',
    output_format: record.output_format || '',
    is_default: record.is_default,
  })
  modalVisible.value = true
}

function resetForms() {
  Object.assign(styleForm, {
    id: '',
    name: '',
    platform: 'xiaohongshu',
    description: '',
    keywords: [],
  })
  Object.assign(promptForm, {
    id: '',
    name: '',
    platform: 'xiaohongshu',
    role: '',
    rules: '',
    output_format: '',
    is_default: false,
  })
}

async function submitModal() {
  if (modalType.value === 'style') {
    if (!styleForm.name.trim()) {
      Message.warning('请输入模板名称')
      return
    }
    modalSaving.value = true
    try {
      const payload = {
        name: styleForm.name.trim(),
        platform: styleForm.platform,
        description: styleForm.description,
        keywords: styleForm.keywords,
      }
      if (modalMode.value === 'create') {
        await api.post('/style-templates/', payload)
      } else {
        await api.put(`/style-templates/${styleForm.id}`, payload)
      }
      Message.success(modalMode.value === 'create' ? '创建成功' : '更新成功')
      modalVisible.value = false
      fetchStyleTemplates()
    } catch {
      Message.error('保存失败')
    } finally {
      modalSaving.value = false
    }
  } else {
    if (!promptForm.name.trim()) {
      Message.warning('请输入模板名称')
      return
    }
    modalSaving.value = true
    try {
      const payload = {
        name: promptForm.name.trim(),
        platform: promptForm.platform,
        role: promptForm.role,
        rules: promptForm.rules,
        output_format: promptForm.output_format,
        is_default: promptForm.is_default,
      }
      if (modalMode.value === 'create') {
        await api.post('/prompt-templates/', payload)
      } else {
        await api.put(`/prompt-templates/${promptForm.id}`, payload)
      }
      Message.success(modalMode.value === 'create' ? '创建成功' : '更新成功')
      modalVisible.value = false
      fetchPromptTemplates()
    } catch {
      Message.error('保存失败')
    } finally {
      modalSaving.value = false
    }
  }
}

async function removeStyle(record: StyleTemplate) {
  try {
    await api.delete(`/style-templates/${record.id}`)
    Message.success('已删除')
    fetchStyleTemplates()
  } catch {
    Message.error('删除失败')
  }
}

async function removePrompt(record: PromptTemplate) {
  try {
    await api.delete(`/prompt-templates/${record.id}`)
    Message.success('已删除')
    fetchPromptTemplates()
  } catch {
    Message.error('删除失败')
  }
}
</script>

<style scoped lang="scss">
.settings-tabs {
  :deep(.arco-tabs-content) {
    padding-top: 16px;
  }
}

.prompt-preview {
  display: block;
  max-width: 480px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #86909c;
}
</style>
