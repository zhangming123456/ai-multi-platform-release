<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconLeft,
  IconRight,
  IconSafe,
  IconDown,
  IconRight as IconRightSmall,
  IconClose,
} from '@arco-design/web-vue/es/icon'
import api from '@/utils/api'

interface RoleRef {
  id: string
  name: string
  display_name: string
  role_type: string
  is_super_admin: boolean
  is_builtin: boolean
}

interface InheritanceNode {
  role: RoleRef
  direct_permissions: Record<string, string>
  level: number
}

interface InheritanceData {
  role: RoleRef
  direct_permissions: Record<string, string>
  ancestor_chain: InheritanceNode[]
  descendant_tree: InheritanceNode[]
}

const props = defineProps<{
  roleId: string
  roleName: string
  roleDisplayName: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const data = ref<InheritanceData | null>(null)

const BUILTIN_COLORS: Record<string, string> = {
  admin: 'red',
  manager: 'orangered',
  operator: 'blue',
  reviewer: 'green',
}

function nodeColor(node: RoleRef): string {
  return BUILTIN_COLORS[node.name] || 'arcoblue'
}

const allDescendantLevels = computed(() => {
  if (!data.value) return new Set<number>()
  const levels = new Set<number>()
  for (const d of data.value.descendant_tree) {
    levels.add(d.level)
  }
  return levels
})

const expandedNodes = ref<Set<string>>(new Set())

function toggleNode(nodeId: string) {
  const next = new Set(expandedNodes.value)
  if (next.has(nodeId)) {
    next.delete(nodeId)
  } else {
    next.add(nodeId)
  }
  expandedNodes.value = next
}

function isExpanded(nodeId: string): boolean {
  return expandedNodes.value.has(nodeId)
}

function groupedPermissions(perms: Record<string, string>): { resource: string; keys: { key: string; name: string }[] }[] {
  const groups: Record<string, { key: string; name: string }[]> = {}
  for (const [key, name] of Object.entries(perms)) {
    const resource = key.split(':')[0]
    if (!groups[resource]) groups[resource] = []
    groups[resource].push({ key, name })
  }
  return Object.entries(groups).map(([resource, keys]) => ({ resource, keys }))
}

async function fetchData() {
  loading.value = true
  try {
    const res = await api.get<InheritanceData>(`/v2/roles/${props.roleId}/inheritance`)
    data.value = res.data
  } catch (e: any) {
    Message.error(e.response?.data?.detail || '加载继承关系失败')
  } finally {
    loading.value = false
  }
}

watch(() => props.roleId, () => {
  if (props.roleId) {
    fetchData()
  }
}, { immediate: true })
</script>

<template>
  <div class="inheritance-view">
    <div class="inheritance-header">
      <div class="flex items-center gap-2">
        <span class="text-[15px] font-semibold text-[#1d1d1f]">继承关系预览</span>
        <span class="text-[12px] text-[#86868B]">{{ roleDisplayName }}</span>
      </div>
      <a-button type="text" size="small" @click="emit('close')">
        <template #icon><IconClose :size="16" /></template>
      </a-button>
    </div>

    <a-spin :loading="loading" class="inheritance-body">
      <div v-if="!data" class="empty-hint">
        <span class="text-[13px] text-[#86868B]">暂无数据</span>
      </div>

      <div v-else class="tree-container">
        <div class="tree-legend">
          <div class="legend-item">
            <div class="legend-dot inherited"></div>
            <span>继承自祖先</span>
          </div>
          <div class="legend-item">
            <div class="legend-dot current"></div>
            <span>当前角色</span>
          </div>
          <div class="legend-item">
            <div class="legend-dot descendant"></div>
            <span>被后代继承</span>
          </div>
        </div>

        <div class="tree-content">
          <div
            v-if="data.ancestor_chain.length > 0 || data.descendant_tree.length > 0"
            class="tree-scroll"
          >
            <div class="tree-chain">
              <div
                v-for="(node, idx) in data.ancestor_chain"
                :key="node.role.id"
                class="tree-node-group"
              >
                <div
                  class="tree-node inherited"
                  @click="toggleNode(node.role.id)"
                >
                  <div class="node-header">
                    <a-tag :color="nodeColor(node.role)" size="small" class="!m-0">
                      {{ node.role.display_name }}
                    </a-tag>
                    <a-tag v-if="node.role.role_type === 'admin'" size="small" color="orangered" class="!m-0">管理</a-tag>
                    <a-tag v-else size="small" color="gray" class="!m-0">普通</a-tag>
                  </div>
                  <div class="node-meta">
                    <span class="meta-text">{{ Object.keys(node.direct_permissions).length }} 项权限</span>
                    <span class="meta-icon">
                      <IconDown v-if="isExpanded(node.role.id)" :size="10" />
                      <IconRightSmall v-else :size="10" />
                    </span>
                  </div>
                </div>

                <div
                  v-if="isExpanded(node.role.id)"
                  class="node-detail"
                >
                  <div v-if="groupedPermissions(node.direct_permissions).length > 0" class="detail-perms">
                    <div
                      v-for="group in groupedPermissions(node.direct_permissions)"
                      :key="group.resource"
                      class="detail-group"
                    >
                      <span class="detail-resource">{{ group.keys[0]?.name || group.resource }}</span>
                      <div class="detail-keys">
                        <code
                          v-for="item in group.keys"
                          :key="item.key"
                          class="detail-key"
                        >{{ item.key }}</code>
                      </div>
                    </div>
                  </div>
                  <div v-else class="detail-empty">无直接权限</div>
                </div>

                <div class="tree-connector">
                  <div class="connector-line"></div>
                  <IconRight :size="12" class="connector-arrow" />
                </div>
              </div>

              <div
                class="tree-node current"
                @click="toggleNode(data.role.id)"
              >
                <div class="node-header">
                  <a-tag :color="nodeColor(data.role)" size="small" class="!m-0">
                    {{ data.role.display_name }}
                  </a-tag>
                  <IconSafe v-if="data.role.is_super_admin" :size="14" class="text-[#ff9500]" />
                  <a-tag v-if="data.role.role_type === 'admin'" size="small" color="orangered" class="!m-0">管理</a-tag>
                  <a-tag v-else size="small" color="gray" class="!m-0">普通</a-tag>
                </div>
                <div class="node-meta">
                  <span class="meta-text">{{ Object.keys(data.direct_permissions).length }} 项权限</span>
                  <span class="meta-icon">
                    <IconDown v-if="isExpanded(data.role.id)" :size="10" />
                    <IconRightSmall v-else :size="10" />
                  </span>
                </div>

                <div
                  v-if="isExpanded(data.role.id)"
                  class="node-detail"
                >
                  <div v-if="groupedPermissions(data.direct_permissions).length > 0" class="detail-perms">
                    <div
                      v-for="group in groupedPermissions(data.direct_permissions)"
                      :key="group.resource"
                      class="detail-group"
                    >
                      <span class="detail-resource">{{ group.keys[0]?.name || group.resource }}</span>
                      <div class="detail-keys">
                        <code
                          v-for="item in group.keys"
                          :key="item.key"
                          class="detail-key"
                        >{{ item.key }}</code>
                      </div>
                    </div>
                  </div>
                  <div v-else class="detail-empty">无直接权限</div>
                </div>
              </div>

              <div
                v-for="(node, idx) in data.descendant_tree"
                :key="node.role.id"
                class="tree-node-group"
              >
                <div class="tree-connector">
                  <div class="connector-line"></div>
                  <IconRight :size="12" class="connector-arrow" />
                </div>

                <div
                  class="tree-node descendant"
                  @click="toggleNode(node.role.id)"
                >
                  <div class="node-header">
                    <a-tag :color="nodeColor(node.role)" size="small" class="!m-0">
                      {{ node.role.display_name }}
                    </a-tag>
                    <a-tag v-if="node.role.role_type === 'admin'" size="small" color="orangered" class="!m-0">管理</a-tag>
                    <a-tag v-else size="small" color="gray" class="!m-0">普通</a-tag>
                  </div>
                  <div class="node-meta">
                    <span class="meta-text">{{ Object.keys(node.direct_permissions).length }} 项权限</span>
                    <span class="meta-icon">
                      <IconDown v-if="isExpanded(node.role.id)" :size="10" />
                      <IconRightSmall v-else :size="10" />
                    </span>
                  </div>
                </div>

                <div
                  v-if="isExpanded(node.role.id)"
                  class="node-detail"
                >
                  <div v-if="groupedPermissions(node.direct_permissions).length > 0" class="detail-perms">
                    <div
                      v-for="group in groupedPermissions(node.direct_permissions)"
                      :key="group.resource"
                      class="detail-group"
                    >
                      <span class="detail-resource">{{ group.keys[0]?.name || group.resource }}</span>
                      <div class="detail-keys">
                        <code
                          v-for="item in group.keys"
                          :key="item.key"
                          class="detail-key"
                        >{{ item.key }}</code>
                      </div>
                    </div>
                  </div>
                  <div v-else class="detail-empty">无直接权限</div>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="tree-empty">
            <div class="tree-node current standalone">
              <div class="node-header">
                <a-tag :color="nodeColor(data.role)" size="small" class="!m-0">
                  {{ data.role.display_name }}
                </a-tag>
                <a-tag v-if="data.role.role_type === 'admin'" size="small" color="orangered" class="!m-0">管理</a-tag>
                <a-tag v-else size="small" color="gray" class="!m-0">普通</a-tag>
              </div>
              <div class="node-meta">
                <span class="meta-text">{{ Object.keys(data.direct_permissions).length }} 项权限</span>
              </div>
            </div>
            <div class="empty-label">
              <IconLeft :size="14" class="text-[#aeaeaf]" />
              <span class="text-[12px] text-[#aeaeaf]">无祖先继承</span>
              <span class="mx-2 text-[#aeaeaf]">·</span>
              <span class="text-[12px] text-[#aeaeaf]">无后代继承</span>
              <IconRight :size="14" class="text-[#aeaeaf]" />
            </div>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.inheritance-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.inheritance-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.inheritance-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.empty-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

.tree-container {
  display: flex;
  flex-direction: column;
}

.tree-legend {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
  padding: 10px 14px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 10px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #636366;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.legend-dot.inherited {
  background: rgba(0, 122, 255, 0.6);
}

.legend-dot.current {
  background: #007AFF;
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.2);
}

.legend-dot.descendant {
  background: rgba(52, 199, 89, 0.8);
}

.tree-content {
  min-height: 120px;
}

.tree-scroll {
  overflow-x: auto;
}

.tree-chain {
  display: flex;
  align-items: flex-start;
  gap: 0;
  min-width: max-content;
  padding: 8px 0 16px;
}

.tree-node-group {
  display: flex;
  align-items: flex-start;
  gap: 0;
}

.tree-node {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 14px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: all 0.2s;
  min-width: 120px;
}

.tree-node:hover {
  box-shadow: 0 4px 16px -8px rgba(0, 0, 0, 0.15);
  border-color: rgba(0, 0, 0, 0.12);
}

.tree-node.inherited {
  border-left: 3px solid rgba(0, 122, 255, 0.4);
}

.tree-node.current {
  border: 2px solid #007AFF;
  background: rgba(0, 122, 255, 0.04);
}

.tree-node.descendant {
  border-left: 3px solid rgba(52, 199, 89, 0.5);
}

.tree-node.standalone {
  margin: 0 auto;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.meta-text {
  font-size: 11px;
  color: #86868b;
}

.meta-icon {
  color: #aeaeaf;
  display: flex;
  align-items: center;
}

.node-detail {
  padding: 8px 10px;
  margin-top: 2px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.detail-perms {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.detail-resource {
  font-size: 11px;
  font-weight: 500;
  color: #1d1d1f;
}

.detail-keys {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
}

.detail-key {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0, 122, 255, 0.08);
  color: #007AFF;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.detail-empty {
  font-size: 11px;
  color: #aeaeaf;
  padding: 4px 0;
}

.tree-connector {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 4px;
  align-self: center;
}

.connector-line {
  width: 16px;
  height: 1px;
  background: rgba(0, 0, 0, 0.15);
}

.connector-arrow {
  color: rgba(0, 0, 0, 0.25);
}

.tree-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 32px 0;
}

.empty-label {
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
