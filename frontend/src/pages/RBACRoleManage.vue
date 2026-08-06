<template>
  <div class="page-main relative">
    <PageHeader title="角色管理" subtitle="管理系统角色、自定义角色与角色继承关系">
      <template #actions>
        <a-button
          v-perm="'roles:manage:write'"
          type="text"
          size="mini"
          class="!text-[#007AFF] !px-0 !h-auto"
          @click="openAdd"
        >
          <template #icon><IconPlus :size="13" /></template>
          创建角色
        </a-button>
      </template>
    </PageHeader>

    <div class="view-mode-toggle mb-5">
      <button
        class="toggle-btn"
        :class="{ active: viewMode === 'card' }"
        @click="viewMode = 'card'"
      >
        <IconApps :size="14" /> 卡片
      </button>
      <button
        class="toggle-btn"
        :class="{ active: viewMode === 'table' }"
        @click="viewMode = 'table'"
      >
        <IconList :size="14" /> 列表
      </button>
    </div>

    <div class="inherit-panel-wrapper" :class="{ 'panel-open': inheritanceRole && !isMobile }">
      <div class="inherit-main-content">
        <a-spin :loading="loading" tip="加载中..." class="w-full">
          <!-- 卡片模式 -->
          <div
            v-if="viewMode === 'card'"
            class="grid gap-4"
            :class="[
              inheritanceRole && !isMobile
                ? 'grid-cols-1 md:grid-cols-2'
                : 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
            ]"
          >
            <div v-for="role in sortedRoles" :key="role.id" class="role-card">
              <div class="p-4 md:p-5">
                <div class="flex items-start justify-between mb-3 gap-2">
                  <div class="flex items-center gap-2 min-w-0 flex-1">
                    <a-tag :color="roleColor(role)" size="small" class="!m-0 truncate max-w-full">
                      {{ role.display_name }}
                    </a-tag>
                    <IconSafe
                      v-if="role.is_super_admin"
                      :size="14"
                      class="text-[#ff9500] shrink-0"
                    />
                  </div>
                  <div class="flex items-center gap-1 shrink-0">
                    <a-button
                      v-perm="'roles:manage:write'"
                      type="text"
                      size="mini"
                      :disabled="role.is_super_admin"
                      @click="openEdit(role)"
                    >
                      <template #icon><IconEdit :size="14" /></template>
                    </a-button>
                    <a-button
                      v-perm="'roles:manage:write'"
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

                <div class="flex flex-wrap items-center gap-1.5 mb-3">
                  <code class="role-code">{{ role.name }}</code>
                  <a-tag v-if="role.is_super_admin" size="small" color="orangered" class="!m-0"
                    >超级</a-tag
                  >
                  <a-tag v-else-if="role.is_builtin" size="small" color="gray" class="!m-0"
                    >内置</a-tag
                  >
                  <a-tag v-else size="small" color="arcoblue" class="!m-0">自定义</a-tag>
                  <a-tag
                    v-if="role.role_type === 'admin'"
                    size="small"
                    color="orangered"
                    class="!m-0"
                    >管理类型</a-tag
                  >
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

                  <div v-if="hasIndirectRelations(role)" class="pt-1">
                    <a-button
                      type="text"
                      size="mini"
                      class="!text-[#007AFF] !text-[11px] !px-1 !h-auto"
                      @click="toggleExpand(role.id)"
                    >
                      <template #icon>
                        <IconDown v-if="isExpanded(role.id)" :size="10" />
                        <IconRight v-else :size="10" />
                      </template>
                      {{ isExpanded(role.id) ? '收起继承链' : '展开继承链' }}
                    </a-button>
                  </div>

                  <div v-if="isExpanded(role.id) && hasIndirectRelations(role)" class="chain-view">
                    <div v-if="indirectAncestors(role).length > 0" class="chain-chunk">
                      <div class="chain-label">间接祖先</div>
                      <div class="chain-flow">
                        <div
                          v-for="(ancestor, idx) in indirectAncestors(role)"
                          :key="ancestor.id"
                          class="chain-node"
                        >
                          <span class="chain-arrow" v-if="idx > 0 || role.parent_roles.length > 0"
                            >←</span
                          >
                          <a-tag size="small" :color="roleColorById(ancestor.id)" class="!m-0">
                            {{ ancestor.display_name }}
                          </a-tag>
                        </div>
                        <span class="chain-arrow" v-if="role.parent_roles.length > 0"> ← </span>
                        <span class="chain-current" v-if="role.parent_roles.length > 0">
                          {{ role.display_name }}
                        </span>
                      </div>
                    </div>

                    <div v-if="indirectDescendants(role).length > 0" class="chain-chunk">
                      <div class="chain-label">间接后代</div>
                      <div class="chain-flow">
                        <span class="chain-current" v-if="role.child_roles.length > 0">
                          {{ role.display_name }}
                        </span>
                        <span class="chain-arrow" v-if="role.child_roles.length > 0">→</span>
                        <div
                          v-for="(descendant, idx) in indirectDescendants(role)"
                          :key="descendant.id"
                          class="chain-node"
                        >
                          <span class="chain-arrow" v-if="idx > 0 || role.child_roles.length > 0"
                            >→</span
                          >
                          <a-tag size="small" :color="roleColorById(descendant.id)" class="!m-0">
                            {{ descendant.display_name }}
                          </a-tag>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div
                    v-if="constraintsForRole(role.id).length > 0"
                    class="pt-2 mt-2 border-t border-black/[0.04]"
                  >
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
                        <a-tag
                          size="small"
                          :color="constraintTypeColor(c.constraint_type)"
                          class="!m-0"
                        >
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

              <div class="role-footer px-4 md:px-5">
                <div class="flex items-center gap-2">
                  <a-tooltip content="查看继承关系" mini>
                    <a-button
                      v-perm="'roles:read'"
                      type="text"
                      size="small"
                      @click="isMobile ? openInheritanceDrawer(role) : openInheritance(role)"
                    >
                      <template #icon><IconEye :size="15" /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="配置继承" mini>
                    <a-button
                      v-perm="'roles:manage:write'"
                      type="text"
                      size="small"
                      :disabled="role.is_super_admin"
                      @click="openParents(role)"
                    >
                      <template #icon><IconLink :size="15" /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="权限配置" mini>
                    <a-button
                      v-perm="'roles:manage:write'"
                      type="text"
                      size="small"
                      @click="$router.push(`/rbac/permissions?role=${role.id}`)"
                    >
                      <template #icon><IconSettings :size="15" /></template>
                    </a-button>
                  </a-tooltip>
                </div>
              </div>
            </div>
          </div>

          <!-- 表格模式 -->
          <div v-if="viewMode === 'table'" class="role-table-wrap">
            <a-table :data="sortedRoles" :pagination="false" :bordered="false" :stripe="true">
              <template #columns>
                <a-table-column
                  title="角色"
                  data-index="display_name"
                  :width="140"
                  :min-width="100"
                >
                  <template #cell="{ record }">
                    <div class="flex items-center gap-2">
                      <a-tag :color="roleColor(record)" size="small" class="!m-0">
                        {{ record.display_name }}
                      </a-tag>
                      <IconSafe
                        v-if="record.is_super_admin"
                        :size="13"
                        class="text-[#ff9500] shrink-0"
                      />
                    </div>
                  </template>
                </a-table-column>
                <a-table-column title="标识" data-index="name" :width="120" :min-width="100">
                  <template #cell="{ record }">
                    <code class="role-code">{{ record.name }}</code>
                  </template>
                </a-table-column>
                <a-table-column
                  title="描述"
                  data-index="description"
                  :ellipsis="true"
                  :min-width="100"
                >
                  <template #cell="{ record }">
                    <span class="text-[13px] text-[#86868B]">
                      {{ record.description || '—' }}
                    </span>
                  </template>
                </a-table-column>
                <a-table-column title="类型" :width="100" :min-width="100">
                  <template #cell="{ record }">
                    <a-tag v-if="record.is_super_admin" size="small" color="orangered" class="!m-0"
                      >超级</a-tag
                    >
                    <a-tag v-else-if="record.is_builtin" size="small" color="gray" class="!m-0"
                      >内置</a-tag
                    >
                    <a-tag v-else size="small" color="arcoblue" class="!m-0">自定义</a-tag>
                  </template>
                </a-table-column>
                <a-table-column title="继承" :width="180" :min-width="100">
                  <template #cell="{ record }">
                    <div class="flex flex-wrap gap-1">
                      <a-tag
                        v-for="parent in record.parent_roles"
                        :key="parent.id"
                        size="small"
                        :color="roleColorById(parent.id)"
                        class="!m-0"
                      >
                        {{ parent.display_name }}
                      </a-tag>
                      <span
                        v-if="record.parent_roles.length === 0"
                        class="text-[12px] text-[#c9c9cc]"
                        >—</span
                      >
                    </div>
                  </template>
                </a-table-column>
                <a-table-column title="约束" :width="140" :min-width="100">
                  <template #cell="{ record }">
                    <div class="flex flex-wrap gap-1">
                      <a-tag
                        v-for="c in constraintsForRole(record.id)"
                        :key="c.id"
                        size="small"
                        :color="constraintTypeColor(c.constraint_type)"
                        class="!m-0"
                      >
                        {{ constraintTypeLabel(c.constraint_type) }}
                      </a-tag>
                      <span
                        v-if="constraintsForRole(record.id).length === 0"
                        class="text-[12px] text-[#c9c9cc]"
                        >—</span
                      >
                    </div>
                  </template>
                </a-table-column>
                <a-table-column title="操作" :width="120" :min-width="100" fixed="right">
                  <template #cell="{ record }">
                    <div class="flex items-center gap-1">
                      <a-tooltip content="查看继承关系" mini>
                        <a-button
                          v-perm="'roles:read'"
                          type="text"
                          size="small"
                          @click="
                            isMobile ? openInheritanceDrawer(record) : openInheritance(record)
                          "
                        >
                          <template #icon><IconEye :size="15" /></template>
                        </a-button>
                      </a-tooltip>
                      <a-tooltip content="配置继承" mini>
                        <a-button
                          v-perm="'roles:manage:write'"
                          type="text"
                          size="small"
                          :disabled="record.is_super_admin"
                          @click="openParents(record)"
                        >
                          <template #icon><IconLink :size="15" /></template>
                        </a-button>
                      </a-tooltip>
                      <a-tooltip content="权限配置" mini>
                        <a-button
                          v-perm="'roles:manage:write'"
                          type="text"
                          size="small"
                          @click="$router.push(`/rbac/permissions?role=${record.id}`)"
                        >
                          <template #icon><IconSettings :size="15" /></template>
                        </a-button>
                      </a-tooltip>
                      <a-tooltip content="编辑" mini>
                        <a-button
                          v-perm="'roles:manage:write'"
                          type="text"
                          size="small"
                          :disabled="record.is_super_admin"
                          @click="openEdit(record)"
                        >
                          <template #icon><IconEdit :size="15" /></template>
                        </a-button>
                      </a-tooltip>
                      <a-tooltip content="删除" mini>
                        <a-button
                          v-perm="'roles:manage:write'"
                          type="text"
                          size="small"
                          status="danger"
                          :disabled="record.is_builtin"
                          @click="deleteRole(record)"
                        >
                          <template #icon><IconDelete :size="15" /></template>
                        </a-button>
                      </a-tooltip>
                    </div>
                  </template>
                </a-table-column>
              </template>
            </a-table>
          </div>

          <a-empty v-if="!loading && roles.length === 0" description="暂无角色数据" class="mt-20" />
        </a-spin>
      </div>

      <div v-if="inheritanceRole && !isMobile" class="inherit-panel">
        <RoleInheritanceTree
          :role-id="inheritanceRole.id"
          :role-name="inheritanceRole.name"
          :role-display-name="inheritanceRole.display_name"
          @close="closeInheritancePanel"
        />
      </div>
    </div>

    <a-drawer
      v-model:visible="inheritanceDrawerVisible"
      title="继承关系预览"
      :width="360"
      :footer="false"
      placement="right"
      @cancel="closeInheritancePanel"
    >
      <RoleInheritanceTree
        v-if="inheritanceRole"
        :role-id="inheritanceRole.id"
        :role-name="inheritanceRole.name"
        :role-display-name="inheritanceRole.display_name"
        @close="closeInheritancePanel"
      />
    </a-drawer>

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
              {{
                newRole.role_type === 'admin'
                  ? '管理类型角色可配置数据库等高级权限'
                  : '普通类型角色不支持数据库相关权限'
              }}
            </span>
          </template>
        </a-form-item>
        <a-form-item label="父角色（可选）">
          <a-select
            v-model="newRole.parent_role_ids"
            placeholder="选择父角色以继承其权限"
            multiple
            :options="
              roles
                .filter((r) => !r.is_super_admin)
                .map((r) => ({ value: r.id, label: r.display_name }))
            "
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
      :width="560"
      :ok-loading="parentsSaving"
      @ok="saveParents"
      ok-text="保存"
    >
      <a-form v-if="parentsRole" :model="{ selectedParents }" layout="vertical">
        <a-form-item label="选择父角色">
          <a-select v-model="selectedParents" placeholder="选择父角色以继承其权限" multiple>
            <a-option
              v-for="opt in availableParentOptions(parentsRole)"
              :key="opt.id"
              :value="opt.id"
              :disabled="isCycleCandidate(opt.id)"
            >
              <div class="flex items-center gap-1.5">
                <a-tag :color="roleColorById(opt.id)" size="small" class="!m-0">
                  {{ opt.display_name }}
                </a-tag>
                <span v-if="isCycleCandidate(opt.id)" class="text-[11px] text-[#ff3b30]">
                  (会形成循环继承)
                </span>
              </div>
            </a-option>
          </a-select>
          <template #extra>
            <span class="text-[11px] text-[#86868b]">
              子角色将自动继承所选父角色及其祖先的全部权限。标记为红色的选项将导致循环继承，不可选择。
            </span>
          </template>
        </a-form-item>

        <a-divider :margin="4" />

        <div class="preview-section">
          <div class="flex items-center gap-1.5 mb-3">
            <IconEye :size="14" class="text-[#007AFF]" />
            <span class="text-[13px] font-medium text-[#1d1d1f]">权限继承预览</span>
          </div>

          <div v-if="!previewedParentId()" class="text-[12px] text-[#86868b] py-2">
            选择一个尚未继承的父角色，查看将获得的权限
          </div>

          <a-spin
            v-else
            :loading="parentPreviewLoading[previewedParentId()!]"
            tip="加载权限..."
            class="w-full"
          >
            <div
              v-if="parentPreviewMap[previewedParentId()!]?._error === 'cycle'"
              class="text-[12px] text-[#ff3b30] py-2"
            >
              无法预览：添加此父角色会形成循环继承
            </div>

            <div v-else-if="groupedPreviewPermissions().length > 0" class="preview-perms">
              <div
                v-for="group in groupedPreviewPermissions()"
                :key="group.resource"
                class="preview-group"
              >
                <span class="preview-resource">{{ group.keys[0]?.name || group.resource }}</span>
                <div class="preview-keys">
                  <code v-for="item in group.keys" :key="item.key" class="preview-key">
                    {{ item.key }}
                  </code>
                </div>
              </div>
            </div>

            <div
              v-else-if="!parentPreviewLoading[previewedParentId()!]"
              class="text-[12px] text-[#86868b] py-2"
            >
              该父角色无任何权限
            </div>
          </a-spin>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconSafe,
  IconSettings,
  IconLink,
  IconExclamationCircle,
  IconDown,
  IconRight,
  IconEye,
  IconApps,
  IconList,
  IconLink as IconLinkMini,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import RoleInheritanceTree from '@/components/rbac/RoleInheritanceTree.vue'
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
  all_ancestors: RoleRef[]
  all_descendants: RoleRef[]
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

const expandedRoles = ref<Set<string>>(new Set())

function toggleExpand(roleId: string) {
  const newSet = new Set(expandedRoles.value)
  if (newSet.has(roleId)) {
    newSet.delete(roleId)
  } else {
    newSet.add(roleId)
  }
  expandedRoles.value = newSet
}

function isExpanded(roleId: string): boolean {
  return expandedRoles.value.has(roleId)
}

function hasIndirectRelations(role: Role): boolean {
  return (
    role.all_ancestors.length > role.parent_roles.length ||
    role.all_descendants.length > role.child_roles.length
  )
}

function indirectAncestors(role: Role): RoleRef[] {
  const directIds = new Set(role.parent_roles.map((p) => p.id))
  return role.all_ancestors.filter((a) => !directIds.has(a.id))
}

function indirectDescendants(role: Role): RoleRef[] {
  const directIds = new Set(role.child_roles.map((c) => c.id))
  return role.all_descendants.filter((d) => !directIds.has(d.id))
}

function isAncestorOf(ancestorId: string, descendantId: string): boolean {
  const descendant = roles.value.find((r) => r.id === descendantId)
  if (!descendant) return false
  return descendant.all_ancestors.some((a) => a.id === ancestorId)
}

const parentsVisible = ref(false)
const parentsRole = ref<Role | null>(null)
const parentsSaving = ref(false)
const selectedParents = ref<string[]>([])
const parentPreviewMap = ref<Record<string, Record<string, string>>>({})
const parentPreviewLoading = ref<Record<string, boolean>>({})

function openParents(role: Role) {
  parentsRole.value = role
  selectedParents.value = role.parent_roles.map((p) => p.id)
  parentPreviewMap.value = {}
  parentPreviewLoading.value = {}
  parentsVisible.value = true
}

async function fetchParentPreview(parentId: string) {
  if (!parentsRole.value) return
  if (parentPreviewMap.value[parentId] !== undefined) return

  parentPreviewLoading.value = { ...parentPreviewLoading.value, [parentId]: true }
  try {
    const res = await api.get<Record<string, string>>(
      `/v2/roles/${parentsRole.value.id}/permissions/preview/${parentId}`,
    )
    parentPreviewMap.value = {
      ...parentPreviewMap.value,
      [parentId]: res.data || {},
    }
  } catch (e: any) {
    if (e.response?.status === 400) {
      parentPreviewMap.value = {
        ...parentPreviewMap.value,
        [parentId]: { _error: 'cycle' } as any,
      }
    } else {
      parentPreviewMap.value = {
        ...parentPreviewMap.value,
        [parentId]: {},
      }
    }
  } finally {
    parentPreviewLoading.value = { ...parentPreviewLoading.value, [parentId]: false }
  }
}

function previewedParentId(): string | null {
  if (!selectedParents.value.length) return null
  const currentIds = parentsRole.value
    ? new Set(parentsRole.value.parent_roles.map((p) => p.id))
    : new Set<string>()
  for (const id of selectedParents.value) {
    if (!currentIds.has(id)) return id
  }
  const allCurrentIds = new Set(parentsRole.value?.parent_roles.map((p) => p.id) || [])
  return selectedParents.value.find((id) => !allCurrentIds.has(id)) || null
}

watch(
  () => selectedParents.value,
  (newVal) => {
    if (!parentsVisible.value || !parentsRole.value) return
    const currentIds = new Set(parentsRole.value.parent_roles.map((p) => p.id))
    const newlyAdded = newVal.filter((id) => !currentIds.has(id))
    for (const id of newlyAdded) {
      fetchParentPreview(id)
    }
  },
  { immediate: false },
)

function groupedPreviewPermissions(): {
  resource: string
  keys: { key: string; name: string }[]
}[] {
  const preview = parentPreviewMap.value
  const pid = previewedParentId()
  if (!pid) return []
  const perms = preview[pid]
  if (!perms || (perms as any)._error) return []

  const groups: Record<string, { key: string; name: string }[]> = {}
  for (const [key, name] of Object.entries(perms)) {
    const parts = key.split(':')
    const resource = parts[0]
    if (!groups[resource]) groups[resource] = []
    groups[resource].push({ key, name })
  }
  return Object.entries(groups).map(([resource, keys]) => ({ resource, keys }))
}

function availableParentOptions(role: Role) {
  return roles.value.filter((r) => {
    if (r.id === role.id) return false
    if (r.is_super_admin) return false
    return true
  })
}

function isCycleCandidate(parentId: string): boolean {
  if (!parentsRole.value) return false
  return isAncestorOf(parentsRole.value.id, parentId)
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
      try {
        await api.post(`/v2/roles/${parentsRole.value.id}/parents`, { id: parentId })
      } catch (e: any) {
        const msg = e.response?.data?.detail || '添加父角色失败'
        if (msg.includes('循环继承')) {
          Message.error(`无法添加「${getRoleName(parentId)}」为父角色：会形成循环继承`)
        } else {
          Message.error(msg)
        }
        parentsSaving.value = false
        return
      }
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

function getRoleName(roleId: string): string {
  const role = roles.value.find((r) => r.id === roleId)
  return role ? role.display_name : roleId
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
  return constraints.value.filter((c) => c.is_active && c.roles.some((r) => r.role_id === roleId))
}

function formatConstraintRoles(constraint: Constraint, roleId: string): string {
  const others = constraint.roles
    .filter((r) => r.role_id !== roleId)
    .map((r) => r.role_display_name || r.role_name)
  return others.length > 0 ? others.join('、') : '—'
}

const inheritanceRole = ref<Role | null>(null)
const inheritanceDrawerVisible = ref(false)

function openInheritance(role: Role) {
  inheritanceRole.value = role
  inheritanceDrawerVisible.value = false
}

function closeInheritancePanel() {
  inheritanceRole.value = null
  inheritanceDrawerVisible.value = false
}

function openInheritanceDrawer(role: Role) {
  inheritanceRole.value = role
  inheritanceDrawerVisible.value = true
}

const windowWidth = ref(window.innerWidth)
function onResize() {
  windowWidth.value = window.innerWidth
}
onMounted(() => {
  window.addEventListener('resize', onResize)
})
const isMobile = computed(() => windowWidth.value < 1024)

const viewMode = ref<'card' | 'table'>('card')
</script>

<style scoped lang="scss">
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
  padding-top: 12px;
  padding-bottom: 12px;
  background: rgba(0, 0, 0, 0.01);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}

.role-table-wrap {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 16px;
  overflow: hidden;
}

.role-table-wrap :deep(.arco-table) {
  background: transparent;
}

.role-table-wrap :deep(.arco-table-th) {
  background: rgba(0, 0, 0, 0.02) !important;
  font-size: 12px;
  color: #86868b;
  font-weight: 500;
}

.role-table-wrap :deep(.arco-table-td) {
  font-size: 13px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04) !important;
}

.role-table-wrap :deep(.arco-table-tr:last-child .arco-table-td) {
  border-bottom: none !important;
}

.view-mode-toggle {
  display: inline-flex;
  align-items: center;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 999px;
  padding: 3px;
  gap: 2px;
}

.toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  font-weight: 500;
  color: #86868b;
  background: transparent;
  border: none;
  border-radius: 999px;
  padding: 5px 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.toggle-btn:hover {
  color: #1d1d1f;
}

.toggle-btn.active {
  background: #f2f4f8;
  color: #1d1d1f;
}

.inherit-panel-wrapper {
  display: flex;
  gap: 0;
  min-height: 0;
  align-items: flex-start;
}

.inherit-panel-wrapper.panel-open .inherit-main-content {
  flex: 0 0 65%;
  max-width: 65%;
  padding-right: 20px;
}

.inherit-main-content {
  flex: 1;
  min-width: 0;
  transition: flex 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
}

.inherit-panel {
  position: sticky;
  top: 0;
  flex: 0 0 35%;
  min-width: 320px;
  max-width: 35%;
  max-height: calc(100vh - 100px);
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 20px;
  overflow: hidden;
  animation: panelSlideIn 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
}

@keyframes panelSlideIn {
  from {
    opacity: 0;
    transform: translateX(24px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.chain-view {
  background: rgba(0, 0, 0, 0.02);
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  padding: 12px;
}

.chain-chunk {
  margin-bottom: 8px;
}
.chain-chunk:last-child {
  margin-bottom: 0;
}

.chain-label {
  font-size: 10px;
  color: #aeaeaf;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
}

.chain-flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}

.chain-node {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.chain-arrow {
  font-size: 10px;
  color: #aeaeaf;
}

.chain-current {
  font-size: 11px;
  font-weight: 500;
  color: #007aff;
}

.preview-section {
  max-height: 240px;
  overflow-y: auto;
}

.preview-perms {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-group {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.preview-resource {
  font-size: 12px;
  font-weight: 500;
  color: #1d1d1f;
}

.preview-keys {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.preview-key {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(0, 122, 255, 0.08);
  color: #007aff;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
</style>
