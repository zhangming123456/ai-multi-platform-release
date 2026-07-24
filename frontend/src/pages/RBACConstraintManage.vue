<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconSafe,
  IconPlus,
  IconDelete,
  IconExclamation,
  IconLock,
  IconClockCircle,
  IconUserGroup,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import api from '@/utils/api'
import { usePermissionStore, type RBACRole } from '@/stores/permission'

const permStore = usePermissionStore()

interface SSDConstraint {
  id: string
  name: string
  description: string
  cardinality: number
  role_ids: string[]
  roles?: RBACRole[]
  created_at: string
}

interface DSDConstraint {
  id: string
  name: string
  description: string
  cardinality: number
  role_ids: string[]
  roles?: RBACRole[]
  created_at: string
}

const activeTab = ref<'ssd' | 'dsd'>('ssd')
const loading = ref(false)
const ssdList = ref<SSDConstraint[]>([])
const dsdList = ref<DSDConstraint[]>([])

const ssdModalVisible = ref(false)
const ssdSaving = ref(false)
const newSSD = ref({
  name: '',
  description: '',
  cardinality: 2,
  role_ids: [] as string[],
})

const dsdModalVisible = ref(false)
const dsdSaving = ref(false)
const newDSD = ref({
  name: '',
  description: '',
  cardinality: 2,
  role_ids: [] as string[],
})

const BUILTIN_COLORS: Record<string, string> = {
  super_admin: 'red',
  admin: 'orangered',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
  auditor: 'cyan',
}

function roleColor(roleName: string): string {
  return BUILTIN_COLORS[roleName] || 'arcoblue'
}

const availableRoles = computed(() => permStore.roles.filter(r => !r.is_super_admin))

async function loadData() {
  loading.value = true
  try {
    await permStore.fetchRoles()
    const [ssdRes, dsdRes] = await Promise.all([
      api.get<SSDConstraint[]>('/v2/constraints/ssd'),
      api.get<DSDConstraint[]>('/v2/constraints/dsd'),
    ])
    ssdList.value = Array.isArray(ssdRes.data) ? ssdRes.data : []
    dsdList.value = Array.isArray(dsdRes.data) ? dsdRes.data : []
    await loadConstraintRoles()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载约束失败')
  } finally {
    loading.value = false
  }
}

async function loadConstraintRoles() {
  for (const ssd of ssdList.value) {
    ssd.roles = ssd.role_ids
      .map(id => permStore.roles.find(r => r.id === id))
      .filter(Boolean) as RBACRole[]
  }
  for (const dsd of dsdList.value) {
    dsd.roles = dsd.role_ids
      .map(id => permStore.roles.find(r => r.id === id))
      .filter(Boolean) as RBACRole[]
  }
}

onMounted(() => {
  loadData()
})

async function createSSD() {
  if (!newSSD.value.name || newSSD.value.role_ids.length < 2) return
  ssdSaving.value = true
  try {
    await api.post('/v2/constraints/ssd', newSSD.value)
    Message.success('SSD 约束创建成功')
    ssdModalVisible.value = false
    newSSD.value = { name: '', description: '', cardinality: 2, role_ids: [] }
    await loadData()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    ssdSaving.value = false
  }
}

async function createDSD() {
  if (!newDSD.value.name || newDSD.value.role_ids.length < 2) return
  dsdSaving.value = true
  try {
    await api.post('/v2/constraints/dsd', newDSD.value)
    Message.success('DSD 约束创建成功')
    dsdModalVisible.value = false
    newDSD.value = { name: '', description: '', cardinality: 2, role_ids: [] }
    await loadData()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '创建失败')
  } finally {
    dsdSaving.value = false
  }
}

async function deleteSSD(id: string) {
  try {
    await api.delete(`/v2/constraints/ssd/${id}`)
    Message.success('SSD 约束已删除')
    await loadData()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}

async function deleteDSD(id: string) {
  try {
    await api.delete(`/v2/constraints/dsd/${id}`)
    Message.success('DSD 约束已删除')
    await loadData()
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '删除失败')
  }
}
</script>

<template>
  <div>
    <PageHeader title="职责分离约束 (RBAC2)" subtitle="静态职责分离(SSD)与动态职责分离(DSD)，确保系统权限安全">
      <template #actions>
        <a-radio-group v-model="activeTab" type="button" size="large">
          <a-radio value="ssd">静态职责分离 (SSD)</a-radio>
          <a-radio value="dsd">动态职责分离 (DSD)</a-radio>
        </a-radio-group>
        <a-button
          v-if="activeTab === 'ssd'"
          type="primary"
          @click="ssdModalVisible = true"
        >
          <template #icon><IconPlus /></template>
          新建 SSD 约束
        </a-button>
        <a-button
          v-else
          type="primary"
          @click="dsdModalVisible = true"
        >
          <template #icon><IconPlus /></template>
          新建 DSD 约束
        </a-button>
      </template>
    </PageHeader>

    <a-spin :loading="loading" class="w-full">
      <div
        class="bg-gradient-to-r from-[#ff9500]/[0.06] to-[#ff3b30]/[0.06] border border-[#ff9500]/[0.15] rounded-2xl p-5 mb-5 flex items-start gap-3"
      >
        <IconExclamation :size="22" class="text-[#ff9500] mt-0.5 shrink-0" />
        <div>
          <p class="text-[14px] font-semibold text-[#1D1D1F] m-0">
            {{ activeTab === 'ssd' ? '静态职责分离 (SSD)' : '动态职责分离 (DSD)' }}
          </p>
          <p class="text-[12px] text-[#636366] m-0 mt-1 leading-relaxed">
            <template v-if="activeTab === 'ssd'">
              SSD 约束确保用户不能被分配冲突集中的多个角色。例如：一个用户不能同时拥有"运营者"和"审核员"角色，防止舞弊风险。
            </template>
            <template v-else>
              DSD 约束确保用户在同一会话中不能同时激活冲突集中的多个角色。用户可以被分配多个冲突角色，但在使用时只能选择其中一组。
            </template>
          </p>
        </div>
      </div>

      <div v-if="activeTab === 'ssd'" class="space-y-4">
        <div v-if="ssdList.length === 0" class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-12 text-center">
          <IconLock :size="36" class="text-[#c7c7cc] mx-auto mb-4" />
          <p class="text-[14px] text-[#86868B]">暂无 SSD 约束</p>
          <p class="text-[12px] text-[#c7c7cc] mt-1">点击右上角按钮创建第一个约束</p>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div
            v-for="ssd in ssdList"
            :key="ssd.id"
            class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5"
          >
            <div class="px-5 py-4 border-b border-black/[0.04] flex items-center justify-between">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-9 h-9 rounded-xl bg-[#ff3b30]/[0.08] flex items-center justify-center shrink-0">
                  <IconSafe :size="17" class="text-[#ff3b30]" />
                </div>
                <div class="min-w-0">
                  <p class="text-[14px] font-semibold text-[#1D1D1F] m-0 truncate">{{ ssd.name }}</p>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                    基数: {{ ssd.cardinality }} · {{ ssd.role_ids.length }} 个角色
                  </p>
                </div>
              </div>
              <a-popconfirm
                content="确定要删除该约束吗？"
                @ok="deleteSSD(ssd.id)"
              >
                <a-button type="text" size="small" status="danger" class="!px-2">
                  <template #icon><IconDelete :size="14" /></template>
                </a-button>
              </a-popconfirm>
            </div>
            <div v-if="ssd.description" class="px-5 py-3 border-b border-black/[0.04] bg-black/[0.01]">
              <p class="text-[12px] text-[#636366] m-0">{{ ssd.description }}</p>
            </div>
            <div class="p-4">
              <p class="text-[11px] text-[#86868B] font-medium mb-2.5 flex items-center gap-1.5">
                <IconUserGroup :size="12" />
                冲突角色集
              </p>
              <div class="flex flex-wrap gap-1.5">
                <a-tag
                  v-for="role in ssd.roles"
                  :key="role.id"
                  :color="roleColor(role.name)"
                  size="small"
                  class="!m-0"
                >
                  {{ role.display_name }}
                </a-tag>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="space-y-4">
        <div v-if="dsdList.length === 0" class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] p-12 text-center">
          <IconClockCircle :size="36" class="text-[#c7c7cc] mx-auto mb-4" />
          <p class="text-[14px] text-[#86868B]">暂无 DSD 约束</p>
          <p class="text-[12px] text-[#c7c7cc] mt-1">点击右上角按钮创建第一个约束</p>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div
            v-for="dsd in dsdList"
            :key="dsd.id"
            class="bg-white/80 backdrop-blur-xl rounded-2xl border border-black/[0.05] overflow-hidden transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5"
          >
            <div class="px-5 py-4 border-b border-black/[0.04] flex items-center justify-between">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-9 h-9 rounded-xl bg-[#007aff]/[0.08] flex items-center justify-center shrink-0">
                  <IconClockCircle :size="17" class="text-[#007aff]" />
                </div>
                <div class="min-w-0">
                  <p class="text-[14px] font-semibold text-[#1D1D1F] m-0 truncate">{{ dsd.name }}</p>
                  <p class="text-[11px] text-[#86868B] m-0 mt-0.5">
                    基数: {{ dsd.cardinality }} · {{ dsd.role_ids.length }} 个角色
                  </p>
                </div>
              </div>
              <a-popconfirm
                content="确定要删除该约束吗？"
                @ok="deleteDSD(dsd.id)"
              >
                <a-button type="text" size="small" status="danger" class="!px-2">
                  <template #icon><IconDelete :size="14" /></template>
                </a-button>
              </a-popconfirm>
            </div>
            <div v-if="dsd.description" class="px-5 py-3 border-b border-black/[0.04] bg-black/[0.01]">
              <p class="text-[12px] text-[#636366] m-0">{{ dsd.description }}</p>
            </div>
            <div class="p-4">
              <p class="text-[11px] text-[#86868B] font-medium mb-2.5 flex items-center gap-1.5">
                <IconUserGroup :size="12" />
                会话互斥角色
              </p>
              <div class="flex flex-wrap gap-1.5">
                <a-tag
                  v-for="role in dsd.roles"
                  :key="role.id"
                  :color="roleColor(role.name)"
                  size="small"
                  class="!m-0"
                >
                  {{ role.display_name }}
                </a-tag>
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-spin>

    <a-modal
      v-model:visible="ssdModalVisible"
      title="新建 SSD 约束"
      :width="520"
      @ok="createSSD"
      :ok-loading="ssdSaving"
      ok-text="创建"
    >
      <a-form :model="newSSD" layout="vertical">
        <a-form-item label="约束名称" required>
          <a-input v-model="newSSD.name" placeholder="例如：运营审核分离" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea
            v-model="newSSD.description"
            placeholder="描述该约束的用途和业务背景"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>
        <a-form-item label="基数 (cardinality)">
          <a-input-number v-model="newSSD.cardinality" :min="2" style="width: 100%" />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              最多允许用户同时拥有的冲突角色数量，默认为 2（即互斥）
            </span>
          </template>
        </a-form-item>
        <a-form-item label="冲突角色集" required>
          <a-select v-model="newSSD.role_ids" multiple placeholder="选择互斥的角色">
            <a-option v-for="role in availableRoles" :key="role.id" :value="role.id">
              {{ role.display_name }}
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              用户不能同时被分配所选集中的多个角色
            </span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:visible="dsdModalVisible"
      title="新建 DSD 约束"
      :width="520"
      @ok="createDSD"
      :ok-loading="dsdSaving"
      ok-text="创建"
    >
      <a-form :model="newDSD" layout="vertical">
        <a-form-item label="约束名称" required>
          <a-input v-model="newDSD.name" placeholder="例如：会话角色互斥" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea
            v-model="newDSD.description"
            placeholder="描述该约束的用途和业务背景"
            :auto-size="{ minRows: 2, maxRows: 4 }"
          />
        </a-form-item>
        <a-form-item label="基数 (cardinality)">
          <a-input-number v-model="newDSD.cardinality" :min="2" style="width: 100%" />
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              同一会话中最多允许激活的冲突角色数量，默认为 2
            </span>
          </template>
        </a-form-item>
        <a-form-item label="会话互斥角色" required>
          <a-select v-model="newDSD.role_ids" multiple placeholder="选择会话中互斥的角色">
            <a-option v-for="role in availableRoles" :key="role.id" :value="role.id">
              {{ role.display_name }}
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              用户在同一会话中不能同时激活所选集中的多个角色
            </span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
