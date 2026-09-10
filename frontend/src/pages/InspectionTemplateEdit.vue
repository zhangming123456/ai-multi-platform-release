<template>
  <div class="page-main page-main-flush">
    <PageHeader
      :title="isEdit ? '编辑检查表模板' : '新建检查表模板'"
      subtitle="配置模板名称与检查项，支持检查标准、标准图、评分方式与必填项设置"
    >
      <template #actions>
        <a-tooltip content="从素材库添加" mini>
          <a-button
            size="mini"
            type="text"
            class="!text-[#007AFF]"
            @click="() => openMaterialPicker()"
          >
            <template #icon><IconStorage :size="13" /></template>
            <span class="hidden md:inline">从素材库添加</span>
          </a-button>
        </a-tooltip>
        <a-tooltip content="AI 识别图片或描述，智能添加检查项" mini>
          <a-button
            v-perm="'inspection:template:ai_create'"
            size="mini"
            type="text"
            class="!text-[#007AFF]"
            @click="aiGeneratorVisible = true"
          >
            <template #icon><IconRobot :size="13" /></template>
            <span class="hidden md:inline">AI 智能添加</span>
          </a-button>
        </a-tooltip>
      </template>
    </PageHeader>

    <a-spin :loading="loading" tip="加载中..." class="w-full px-4 md:px-6 lg:px-8 pb-24">
      <div ref="checkItemsRef" class="max-w-4xl space-y-4 relative">
        <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
          <div class="text-[15px] font-semibold text-[#1D1D1F] mb-4">基本信息</div>
          <a-form :model="form" layout="vertical">
            <a-form-item
              label="模板名称"
              required
              :class="sectionErrors.has('name') ? 'shake-error-wrapper' : ''"
            >
              <a-input
                v-model="form.name"
                placeholder="请输入模板名称（20 字以内）"
                :maxlength="20"
                show-word-limit
              />
            </a-form-item>
            <a-form-item label="模板描述">
              <a-textarea
                v-model="form.description"
                placeholder="模板用途或说明（50 字以内）..."
                :auto-size="{ minRows: 2, maxRows: 5 }"
                :maxlength="50"
                show-word-limit
                class="!rounded-xl"
                @input="(val: any) => onMultilineInput(form, 'description', val)"
              />
            </a-form-item>
            <a-form-item label="计分方式" required>
              <div class="flex flex-col gap-1.5 items-start w-full">
                <a-radio-group v-model="form.scoring_mode" type="button" size="small">
                  <a-radio value="additive">加分制</a-radio>
                  <a-radio value="deductive">减分制</a-radio>
                </a-radio-group>
                <div class="text-[11px] text-[#86868b] leading-relaxed w-full">
                  加分制：从 0
                  分开始累加各项得分，总分即实得分数；减分制：以满分为基数，扣减各项未得满分部分的差额。
                </div>
              </div>
            </a-form-item>
            <a-form-item v-if="isEdit" label="启用状态">
              <a-switch v-model="form.is_active" checked-text="启用" unchecked-text="停用" />
            </a-form-item>
          </a-form>
        </div>

        <template
          v-if="
            groupedItems.length === 0 ||
            (groupedItems.length === 1 && groupedItems[0].items.length === 0)
          "
        >
          <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
            <div
              class="text-[13px] text-[#86868b] py-6 text-center flex flex-col items-center gap-3"
            >
              <span>暂无检查项，点击右侧浮动按钮或从素材库批量添加</span>
              <a-button size="small" type="primary" @click="() => openMaterialPicker()">
                <template #icon><IconStorage :size="13" /></template>
                从素材库批量添加
              </a-button>
            </div>
          </div>
        </template>
        <template v-else>
          <div
            v-for="(group, gIndex) in groupedItems"
            :key="'g-' + gIndex"
            class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl overflow-hidden"
            :class="
              (sectionErrors.has('items') && gIndex === 0) || categoryErrors.has(gIndex)
                ? 'shake-error-wrapper'
                : ''
            "
          >
            <!-- 分类头部：Collapse Header 样式，整行可点击切换 -->
            <div
              class="flex items-center gap-2 px-5 py-4 cursor-pointer select-none transition-colors hover:bg-[#F5F5F7]/60"
              @click="toggleCategory(gIndex)"
            >
              <!-- 展开图标：左侧三角形，展开朝下，收起朝右 -->
              <span
                class="flex items-center justify-center w-5 h-5 shrink-0 text-[#86868b] transition-transform duration-300 ease-out"
                :class="collapsedCategories.has(gIndex) ? '-rotate-90' : ''"
              >
                <icon-caret-down :size="14" />
              </span>
              <div class="group flex items-center gap-2 flex-1 min-w-0" @click.stop>
                <template v-if="editingCategoryIndex === gIndex">
                  <a-input
                    :model-value="group.category"
                    size="small"
                    placeholder="分类名称（20 字以内）"
                    :maxlength="20"
                    class="category-name-input"
                    :style="{ width: categoryInputWidth(group.category) }"
                    @blur="finishEditCategory(gIndex)"
                    @press-enter="finishEditCategory(gIndex)"
                    @input="(val: any) => updateGroupCategory(gIndex, val)"
                  />
                </template>
                <template v-else>
                  <span
                    class="text-[15px] font-semibold text-[#1D1D1F] cursor-pointer hover:text-[#165DFF] transition-colors truncate"
                    @click="[setActiveCategory(gIndex), startEditCategory(gIndex)]"
                  >
                    {{ group.category || '未命名分类' }}
                  </span>
                  <span
                    class="opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer text-[#86868b] hover:text-[#165DFF] shrink-0"
                    @click="startEditCategory(gIndex)"
                  >
                    <IconEdit :size="13" />
                  </span>
                </template>
                <a-tag size="small" color="arcoblue">{{ group.items.length }} 项</a-tag>
                <a-tooltip content="从素材库快速添加到该分类" mini>
                  <a-button
                    size="mini"
                    type="text"
                    class="!text-[#722ED1] hover:!text-[#722ED1]"
                    @click="[setActiveCategory(gIndex), openMaterialPicker(group.category || '')]"
                  >
                    <template #icon><IconStorage :size="13" /></template>
                    <span class="hidden md:inline">从素材库添加</span>
                  </a-button>
                </a-tooltip>
                <a-popconfirm
                  v-if="groupedItems.length > 1"
                  content="合并后该分类下的所有检查项将移入其他分类"
                  @ok="removeGroup(gIndex)"
                >
                  <a-button size="mini" status="danger" type="text">
                    <template #icon><IconDelete :size="13" /></template>
                  </a-button>
                </a-popconfirm>
              </div>
            </div>

            <!-- 折叠内容区：带高度展开过渡动画 -->
            <transition name="collapse">
              <div v-show="!collapsedCategories.has(gIndex)">
                <div class="px-5 pb-5">
                  <!-- 分类前置条件配置 -->
                  <div class="rounded-xl border border-[#E5E5EA] bg-[#F5F5F7]/50 p-3 mb-3">
                    <div class="flex items-center justify-between">
                      <div class="flex items-center gap-2">
                        <span class="text-[13px] font-medium text-[#1D1D1F]">检查前置条件</span>
                        <span class="text-[11px] text-[#86868b]"
                          >开启后，检查该分类前需先确认前置条件</span
                        >
                      </div>
                      <a-switch
                        :model-value="getCategoryPreconditionEnabled(gIndex)"
                        size="small"
                        @change="(val: any) => setCategoryPreconditionEnabled(gIndex, val)"
                      />
                    </div>
                    <div v-if="getCategoryPreconditionEnabled(gIndex)" class="mt-2">
                      <a-textarea
                        :model-value="getCategoryPrecondition(gIndex)"
                        placeholder="请输入前置条件内容（50 字以内）"
                        :auto-size="{ minRows: 1, maxRows: 3 }"
                        :maxlength="50"
                        show-word-limit
                        size="small"
                        class="!rounded-lg"
                        @input="(val: any) => setCategoryPrecondition(gIndex, val)"
                      />
                    </div>
                  </div>
                  <div
                    v-for="(item, index) in group.items"
                    :key="item.key"
                    :ref="(el: any) => setItemRef(item.key, el)"
                    class="rounded-xl border bg-white p-4 mb-3 transition-colors duration-300 last:mb-0"
                    :class="
                      formErrors.has(item.key)
                        ? '!border-[#FF3B30] shake-error'
                        : 'border-[#E5E5EA]'
                    "
                  >
                    <div class="flex items-center justify-between mb-2">
                      <div
                        class="flex items-center gap-2 cursor-pointer"
                        @click="setActivePosition(gIndex, index)"
                      >
                        <a-tag size="small" color="arcoblue" class="!m-0"
                          >第 {{ index + 1 }} 项</a-tag
                        >
                        <span
                          class="text-[12px] text-[#86868b] truncate max-w-[200px]"
                          :title="item.title"
                          >{{ item.title || '未命名' }}</span
                        >
                      </div>
                      <div class="flex items-center gap-1">
                        <a-tooltip content="上移">
                          <a-button
                            type="text"
                            size="mini"
                            :disabled="index === 0"
                            @click="moveItemInGroup(gIndex, index, -1)"
                          >
                            <template #icon><IconArrowUp :size="14" /></template>
                          </a-button>
                        </a-tooltip>
                        <a-tooltip content="下移">
                          <a-button
                            type="text"
                            size="mini"
                            :disabled="index === group.items.length - 1"
                            @click="moveItemInGroup(gIndex, index, 1)"
                          >
                            <template #icon><IconArrowDown :size="14" /></template>
                          </a-button>
                        </a-tooltip>
                        <a-popconfirm
                          content="确定删除该检查项？"
                          @ok="removeItemFromGroup(gIndex, index)"
                        >
                          <a-tooltip content="删除">
                            <a-button type="text" size="mini" status="danger">
                              <template #icon><IconDelete :size="14" /></template>
                            </a-button>
                          </a-tooltip>
                        </a-popconfirm>
                      </div>
                    </div>

                    <a-form :model="item" layout="vertical" class="!mb-0" size="mini">
                      <a-form-item label="标题" required field-class="!mb-2">
                        <a-input
                          v-model="item.title"
                          placeholder="如：门头形象（100 字以内）"
                          :maxlength="100"
                          show-word-limit
                          size="small"
                          @blur="onTitleBlur(item)"
                        />
                      </a-form-item>

                      <template v-if="expandedItemKeys.has(item.key)">
                        <a-form-item label="检查标准与标准图" field-class="!mb-2">
                          <AttachmentInputArea
                            :ref="(el) => setStandardAreaRef(item, el)"
                            v-model="item.standard"
                            v-model:file-list="item.standardImages"
                            :upload="uploadImageFile"
                            :max-count="MAX_STANDARD_IMAGES"
                            :max-length="500"
                            :min-rows="2"
                            :max-rows="4"
                            placeholder="填写该项的检查标准（500 字以内）..."
                          />
                          <template #extra>
                            <div>支持拖拽 / 粘贴图片，或粘贴图片 URL 自动识别为标准图</div>
                          </template>
                        </a-form-item>
                      </template>

                      <a-form-item label="评分方式" required field-class="!mb-2">
                        <a-radio-group
                          v-model="item.score_type"
                          type="button"
                          size="small"
                          @change="() => onScoreTypeChange(item)"
                        >
                          <a-radio value="score">分值评分</a-radio>
                          <a-radio value="pass_fail">选项评分</a-radio>
                        </a-radio-group>
                      </a-form-item>

                      <a-form-item
                        :label="item.score_type === 'score' ? '评分选项' : '选项设置'"
                        field-class="!mb-2"
                      >
                        <div class="w-full rounded-lg border border-[#E5E5EA] p-2.5">
                          <div
                            v-for="(opt, optIndex) in item.score_options"
                            :key="optIndex"
                            class="flex items-center gap-1.5 mb-1.5"
                          >
                            <div class="flex items-center">
                              <span class="text-[11px] text-[#86868b] w-7 shrink-0">
                                {{ item.score_type === 'pass_fail' ? '序号' : '分值' }}
                              </span>
                              <a-select
                                v-if="item.score_type === 'score'"
                                :model-value="Number(opt.score)"
                                size="mini"
                                class="!w-15"
                                placeholder="选择分值"
                                @change="(val: any) => onOptionScoreSelect(item, optIndex, val)"
                              >
                                <a-option
                                  v-for="v in ALLOWED_SCORE_VALUES"
                                  :key="v"
                                  :value="v"
                                  :disabled="
                                    item.score_options.some(
                                      (o, i) => i !== optIndex && Number(o.score) === v,
                                    )
                                  "
                                >
                                  {{ v }}
                                </a-option>
                              </a-select>
                              <span
                                v-else
                                class="inline-flex items-center justify-center rounded-md text-[#1D1D1F] text-[13px] font-semibold h-[28px] px-0.5 select-none tabular-nums"
                              >
                                {{ optIndex + 1 }}
                              </span>
                            </div>
                            <span class="text-[11px] text-[#86868b] w-7 shrink-0">
                              {{ item.score_type === 'pass_fail' ? '选项' : '描述' }}
                            </span>
                            <a-input
                              v-model="opt.label"
                              :placeholder="
                                item.score_type === 'pass_fail' ? '选项名（必填）' : '如：0分'
                              "
                              size="mini"
                              class="flex-1"
                            />
                            <a-button
                              type="text"
                              size="mini"
                              status="danger"
                              :disabled="item.score_options.length <= 1"
                              @click="removeOption(item, optIndex)"
                            >
                              <template #icon><IconDelete :size="12" /></template>
                            </a-button>
                          </div>
                          <a-button
                            type="outline"
                            size="mini"
                            :disabled="!canAddOption(item)"
                            @click="addOption(item)"
                          >
                            <template #icon><IconPlus :size="10" /></template>
                            添加选项
                          </a-button>
                          <div class="text-[11px] text-[#FF9500] mt-1.5 leading-relaxed">
                            {{
                              item.score_type === 'pass_fail'
                                ? `选项评分不参与分数计算，仅记录所选序号（1, 2, 3...）；选项名必填且不重复。`
                                : `分值仅允许 ${ALLOWED_SCORE_VALUES.join('、')}，分值必填不重复；描述默认「分值+分」，自动排序；必须包含最小值 ${MIN_SCORE} 与最大值 ${MAX_SCORE}。`
                            }}
                          </div>
                        </div>
                      </a-form-item>

                      <template v-if="expandedItemKeys.has(item.key)">
                        <a-form-item field-class="!mb-2">
                          <div class="flex flex-wrap items-center gap-2">
                            <div
                              class="flex items-center justify-between rounded-lg bg-[#F5F5F7] px-3 py-2"
                            >
                              <div class="flex items-center gap-2">
                                <div class="flex flex-col gap-3">
                                  <div class="flex flex-col gap-1.5">
                                    <div class="text-[13px] font-medium text-[#1D1D1F]">
                                      问题描述
                                    </div>
                                    <div class="text-[11px] text-[#86868b]">
                                      控制巡店时该字段是否展示与必填
                                    </div>
                                  </div>
                                  <div class="flex items-center gap-3">
                                    <div class="flex items-center gap-1.5">
                                      <span class="text-[11px] text-[#86868b]">显示</span>
                                      <a-switch v-model="item.show_remark" size="small" />
                                    </div>
                                    <div
                                      class="flex items-center gap-1.5"
                                      :class="!item.show_remark ? 'opacity-40' : ''"
                                    >
                                      <span class="text-[11px] text-[#86868b]">必填</span>
                                      <a-switch
                                        :model-value="
                                          item.show_remark ? item.require_remark : false
                                        "
                                        :disabled="!item.show_remark"
                                        size="small"
                                        @change="(val: any) => (item.require_remark = val)"
                                      />
                                    </div>
                                  </div>
                                </div>
                              </div>
                            </div>
                            <div
                              class="flex items-center justify-between rounded-lg bg-[#F5F5F7] px-3 py-2"
                            >
                              <div class="flex items-center gap-2">
                                <div class="flex flex-col gap-3">
                                  <div class="flex flex-col gap-1.5">
                                    <div class="text-[13px] font-medium text-[#1D1D1F]">
                                      巡店图片
                                    </div>
                                    <div class="text-[11px] text-[#86868b]">
                                      控制巡店时该字段是否展示与必填
                                    </div>
                                  </div>
                                  <div class="flex items-center gap-3">
                                    <div class="flex items-center gap-1.5">
                                      <span class="text-[11px] text-[#86868b]">显示</span>
                                      <a-switch v-model="item.show_photo" size="small" />
                                    </div>
                                    <div
                                      class="flex items-center gap-1.5"
                                      :class="!item.show_photo ? 'opacity-40' : ''"
                                    >
                                      <span class="text-[11px] text-[#86868b]">必填</span>
                                      <a-switch
                                        :model-value="item.show_photo ? item.require_photo : false"
                                        :disabled="!item.show_photo"
                                        size="small"
                                        @change="(val: any) => (item.require_photo = val)"
                                      />
                                    </div>
                                  </div>
                                </div>
                              </div>
                            </div>
                          </div>
                        </a-form-item>
                      </template>
                    </a-form>

                    <div
                      class="flex items-center justify-center gap-2 mt-2 pt-2 border-t border-[#E5E5EA]"
                    >
                      <div class="relative">
                        <a-button
                          v-if="!expandedItemKeys.has(item.key)"
                          type="text"
                          size="small"
                          @click="expandItem(item.key)"
                        >
                          <template #icon><IconPlus :size="12" /></template>
                          高级
                        </a-button>
                        <transition name="fade-slide">
                          <div
                            v-if="guideAdvancedKey === item.key"
                            class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 whitespace-nowrap rounded-lg bg-[#165DFF] text-white text-[12px] px-3 py-1.5 shadow-lg cursor-pointer z-30"
                            @click="expandItem(item.key)"
                          >
                            点击「高级」编写检查标准
                            <div
                              class="absolute top-full left-1/2 -translate-x-1/2 w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-[#165DFF]"
                            ></div>
                          </div>
                        </transition>
                      </div>
                      <a-button
                        type="text"
                        size="small"
                        @click="addItemAfterInGroup(gIndex, index)"
                      >
                        <template #icon><IconPlus :size="12" /></template>
                        在此项后添加
                      </a-button>
                    </div>
                  </div>
                </div>
              </div>
            </transition>
          </div>
        </template>

        <transition name="fade-slide">
          <div
            v-if="showFloatingBar"
            class="fixed z-40 flex flex-col items-center gap-1.5"
            :style="{ right: barRight + 'px', top: '50%', transform: 'translateY(-50%)' }"
          >
            <a-tooltip content="从素材库批量添加" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                @click="() => openMaterialPicker()"
              >
                <template #icon><IconStorage :size="16" class="text-[#722ED1]" /></template>
              </a-button>
            </a-tooltip>
            <div class="w-[1px] h-2 bg-[#E5E5EA]" />
            <a-tooltip content="向上添加分类" position="left">
              <a-button shape="circle" size="small" class="!shadow-md" @click="addCategoryAbove">
                <template #icon><IconMindMapping :size="16" class="text-[#00B42A]" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="向上添加检查项" position="left">
              <a-button shape="circle" size="small" class="!shadow-md" @click="addItemAbove">
                <template #icon><IconOrderedList :size="16" class="text-[#165DFF]" /></template>
              </a-button>
            </a-tooltip>
            <div class="w-[1px] h-2 bg-[#E5E5EA]" />
            <a-tooltip content="上一个分类" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeCategory === 0"
                @click="navCategory(-1)"
              >
                <template #icon><IconDoubleUp :size="16" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="上一个检查项" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeItem === 0"
                @click="navItem(-1)"
              >
                <template #icon><IconUp :size="14" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip
              :content="`第 ${activeCategory + 1}/${activeItemInGroup + 1} 项`"
              position="left"
            >
              <div
                class="flex flex-col gap-1 items-center justify-center rounded-xl bg-white border border-[#E5E5EA] shadow-md px-1 py-1.5 select-none"
              >
                <span class="text-[13px] font-bold text-[#165DFF] leading-none tabular-nums">
                  {{ activeCategory + 1 }}
                </span>
                <div class="border border-[#E5E5EA] w-full" />
                <span class="text-[13px] font-bold text-[#165DFF] leading-none tabular-nums">
                  {{ activeItemInGroup + 1 }}
                </span>
              </div>
            </a-tooltip>
            <a-tooltip content="下一个检查项" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeItem >= totalItems - 1"
                @click="navItem(1)"
              >
                <template #icon><IconDown :size="14" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="下一个分类" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeCategory >= groupedItems.length - 1"
                @click="navCategory(1)"
              >
                <template #icon><IconDoubleDown :size="16" /></template>
              </a-button>
            </a-tooltip>
            <div class="w-[1px] h-2 bg-[#E5E5EA]" />
            <a-tooltip content="向下添加检查项" position="left">
              <a-button shape="circle" size="small" class="!shadow-md" @click="addItemBelow">
                <template #icon><IconOrderedList :size="16" class="text-[#165DFF]" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="向下添加分类" position="left">
              <a-button shape="circle" size="small" class="!shadow-md" @click="addCategoryBelow">
                <template #icon><IconMindMapping :size="16" class="text-[#00B42A]" /></template>
              </a-button>
            </a-tooltip>
          </div>
        </transition>
      </div>
    </a-spin>

    <div
      class="fixed bottom-0 right-0 z-30 border-t border-[#E5E5EA] bg-white/90 backdrop-blur-xl px-4 md:px-6 lg:px-8 py-3 flex items-center justify-between transition-all duration-350 ease-out"
      :style="{ left: 'var(--f-aside-width)' }"
    >
      <span class="text-[13px] text-[#86868b]">
        共 {{ items.length }} 个检查项，{{ groupedItems.length }} 个分类
      </span>
      <div class="flex items-center gap-3">
        <a-button @click="goBack">返回</a-button>
        <a-button
          v-perm="isEdit ? 'inspection:template:update:write' : 'inspection:template:create:write'"
          type="primary"
          :loading="saving"
          @click="handleSave"
        >
          保存
        </a-button>
      </div>
    </div>

    <!-- 从素材库批量添加弹窗 -->
    <a-modal
      v-model:visible="materialPickerVisible"
      :title="
        materialPickerTargetCategory !== null
          ? `从素材库添加到「${materialPickerTargetCategory || '未命名分类'}」`
          : '从素材库批量添加'
      "
      width="880px"
      :mask-closable="false"
      :ok-text="`添加选中的 ${selectedMaterialsMap.size} 项${materialPickerTargetCategory !== null ? '到该分类' : ''}`"
      :ok-button-props="{ disabled: selectedMaterialsMap.size === 0 }"
      @ok="confirmAddFromMaterials"
      @cancel="materialPickerVisible = false"
    >
      <div class="flex items-center justify-between mb-3">
        <a-input-search
          v-model="materialKeyword"
          placeholder="搜索标题 / 分类 / 检查标准"
          allow-clear
          class="!w-72"
          @search="onMaterialSearch"
          @clear="onMaterialSearch"
        />
        <span class="text-[12px] text-[#86868b]"
          >已选 {{ selectedMaterialsMap.size }} 项（跨页累计）</span
        >
      </div>
      <a-spin :loading="materialLoading" tip="加载中..." class="w-full">
        <a-table
          :columns="materialColumns"
          :data="materialList"
          :bordered="false"
          :hoverable="true"
          :pagination="false"
          :row-key="'id'"
          :row-selection="{ type: 'checkbox', showCheckedAll: true }"
          :selected-keys="selectedMaterialIds"
          :scroll="{ y: 360 }"
          @selection-change="onMaterialSelectionChange"
        >
          <template #matTitle="{ record }">
            <span class="text-[13px] font-medium text-[#1D1D1F]">{{ record.title }}</span>
          </template>
          <template #matCategory="{ record }">
            <a-tag v-if="record.category" size="small" color="arcoblue" class="!m-0">{{
              record.category
            }}</a-tag>
            <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
          </template>
          <template #matStandard="{ record }">
            <span class="text-[12px] text-[#86868b]">{{ record.standard || '--' }}</span>
          </template>
          <template #matImage="{ record }">
            <a-image
              v-if="record.standard_images?.length"
              :src="record.standard_images?.[0]"
              :preview-src="record.standard_images?.[0]"
              :width="40"
              :height="40"
              fit="cover"
              class="!rounded-md overflow-hidden"
            />
            <span v-else class="text-[12px] text-[#C7C7CC]">--</span>
          </template>
          <template #matScore="{ record }">
            <span class="text-[12px] text-[#1D1D1F]">{{
              record.score_type === 'pass_fail' ? '选项评分' : '分值评分'
            }}</span>
          </template>
          <template #empty>
            <a-empty description="暂无检查项素材，请先在「巡店管理 > 素材管理」中创建" />
          </template>
        </a-table>
      </a-spin>
      <div class="flex justify-end mt-3">
        <a-pagination
          :total="materialTotal"
          :current="materialPage"
          :page-size="materialPageSize"
          show-total
          show-page-size
          :page-size-options="[10, 20, 50]"
          @change="onMaterialPageChange"
          @page-size-change="onMaterialPageSizeChange"
        />
      </div>
    </a-modal>

    <AITemplateItemGenerator
      v-model:visible="aiGeneratorVisible"
      :existing-categories="existingCategories"
      @confirm="confirmAIGeneratedItems"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconPlus,
  IconDelete,
  IconArrowUp,
  IconArrowDown,
  IconEdit,
  IconDoubleUp,
  IconDoubleDown,
  IconUp,
  IconDown,
  IconMindMapping,
  IconOrderedList,
  IconCaretDown,
  IconStorage,
  IconRobot,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import AITemplateItemGenerator from '@/components/AITemplateItemGenerator.vue'
import type { AIGeneratedItem } from '@/components/AITemplateItemGenerator.types'
import { uploadImageFile } from '@/composables/useFileUpload'
import type { InspectionTemplate, InspectionMaterial, Paginated } from '@/types'
import api, { getApiErrorDetail } from '@/utils/api'
import { useResizeObserver } from '@vueuse/core'

interface ScoreOptionRow {
  score: number
  label: string
}

interface InspectionTemplateItemPayload {
  category: string
  title: string
  standard: string
  standard_images: string[]
  score_type: 'score' | 'pass_fail'
  score_options: { score: number; label: string }[]
  require_remark: boolean
  require_photo: boolean
  show_remark: boolean
  show_photo: boolean
  category_precondition_enabled: boolean
  category_precondition: string
}

interface ItemRow {
  key: number
  id: string
  category: string
  title: string
  standard: string
  standard_images: string[]
  standardImages: string[]
  score_type: 'score' | 'pass_fail'
  score_options: ScoreOptionRow[]
  require_remark: boolean
  require_photo: boolean
  show_remark: boolean
  show_photo: boolean
  category_precondition_enabled: boolean
  category_precondition: string
}

const ALLOWED_SCORE_VALUES = [0, 0.5, 1, 2, 3, 4, 5]
const MIN_SCORE = 0
const MAX_SCORE = 5

const DEFAULT_SCORE_OPTIONS: ScoreOptionRow[] = [
  { score: 0, label: '0分' },
  { score: 5, label: '5分' },
]

const DEFAULT_PASS_FAIL_OPTIONS: ScoreOptionRow[] = [
  { score: 1, label: '合格' },
  { score: 2, label: '不合格' },
]

function cloneOptions(options: ScoreOptionRow[]): ScoreOptionRow[] {
  return options.map((o) => ({ score: o.score, label: o.label }))
}

function sortOptionsByScore(options: ScoreOptionRow[]): ScoreOptionRow[] {
  return [...options].sort((a, b) => Number(a.score) - Number(b.score))
}

function defaultLabelForScore(score: number | string): string {
  const n = Number(score)
  if (Number.isInteger(n)) return `${n}分`
  return `${n}分`
}

function looksLikeAutoLabel(label: string, currentScore: number | string): boolean {
  const n = Number(currentScore)
  return label.trim() === defaultLabelForScore(n)
}

const route = useRoute()
const router = useRouter()

// 侧边栏宽度偏移：让底部固定保存栏与顶部导航对齐
const sidebarOffset = ref(248)
const isMobileView = ref(false)
let sidebarObserver: MutationObserver | null = null

function updateSidebarOffset() {
  isMobileView.value = window.innerWidth < 768
  if (isMobileView.value) {
    sidebarOffset.value = 0
    return
  }
  // 读取 main 元素的实际 left 偏移（与 AppLayout 的 marginLeft 一致）
  const mainEl = document.querySelector('main')
  if (mainEl) {
    const rect = mainEl.getBoundingClientRect()
    sidebarOffset.value = rect.left
  } else {
    // 退回默认：根据侧边栏折叠状态判断
    const aside = document.querySelector('aside')
    sidebarOffset.value = aside ? aside.getBoundingClientRect().width : 248
  }
}

function startSidebarObserver() {
  const aside = document.querySelector('aside')
  if (!aside) return
  sidebarObserver = new MutationObserver(() => {
    updateSidebarOffset()
  })
  sidebarObserver.observe(aside, { attributes: true, attributeFilter: ['style', 'class'] })
}

function stopSidebarObserver() {
  if (sidebarObserver) {
    sidebarObserver.disconnect()
    sidebarObserver = null
  }
}

const loading = ref(false)
const saving = ref(false)
const formErrors = ref(new Set<number>())
const sectionErrors = ref(new Set<string>())
const collapsedCategories = ref(new Set<number>())
const categoryErrors = ref(new Set<number>())
const showFloatingBar = ref(false)
const barRight = ref(0)
const editingCategoryIndex = ref(-1)
const expandedItemKeys = ref(new Set<number>())
const guideAdvancedKey = ref<number | null>(null)
let guideAdvancedShown = false
let guideAdvancedTimer: ReturnType<typeof setTimeout> | null = null
const activeCategory = ref(0)
const activeItem = ref(0)

const totalItems = computed(() => items.value.length)
const activeItemInGroup = computed(() => {
  let count = 0
  for (let g = 0; g < activeCategory.value; g++) {
    count += groupedItems.value[g].items.length
  }
  return Math.max(0, activeItem.value - count)
})

let programmaticScrollUntil = 0
function markProgrammaticScroll() {
  programmaticScrollUntil = Date.now() + 900
}

let scrollRafId: number | null = null

const checkItemsRef = ref<HTMLElement | undefined>()

useResizeObserver(checkItemsRef, handleBarRight)

function handleBarRight() {
  // 计算浮动栏 right 位置，使其贴近检查项区域右侧
  const container = document.querySelector('.max-w-4xl.relative') as HTMLElement | null
  if (container) {
    const rect = container.getBoundingClientRect()
    barRight.value = Math.max(0, window.innerWidth - rect.right - 30)
  }
}

function onScroll() {
  const mainEl = document.querySelector('main')
  if (!mainEl) return
  showFloatingBar.value = mainEl.scrollTop > 180
  handleBarRight()
  if (Date.now() < programmaticScrollUntil) return
  if (scrollRafId !== null) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    updatePositionFromViewport()
  })
}

function updatePositionFromViewport() {
  const viewportCenter = window.innerHeight / 2
  let bestKey: number | null = null
  let bestDist = Infinity
  for (const item of items.value) {
    const el = itemEls[item.key]
    if (!el) continue
    const rect = el.getBoundingClientRect()
    if (rect.height === 0) continue
    const center = rect.top + rect.height / 2
    const dist = Math.abs(center - viewportCenter)
    if (dist < bestDist) {
      bestDist = dist
      bestKey = item.key
    }
  }
  if (bestKey === null) return
  for (let g = 0; g < groupedItems.value.length; g++) {
    const grp = groupedItems.value[g]
    for (let i = 0; i < grp.items.length; i++) {
      if (grp.items[i].key === bestKey) {
        if (activeCategory.value !== g) activeCategory.value = g
        let count = 0
        for (let gg = 0; gg < g; gg++) count += groupedItems.value[gg].items.length
        const target = count + i
        if (activeItem.value !== target) activeItem.value = target
        return
      }
    }
  }
}

function scrollToGroup(gIndex: number) {
  markProgrammaticScroll()
  const groups = document.querySelectorAll(
    '.rounded-2xl.border.border-\\[\\#E5E5EA\\].bg-white\\/70',
  )
  const target = groups[gIndex + 1]
  if (target) target.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function scrollToItemInGroup(gIndex: number, itemIndex: number) {
  markProgrammaticScroll()
  const group = groupedItems.value[gIndex]
  if (!group) return
  const item = group.items[itemIndex]
  if (!item) return
  const el = itemEls[item.key]
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function navCategory(delta: number) {
  const target = activeCategory.value + delta
  if (target < 0 || target >= groupedItems.value.length) return
  activeCategory.value = target
  activeItem.value = 0
  collapsedCategories.value = new Set()
  scrollToGroup(target)
}

function setActiveCategory(gIndex: number) {
  activeCategory.value = gIndex
  let count = 0
  for (let g = 0; g < gIndex; g++) {
    count += groupedItems.value[g].items.length
  }
  activeItem.value = count
}

function setActivePosition(gIndex: number, itemIndex: number) {
  activeCategory.value = gIndex
  let count = 0
  for (let g = 0; g < gIndex; g++) {
    count += groupedItems.value[g].items.length
  }
  activeItem.value = count + itemIndex
}

function navItem(delta: number) {
  const target = activeItem.value + delta
  if (target < 0 || target >= totalItems.value) return
  activeItem.value = target
  let count = 0
  for (let g = 0; g < groupedItems.value.length; g++) {
    const group = groupedItems.value[g]
    if (count + group.items.length > target) {
      activeCategory.value = g
      collapsedCategories.value = new Set()
      const itemIndex = target - count
      scrollToItemInGroup(g, itemIndex)
      return
    }
    count += group.items.length
  }
}

const itemEls: Record<number, HTMLElement | null> = {}

function setItemRef(key: number, el: unknown) {
  if (el) itemEls[key] = el as HTMLElement
}

function highlightSection(sectionKey: string) {
  sectionErrors.value = new Set([sectionKey])
  markProgrammaticScroll()
  nextTick(() => {
    const el = document.querySelector('.shake-error-wrapper')
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
  setTimeout(() => {
    sectionErrors.value = new Set()
  }, 1200)
}

function startEditCategory(gIndex: number) {
  editingCategoryIndex.value = gIndex
  nextTick(() => {
    const input = document.querySelector('.category-name-input input') as HTMLInputElement | null
    if (input) {
      input.focus()
      input.select()
    }
  })
}

// 使用 canvas 实时测量文本宽度，让输入框随内容自适应
// Arco a-input size="small" 字体 14px，水平 padding 12px，边框 1px
let measureCtx: CanvasRenderingContext2D | null = null
const CATEGORY_INPUT_FONT =
  '14px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif'
// 左右 padding 各 12px + 边框 1px*2 + 光标余量 2px
const CATEGORY_INPUT_EXTRA = 12 + 12 + 1 * 2 + 2

function categoryInputWidth(value: string): string {
  if (!measureCtx) {
    const canvas = document.createElement('canvas')
    measureCtx = canvas.getContext('2d')
  }
  if (!measureCtx) return '160px'
  measureCtx.font = CATEGORY_INPUT_FONT
  const text = value || ''
  const textWidth = measureCtx.measureText(text).width
  const placeholderWidth = measureCtx.measureText('分类名称（20 字以内）').width
  const baseWidth = Math.max(textWidth, placeholderWidth) + CATEGORY_INPUT_EXTRA
  return `${Math.min(360, Math.max(120, baseWidth))}px`
}

function finishEditCategory(gIndex: number) {
  clearCategoryError(gIndex)
  editingCategoryIndex.value = -1
}

function toggleCategory(gIndex: number) {
  const next = new Set(collapsedCategories.value)
  if (next.has(gIndex)) {
    next.delete(gIndex)
  } else {
    next.add(gIndex)
  }
  collapsedCategories.value = next
}

function expandItem(key: number) {
  dismissGuideAdvanced()
  const next = new Set(expandedItemKeys.value)
  next.add(key)
  expandedItemKeys.value = next
}

function autoExpandItemsWithData() {
  const next = new Set(expandedItemKeys.value)
  for (const item of items.value) {
    if (
      item.standard ||
      item.standardImages.length > 0 ||
      item.require_remark ||
      item.require_photo
    ) {
      next.add(item.key)
    }
  }
  expandedItemKeys.value = next
}

function onTitleBlur(item: ItemRow) {
  if (guideAdvancedShown) return
  if (!item.title.trim()) return
  if (expandedItemKeys.value.has(item.key)) return
  guideAdvancedShown = true
  guideAdvancedKey.value = item.key
  if (guideAdvancedTimer) clearTimeout(guideAdvancedTimer)
  guideAdvancedTimer = setTimeout(() => {
    guideAdvancedKey.value = null
  }, 6000)
}

function dismissGuideAdvanced() {
  guideAdvancedKey.value = null
  if (guideAdvancedTimer) {
    clearTimeout(guideAdvancedTimer)
    guideAdvancedTimer = null
  }
}

function clearCategoryError(gIndex: number) {
  if (categoryErrors.value.has(gIndex)) {
    const next = new Set(categoryErrors.value)
    next.delete(gIndex)
    categoryErrors.value = next
  }
}

// 读取分类前置条件开关状态（取分类下首条 item）
function getCategoryPreconditionEnabled(gIndex: number): boolean {
  const group = groupedItems.value[gIndex]
  if (!group || group.items.length === 0) return false
  return group.items[0].category_precondition_enabled
}

// 读取分类前置条件内容（取分类下首条 item）
function getCategoryPrecondition(gIndex: number): string {
  const group = groupedItems.value[gIndex]
  if (!group || group.items.length === 0) return ''
  return group.items[0].category_precondition || ''
}

// 更新分类前置条件开关，同步到同分类下所有 item
function setCategoryPreconditionEnabled(gIndex: number, val: unknown) {
  const enabled = !!val
  const group = groupedItems.value[gIndex]
  if (!group) return
  for (const item of group.items) {
    item.category_precondition_enabled = enabled
    if (!enabled) {
      item.category_precondition = ''
    }
  }
}

// 更新分类前置条件内容，同步到同分类下所有 item
function setCategoryPrecondition(
  gIndex: number,
  val: string | { value?: string } | null | undefined,
) {
  const raw = typeof val === 'string' ? val : (val?.value ?? '')
  const group = groupedItems.value[gIndex]
  if (!group) return
  for (const item of group.items) {
    item.category_precondition = raw
  }
}

// 多行文本输入清洗：禁止连续换行、换行+空格+换行
function sanitizeMultiline(val: string): string {
  if (!val) return val
  // 将"换行+可选空格/tab+换行"模式（含连续换行）合并为单个换行
  return val.replace(/(\n[ \t]*)+\n/g, '\n')
}

function onMultilineInput(
  target: Record<string, unknown>,
  field: string,
  val: string | { value?: string } | null | undefined,
) {
  const raw = typeof val === 'string' ? val : (val?.value ?? '')
  const cleaned = sanitizeMultiline(raw)
  if (target && typeof target === 'object') {
    target[field] = cleaned
  }
}

function shakeItemKey(key: number) {
  formErrors.value = new Set([key])
  // 找到该项所在的分类索引，若被折叠则先展开，再滚动到可视范围
  let gIndex = -1
  for (let g = 0; g < groupedItems.value.length; g++) {
    if (groupedItems.value[g].items.some((it) => it.key === key)) {
      gIndex = g
      break
    }
  }
  const wasCollapsed = gIndex >= 0 && collapsedCategories.value.has(gIndex)
  if (wasCollapsed) {
    const nextCollapsed = new Set(collapsedCategories.value)
    nextCollapsed.delete(gIndex)
    collapsedCategories.value = nextCollapsed
  }
  markProgrammaticScroll()
  nextTick(() => {
    const el = itemEls[key]
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
  setTimeout(() => {
    formErrors.value = new Set()
  }, 1200)
}
const isEdit = computed(() => route.name === 'InspectionTemplateEdit')

const form = ref({
  name: '',
  description: '',
  is_active: true,
  scoring_mode: 'additive' as 'additive' | 'deductive',
})
const items = ref<ItemRow[]>([])
let itemKeySeed = 0

const aiGeneratorVisible = ref(false)
const existingCategories = computed<string[]>(() =>
  groupedItems.value.map((g) => g.category.trim()).filter((c) => c !== ''),
)

const standardAreaRefs = new Map<number, { flushPending: () => Promise<boolean> }>()
function setStandardAreaRef(item: ItemRow, el: unknown) {
  if (el) {
    standardAreaRefs.set(item.key, el as { flushPending: () => Promise<boolean> })
  } else {
    standardAreaRefs.delete(item.key)
  }
}

interface ItemGroup {
  category: string
  items: ItemRow[]
}

const groupedItems = computed<ItemGroup[]>(() => {
  const map = new Map<string, ItemRow[]>()
  for (const item of items.value) {
    const key = item.category.trim() || ''
    const arr = map.get(key)
    if (arr) {
      arr.push(item)
    } else {
      map.set(key, [item])
    }
  }
  const groups: ItemGroup[] = []
  for (const [cat, list] of map) {
    groups.push({ category: cat, items: list })
  }
  if (groups.length === 0) {
    groups.push({ category: '', items: [] })
  }
  return groups
})

function syncItemsFromGroups(groups: ItemGroup[]) {
  const result: ItemRow[] = []
  for (const g of groups) {
    for (const item of g.items) {
      item.category = g.category
      result.push(item)
    }
  }
  items.value = result
}

function activeItemIndexInGroup(): number {
  let count = 0
  for (let g = 0; g < activeCategory.value; g++) {
    count += groupedItems.value[g].items.length
  }
  return Math.max(0, activeItem.value - count)
}

function insertItemInGroup(gIndex: number, index: number) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  groups[gIndex].items.splice(index, 0, createRow())
  syncItemsFromGroups(groups)
  activeCategory.value = gIndex
  let count = 0
  for (let g = 0; g < gIndex; g++) {
    count += groups[g].items.length
  }
  activeItem.value = count + index
  markProgrammaticScroll()
  nextTick(() => {
    const items = document.querySelectorAll('.max-w-4xl.relative .rounded-xl.border.bg-white.p-4')
    let offset = 0
    for (let g = 0; g < gIndex; g++) {
      const grp = groupedItems.value[g]
      if (!collapsedCategories.value.has(g)) offset += grp.items.length
    }
    const target = items[offset + index]
    if (target) target.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

function addItemAbove() {
  const gIndex = Math.min(activeCategory.value, groupedItems.value.length - 1)
  const idx = Math.min(activeItemIndexInGroup(), groupedItems.value[gIndex].items.length)
  insertItemInGroup(gIndex, idx)
}

function addItemBelow() {
  const gIndex = Math.min(activeCategory.value, groupedItems.value.length - 1)
  const idx = Math.min(activeItemIndexInGroup() + 1, groupedItems.value[gIndex].items.length)
  insertItemInGroup(gIndex, idx)
}

function removeItemFromGroup(gIndex: number, index: number) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  groups[gIndex].items.splice(index, 1)
  if (groups[gIndex].items.length === 0 && groups.length > 1) {
    groups.splice(gIndex, 1)
  }
  syncItemsFromGroups(groups)
}

function moveItemInGroup(gIndex: number, index: number, delta: number) {
  const target = index + delta
  const group = groupedItems.value[gIndex]
  if (target < 0 || target >= group.items.length) return
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  const arr = groups[gIndex].items
  const tmp = arr[index]
  arr[index] = arr[target]
  arr[target] = tmp
  syncItemsFromGroups(groups)
}

function addItemAfterInGroup(gIndex: number, index: number) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  groups[gIndex].items.splice(index + 1, 0, createRow())
  syncItemsFromGroups(groups)
  markProgrammaticScroll()
  nextTick(() => {
    const items = document.querySelectorAll('.max-w-4xl.relative .rounded-xl.border.bg-white.p-4')
    let offset = 0
    for (let g = 0; g < gIndex; g++) {
      const grp = groupedItems.value[g]
      if (!collapsedCategories.value.has(g)) offset += grp.items.length
    }
    const target = items[offset + index + 1]
    if (target) target.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

function updateGroupCategory(gIndex: number, val: string) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  groups[gIndex].category = val
  syncItemsFromGroups(groups)
}

function removeGroup(gIndex: number) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  const removed = groups.splice(gIndex, 1)[0]
  if (groups.length === 0) {
    groups.push({ category: '', items: [] })
  }
  groups[0].items.push(...removed.items)
  syncItemsFromGroups(groups)
}

function insertCategoryAt(gIndex: number) {
  const groups = groupedItems.value.map((g) => ({ category: g.category, items: [...g.items] }))
  const existingNames = groups.map((g) => g.category)
  let name = '未命名分类 1'
  let n = 1
  while (existingNames.includes(name)) {
    n += 1
    name = `未命名分类 ${n}`
  }
  const idx = Math.max(0, Math.min(gIndex, groups.length))
  groups.splice(idx, 0, { category: name, items: [createRow()] })
  syncItemsFromGroups(groups)
  activeCategory.value = idx
  let count = 0
  for (let g = 0; g < idx; g++) {
    count += groups[g].items.length
  }
  activeItem.value = count
  markProgrammaticScroll()
  nextTick(() => {
    const cards = document.querySelectorAll('.max-w-4xl.relative > .rounded-2xl')
    const target = cards[idx + 1] as HTMLElement | undefined
    if (target) target.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function addCategoryAbove() {
  insertCategoryAt(activeCategory.value)
}

function addCategoryBelow() {
  insertCategoryAt(activeCategory.value + 1)
}

function createRow(): ItemRow {
  itemKeySeed += 1
  return {
    key: itemKeySeed,
    id: '',
    category: '',
    title: '',
    standard: '',
    standard_images: [],
    score_type: 'score',
    score_options: cloneOptions(DEFAULT_SCORE_OPTIONS),
    require_remark: false,
    require_photo: false,
    show_remark: true,
    show_photo: true,
    standardImages: [],
    category_precondition_enabled: false,
    category_precondition: '',
  }
}

function addItem() {
  items.value.push(createRow())
}

function onScoreTypeChange(item: ItemRow) {
  if (item.score_type === 'pass_fail') {
    item.score_options = [
      { score: 1, label: '合格' },
      { score: 2, label: '不合格' },
    ]
  } else {
    item.score_options = cloneOptions(DEFAULT_SCORE_OPTIONS)
  }
}

function onOptionScoreSelect(item: ItemRow, optIndex: number, newScore: number | string) {
  const opt = item.score_options[optIndex]
  if (!opt) return
  const oldScore = opt.score
  opt.score = Number(newScore)
  if (item.score_type === 'score' && looksLikeAutoLabel(opt.label, oldScore)) {
    opt.label = defaultLabelForScore(opt.score)
  }
  item.score_options = sortOptionsByScore(item.score_options)
}

function addOption(item: ItemRow) {
  if (item.score_type === 'pass_fail') {
    const nextNo = item.score_options.length + 1
    item.score_options.push({ score: nextNo, label: '' })
  } else {
    const used = new Set(item.score_options.map((o) => Number(o.score)))
    const available = ALLOWED_SCORE_VALUES.filter((v) => !used.has(v))
    let nextScore = available[0] ?? MIN_SCORE
    if (available.length === 0) {
      nextScore = ALLOWED_SCORE_VALUES[ALLOWED_SCORE_VALUES.length - 1]
    }
    item.score_options.push({ score: nextScore, label: defaultLabelForScore(nextScore) })
    item.score_options = sortOptionsByScore(item.score_options)
  }
}

function canAddOption(item: ItemRow): boolean {
  if (item.score_type !== 'score') return true
  const used = new Set(item.score_options.map((o) => Number(o.score)))
  return ALLOWED_SCORE_VALUES.some((v) => !used.has(v))
}

function removeOption(item: ItemRow, index: number) {
  if (item.score_options.length <= 1) return
  item.score_options.splice(index, 1)
}

// ===== 检查标准 + 标准图一体化输入区（AttachmentInputArea 公共组件） =====
const MAX_STANDARD_IMAGES = 5

async function fetchDetail() {
  try {
    const res = await api.get<InspectionTemplate>(`/inspection-templates/${route.params.id}`)
    const data = res.data
    form.value = {
      name: data.name || '',
      description: data.description || '',
      is_active: data.is_active,
      scoring_mode: data.scoring_mode === 'deductive' ? 'deductive' : 'additive',
    }
    items.value = (data.items || []).map((it) => {
      itemKeySeed += 1
      const standardImages: string[] = it.standard_images ?? []
      const scoreType = it.score_type === 'pass_fail' ? 'pass_fail' : 'score'
      return {
        key: itemKeySeed,
        id: it.id,
        category: it.category || '',
        title: it.title || '',
        standard: it.standard || '',
        standard_images: standardImages,
        standardImages,
        score_type: scoreType,
        score_options:
          it.score_options && it.score_options.length > 0
            ? it.score_options.map((o) => ({ score: o.score, label: o.label || '' }))
            : cloneOptions(
                scoreType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS,
              ),
        require_remark: it.require_remark,
        require_photo: it.require_photo,
        show_remark: it.show_remark ?? true,
        show_photo: it.show_photo ?? true,
        category_precondition_enabled: it.category_precondition_enabled ?? false,
        category_precondition: it.category_precondition || '',
      }
    })
    autoExpandItemsWithData()
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载模板失败')
  }
}

async function handleSave() {
  if (!form.value.name.trim()) {
    highlightSection('name')
    Message.warning('请填写模板名称')
    return
  }
  if (items.value.length === 0) {
    highlightSection('items')
    Message.warning('请至少添加一个检查项')
    return
  }
  for (let g = 0; g < groupedItems.value.length; g++) {
    const group = groupedItems.value[g]
    if (!group.category.trim()) {
      const nextCollapsed = new Set(collapsedCategories.value)
      nextCollapsed.delete(g)
      collapsedCategories.value = nextCollapsed
      const errs = new Set(categoryErrors.value)
      errs.add(g)
      categoryErrors.value = errs
      markProgrammaticScroll()
      nextTick(() => {
        const cards = document.querySelectorAll('.max-w-4xl.relative > .rounded-2xl')
        const target = cards[g + 1]
        const input = target?.querySelector('.category-name-input input') as HTMLInputElement | null
        if (input) {
          input.scrollIntoView({ behavior: 'smooth', block: 'center' })
          input.focus()
          input.select()
        } else if (target) {
          target.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      })
      Message.warning('请填写所有分类名称')
      return
    }
    // 前置条件开启时内容必填
    if (
      group.items.length > 0 &&
      group.items[0].category_precondition_enabled &&
      !group.items[0].category_precondition.trim()
    ) {
      const nextCollapsed = new Set(collapsedCategories.value)
      nextCollapsed.delete(g)
      collapsedCategories.value = nextCollapsed
      const errs = new Set(categoryErrors.value)
      errs.add(g)
      categoryErrors.value = errs
      markProgrammaticScroll()
      nextTick(() => {
        const cards = document.querySelectorAll('.max-w-4xl.relative > .rounded-2xl')
        const target = cards[g + 1]
        const textarea = target?.querySelector(
          'textarea[placeholder="请输入前置条件内容（50 字以内）"]',
        ) as HTMLTextAreaElement | null
        if (textarea) {
          textarea.scrollIntoView({ behavior: 'smooth', block: 'center' })
          textarea.focus()
        } else if (target) {
          target.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      })
      Message.warning(`分类「${group.category || '未命名分类'}」的前置条件内容不能为空`)
      return
    }
  }
  for (let i = 0; i < items.value.length; i++) {
    const item = items.value[i]
    if (!item.title.trim()) {
      shakeItemKey(item.key)
      Message.warning(`第 ${i + 1} 项检查项标题不能为空`)
      return
    }
    if (!item.score_options || item.score_options.length === 0) {
      shakeItemKey(item.key)
      Message.warning(`第 ${i + 1} 项「${item.title}」的评分选项不能为空`)
      return
    }
    for (const opt of item.score_options) {
      if (!opt.label.trim()) {
        shakeItemKey(item.key)
        Message.warning(`第 ${i + 1} 项「${item.title}」的评分选项描述不能为空`)
        return
      }
    }
    const labels = item.score_options.map((o) => o.label.trim())
    if (new Set(labels).size !== labels.length) {
      shakeItemKey(item.key)
      Message.warning(`第 ${i + 1} 项「${item.title}」的选项描述不能重复`)
      return
    }
    if (item.score_type === 'score') {
      const scores = item.score_options.map((o) => Number(o.score))
      for (const s of scores) {
        if (!ALLOWED_SCORE_VALUES.includes(s)) {
          shakeItemKey(item.key)
          Message.warning(
            `第 ${i + 1} 项「${item.title}」的分值仅允许 ${ALLOWED_SCORE_VALUES.join('、')}`,
          )
          return
        }
      }
      if (new Set(scores).size !== scores.length) {
        shakeItemKey(item.key)
        Message.warning(`第 ${i + 1} 项「${item.title}」的分值不能重复`)
        return
      }
      if (!scores.includes(MIN_SCORE) || !scores.includes(MAX_SCORE)) {
        shakeItemKey(item.key)
        Message.warning(
          `第 ${i + 1} 项「${item.title}」必须包含最小值 ${MIN_SCORE} 与最大值 ${MAX_SCORE}`,
        )
        return
      }
      for (let j = 1; j < scores.length; j++) {
        if (scores[j] <= scores[j - 1]) {
          shakeItemKey(item.key)
          Message.warning(`第 ${i + 1} 项「${item.title}」的选项分值必须从小到大排列`)
          return
        }
      }
    }
  }
  for (const areaRef of standardAreaRefs.values()) {
    if (!(await areaRef.flushPending())) return
  }
  saving.value = true
  try {
    const itemPayloads: InspectionTemplateItemPayload[] = []
    for (const item of items.value) {
      itemPayloads.push({
        category: item.category.trim(),
        title: item.title.trim(),
        standard: item.standard,
        standard_images: item.standardImages,
        score_type: item.score_type,
        score_options: item.score_options.map((o) => ({
          score: Number(o.score) || 0,
          label: o.label.trim(),
        })),
        require_remark: item.require_remark,
        require_photo: item.require_photo,
        show_remark: item.show_remark,
        show_photo: item.show_photo,
        category_precondition_enabled: item.category_precondition_enabled,
        category_precondition: item.category_precondition_enabled
          ? item.category_precondition.trim()
          : '',
      })
    }
    const payload = {
      name: form.value.name.trim(),
      description: form.value.description,
      is_active: form.value.is_active,
      scoring_mode: form.value.scoring_mode,
      items: itemPayloads,
    }
    if (isEdit.value) {
      await api.put(`/inspection-templates/${route.params.id}`, payload)
      Message.success('模板已更新')
    } else {
      await api.post('/inspection-templates/', payload)
      Message.success('模板已创建')
    }
    router.push({ name: 'InspectionTemplateList' })
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '保存失败')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push({ name: 'InspectionTemplateList' })
}

// ============ 从素材库批量添加 ============
const materialPickerVisible = ref(false)
const materialLoading = ref(false)
const materialList = ref<InspectionMaterial[]>([])
const materialTotal = ref(0)
const materialPage = ref(1)
const materialPageSize = ref(10)
const materialKeyword = ref('')
const selectedMaterialIds = ref<string[]>([])
const selectedMaterialsMap = ref<Map<string, InspectionMaterial>>(new Map())
// 目标分类：指定时，所选素材全部添加到该分类；为空时按素材自身分类归组
const materialPickerTargetCategory = ref<string | null>(null)

const materialColumns = [
  { title: '素材标题', dataIndex: 'title', slotName: 'matTitle', width: 200 },
  { title: '分类', dataIndex: 'category', slotName: 'matCategory', width: 120 },
  {
    title: '检查标准',
    dataIndex: 'standard',
    slotName: 'matStandard',
    ellipsis: true,
    tooltip: true,
  },
  { title: '标准图', dataIndex: 'standard_images', slotName: 'matImage', width: 80 },
  { title: '评分', dataIndex: 'score_type', slotName: 'matScore', width: 100 },
]

async function fetchMaterials() {
  materialLoading.value = true
  try {
    const params: Record<string, unknown> = {
      page: materialPage.value,
      page_size: materialPageSize.value,
    }
    if (materialKeyword.value.trim()) params.keyword = materialKeyword.value.trim()
    const res = await api.get<Paginated<InspectionMaterial>>('/inspection-materials/', { params })
    materialList.value = res.data.items || []
    materialTotal.value = res.data.total || 0
  } catch (e) {
    Message.error(getApiErrorDetail(e) || '加载素材库失败')
  } finally {
    materialLoading.value = false
  }
}

// targetCategory: 指定目标分类名（添加到该分类）；不传则按素材自身分类归组
function openMaterialPicker(targetCategory?: string) {
  materialPickerVisible.value = true
  selectedMaterialIds.value = []
  selectedMaterialsMap.value = new Map()
  materialPage.value = 1
  materialKeyword.value = ''
  materialPickerTargetCategory.value = typeof targetCategory === 'string' ? targetCategory : null
  fetchMaterials()
}

function onMaterialSearch() {
  materialPage.value = 1
  fetchMaterials()
}

function onMaterialPageChange(p: number) {
  materialPage.value = p
  fetchMaterials()
}

function onMaterialPageSizeChange(size: number) {
  materialPageSize.value = size
  materialPage.value = 1
  fetchMaterials()
}

function onMaterialSelectionChange(keys: (string | number)[]) {
  const currentIds = new Set(materialList.value.map((m) => m.id))
  const keySet = new Set(keys)
  for (const m of materialList.value) {
    if (keySet.has(m.id)) {
      selectedMaterialsMap.value.set(m.id, m)
    } else if (currentIds.has(m.id)) {
      selectedMaterialsMap.value.delete(m.id)
    }
  }
  selectedMaterialIds.value = Array.from(selectedMaterialsMap.value.keys())
}

function createRowFromMaterial(m: InspectionMaterial, fallbackCategory: string): ItemRow {
  itemKeySeed += 1
  const scoreType = m.score_type === 'pass_fail' ? 'pass_fail' : 'score'
  const options =
    m.score_options && m.score_options.length > 0
      ? m.score_options.map((o) => ({ score: o.score, label: o.label || '' }))
      : cloneOptions(scoreType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS)
  const standardImages: string[] = m.standard_images ?? []
  return {
    key: itemKeySeed,
    id: '',
    category: (m.category || '').trim() || fallbackCategory,
    title: m.title || '',
    standard: m.standard || '',
    standard_images: standardImages,
    standardImages,
    score_type: scoreType,
    score_options: options,
    require_remark: false,
    require_photo: false,
    show_remark: true,
    show_photo: true,
    category_precondition_enabled: false,
    category_precondition: '',
  }
}

function confirmAddFromMaterials() {
  if (selectedMaterialsMap.value.size === 0) {
    Message.warning('请至少勾选一个素材')
    return
  }
  const groups = groupedItems.value
    .filter((g) => g.items.length > 0)
    .map((g) => ({ category: g.category, items: [...g.items] }))
  const targetCategory = materialPickerTargetCategory.value
  const fallbackCategory = groupedItems.value[activeCategory.value]?.category || ''
  let insertCount = 0
  let firstNewKey = 0
  for (const m of selectedMaterialsMap.value.values()) {
    const row = createRowFromMaterial(m, fallbackCategory)
    if (firstNewKey === 0) firstNewKey = row.key
    // 指定目标分类时，强制归入该分类；否则按素材自身分类归组
    const cat =
      targetCategory !== null ? targetCategory : (m.category || '').trim() || fallbackCategory
    row.category = cat
    let idx = groups.findIndex((g) => g.category === cat)
    if (idx === -1) {
      groups.push({ category: cat, items: [] })
      idx = groups.length - 1
    }
    groups[idx].items.push(row)
    insertCount += 1
  }
  if (groups.length === 0) {
    groups.push({ category: '', items: [] })
  }
  syncItemsFromGroups(groups)
  materialPickerVisible.value = false
  Message.success(`已添加 ${insertCount} 个检查项`)
  collapsedCategories.value = new Set()
  if (firstNewKey > 0) {
    markProgrammaticScroll()
    nextTick(() => {
      const el = itemEls[firstNewKey]
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    })
  }
}

function confirmAIGeneratedItems(result: {
  name: string
  description: string
  items: AIGeneratedItem[]
}) {
  const aiItems = result.items
  if (!aiItems.length && !result.name && !result.description) return
  if (result.name && !form.value.name.trim()) {
    form.value.name = result.name
  }
  if (result.description && !form.value.description.trim()) {
    form.value.description = result.description
  }
  if (!aiItems.length) return
  const groups = groupedItems.value
    .filter((g) => g.items.length > 0)
    .map((g) => ({ category: g.category, items: [...g.items] }))
  let firstNewKey = 0
  for (const it of aiItems) {
    itemKeySeed += 1
    const category = it.category.trim() || ''
    const standardImages: string[] = it.standard_images ?? []
    const scoreType = it.score_type === 'pass_fail' ? 'pass_fail' : 'score'
    const options =
      it.score_options && it.score_options.length > 0
        ? it.score_options.map((o) => ({ score: o.score, label: o.label || '' }))
        : cloneOptions(
            scoreType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS,
          )
    const row: ItemRow = {
      key: itemKeySeed,
      id: '',
      category,
      title: it.title || '',
      standard: it.standard || '',
      standard_images: standardImages,
      standardImages,
      score_type: scoreType,
      score_options: options,
      require_remark: false,
      require_photo: false,
      show_remark: true,
      show_photo: true,
      category_precondition_enabled: false,
      category_precondition: '',
    }
    if (firstNewKey === 0) firstNewKey = row.key
    let idx = groups.findIndex((g) => g.category === category)
    if (idx === -1) {
      groups.push({ category, items: [] })
      idx = groups.length - 1
    }
    groups[idx].items.push(row)
  }
  if (groups.length === 0) {
    groups.push({ category: '', items: [] })
  }
  syncItemsFromGroups(groups)
  Message.success(`已添加 ${aiItems.length} 个检查项`)
  collapsedCategories.value = new Set()
  if (firstNewKey > 0) {
    markProgrammaticScroll()
    nextTick(() => {
      const el = itemEls[firstNewKey]
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    })
  }
}

onMounted(async () => {
  loading.value = true
  try {
    if (isEdit.value) {
      await fetchDetail()
    } else {
      addItem()
    }
  } finally {
    loading.value = false
  }
  const mainEl = document.querySelector('main')
  if (mainEl) mainEl.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll)
  updateSidebarOffset()
  window.addEventListener('resize', updateSidebarOffset)
  nextTick(() => startSidebarObserver())
})

onUnmounted(() => {
  const mainEl = document.querySelector('main')
  if (mainEl) mainEl.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
  window.removeEventListener('resize', updateSidebarOffset)
  stopSidebarObserver()
})
</script>

<style scoped>
@keyframes shake {
  0%,
  100% {
    transform: translateX(0);
  }
  10%,
  50%,
  90% {
    transform: translateX(-3px);
  }
  30%,
  70% {
    transform: translateX(3px);
  }
}
.shake-error {
  animation: shake 0.45s ease-in-out;
}
.shake-error-wrapper {
  animation: shake 0.45s ease-in-out;
  border-color: #ff3b30 !important;
  outline: 2px solid rgba(255, 59, 48, 0.3);
  border-radius: 8px;
}
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.25s ease;
}
.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(8px);
}

.tree-toggle {
  cursor: pointer;
  display: inline-flex;
  flex-shrink: 0;
}

.page-main-flush {
  min-height: 100%;
}

/* Collapse 展开/收起高度过渡 */
.collapse-enter-active,
.collapse-leave-active {
  transition: all 0.32s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}
.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  transform: translateY(-4px);
  max-height: 0;
}
.collapse-enter-to,
.collapse-leave-from {
  opacity: 1;
  transform: translateY(0);
  max-height: 3000px;
}

/* 检查标准 + 标准图一体化输入区（AttachmentInputArea 公共组件内置样式） */
</style>
