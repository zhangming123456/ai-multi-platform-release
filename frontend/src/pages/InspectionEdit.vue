<template>
  <div class="page-main">
    <PageHeader
      :title="isEdit ? '编辑巡店' : '发起巡店'"
      subtitle="选择检查表模板并评分，支持上传照片并使用 AI 生成检查报告"
    />

    <div
      v-if="isEdit && !isEditable"
      class="mx-4 md:mx-6 lg:mx-8 mt-4 px-4 py-3 rounded-lg bg-[#FFF3E0] text-[#F57C00] text-[13px] flex items-center gap-2"
    >
      <IconExclamationCircle :size="16" />
      当前巡店已进入整改流程，不可编辑。
    </div>

    <DefineAIInspectForm v-slot="{ selectSize, fromDrawer, hintClass, showHeader, wrapperClass }">
      <div :class="[wrapperClass]">
        <div v-if="showHeader" class="flex items-center gap-2 mb-1">
          <span class="text-[15px] font-semibold text-[#1D1D1F]">AI 巡店</span>
          <a-tag color="arcoblue" size="small">实时分析</a-tag>
        </div>
        <div class="text-[12px] text-[#86868b] mb-4">
          检查项内图片、检查项内反馈问题、AI 巡店图片与巡店关键词至少提供一种，AI
          将按检查表标准分析并生成问题与备注、AI 整改建议。
        </div>
        <a-form :model="form" layout="vertical">
          <a-form-item label="模型">
            <ModelSelect
              :model-value="activeModelKey"
              :auto-select="false"
              :require-vision="selectedHasVision"
              :size="selectSize as any"
              @change="onModelChange"
            />
          </a-form-item>
          <a-form-item label="巡店现场描述">
            <div
              class="feedback-input-area"
              :class="{ 'feedback-input-area--dragging': isAIDragging }"
              @dragenter="handleAIDragEnter"
              @dragleave="handleAIDragLeave"
              @dragover="handleAIDragOver"
              @drop="handleAIDrop"
            >
              <!-- 已上传图片预览 -->
              <div v-if="aiPhotos.length > 0" class="feedback-thumbs">
                <div v-for="(photo, pIdx) in aiPhotos" :key="photo.uid" class="feedback-thumb-item">
                  <img :src="photo.url" class="feedback-thumb-img" />
                  <button type="button" class="feedback-thumb-remove" @click="removeAIPhoto(pIdx)">
                    <IconClose :size="12" />
                  </button>
                  <span class="feedback-thumb-name">{{ photo.name }}</span>
                </div>
              </div>

              <!-- 文本输入区 -->
              <a-textarea
                v-model="aiKeywords"
                placeholder="填写巡店现场发现，例如：门口地垫破损、消防通道被遮挡、收银台前有杂物堆放…&#10;支持拖拽/粘贴图片，或粘贴图片 URL 自动识别"
                :auto-size="{ minRows: 3, maxRows: 6 }"
                :max-length="500"
                show-word-limit
                class="feedback-textarea"
                @paste="handleAIPaste"
                @input="handleAIInput"
                @keydown="handleAIKeydown"
              />

              <!-- 工具栏 -->
              <div class="feedback-toolbar">
                <div class="feedback-toolbar-left">
                  <button
                    type="button"
                    class="feedback-toolbar-btn"
                    :disabled="aiPhotos.length >= 10"
                    @click="triggerAIUpload(fromDrawer)"
                  >
                    <IconImage :size="16" />
                  </button>
                  <span class="text-[12px] text-[#86868b]">图片</span>
                  <span class="text-[11px] text-[#86868b]">{{ aiPhotos.length }}/10</span>
                  <span class="text-[11px] text-[#86868b]">单张≤10MB</span>
                </div>
              </div>

              <!-- 隐藏文件输入 -->
              <input
                :ref="(el: any) => (fromDrawer ? setAIDrawerFileInput(el) : setAIFileInput(el))"
                type="file"
                accept="image/*"
                multiple
                class="hidden"
                @change="(e: Event) => onAIFileInputChange(e)"
              />
            </div>
          </a-form-item>
        </a-form>
        <a-button
          v-perm="'inspection:ai:write'"
          type="primary"
          long
          size="large"
          :loading="isAnalyzing"
          :disabled="
            !form.store_id ||
            !form.template_id ||
            (allRowPhotos.length === 0 &&
              !hasRowComments &&
              aiPhotos.length === 0 &&
              !aiKeywords.trim()) ||
            isAnalyzing ||
            !scrolledToBottom
          "
          @click="handleAnalyze"
        >
          <template #icon v-if="!isAnalyzing"><IconRobot /></template>
          {{ isAnalyzing ? 'AI 正在分析...' : '开始 AI 巡店' }}
        </a-button>
        <div :class="['text-[12px] text-[#86868b] mt-3', hintClass]">
          <template v-if="!form.store_id">{{
            fromDrawer ? '请先选择巡店门店。' : '请先选择门店。'
          }}</template>
          <template v-else-if="!form.template_id">请先选择检查表模板。</template>
          <template
            v-else-if="
              allRowPhotos.length === 0 &&
              !hasRowComments &&
              aiPhotos.length === 0 &&
              !aiKeywords.trim()
            "
            >请至少提供一种巡店依据：检查项内图片、检查项内反馈问题、AI 巡店图片或关键词。</template
          >
          <template v-else-if="!scrolledToBottom">请滚动至页面最底部以解锁 AI 巡店功能。</template>
          <template v-else>本次将基于 {{ aiBasisDescription }} 进行实时分析。</template>
        </div>
      </div>
    </DefineAIInspectForm>

    <DefineAIReportBlock v-slot="{ wrapperClass }">
      <div
        v-if="aiReport"
        :class="[
          'rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5',
          wrapperClass,
        ]"
      >
        <div class="flex items-center justify-between mb-3">
          <span class="text-[15px] font-semibold text-[#1D1D1F]">AI 巡店分析报告</span>
          <button
            type="button"
            class="text-[12px] text-[#007AFF] hover:text-[#0056cc] transition-colors"
            @click="copyAIReport"
          >
            <IconCopy :size="14" class="inline-block mr-1" />复制报告
          </button>
        </div>

        <div
          v-if="aiReport.summary"
          class="mb-4 rounded-xl bg-[#007AFF]/5 px-4 py-3 text-[13px] leading-relaxed text-[#1D1D1F]"
        >
          <span class="font-semibold text-[#007AFF]">总结：</span>{{ aiReport.summary }}
        </div>

        <div v-if="aiReport.high_risk_problems?.length" class="mb-4">
          <div class="flex items-center gap-2 mb-2">
            <span class="inline-block w-2 h-2 rounded-full bg-[#FF3B30]"></span>
            <span class="text-[13px] font-semibold text-[#FF3B30]">高危风险问题</span>
          </div>
          <div
            v-for="(p, i) in aiReport.high_risk_problems"
            :key="'hr' + i"
            class="mb-2 last:mb-0 rounded-lg border border-[#FF3B30]/20 bg-[#FF3B30]/5 px-3 py-2"
          >
            <div class="text-[13px] font-medium text-[#1D1D1F]">
              {{ p.item_name || '高危问题' }}
            </div>
            <div class="text-[12px] text-[#3C3C43] mt-0.5">{{ p.desc }}</div>
          </div>
        </div>

        <div v-if="aiReport.main_problems?.length" class="mb-4">
          <div class="flex items-center gap-2 mb-2">
            <span class="inline-block w-2 h-2 rounded-full bg-[#FF9500]"></span>
            <span class="text-[13px] font-semibold text-[#FF9500]">主要问题</span>
          </div>
          <div
            v-for="(p, i) in aiReport.main_problems"
            :key="'mp' + i"
            class="mb-2 last:mb-0 rounded-lg border border-[#E5E5EA] bg-white/60 px-3 py-2"
          >
            <div class="flex items-center gap-2">
              <span class="text-[13px] font-medium text-[#1D1D1F]">{{
                p.item_name || '问题'
              }}</span>
              <a-tag v-if="p.level" size="small" :color="levelColor(p.level)">{{ p.level }}</a-tag>
            </div>
            <div class="text-[12px] text-[#3C3C43] mt-0.5">{{ p.desc }}</div>
          </div>
        </div>

        <div v-if="aiReport.priority_suggest?.length" class="mb-4">
          <div class="flex items-center gap-2 mb-2">
            <span class="inline-block w-2 h-2 rounded-full bg-[#007AFF]"></span>
            <span class="text-[13px] font-semibold text-[#007AFF]">优先整改建议</span>
          </div>
          <div
            v-for="(s, i) in aiReport.priority_suggest"
            :key="'ps' + i"
            class="mb-2 last:mb-0 rounded-lg bg-[#007AFF]/5 px-3 py-2"
          >
            <div class="text-[13px] font-medium text-[#1D1D1F]">{{ i + 1 }}. {{ s.title }}</div>
            <div class="text-[12px] text-[#3C3C43] mt-0.5">{{ s.desc }}</div>
          </div>
        </div>

        <div v-if="aiReport.business_suggest?.length">
          <div class="flex items-center gap-2 mb-2">
            <span class="inline-block w-2 h-2 rounded-full bg-[#34C759]"></span>
            <span class="text-[13px] font-semibold text-[#34C759]">运营优化建议</span>
          </div>
          <div
            v-for="(s, i) in aiReport.business_suggest"
            :key="'bs' + i"
            class="mb-2 last:mb-0 rounded-lg bg-[#34C759]/5 px-3 py-2"
          >
            <div class="text-[13px] font-medium text-[#1D1D1F]">{{ s.title }}</div>
            <div class="text-[12px] text-[#3C3C43] mt-0.5">{{ s.desc }}</div>
          </div>
        </div>
      </div>
    </DefineAIReportBlock>

    <DefineAiInspectPanel v-slot="{ wrapperClass }">
      <div class="ai-inspect-panel" :class="wrapperClass">
        <div class="ai-inspect-panel__bar">
          <div class="flex items-center gap-2">
            <IconCode :size="14" style="color: #8e8e93" />
            <span class="ai-inspect-panel__title">调用日志</span>
            <span v-if="isAnalyzing" class="log-live">
              <span class="log-live__pulse"></span>
              实时分析中
            </span>
            <span v-else-if="aiLogs.length > 0" class="log-idle">空闲</span>
          </div>
          <div class="flex items-center gap-1">
            <span class="ai-inspect-panel__count">{{ aiLogs.length }} 条</span>
            <button class="ai-inspect-panel__btn" title="清空日志" @click="clearAILogs">
              <IconDelete :size="13" />
            </button>
          </div>
        </div>
        <!-- 日志打印 -->
        <div class="ai-inspect-panel__body-wrap">
          <div class="ai-inspect-panel__body">
            <div class="ai-inspect-panel__body-main">
              <div v-if="aiLogs.length === 0" class="ai-inspect-panel__empty">
                <span class="ai-inspect-panel__prompt">➜</span>
                点击「开始 AI 巡店」后，这里将实时输出调用日志
              </div>
              <template v-for="entry in aiLogs" :key="entry.id">
                <div
                  class="log-line"
                  :class="[`log-line--${entry.level}`, { 'log-line--has-detail': entry.detail }]"
                >
                  <div class="log-line__row" @click="toggleLogDetail(entry.id)">
                    <span class="log-line__time">{{ entry.time }}</span>
                    <span class="log-line__level">{{ levelText(entry.level) }}</span>
                    <span class="log-line__msg">{{ entry.message }}</span>
                    <span v-if="entry.detail" class="log-line__toggle">
                      {{ logExpanded.has(entry.id) ? '▾' : '▸' }}
                    </span>
                  </div>
                </div>
                <div
                  v-if="entry.detail && logExpanded.has(entry.id)"
                  class="log-line"
                  :class="[`log-line--${entry.level}`, { 'log-line--has-detail': entry.detail }]"
                >
                  <div class="log-line__row">
                    <span class="log-line__time">{{ entry.time }}</span>
                    <span class="log-line__level">{{ levelText(entry.level) }}</span>
                    <pre class="log-line__detail">{{ entry.detail }}</pre>
                  </div>
                </div>
              </template>
              <div v-if="isAnalyzing" class="log-line log-line--cursor">
                <span class="ai-inspect-panel__prompt">➜</span>
                <span class="log-cursor"></span>
              </div>
            </div>
          </div>
        </div>
        <!-- 流式输出预览 -->
        <div v-if="aiStreamingText" class="ai-inspect-stream">
          <div class="ai-inspect-stream__header">
            <span class="text-[13px] font-medium">{{
              aiStreaming ? 'AI 正在输出' : 'AI 输出预览'
            }}</span>
            <span v-if="aiStreaming" class="ai-inspect-stream__badge">实时</span>
          </div>
          <div class="ai-inspect-stream__text">
            {{ aiStreamingText }}<span v-if="aiStreaming" class="streaming-cursor"></span>
          </div>
        </div>
      </div>
    </DefineAiInspectPanel>

    <div class="px-4 md:px-6 lg:px-8 flex-1 pb-24">
      <a-spin :loading="loading" tip="加载中..." class="w-full">
        <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
          <div class="xl:col-span-2 space-y-4">
            <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
              <div class="flex items-center justify-between mb-4">
                <div class="text-[15px] font-semibold text-[#1D1D1F]">基本信息</div>
                <a-tag v-if="form.ai_generated" size="small" color="green" class="!m-0">
                  <IconRobot :size="12" /> AI 巡店生成
                </a-tag>
              </div>
              <a-form :model="form" layout="vertical">
                <a-row :gutter="16">
                  <a-col :span="24" :lg="10">
                    <a-form-item label="巡店门店" required>
                      <a-select
                        v-model="form.store_id"
                        placeholder="请选择门店"
                        :options="storeOptions"
                        @change="onStoreChange"
                      />
                    </a-form-item>
                  </a-col>
                  <a-col :span="24" :lg="8">
                    <a-form-item label="检查时间">
                      <a-date-picker
                        v-model="form.checked_at"
                        show-time
                        :placeholder="'选择检查时间'"
                        style="width: 100%"
                      />
                    </a-form-item>
                  </a-col>
                  <a-col :span="24" :lg="6">
                    <a-form-item label="检查人">
                      <a-input :model-value="inspectorName" disabled />
                    </a-form-item>
                  </a-col>
                </a-row>
                <a-form-item class="flex-wrap" label="检查表模板" required>
                  <a-select
                    v-model="form.template_id"
                    placeholder="请选择检查表模板"
                    :options="templateOptions"
                    :disabled="isEdit && scoreRows.length > 0"
                    @change="onTemplateChange"
                  />
                  <template #extra>
                    <div class="text-[12px] text-[#86868b] mt-1">
                      AI
                      巡店需选择检查表模板；选中的模板检查项会展开到下方进行评分；编辑已保存的巡店时模板不可切换。
                    </div>
                  </template>
                </a-form-item>
                <a-form-item label="检查主题">
                  <a-input v-model="form.title" placeholder="巡店检查主题" maxlength="100" />
                </a-form-item>
              </a-form>
            </div>

            <div
              ref="checkItemsRef"
              class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5"
            >
              <div class="flex items-center justify-between mb-4">
                <div class="text-[15px] font-semibold text-[#1D1D1F]">检查项评分</div>
                <a-tag :color="totalPassed ? 'green' : 'red'" size="small" v-if="maxTotal > 0">
                  {{ totalPassed ? '合格' : '不合格' }} · {{ totalScore.toFixed(1) }} /
                  {{ maxTotal }}
                </a-tag>
              </div>

              <div
                v-if="scoreRows.length === 0"
                class="text-[13px] text-[#86868b] py-6 text-center"
              >
                暂无检查项，请先选择检查表模板
              </div>

              <div
                v-for="catGroup in categoryGroups"
                :key="catGroup.category"
                class="mb-4 border border-[#E5E5EA] rounded-2xl bg-white overflow-hidden transition-colors"
                :class="
                  catGroup.precondition_enabled && !catGroup.precondition_checked
                    ? 'border-[#FF9500]/60 bg-[#FFF8EC]/50'
                    : ''
                "
              >
                <!-- 分类标题栏 -->
                <div
                  class="flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-[#F5F5F7] to-white border-b border-[#E5E5EA] cursor-pointer select-none"
                  @click="toggleCategory(catGroup)"
                >
                  <div
                    class="text-[#86868b] transition-transform duration-200 shrink-0"
                    :class="!catGroup.collapsed ? 'rotate-90' : ''"
                  >
                    <IconCaretRight :size="14" />
                  </div>
                  <div class="flex items-center gap-2 min-w-0 flex-1">
                    <span class="text-[14px] font-semibold text-[#1D1D1F] truncate">
                      {{ catGroup.category }}
                    </span>
                    <a-tag size="small" color="arcoblue" class="!m-0 shrink-0">
                      {{ catGroup.rows.length }} 项
                    </a-tag>
                    <a-tag
                      v-if="catGroup.precondition_enabled"
                      size="small"
                      :color="catGroup.precondition_checked ? 'green' : 'orange'"
                      class="!m-0 shrink-0"
                    >
                      {{ catGroup.precondition_checked ? '前置条件已满足' : '含前置条件' }}
                    </a-tag>
                  </div>

                  <!-- 前置条件勾选区 -->
                  <div
                    v-if="catGroup.precondition_enabled"
                    class="flex items-center gap-2 shrink-0 ml-2"
                    @click.stop
                  >
                    <a-checkbox v-model="catGroup.precondition_checked" size="small">
                      <span class="text-[12px] text-[#1D1D1F]">
                        {{ catGroup.precondition || '已满足前置条件' }}
                      </span>
                    </a-checkbox>
                  </div>
                </div>

                <!-- 前置条件提示（未勾选时） -->
                <div
                  v-if="
                    catGroup.precondition_enabled &&
                    !catGroup.precondition_checked &&
                    !catGroup.collapsed
                  "
                  class="px-4 py-2 bg-[#FFF8EC] border-b border-[#FFE9C2] flex items-center gap-2"
                >
                  <span class="text-[12px] text-[#FF9500]">
                    ⚠️ 请先勾选上方前置条件，方可对该分类下的检查项进行评分操作。
                  </span>
                </div>

                <!-- 分类下的检查项列表 -->
                <div v-show="!catGroup.collapsed" class="p-3 space-y-3">
                  <div
                    v-for="(row, idxInCat) in catGroup.rows"
                    :key="row.item_id"
                    :ref="(el: any) => setRowRef(catGroup.rowIndices[idxInCat], el)"
                    class="rounded-xl border border-[#E5E5EA] bg-[#FAFAFA] p-4 transition-all duration-300"
                    :class="[
                      activeRowIndex === catGroup.rowIndices[idxInCat]
                        ? '!border-[#165DFF] shadow-sm bg-white'
                        : '',
                      !isCategoryEditable(catGroup) ? 'opacity-60 pointer-events-none' : '',
                    ]"
                  >
                    <div class="flex flex-col gap-2">
                      <div class="flex items-center justify-between gap-2">
                        <div class="flex items-center gap-2 min-w-0">
                          <span class="text-[13px] font-medium text-[#1D1D1F]">{{
                            row.item_name
                          }}</span>
                          <a-tag v-if="row.ai_generated" size="small" color="green" class="!m-0">
                            <IconRobot :size="12" /> AI
                          </a-tag>
                          <a-tag
                            v-if="row.score_type === 'pass_fail'"
                            size="small"
                            color="purple"
                            class="!m-0"
                          >
                            选项评分
                          </a-tag>
                          <a-tag v-else size="small" color="arcoblue" class="!m-0"
                            >分值评分 · 满分 {{ row.max_score }} 分</a-tag
                          >
                        </div>
                        <span
                          class="text-[13px] font-semibold shrink-0"
                          :class="isRowPassed(row) ? 'text-[#34C759]' : 'text-[#FF3B30]'"
                        >
                          {{ scoreLabel(row) }}
                          <span
                            v-if="row.score_type !== 'pass_fail'"
                            class="text-[#86868b] font-normal text-[12px]"
                            >/ {{ row.max_score }}</span
                          >
                        </span>
                      </div>

                      <div
                        v-if="row.standard || row.standard_image"
                        class="flex items-start gap-3 rounded-lg bg-[#F5F5F7] p-2.5"
                      >
                        <a-image
                          v-if="row.standard_image"
                          :src="row.standard_image"
                          :preview-src="row.standard_image"
                          :width="56"
                          :height="56"
                          fit="cover"
                          class="!rounded-lg overflow-hidden shrink-0 !w-14 !h-14"
                        />
                        <div
                          v-if="row.standard"
                          class="text-[12px] text-[#86868b] leading-relaxed flex-1 min-w-0"
                        >
                          检查标准：{{ row.standard }}
                        </div>
                      </div>

                      <!-- 评分下拉 -->
                      <div class="flex items-center gap-2">
                        <span class="text-[12px] text-[#86868b] shrink-0">评分</span>
                        <div class="w-40 shrink-0">
                          <a-select
                            :model-value="Number(row.score)"
                            size="small"
                            class="!w-full"
                            :disabled="!isCategoryEditable(catGroup)"
                            @change="(val: any) => onScoreOptionChange(row, val)"
                          >
                            <a-option
                              v-for="opt in scoreSelectOptions(row)"
                              :key="opt.score"
                              :value="opt.score"
                            >
                              {{ opt.label }}
                            </a-option>
                          </a-select>
                        </div>
                      </div>

                      <!-- 反馈问题 + 巡店图片（创作内容风格一体化输入区） -->
                      <AttachmentInputArea
                        v-if="row.show_remark || row.show_photo"
                        :ref="(el: any) => setScoreAreaRef(row, el)"
                        v-model="row.photos"
                        v-model:text="row.comment"
                        :upload="uploadImageFile"
                        :max-count="MAX_ROW_PHOTOS"
                        :disabled="!isCategoryEditable(catGroup)"
                        :show-textarea="row.show_remark"
                        :show-images="row.show_photo"
                        :max-length="300"
                        :min-rows="3"
                        :max-rows="8"
                        :placeholder="commentPlaceholder(row)"
                        @update:text="onRowCommentInput(row, $event)"
                      >
                        <template #toolbar-right>
                          <a-button
                            v-if="form.store_id && isCategoryEditable(catGroup)"
                            size="mini"
                            type="text"
                            :loading="row._aiScoring"
                            :disabled="!selectedPlanId || !selectedModelId"
                            class="!text-[#007AFF] !px-1.5 !h-5 !text-[11px]"
                            @click.stop="handleSingleItemAI(row)"
                          >
                            <template #icon><IconRobot :size="12" /></template>
                            AI 生成
                          </a-button>
                          <span
                            v-if="isRowPassed(row) && row.require_remark"
                            class="text-[12px] text-[#FF3B30]"
                            >反馈必填</span
                          >
                          <span v-else-if="!isRowPassed(row)" class="text-[12px] text-[#FF3B30]"
                            >反馈必填</span
                          >
                          <span
                            v-if="isRowPassed(row) && row.require_photo"
                            class="text-[12px] text-[#FF3B30]"
                            >图片必填</span
                          >
                          <span v-else-if="!isRowPassed(row)" class="text-[12px] text-[#FF3B30]"
                            >图片必填</span
                          >
                        </template>
                      </AttachmentInputArea>

                      <!-- AI 整改建议 -->
                      <div
                        v-if="row.ai_suggestion"
                        class="rounded-lg border border-[#E8F1FF] bg-[#F0F7FF] p-2.5"
                      >
                        <div class="flex items-center gap-1.5 mb-1">
                          <IconRobot :size="13" class="text-[#007AFF]" />
                          <span class="text-[12px] font-medium text-[#1D1D1F]">AI 整改建议</span>
                        </div>
                        <div class="text-[12px] leading-relaxed text-[#3C3C43] whitespace-pre-wrap">
                          {{ row.ai_suggestion }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
              <div class="flex items-center justify-between mb-4">
                <div class="text-[15px] font-semibold text-[#1D1D1F]">问题与备注</div>
                <a-tag
                  v-if="form.issues && form.issues === aiOriginalIssues"
                  size="small"
                  color="green"
                  class="!m-0"
                >
                  <IconRobot :size="12" /> AI 生成
                </a-tag>
              </div>
              <a-textarea
                v-model="form.issues"
                placeholder="记录巡店中发现的问题与备注..."
                :auto-size="{ minRows: 3, maxRows: 8 }"
                class="!rounded-xl"
              />
            </div>

            <div class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5">
              <div class="flex items-center justify-between mb-3">
                <div class="text-[15px] font-semibold text-[#1D1D1F]">AI 整改建议</div>
                <a-tag
                  v-if="form.suggestion && form.suggestion === aiOriginalSuggestion"
                  size="small"
                  color="green"
                  class="!m-0"
                >
                  <IconRobot :size="12" /> AI 生成
                </a-tag>
              </div>
              <a-textarea
                v-model="form.suggestion"
                placeholder="AI 生成的整改建议会展示在这里，可手动编辑..."
                :auto-size="{ minRows: 3, maxRows: 8 }"
                class="!rounded-xl"
              />
              <div class="text-[12px] text-[#86868b] mt-1.5">
                AI 巡店分析生成的整改建议（可手动修改）
              </div>
            </div>
          </div>

          <!-- 大屏：右侧 AI 巡店实时对话面板 -->
          <div class="hidden xl:block" style="padding-left: 15px">
            <div class="sticky top-4 space-y-4">
              <!-- 实时对话 / 日志终端 -->
              <ReuseAiInspectPanel ref="aiInspectPanelRef" />
              <!-- AI 巡店分析报告 -->
              <ReuseAIReportBlock />
              <ReuseAIInspectForm
                select-size="small"
                :show-header="true"
                wrapper-class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5"
              />
            </div>
          </div>
        </div>

        <!-- AI 巡店 Drawer（小屏 + 快捷操作入口共用） -->
        <a-drawer
          v-model:visible="aiDrawerVisible"
          title="AI 巡店"
          :width="420"
          :footer="false"
          placement="right"
        >
          <div class="space-y-4">
            <!-- Drawer 内实时日志终端 -->
            <ReuseAiInspectPanel ref="aiDrawerInspectPanelRef" />
            <!-- AI 巡店分析报告 -->
            <ReuseAIReportBlock wrapper-class="mb-4" />
            <ReuseAIInspectForm
              select-size="small"
              :show-header="true"
              wrapper-class="rounded-2xl border border-[#E5E5EA] bg-white/70 backdrop-blur-xl p-5"
            />
          </div>
        </a-drawer>

        <transition name="fade-slide">
          <div
            v-if="showFloatingBar && scoreRows.length > 0"
            class="fixed z-40 flex flex-col items-center gap-1.5"
            :style="{ right: barRight + 'px', top: '50%', transform: 'translateY(-50%)' }"
          >
            <!-- 小屏：AI 巡店作为快捷操作第一项 -->
            <template v-if="!isLargeScreen">
              <a-tooltip content="AI 巡店" position="left">
                <a-button
                  v-perm="'inspection:ai:write'"
                  shape="circle"
                  size="small"
                  class="!shadow-md"
                  :disabled="isAnalyzing"
                  @click="aiDrawerVisible = true"
                >
                  <template #icon><IconRobot :size="14" class="text-white" /></template>
                </a-button>
              </a-tooltip>
              <div class="w-[1px] h-2 bg-[#E5E5EA]" />
            </template>
            <a-tooltip content="上一个检查项" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeRowIndex <= 0"
                @click="navRow(-1)"
              >
                <template #icon><IconUp :size="14" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip :content="`第 ${activeRowIndex + 1}/${scoreRows.length} 项`" position="left">
              <div
                class="flex flex-col gap-1 items-center justify-center rounded-xl bg-white border border-[#E5E5EA] shadow-md px-1 py-1.5 select-none"
              >
                <span class="text-[13px] font-bold text-[#165DFF] leading-none tabular-nums">
                  {{ activeRowIndex + 1 }}
                </span>
                <div class="border border-[#E5E5EA] w-full" />
                <span class="text-[13px] font-bold text-[#165DFF] leading-none tabular-nums">
                  {{ scoreRows.length }}
                </span>
              </div>
            </a-tooltip>
            <a-tooltip content="下一个检查项" position="left">
              <a-button
                shape="circle"
                size="small"
                class="!shadow-md"
                :disabled="activeRowIndex >= scoreRows.length - 1"
                @click="navRow(1)"
              >
                <template #icon><IconDown :size="14" /></template>
              </a-button>
            </a-tooltip>
            <div class="w-[1px] h-2 bg-[#E5E5EA]" />
            <template v-if="activeRowEditable">
              <a-tooltip content="一键满分" position="left">
                <a-button shape="circle" size="small" class="!shadow-md" @click="setRowFullScore">
                  <template #icon><IconCheck :size="14" class="text-[#34C759]" /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip content="一键清零" position="left">
                <a-button shape="circle" size="small" class="!shadow-md" @click="setRowZeroScore">
                  <template #icon><IconClose :size="14" class="text-[#FF3B30]" /></template>
                </a-button>
              </a-tooltip>
            </template>
          </div>
        </transition>
      </a-spin>
    </div>

    <div
      class="fixed bottom-0 right-0 z-30 border-t border-[#E5E5EA] bg-white/90 backdrop-blur-xl px-4 md:px-6 lg:px-8 py-3 flex items-center justify-between transition-all duration-350 ease-out"
      :style="{ left: 'var(--f-aside-width)' }"
    >
      <span class="text-[13px] text-[#86868b]">
        共 {{ scoreRows.length }} 个检查项，{{ categoryGroups.length }} 个分类
      </span>

      <div class="flex items-center gap-3">
        <a-button @click="goBack">返回</a-button>
        <a-button
          v-perm="isEdit ? 'inspection:update:write' : 'inspection:create:write'"
          type="primary"
          :loading="saving"
          :disabled="!isEditable || saving"
          @click="handleSave"
        >
          保存
        </a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch, unref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import { useResizeObserver, createReusableTemplate, useScroll, unrefElement } from '@vueuse/core'
import {
  IconRobot,
  IconUp,
  IconDown,
  IconCheck,
  IconClose,
  IconCaretRight,
  IconImage,
  IconCode,
  IconCopy,
  IconDelete,
  IconExclamationCircle,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import AttachmentInputArea from '@/components/AttachmentInputArea.vue'
import ModelSelect, { type ModelSelectValue } from '@/components/shared/ModelSelect.vue'
import { uploadImageFile } from '@/composables/useFileUpload'
import { extractImageUrlsFromText, cleanUrlsFromText } from '@/composables/useUrlExtractor'
import { useTokenPlanStore, parseModelField } from '@/stores/tokenPlan'
import { useUserStore } from '@/stores/user'
import type {
  Inspection,
  InspectionItem,
  InspectionScore,
  InspectionTemplate,
  InspectionTemplateItem,
  ScoreOption,
  Store,
} from '@/types'
import api from '@/utils/api'

const [DefineAIInspectForm, ReuseAIInspectForm] = createReusableTemplate<{
  selectSize?: string
  fromDrawer?: boolean
  hintClass?: string
  showHeader?: boolean
  wrapperClass?: string
}>()

const [DefineAIReportBlock, ReuseAIReportBlock] = createReusableTemplate<{
  wrapperClass?: string
}>()

const [DefineAiInspectPanel, ReuseAiInspectPanel] = createReusableTemplate<{
  wrapperClass?: string
}>()

interface PhotoItem {
  uid: string
  name: string
  url: string
  status: 'done' | 'init'
  file?: File
}

type LogLevel = 'info' | 'req' | 'ok' | 'err' | 'warn'

interface LogEntry {
  id: string
  time: string
  level: LogLevel
  message: string
  detail?: string
}

const logExpanded = reactive(new Set<string>())

const DEFAULT_SCORE_OPTIONS: ScoreOption[] = [
  { score: 0, label: '0分' },
  { score: 2, label: '2分' },
  { score: 5, label: '5分' },
]

const DEFAULT_PASS_FAIL_OPTIONS: ScoreOption[] = [
  { score: 1, label: '合格' },
  { score: 2, label: '不合格' },
]

function cloneOptions(options: ScoreOption[]): ScoreOption[] {
  return options.map((o) => ({ score: o.score, label: o.label }))
}

interface ScoreRow {
  item_id: string
  item_name: string
  category: string
  category_precondition_enabled: boolean
  category_precondition: string
  standard: string
  standard_image: string
  score_type: 'score' | 'pass_fail'
  max_score: number
  score_options: ScoreOption[] | null
  score: number
  require_remark: boolean
  require_photo: boolean
  show_remark: boolean
  show_photo: boolean
  comment: string
  ai_generated: boolean
  photos: string[]
  ai_suggestion?: string
  _aiScoring?: boolean
  _aiScore?: number
  _aiComment?: string
}

const route = useRoute()
const router = useRouter()
const tokenPlanStore = useTokenPlanStore()
const userStore = useUserStore()

const loading = ref(false)
const saving = ref(false)
const isEdit = computed(() => !!route.params.id)
const isEditable = computed(
  () => !isEdit.value || form.value.status === 'draft' || !form.value.status,
)
// 屏幕尺寸判断：Tailwind xl = 1280px 及以上视为大屏
const isLargeScreen = ref(typeof window !== 'undefined' ? window.innerWidth >= 1280 : true)

function onWindowResize() {
  isLargeScreen.value = window.innerWidth >= 1280
  onScroll()
}

const form = ref({
  store_id: '',
  template_id: '',
  title: '',
  checked_at: '',
  issues: '',
  suggestion: '',
  ai_generated: false,
  status: 'completed' as 'draft' | 'pending' | 'rectifying' | 'closed',
})

const storeOptions = ref<{ label: string; value: string }[]>([])
const templateOptions = ref<{ label: string; value: string }[]>([])
const scoreRows = ref<ScoreRow[]>([])

const scoreAreaRefs = new Map<string, { flushPending: () => Promise<boolean> }>()
function setScoreAreaRef(row: ScoreRow, el: unknown) {
  if (el) {
    scoreAreaRefs.set(row.item_id, el as { flushPending: () => Promise<boolean> })
  } else {
    scoreAreaRefs.delete(row.item_id)
  }
}

// 分类分组状态
interface CategoryGroup {
  category: string
  precondition_enabled: boolean
  precondition: string
  precondition_checked: boolean
  collapsed: boolean
  rows: ScoreRow[]
  rowIndices: number[] // 在 scoreRows 中的原始索引
}
const categoryGroups = ref<CategoryGroup[]>([])
// 分类展开 keys（与 v-model:open-keys 配合）
const categoryOpenKeys = ref<string[]>([])

// 分类分组构建：根据 scoreRows 的 category 字段聚合
function rebuildCategoryGroups() {
  const groups = new Map<string, CategoryGroup>()
  // 保留上一次的勾选/折叠状态
  const prevMap = new Map<string, CategoryGroup>()
  for (const g of categoryGroups.value) prevMap.set(g.category, g)
  const rows = scoreRows.value
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i]
    const catName = row.category || '未分类'
    let g = groups.get(catName)
    if (!g) {
      const prev = prevMap.get(catName)
      g = {
        category: catName,
        precondition_enabled: !!row.category_precondition_enabled,
        precondition: row.category_precondition || '',
        precondition_checked: prev?.precondition_checked ?? false,
        collapsed: prev?.collapsed ?? false,
        rows: [],
        rowIndices: [],
      }
      groups.set(catName, g)
    }
    // 取第一个开启前置条件的配置（同分类下应一致）
    if (!g.precondition_enabled && row.category_precondition_enabled) {
      g.precondition_enabled = true
      g.precondition = row.category_precondition || ''
    }
    g.rows.push(row)
    g.rowIndices.push(i)
  }
  categoryGroups.value = Array.from(groups.values())
  // 仅在首次构建（无 prev）时设置默认展开；后续重建保留用户操作过的状态
  if (prevMap.size === 0) {
    categoryOpenKeys.value = categoryGroups.value
      .filter((g) => !g.precondition_enabled)
      .map((g) => g.category)
    for (const g of categoryGroups.value) {
      g.collapsed = !categoryOpenKeys.value.includes(g.category)
    }
  } else {
    // 同步 openKeys 与当前 collapsed 状态
    categoryOpenKeys.value = categoryGroups.value.filter((g) => !g.collapsed).map((g) => g.category)
  }
}

// 监听 scoreRows 变化重建分组
watch(
  scoreRows,
  () => {
    rebuildCategoryGroups()
  },
  { deep: true, immediate: true },
)

// 切换分类展开
function toggleCategory(cat: CategoryGroup) {
  cat.collapsed = !cat.collapsed
  if (cat.collapsed) {
    categoryOpenKeys.value = categoryOpenKeys.value.filter((k) => k !== cat.category)
  } else {
    if (!categoryOpenKeys.value.includes(cat.category)) {
      categoryOpenKeys.value.push(cat.category)
    }
  }
}

// 判断分类下检查项是否可编辑：前置条件未开启 或 已勾选
function isCategoryEditable(cat: CategoryGroup): boolean {
  if (!cat.precondition_enabled) return true
  return cat.precondition_checked
}

// 汇总所有检查项的 photos，供 AI 分析使用
const allRowPhotos = computed<PhotoItem[]>(() => {
  const result: PhotoItem[] = []
  for (const row of scoreRows.value) {
    for (const url of row.photos) {
      result.push({
        uid: `${row.item_id}-${url}`,
        name: url.split('/').pop() || url,
        url,
        status: 'done' as const,
      })
    }
  }
  return result
})

// 检查项内是否已填写反馈问题
const hasRowComments = computed(() => scoreRows.value.some((r) => r.comment.trim()))

const selectedPlanId = ref('')
const selectedModelId = ref('')
const aiDrawerVisible = ref(false)

// AI 巡店实时对话状态
const isAnalyzing = ref(false)
const aiStreaming = ref(false)
const aiStreamingText = ref('')
interface AIProblemItem {
  item_id: string
  item_name: string
  level: string
  desc: string
}
interface AISuggestionItem {
  title: string
  desc: string
}
interface AIReportSummary {
  summary: string
  high_risk_problems: AIProblemItem[]
  main_problems: AIProblemItem[]
  priority_suggest: AISuggestionItem[]
  business_suggest: AISuggestionItem[]
}
const aiReport = ref<AIReportSummary | null>(null)
const aiKeywords = ref('')
const aiPhotos = ref<PhotoItem[]>([])

// 本次 AI 巡店的依据描述（照片 / 检查项内反馈问题 / 巡店关键词）
const aiBasisDescription = computed(() => {
  const parts: string[] = []
  const totalPhotos = allRowPhotos.value.length + aiPhotos.value.length
  if (totalPhotos > 0) {
    parts.push(
      allRowPhotos.value.length > 0
        ? `${totalPhotos} 张照片（其中检查项 ${allRowPhotos.value.length} 张）`
        : `${totalPhotos} 张照片`,
    )
  }
  if (hasRowComments.value) parts.push('检查项内反馈问题')
  if (aiKeywords.value.trim()) parts.push('巡店关键词')
  return parts.join('、')
})
const aiLogs = ref<LogEntry[]>([])
const scrolledToBottom = ref(false)
let aiLogSeq = 0

// AI 生成标记追踪：记录 AI 生成的问题/建议原始值，用于识别用户手动修改后取消标记
const aiOriginalIssues = ref('')
const aiOriginalSuggestion = ref('')

// AI 面板图片上传（大屏 / Drawer 各自持有隐藏 input）
const aiFileInput = ref<HTMLInputElement | null>(null)
const aiDrawerFileInput = ref<HTMLInputElement | null>(null)

function setAIFileInput(el: any) {
  aiFileInput.value = el as HTMLInputElement | null
}

function setAIDrawerFileInput(el: any) {
  aiDrawerFileInput.value = el as HTMLInputElement | null
}

function triggerAIUpload(drawer = false) {
  if (aiPhotos.value.length >= 10) {
    Message.warning('最多上传 10 张图片')
    return
  }
  const input = drawer ? aiDrawerFileInput.value : aiFileInput.value
  if (input) {
    input.value = ''
    input.click()
  }
}

function levelColor(level: string): string {
  switch (level) {
    case '高':
      return 'red'
    case '中':
      return 'orange'
    default:
      return 'gray'
  }
}

function formatAIReportText(report: AIReportSummary): string {
  const lines: string[] = []
  if (report.summary) lines.push(`【总结】${report.summary}`)
  if (report.high_risk_problems?.length) {
    lines.push('\n【高危风险问题】')
    report.high_risk_problems.forEach((p) =>
      lines.push(`- ${p.item_name ? `[${p.item_name}] ` : ''}${p.desc}`),
    )
  }
  if (report.main_problems?.length) {
    lines.push('\n【主要问题】')
    report.main_problems.forEach((p) =>
      lines.push(
        `- ${p.item_name ? `[${p.item_name}] ` : ''}${p.level ? `(${p.level}) ` : ''}${p.desc}`,
      ),
    )
  }
  if (report.priority_suggest?.length) {
    lines.push('\n【优先整改建议】')
    report.priority_suggest.forEach((s, i) => lines.push(`${i + 1}. ${s.title}：${s.desc}`))
  }
  if (report.business_suggest?.length) {
    lines.push('\n【运营优化建议】')
    report.business_suggest.forEach((s) => lines.push(`- ${s.title}：${s.desc}`))
  }
  return lines.join('\n')
}

function copyAIReport() {
  if (!aiReport.value) return
  navigator.clipboard.writeText(formatAIReportText(aiReport.value)).then(
    () => Message.success('报告已复制到剪贴板'),
    () => Message.error('复制失败，请手动复制'),
  )
}

async function addAIPhotos(files: File[]) {
  const imageFiles = files.filter((f) => f.type.startsWith('image/'))
  if (imageFiles.length === 0) return
  const remaining = 10 - aiPhotos.value.length
  if (remaining <= 0) {
    Message.warning('最多上传 10 张图片')
    return
  }
  const accept = imageFiles.slice(0, remaining)
  let hasOversize = false
  for (const file of accept) {
    if (file.size > MAX_IMAGE_SIZE) {
      hasOversize = true
      continue
    }
    const fileToUse = file.size > 500 * 1024 ? await compressImage(file) : file
    aiPhotos.value.push({
      uid: `ai-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name: file.name,
      url: URL.createObjectURL(fileToUse),
      status: 'init',
      file: fileToUse,
    })
  }
  if (hasOversize) Message.warning('单张图片大小不能超过 10MB')
}

async function onAIFileInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  await addAIPhotos(Array.from(input.files))
  input.value = ''
}

function removeAIPhoto(index: number) {
  aiPhotos.value.splice(index, 1)
}

const isAIDragging = ref(false)
let aiDragDepth = 0

function handleAIPaste(e: ClipboardEvent) {
  const files = Array.from(e.clipboardData?.files || []).filter((f) => f.type.startsWith('image/'))
  if (files.length > 0) {
    e.preventDefault()
    addAIPhotos(files)
    return
  }
  nextTick(() => {
    extractImageLinksFromAIKeywords()
    normalizeAIKeywordsNewlines()
  })
}

function handleAIInput(val: string | any) {
  if (typeof val !== 'string') return
  nextTick(() => {
    normalizeAIKeywordsNewlines()
    extractImageLinksFromAIKeywords()
  })
}

function handleAIKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.ctrlKey && !e.metaKey && !e.shiftKey && !e.altKey) {
    const target = e.target as HTMLTextAreaElement
    const before = target.value.slice(0, target.selectionStart)
    if (before.endsWith('\n\n')) {
      e.preventDefault()
    }
  }
}

function extractImageLinksFromAIKeywords() {
  const text = aiKeywords.value
  const urls = extractImageUrlsFromText(text)
  if (urls.length === 0) return
  const existing = new Set(aiPhotos.value.map((p) => p.url))
  const remaining = 10 - aiPhotos.value.length
  let addedCount = 0
  for (const url of urls) {
    if (existing.has(url)) continue
    if (addedCount >= remaining) break
    const path = url.split(/[?#]/)[0]
    let name = path.split('/').pop() || url
    try {
      name = decodeURIComponent(name)
    } catch {
      /* keep original */
    }
    aiPhotos.value.push({
      uid: `ai-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name,
      url,
      status: 'done',
    })
    existing.add(url)
    addedCount++
  }
  const cleaned = cleanUrlsFromText(text)
  if (cleaned !== text) aiKeywords.value = cleaned
}

function normalizeAIKeywordsNewlines() {
  const text = aiKeywords.value
  const normalized = text.replace(/\n{3,}/g, '\n\n')
  if (normalized !== text) {
    aiKeywords.value = normalized
  }
}

function handleAIDragEnter(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files')) return
  aiDragDepth++
  isAIDragging.value = true
}
function handleAIDragLeave() {
  aiDragDepth--
  if (aiDragDepth <= 0) {
    aiDragDepth = 0
    isAIDragging.value = false
  }
}
function handleAIDragOver(e: DragEvent) {
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
}
function handleAIDrop(e: DragEvent) {
  aiDragDepth = 0
  isAIDragging.value = false
  if (!e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
  addAIPhotos(Array.from(e.dataTransfer.files))
}

function pushAILog(level: LogLevel, message: string, detail?: string) {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  aiLogs.value.push({
    id: String(++aiLogSeq),
    time: `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${String(
      now.getMilliseconds(),
    ).padStart(3, '0')}`,
    level,
    message,
    detail,
  })
  if (aiLogs.value.length > 200) {
    aiLogs.value.splice(0, aiLogs.value.length - 200)
  }
}

function clearAILogs() {
  aiLogs.value = []
  logExpanded.clear()
}

function toggleLogDetail(id: string) {
  if (logExpanded.has(id)) {
    logExpanded.delete(id)
  } else {
    logExpanded.add(id)
  }
}

// 日志终端滚动容器（大屏面板 + Drawer）
const aiInspectPanelRef = ref<InstanceType<typeof ReuseAiInspectPanel>>()
const logBodyEl = computed<HTMLElement | undefined>(() => {
  const el = unrefElement(aiInspectPanelRef)
  return el?.querySelector('.ai-inspect-panel__body') ?? undefined
})
const logBodyMainEl = computed<HTMLElement | undefined>(() => {
  const el = unrefElement(aiInspectPanelRef)
  return el?.querySelector('.ai-inspect-panel__body-main') ?? undefined
})
const { y: logBodyY } = useScroll(logBodyEl, {
  behavior: 'smooth',
})

useResizeObserver(logBodyMainEl, () => {
  const el = unref(logBodyEl)
  // 日志实时滚动到底部，确保光标始终处于可视范围
  // 仅在用户已经处于底部时才自动滚动，避免打断用户查看历史日志
  if (el) {
    logBodyY.value = el.scrollHeight - el.clientHeight
  }
})

const aiDrawerInspectPanelRef = ref<InstanceType<typeof ReuseAiInspectPanel>>()
const logDrawerBodyEl = computed<HTMLElement | undefined>(() => {
  const el = unrefElement(aiDrawerInspectPanelRef)
  return el?.querySelector('.ai-inspect-panel__body') ?? undefined
})
const logDrawerBodyMainEl = computed<HTMLElement | undefined>(() => {
  const el = unrefElement(aiDrawerInspectPanelRef)
  return el?.querySelector('.ai-inspect-panel__body-main') ?? undefined
})
const { y: logDrawerBodyY } = useScroll(logDrawerBodyEl, {
  behavior: 'smooth',
})

useResizeObserver(logDrawerBodyMainEl, () => {
  const el = unref(logDrawerBodyEl)
  // 日志实时滚动到底部，确保光标始终处于可视范围
  // 仅在用户已经处于底部时才自动滚动，避免打断用户查看历史日志
  if (el) {
    logDrawerBodyY.value = el.scrollHeight - el.clientHeight
  }
})

function levelText(level: LogLevel): string {
  const map: Record<LogLevel, string> = {
    info: 'INFO',
    req: 'REQ',
    ok: 'OK',
    err: 'ERR',
    warn: 'WARN',
  }
  return map[level] || level.toUpperCase()
}
const selectedHasVision = ref(true)

const activeModelKey = computed(() => {
  if (!selectedPlanId.value) return ''
  return selectedModelId.value ? `${selectedPlanId.value}:${selectedModelId.value}` : ''
})

const selectedModelSupportsVision = computed(() => {
  if (!selectedPlanId.value || !selectedModelId.value) return false
  const plan = tokenPlanStore.enabledPlans.find((p) => p.id === selectedPlanId.value)
  if (!plan) return false
  const models = parseModelField(plan.model)
  const model = models.find((m) => m.id === selectedModelId.value)
  return model ? model.types.includes('vision') : false
})

function onModelChange(val: ModelSelectValue) {
  selectedPlanId.value = val.planId
  selectedModelId.value = val.modelId
  selectedHasVision.value = val.hasVision
}

const inspectorName = computed(
  () => userStore.userInfo?.nickname || userStore.userInfo?.username || '',
)

const scoringRows = computed(() => scoreRows.value.filter((r) => r.score_type !== 'pass_fail'))
const maxTotal = computed(() => scoringRows.value.reduce((sum, r) => sum + r.max_score, 0))
const totalScore = computed(() =>
  scoringRows.value.reduce((sum, r) => sum + (Number(r.score) || 0), 0),
)
const totalPassed = computed(() =>
  maxTotal.value > 0 ? totalScore.value >= maxTotal.value * 0.8 : false,
)

function isRowPassed(row: ScoreRow): boolean {
  if (row.score_type === 'pass_fail') {
    const opts = row.score_options || []
    if (opts.length === 0) return false
    return Number(row.score) === opts[0].score
  }
  return row.max_score > 0 ? Number(row.score) >= row.max_score : Number(row.score) > 0
}

function scoreLabel(row: ScoreRow): string {
  if (row.score_type === 'pass_fail') {
    const opt = (row.score_options || []).find((o) => o.score === Number(row.score))
    if (opt) return opt.label
    const opts = row.score_options || []
    return opts.length > 0 ? opts[0].label : '—'
  }
  return String(row.score)
}

function onScoreOptionChange(row: ScoreRow, val: any) {
  row.score = Number(val)
  row.ai_generated = false
}

// 生成评分下拉选项：pass_fail 用 score_options；分值评分生成 0~max 的整数选项
function scoreSelectOptions(row: ScoreRow): { score: number; label: string }[] {
  if (row.score_options && row.score_options.length > 0) {
    return row.score_options.map((o) => ({ score: o.score, label: o.label }))
  }
  if (row.score_type === 'pass_fail') {
    return cloneOptions(DEFAULT_PASS_FAIL_OPTIONS)
  }
  const max = row.max_score || 5
  const opts: { score: number; label: string }[] = []
  for (let i = 0; i <= max; i++) {
    opts.push({ score: i, label: `${i} 分` })
  }
  return opts
}

// 备注输入框 placeholder：根据是否满分 + require_remark 组合文案
function commentPlaceholder(row: ScoreRow): string {
  const passed = isRowPassed(row)
  if (passed) {
    // 满分：根据配置显示必填或选填
    return row.require_remark
      ? '（必填）请输入反馈问题，300 字内'
      : '（选填）请输入反馈问题，300 字内'
  }
  // 非满分：必填
  return '（必填）请输入反馈问题，300 字内'
}

async function fetchStores() {
  try {
    const res = await api.get<Store[]>('/stores/all')
    storeOptions.value = (res.data || []).map((s) => ({ label: s.name, value: s.id }))
  } catch {
    storeOptions.value = []
  }
}

async function fetchTemplates() {
  try {
    const res = await api.get<InspectionTemplate[]>('/inspection-templates/all')
    templateOptions.value = (res.data || []).map((t) => ({ label: t.name, value: t.id }))
  } catch {
    templateOptions.value = []
  }
}

async function fetchItems() {
  try {
    const res = await api.get<InspectionItem[]>('/inspections/items')
    scoreRows.value = (res.data || []).map((item) => ({
      item_id: item.id,
      item_name: item.name,
      category: item.category || '',
      category_precondition_enabled: false,
      category_precondition: '',
      standard: '',
      standard_image: '',
      score_type: 'score' as const,
      max_score: item.max_score,
      score_options: cloneOptions(DEFAULT_SCORE_OPTIONS),
      score: item.max_score, // 默认满分
      require_remark: false,
      require_photo: false,
      show_remark: true,
      show_photo: true,
      comment: '',
      ai_generated: false,
      ai_suggestion: '',
      photos: [],
    }))
  } catch {
    scoreRows.value = []
  }
}

function rowFromTemplateItem(item: InspectionTemplateItem): ScoreRow {
  const scoreType = item.score_type === 'pass_fail' ? 'pass_fail' : 'score'
  const options =
    item.score_options && item.score_options.length > 0
      ? item.score_options.map((o) => ({ score: o.score, label: o.label }))
      : cloneOptions(scoreType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS)
  return {
    item_id: item.id,
    item_name: item.title,
    category: item.category || '',
    category_precondition_enabled: item.category_precondition_enabled ?? false,
    category_precondition: item.category_precondition || '',
    standard: item.standard || '',
    standard_image: item.standard_image || '',
    score_type: scoreType,
    max_score: item.max_score || 5,
    score_options: options,
    score: scoreType === 'pass_fail' && options.length > 0 ? options[0].score : item.max_score || 5,
    require_remark: item.require_remark,
    require_photo: item.require_photo,
    show_remark: item.show_remark ?? true,
    show_photo: item.show_photo ?? true,
    comment: '',
    ai_generated: false,
    ai_suggestion: '',
    photos: [],
  }
}

async function loadTemplateItems(templateId: string) {
  if (!templateId) {
    await fetchItems()
    return
  }
  try {
    const res = await api.get<InspectionTemplate>(`/inspection-templates/${templateId}`)
    scoreRows.value = (res.data.items || []).map(rowFromTemplateItem)
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载模板检查项失败')
    scoreRows.value = []
  }
}

async function onTemplateChange(templateId: any) {
  await loadTemplateItems(templateId || '')
}

async function fetchDetail() {
  try {
    const res = await api.get<Inspection>(`/inspections/${route.params.id}`)
    const data = res.data
    form.value = {
      store_id: data.store_id,
      template_id: data.template_id || '',
      title: data.title,
      checked_at: data.checked_at || '',
      issues: data.issues || '',
      suggestion: data.suggestion || '',
      ai_generated: !!data.ai_generated,
      status: data.status,
    }
    aiOriginalIssues.value = data.issues || ''
    aiOriginalSuggestion.value = data.suggestion || ''
    if (data.ai_summary) {
      aiReport.value = {
        summary: data.ai_summary.summary || '',
        high_risk_problems: data.ai_summary.high_risk_problems || [],
        main_problems: data.ai_summary.main_problems || [],
        priority_suggest: data.ai_summary.priority_suggest || [],
        business_suggest: data.ai_summary.business_suggest || [],
      }
    }
    if (data.template_id && !templateOptions.value.some((o) => o.value === data.template_id)) {
      templateOptions.value.push({
        label: data.template_name || '当前模板',
        value: data.template_id,
      })
    }
    if (data.scores && data.scores.length > 0) {
      scoreRows.value = data.scores.map((s) => {
        const scoreType = s.score_type === 'pass_fail' ? 'pass_fail' : 'score'
        return {
          item_id: s.item_id,
          item_name: s.item_name,
          category: s.category || '',
          category_precondition_enabled: false,
          category_precondition: '',
          standard: s.standard || '',
          standard_image: s.standard_image || '',
          score_type: scoreType,
          max_score: s.max_score,
          score_options:
            s.score_options && s.score_options.length > 0
              ? s.score_options.map((o) => ({ score: o.score, label: o.label }))
              : cloneOptions(
                  scoreType === 'pass_fail' ? DEFAULT_PASS_FAIL_OPTIONS : DEFAULT_SCORE_OPTIONS,
                ),
          score: s.score,
          require_remark: s.require_remark,
          require_photo: s.require_photo,
          show_remark: s.show_remark ?? true,
          show_photo: s.show_photo ?? true,
          comment: s.comment || '',
          ai_generated: !!s.ai_generated,
          ai_suggestion: s.ai_suggestion || '',
          photos: s.photos || [],
        }
      })
      // 如果有 template_id，尝试从模板拉取分类前置条件覆盖默认值
      if (data.template_id) {
        try {
          const tplRes = await api.get<InspectionTemplate>(
            `/inspection-templates/${data.template_id}`,
          )
          const tplItems = tplRes.data.items || []
          const tplMap = new Map(tplItems.map((t) => [t.id, t]))
          for (const row of scoreRows.value) {
            const tpl = tplMap.get(row.item_id)
            if (tpl) {
              row.category = tpl.category || row.category
              row.category_precondition_enabled = tpl.category_precondition_enabled ?? false
              row.category_precondition = tpl.category_precondition || ''
            }
          }
        } catch {
          // 忽略模板加载失败，继续使用默认值
        }
      }
    }
    if (data.store_id && !storeOptions.value.some((o) => o.value === data.store_id)) {
      storeOptions.value.push({ label: data.store_name || '未知门店', value: data.store_id })
    }
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '加载巡店记录失败')
  }
}

function onStoreChange(storeId: any) {
  const store = storeOptions.value.find((o) => o.value === storeId)
  if (store && !form.value.title.trim()) {
    form.value.title = `${store.label}巡店检查`
  }
}

const MAX_IMAGE_SIZE = 10 * 1024 * 1024 // 10MB
const MAX_ROW_PHOTOS = 10

// ===== 反馈问题 + 巡店图片一体化输入区（AttachmentInputArea 公共组件） =====
// 用户编辑文本后标记为非 AI 生成内容（图片提取 / 换行归一化由组件内置处理）
function onRowCommentInput(row: ScoreRow, _val: string) {
  row.ai_generated = false
}

async function compressImage(file: File, maxSize = 1600, quality = 0.8): Promise<File> {
  if (!file.type.startsWith('image/')) return file
  return new Promise((resolve) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      let { width, height } = img
      if (width <= maxSize && height <= maxSize) {
        resolve(file)
        return
      }
      if (width > height) {
        height = Math.round((height * maxSize) / width)
        width = maxSize
      } else {
        width = Math.round((width * maxSize) / height)
        height = maxSize
      }
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        resolve(file)
        return
      }
      ctx.drawImage(img, 0, 0, width, height)
      canvas.toBlob(
        (blob) => {
          if (!blob) {
            resolve(file)
            return
          }
          const compressed = new File([blob], file.name, { type: 'image/jpeg' })
          resolve(compressed)
        },
        'image/jpeg',
        quality,
      )
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(file)
    }
    img.src = url
  })
}

async function fileToBase64(file: File): Promise<{ data: string; mime_type: string }> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve({ data: result.split(',')[1] || '', mime_type: file.type || 'image/jpeg' })
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

function applyAIResult(data: {
  scores?: InspectionScore[]
  issues?: string
  suggestion?: string
  summary?: string
  high_risk_problems?: AIProblemItem[]
  main_problems?: AIProblemItem[]
  priority_suggest?: AISuggestionItem[]
  business_suggest?: AISuggestionItem[]
}) {
  if (data.scores && data.scores.length > 0) {
    const aiMap = new Map(data.scores.map((s: InspectionScore) => [s.item_id, s]))
    for (const row of scoreRows.value) {
      const aiRow = aiMap.get(row.item_id)
      if (aiRow) {
        row.score = aiRow.score
        row.comment = aiRow.comment || ''
        row._aiScore = aiRow.score
        row._aiComment = aiRow.comment || ''
        row.ai_generated = true
      }
    }
  }
  if (data.issues) form.value.issues = data.issues
  if (data.suggestion) form.value.suggestion = data.suggestion
  const hasSummary =
    !!data.summary?.trim() ||
    (data.high_risk_problems?.length ?? 0) > 0 ||
    (data.main_problems?.length ?? 0) > 0 ||
    (data.priority_suggest?.length ?? 0) > 0 ||
    (data.business_suggest?.length ?? 0) > 0
  if (hasSummary) {
    aiReport.value = {
      summary: data.summary || '',
      high_risk_problems: data.high_risk_problems || [],
      main_problems: data.main_problems || [],
      priority_suggest: data.priority_suggest || [],
      business_suggest: data.business_suggest || [],
    }
  }
  aiOriginalIssues.value = form.value.issues
  aiOriginalSuggestion.value = form.value.suggestion
  form.value.ai_generated = true
}

async function handleAnalyze() {
  if (!form.value.store_id) {
    Message.warning('请先选择巡店门店')
    return
  }
  if (!form.value.template_id) {
    Message.warning('请先选择检查表模板')
    return
  }
  const keywords = aiKeywords.value.trim()
  if (
    allRowPhotos.value.length === 0 &&
    !hasRowComments.value &&
    aiPhotos.value.length === 0 &&
    !keywords
  ) {
    Message.warning('请至少提供一种巡店依据：检查项内图片、检查项内反馈问题、AI 巡店图片或关键词')
    return
  }
  if (!selectedPlanId.value || !selectedModelId.value) {
    Message.warning('请选择模型服务商和模型')
    return
  }
  const hasPhotos = allRowPhotos.value.length > 0 || aiPhotos.value.length > 0
  if (hasPhotos && !selectedModelSupportsVision.value) {
    Message.warning('当前模型不支持视觉理解，无法分析图片。请在模型配置中选用支持视觉理解的模型')
    return
  }
  const photos: { data: string; mime_type: string; url?: string }[] = []
  for (const item of [...allRowPhotos.value, ...aiPhotos.value]) {
    if (item.file) {
      const encoded = await fileToBase64(item.file)
      photos.push(encoded)
    } else if (item.url) {
      photos.push({ data: '', mime_type: 'image/jpeg', url: item.url })
    }
  }

  // 重新 AI 巡店前：先移除旧的 AI 生成标记，等待新的分析结果重新标记
  removeAIResult()
  clearAILogs()
  isAnalyzing.value = true
  aiStreaming.value = true
  aiStreamingText.value = ''

  // 将当前检查项组装为 skills 技能规范，向 AI 明确巡店标准与评分规则
  const skills = scoreRows.value.map((row) => ({
    item_id: row.item_id,
    name: row.item_name,
    category: row.category,
    standard: row.standard,
    standard_image: row.standard_image,
    score_type: row.score_type,
    max_score: row.max_score,
    score_options: (row.score_options || []).map((o) => ({ score: o.score, label: o.label })),
    require_remark: row.require_remark,
    require_photo: row.require_photo,
  }))
  // 规范 AI 返回格式（JSON schema）
  const responseSchema = {
    type: 'object',
    properties: {
      scores: {
        type: 'array',
        description: '仅包含与巡店关键词/图片有关联的检查项评分，无关项不返回',
        items: {
          type: 'object',
          properties: {
            item_id: { type: 'string', description: '检查项 ID' },
            score: { type: 'number', description: '得分，必须在允许的分值范围内' },
            comment: { type: 'string', description: '该检查项的反馈问题说明' },
          },
          required: ['item_id', 'score', 'comment'],
        },
      },
      issues: {
        type: 'string',
        description: '问题与备注：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项',
      },
      suggestion: {
        type: 'string',
        description: 'AI 整改建议：针对每个问题给出具体、可执行的整改措施与提升建议',
      },
      summary: {
        type: 'string',
        description: '整体一句话概括本次巡店情况，30-60字',
      },
      high_risk_problems: {
        type: 'array',
        description: '高危风险问题列表（如消防、食安等需立即处理的问题）',
        items: {
          type: 'object',
          properties: {
            item_id: {
              type: 'string',
              description: '关联的检查项 ID，优先填写；无法关联时可为空字符串',
            },
            item_name: { type: 'string', description: '关联的检查项名称，无法关联时可为空字符串' },
            level: { type: 'string', description: '严重程度：高/中/低' },
            desc: { type: 'string', description: '问题描述，说明具体现象与位置' },
          },
          required: ['item_id', 'item_name', 'level', 'desc'],
        },
      },
      main_problems: {
        type: 'array',
        description: '主要问题汇总列表（除高危外需要整改的问题）',
        items: {
          type: 'object',
          properties: {
            item_id: {
              type: 'string',
              description: '关联的检查项 ID，优先填写；无法关联时可为空字符串',
            },
            item_name: { type: 'string', description: '关联的检查项名称，无法关联时可为空字符串' },
            level: { type: 'string', description: '严重程度：高/中/低' },
            desc: { type: 'string', description: '问题描述，说明具体现象与位置' },
          },
          required: ['item_id', 'item_name', 'level', 'desc'],
        },
      },
      priority_suggest: {
        type: 'array',
        description: '优先整改建议列表（3条，针对最严重问题）',
        items: {
          type: 'object',
          properties: {
            title: { type: 'string', description: '建议标题，简短明确' },
            desc: { type: 'string', description: '建议说明，具体可执行' },
          },
          required: ['title', 'desc'],
        },
      },
      business_suggest: {
        type: 'array',
        description: '门店运营优化建议列表',
        items: {
          type: 'object',
          properties: {
            title: { type: 'string', description: '建议标题，简短明确' },
            desc: { type: 'string', description: '建议说明，具体可执行' },
          },
          required: ['title', 'desc'],
        },
      },
    },
    required: [
      'scores',
      'issues',
      'suggestion',
      'summary',
      'high_risk_problems',
      'main_problems',
      'priority_suggest',
      'business_suggest',
    ],
  }

  const startedAt = performance.now()
  let analysisFailed = false
  pushAILog('info', `开始 AI 巡店分析 · ${keywords ? '关键词 + ' : ''}${photos.length} 张照片`)
  pushAILog('info', `使用模型配置（${selectedModelId.value}）`)

  try {
    const response = await fetch('/api/inspections/ai-analyze-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({
        store_id: form.value.store_id,
        template_id: form.value.template_id,
        plan_id: selectedPlanId.value,
        model_id: selectedModelId.value,
        photos,
        keywords,
        skills,
        response_schema: responseSchema,
      }),
    })

    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        router.push('/login')
        return
      }
      const errorData = await response.json().catch(() => ({}))
      const detail = errorData.detail
      let errorMsg = 'AI 巡店分析失败，请重试'
      if (typeof detail === 'object' && detail?.message) {
        errorMsg = detail.message
        if (Array.isArray(detail.available_plans) && detail.available_plans.length > 0) {
          const planNames = detail.available_plans
            .map(
              (p: { display_name?: string; name: string; provider: string; model: string }) =>
                `${p.display_name || p.name}（${p.provider}/${p.model}）`,
            )
            .join('、')
          Message.warning(`可前往设置切换至：${planNames}`)
        }
      } else if (typeof detail === 'string') {
        errorMsg = detail
      }
      pushAILog('err', `HTTP ${response.status} · ${errorMsg}`)
      Message.error(errorMsg)
      return
    }

    pushAILog('req', 'POST /api/inspections/ai-analyze-stream → SSE 连接已建立')

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    let resultData: {
      scores: InspectionScore[]
      issues?: string
      suggestion?: string
      summary?: string
      high_risk_problems?: AIProblemItem[]
      main_problems?: AIProblemItem[]
      priority_suggest?: AISuggestionItem[]
      business_suggest?: AISuggestionItem[]
    } | null = null

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const events = buffer.split('\n\n')
      buffer = events.pop() || ''

      for (const eventStr of events) {
        if (!eventStr.trim()) continue
        const lines = eventStr.split('\n')
        let eventType = ''
        let dataStr = ''
        for (const line of lines) {
          if (line.startsWith('event:')) eventType = line.slice(6).trim()
          else if (line.startsWith('data:')) dataStr = line.slice(5).trim()
        }
        if (!eventType || !dataStr) continue

        const payload = JSON.parse(dataStr)

        switch (eventType) {
          case 'log':
            pushAILog(payload.level || 'info', payload.message || '')
            break
          case 'request_data':
            pushAILog('req', '向模型发送请求', payload.detail || JSON.stringify(payload))
            break
          case 'response_data':
            pushAILog('ok', '收到模型响应', payload.detail || JSON.stringify(payload))
            break
          case 'chunk':
            aiStreamingText.value += payload.text || ''
            break
          case 'result':
            resultData = {
              scores: Array.isArray(payload.scores) ? payload.scores : [],
              issues: payload.issues || '',
              suggestion: payload.suggestion || '',
              summary: payload.summary || '',
              high_risk_problems: Array.isArray(payload.high_risk_problems)
                ? payload.high_risk_problems
                : [],
              main_problems: Array.isArray(payload.main_problems) ? payload.main_problems : [],
              priority_suggest: Array.isArray(payload.priority_suggest)
                ? payload.priority_suggest
                : [],
              business_suggest: Array.isArray(payload.business_suggest)
                ? payload.business_suggest
                : [],
            }
            pushAILog('ok', '已获取 AI 巡店分析结果')
            break
          case 'error':
            analysisFailed = true
            pushAILog('err', payload.message || 'AI 巡店分析失败')
            if (payload.available_plans?.length > 0) {
              const planNames = payload.available_plans
                .map(
                  (p: { display_name?: string; name: string; provider: string; model: string }) =>
                    `${p.display_name || p.name}（${p.provider}/${p.model}）`,
                )
                .join('、')
              Message.warning(`可前往设置切换至：${planNames}`)
            }
            Message.error(payload.message || 'AI 巡店分析失败')
            break
          case 'complete':
            pushAILog('ok', 'AI 巡店分析结束')
            break
        }
      }
    }

    if (resultData) {
      applyAIResult(resultData)
      const total = Math.round(performance.now() - startedAt)
      pushAILog('ok', `已生成问题与建议 ，相关检查项已评分 · 总耗时 ${(total / 1000).toFixed(1)}s`)
      Message.success('AI 巡店分析完成，已生成问题与整改建议，相关检查项已评分')
    } else {
      analysisFailed = true
      pushAILog('err', '未获取到有效分析结果')
      Message.error('未获取到 AI 巡店分析结果，请重试')
    }
  } catch (error: unknown) {
    analysisFailed = true
    const err = error as { message?: string }
    pushAILog('err', err.message || '网络异常')
    Message.error(err.message || 'AI 巡店分析失败，请重试')
  } finally {
    isAnalyzing.value = false
    aiStreaming.value = false
    if (!analysisFailed) aiStreamingText.value = ''
  }
}

// 重新 AI 巡店前：移除旧的 AI 生成标记（评分、问题、建议），等待新的分析结果重新标记
function removeAIResult() {
  for (const row of scoreRows.value) {
    row.ai_generated = false
  }
  form.value.ai_generated = false
  aiOriginalIssues.value = ''
  aiOriginalSuggestion.value = ''
}

// 计算记录级 AI 生成标记：任一检查项由 AI 评分，或问题/建议仍是 AI 生成的原始内容
function computeRecordAIGenerated(): boolean {
  const hasAIScore = scoreRows.value.some((r) => r.ai_generated)
  const issuesStillAI = !!aiOriginalIssues.value && form.value.issues === aiOriginalIssues.value
  const suggestionStillAI =
    !!aiOriginalSuggestion.value && form.value.suggestion === aiOriginalSuggestion.value
  return hasAIScore || issuesStillAI || suggestionStillAI
}

async function handleSingleItemAI(row: ScoreRow) {
  if (!selectedPlanId.value || !selectedModelId.value) {
    Message.warning('请先在 AI 设置中选择模型配置')
    return
  }
  const hasImageInput = row.standard_image || row.photos.length > 0
  if (hasImageInput && !selectedModelSupportsVision.value) {
    Message.warning('当前模型不支持视觉理解，无法分析图片。请在模型配置中选用支持视觉理解的模型')
    return
  }
  row._aiScoring = true
  row.ai_suggestion = ''

  const scoreIsFromAI = row._aiScore !== undefined && row.score === row._aiScore
  const commentIsFromAI = row._aiComment !== undefined && row.comment === row._aiComment

  const requestBody = {
    item_name: row.item_name,
    item_id: row.item_id,
    standard: row.standard || '',
    standard_image: row.standard_image || '',
    score_type: row.score_type,
    max_score: row.max_score,
    score_options: row.score_options || [],
    comment: commentIsFromAI ? '' : row.comment || '',
    current_score: scoreIsFromAI ? 0 : row.score || 0,
    keywords: aiKeywords.value.trim(),
    plan_id: selectedPlanId.value,
    model_id: selectedModelId.value,
  }

  pushAILog('info', `[${row.item_name}] 开始 AI 生成…`)

  try {
    const photos: { data: string; mime_type: string; url?: string }[] = []
    for (const url of row.photos) {
      photos.push({ data: '', mime_type: 'image/jpeg', url })
    }
    const bodyWithPhotos = {
      ...requestBody,
      photos,
    }

    pushAILog(
      'req',
      `[${row.item_name}] POST /api/inspections/ai-analyze-item-stream`,
      JSON.stringify(bodyWithPhotos, null, 2),
    )

    const response = await fetch('/api/inspections/ai-analyze-item-stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(bodyWithPhotos),
    })

    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        router.push('/login')
        return
      }
      const errorData = await response.json().catch(() => ({}))
      const detail = errorData.detail
      let errorMsg = 'AI 生成失败'
      if (typeof detail === 'string') errorMsg = detail
      pushAILog('err', `[${row.item_name}] HTTP ${response.status} · ${errorMsg}`)
      Message.error(errorMsg)
      return
    }

    pushAILog('req', `[${row.item_name}] SSE 连接已建立`)

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const events = buffer.split('\n\n')
      buffer = events.pop() || ''

      for (const eventStr of events) {
        if (!eventStr.trim()) continue
        const lines = eventStr.split('\n')
        let eventType = ''
        let dataStr = ''
        for (const line of lines) {
          if (line.startsWith('event:')) eventType = line.slice(6).trim()
          else if (line.startsWith('data:')) dataStr = line.slice(5).trim()
        }
        if (!eventType || !dataStr) continue

        const payload = JSON.parse(dataStr)

        switch (eventType) {
          case 'log':
            pushAILog(payload.level || 'info', `[${row.item_name}] ${payload.message || ''}`)
            break
          case 'request_data':
            pushAILog(
              'req',
              `[${row.item_name}] 向模型发送请求`,
              payload.detail || JSON.stringify(payload),
            )
            break
          case 'response_data':
            pushAILog(
              'ok',
              `[${row.item_name}] 收到模型响应`,
              payload.detail || JSON.stringify(payload),
            )
            break
          case 'chunk':
            aiStreamingText.value += payload.text || ''
            break
          case 'result':
            if (payload.score !== undefined && payload.score !== null) {
              row.score = payload.score
              row._aiScore = payload.score
            }
            if (payload.comment) {
              row.comment = payload.comment
              row._aiComment = payload.comment
            }
            if (payload.suggestion) {
              row.ai_suggestion = payload.suggestion
            }
            row.ai_generated = true
            pushAILog(
              'ok',
              `[${row.item_name}] AI 生成完成 · score=${payload.score}`,
              JSON.stringify(payload, null, 2),
            )
            if (payload.photo_relevance === false) {
              pushAILog(
                'warn',
                `[${row.item_name}] 上传的图片与当前检查项不匹配，建议更换图片后重新分析`,
              )
            }
            Message.success('AI 生成完成')
            break
          case 'error':
            pushAILog('err', `[${row.item_name}] ${payload.message || 'AI 生成失败'}`)
            if (payload.available_plans?.length > 0) {
              const planNames = payload.available_plans
                .map(
                  (p: { display_name?: string; name: string; provider: string; model: string }) =>
                    `${p.display_name || p.name}（${p.provider}/${p.model}）`,
                )
                .join('、')
              Message.warning(`可前往设置切换至：${planNames}`)
            }
            Message.error(payload.message || 'AI 生成失败')
            break
          case 'complete':
            pushAILog('ok', `[${row.item_name}] SSE 流结束`)
            break
        }
      }
    }
  } catch (err: any) {
    pushAILog('err', `[${row.item_name}] 请求异常 · ${err?.message || '未知错误'}`)
    Message.error(err?.message || 'AI 生成请求失败')
  } finally {
    row._aiScoring = false
  }
}

function validateRows(): { msg: string; rowIndex: number } | null {
  // 先检查是否存在 开启了前置条件但未勾选 且分类下有已填写内容的情况
  const unsatisfiedWithFilled: string[] = []
  let unsatisfiedFirstIdx = -1
  for (const g of categoryGroups.value) {
    if (g.precondition_enabled && !g.precondition_checked) {
      const hasContent = g.rows.some((r) => {
        const hasScore =
          r.score_type !== 'pass_fail'
            ? Number(r.score) !== r.max_score
            : r.score_options && r.score_options.length > 0
              ? Number(r.score) !== r.score_options[0].score
              : false
        return hasScore || r.comment.trim() || r.photos.length > 0
      })
      if (hasContent) {
        unsatisfiedWithFilled.push(g.category)
        if (unsatisfiedFirstIdx < 0) unsatisfiedFirstIdx = g.rowIndices[0]
      }
    }
  }
  if (unsatisfiedWithFilled.length > 0) {
    return {
      msg: `分类「${unsatisfiedWithFilled.join('、')}」前置条件未勾选但已有评分内容，请先勾选前置条件后再保存`,
      rowIndex: unsatisfiedFirstIdx,
    }
  }
  // 常规检查项校验
  for (const g of categoryGroups.value) {
    const skipped = g.precondition_enabled && !g.precondition_checked
    for (let i = 0; i < g.rows.length; i++) {
      const row = g.rows[i]
      if (skipped) continue
      const rowIdx = g.rowIndices[i]
      const full = isRowPassed(row)
      if (full) {
        // 满分：反馈问题根据 require_remark 配置决定必填；图片根据 require_photo 配置决定必填
        if (row.show_remark && row.require_remark && !row.comment.trim()) {
          return { msg: `「${row.item_name}」满分必填反馈问题`, rowIndex: rowIdx }
        }
        if (row.show_photo && row.require_photo && row.photos.length === 0) {
          return { msg: `「${row.item_name}」满分必填巡店图片`, rowIndex: rowIdx }
        }
      } else {
        // 非满分：反馈问题和图片都必填
        if (row.show_remark && !row.comment.trim()) {
          return { msg: `「${row.item_name}」非满分必填反馈问题`, rowIndex: rowIdx }
        }
        if (row.show_photo && row.photos.length === 0) {
          return { msg: `「${row.item_name}」非满分必填巡店图片`, rowIndex: rowIdx }
        }
      }
      // 图片数量上限校验
      if (row.show_photo && row.photos.length > MAX_ROW_PHOTOS) {
        return { msg: `「${row.item_name}」巡店图片最多 ${MAX_ROW_PHOTOS} 张`, rowIndex: rowIdx }
      }
    }
  }
  return null
}

async function doSave() {
  for (const areaRef of scoreAreaRefs.values()) {
    if (!(await areaRef.flushPending())) return
  }
  saving.value = true
  try {
    const scores = []
    const allPhotoUrls: string[] = []
    for (const r of scoreRows.value) {
      allPhotoUrls.push(...r.photos)
      scores.push({
        item_id: r.item_id,
        score: Number(r.score) || 0,
        comment: r.comment,
        ai_generated: r.ai_generated,
        ai_suggestion: r.ai_suggestion || '',
        photos: r.photos,
      })
    }
    const payload = {
      store_id: form.value.store_id,
      template_id: form.value.template_id,
      title: form.value.title,
      status: form.value.status,
      checked_at: form.value.checked_at,
      issues: form.value.issues,
      suggestion: form.value.suggestion,
      ai_summary: aiReport.value || {
        summary: '',
        high_risk_problems: [],
        main_problems: [],
        priority_suggest: [],
        business_suggest: [],
      },
      ai_generated: computeRecordAIGenerated(),
      photos: allPhotoUrls,
      scores,
    }
    if (isEdit.value) {
      await api.put(`/inspections/${route.params.id}`, payload)
      Message.success('保存成功')
    } else {
      await api.post('/inspections/', payload)
      Message.success('巡店记录已保存')
    }
    router.push({ name: 'InspectionList' })
  } catch (e: any) {
    Message.error(e?.response?.data?.detail || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleSave() {
  if (isEdit.value && !isEditable.value) {
    Message.warning('当前巡店已进入整改流程，不可编辑')
    return
  }
  if (!form.value.store_id) {
    Message.warning('请选择巡店门店')
    return
  }
  const validationError = validateRows()
  if (validationError) {
    // 展开出错行所在的分类（如果是折叠状态）
    const catGroup = categoryGroups.value.find((g) =>
      g.rowIndices.includes(validationError.rowIndex),
    )
    if (catGroup && catGroup.collapsed) {
      catGroup.collapsed = false
    }
    // 等待 DOM 更新后先滚动到出错行，滚动完成后再提示与高亮
    nextTick(() => {
      const el = rowEls[validationError.rowIndex]
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
        // 等待滚动动画完成（约 400ms）后再显示提示与高亮，避免一闪而过
        setTimeout(() => {
          Message.warning(validationError.msg)
          el.classList.add('row-error-flash')
          setTimeout(() => el.classList.remove('row-error-flash'), 2000)
        }, 450)
      } else {
        Message.warning(validationError.msg)
      }
    })
    return
  }
  // 收集前置条件未勾选且无内容的分类，这些分类会以零分/默认状态保存
  const skipCats: string[] = []
  for (const g of categoryGroups.value) {
    if (g.precondition_enabled && !g.precondition_checked) {
      skipCats.push(g.category)
    }
  }
  if (skipCats.length > 0) {
    Modal.confirm({
      title: '前置条件未勾选',
      content: `分类「${skipCats.join('、')}」未勾选前置条件，将以默认（未评分）状态保存，确定继续吗？`,
      okText: '确认保存',
      cancelText: '取消',
      maskClosable: false,
      onOk: () => {
        doSave()
      },
    })
    return
  }
  await doSave()
}

function goBack() {
  router.push({ name: 'InspectionList' })
}

// ===== 浮动快捷操作栏 =====
const showFloatingBar = ref(false)
const barRight = ref(0)
const activeRowIndex = ref(0)
const rowEls: Record<number, HTMLElement | null> = {}

// 当前激活行是否可编辑（用于快捷操作栏显隐满分/清零按钮）
const activeRowEditable = computed(() => {
  const idx = activeRowIndex.value
  const row = scoreRows.value[idx]
  if (!row) return false
  const cat = categoryGroups.value.find((g) => g.rowIndices.includes(idx))
  if (!cat) return true
  return isCategoryEditable(cat)
})
let programmaticScrollUntil = 0
let scrollRafId: number | null = null

function setRowRef(index: number, el: any) {
  rowEls[index] = el as HTMLElement | null
}

const checkItemsRef = ref<HTMLElement | undefined>()

useResizeObserver(checkItemsRef, handleBarRight)

function handleBarRight() {
  // 计算浮动栏 right 位置，使其贴近检查项区域右侧
  const container = document.querySelector('.xl\\:col-span-2') as HTMLElement | null
  if (container) {
    const rect = container.getBoundingClientRect()
    barRight.value = Math.max(0, window.innerWidth - rect.right - 30)
  }
}
function onScroll() {
  const mainEl = document.querySelector('main')
  if (!mainEl) return
  showFloatingBar.value = mainEl.scrollTop > 160 && scoreRows.value.length > 0
  const scrollThreshold = 50
  scrolledToBottom.value =
    mainEl.scrollTop + mainEl.clientHeight >= mainEl.scrollHeight - scrollThreshold
  handleBarRight()
  if (Date.now() < programmaticScrollUntil) return
  if (scrollRafId !== null) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    updateActiveRowFromViewport()
  })
}

function updateActiveRowFromViewport() {
  const viewportCenter = window.innerHeight / 2
  let bestIndex = -1
  let bestDist = Infinity
  for (let i = 0; i < scoreRows.value.length; i++) {
    const el = rowEls[i]
    if (!el) continue
    const rect = el.getBoundingClientRect()
    if (rect.height === 0) continue
    const center = rect.top + rect.height / 2
    const dist = Math.abs(center - viewportCenter)
    if (dist < bestDist) {
      bestDist = dist
      bestIndex = i
    }
  }
  if (bestIndex >= 0 && activeRowIndex.value !== bestIndex) {
    activeRowIndex.value = bestIndex
  }
}

function navRow(delta: number) {
  const target = Math.min(Math.max(0, activeRowIndex.value + delta), scoreRows.value.length - 1)
  if (target === activeRowIndex.value) return
  activeRowIndex.value = target
  programmaticScrollUntil = Date.now() + 900
  nextTick(() => {
    const el = rowEls[target]
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

// 一键满分：分值评分设为 max_score，选项评分选第一个选项（合格）
function setRowFullScore() {
  const row = scoreRows.value[activeRowIndex.value]
  if (!row) return
  if (row.score_type === 'pass_fail') {
    // 选项评分：选第一个选项
    if (row.score_options && row.score_options.length > 0) {
      row.score = row.score_options[0].score
    }
  } else {
    row.score = row.max_score
  }
}

// 一键清零：分值评分设为 0，选项评分选最后一个选项（不合格）
function setRowZeroScore() {
  const row = scoreRows.value[activeRowIndex.value]
  if (!row) return
  if (row.score_type === 'pass_fail') {
    if (row.score_options && row.score_options.length > 1) {
      row.score = row.score_options[row.score_options.length - 1].score
    }
  } else {
    row.score = 0
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await fetchStores()
    await fetchTemplates()
    if (!tokenPlanStore.loaded) {
      await tokenPlanStore.loadPlans()
    }
    if (tokenPlanStore.enabledPlans.length > 0) {
      selectedPlanId.value = tokenPlanStore.activePlanId || tokenPlanStore.enabledPlans[0].id
      const plan = tokenPlanStore.enabledPlans.find((p) => p.id === selectedPlanId.value)
      if (plan) {
        const models = parseModelField(plan.model)
        if (models.length > 0) selectedModelId.value = models[0].id
      }
    }
    if (isEdit.value) {
      await fetchDetail()
    } else {
      form.value.checked_at = nowLocalString()
      if (templateOptions.value.length > 0) {
        form.value.template_id = templateOptions.value[0].value
        await loadTemplateItems(form.value.template_id)
      } else {
        await fetchItems()
      }
    }
  } finally {
    loading.value = false
  }
  const mainEl = document.querySelector('main')
  if (mainEl) mainEl.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onWindowResize)
})

onUnmounted(() => {
  const mainEl = document.querySelector('main')
  if (mainEl) mainEl.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onWindowResize)
})

function nowLocalString() {
  const d = new Date()
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 19)
}
</script>

<style scoped lang="scss">
/* ===== 反馈问题 + 图片上传一体化输入区 ===== */
.feedback-input-area {
  width: 100%;
  border: 1px solid #e5e5ea;
  border-radius: 12px;
  background: #f5f5f7;
  padding: 10px 12px;
}
.feedback-input-area--disabled {
  opacity: 0.6;
  pointer-events: none;
}

.feedback-textarea {
  background-color: transparent;
  border: none;
  box-shadow: none;
  &:focus-within,
  &:hover,
  &.arco-textarea-focus {
    background-color: transparent;
    border: none;
    box-shadow: none;
  }
  :deep() {
    .arco-textarea-mirror,
    .arco-textarea {
      padding: 0;
      border: none;
    }
  }
  :deep(.arco-textarea) {
    border: none;
    background: transparent;
    padding: 0;
    font-size: 13px;
    line-height: 1.6;
    resize: none;
    box-shadow: none;
    &::placeholder {
      color: #c0c0c7;
    }
    &:focus {
      box-shadow: none;
      background: transparent;
    }
  }
  :deep(.arco-textarea-word-limit) {
    margin-top: 0;
    margin-bottom: 0;
    padding: 0;
    color: #86868b;
    font-size: 14px;
    text-align: right;
    bottom: 0;
    right: 0;
  }
}

.feedback-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 0;
  padding-top: 6px;
  border-top: 1px solid #e5e5ea;
}
.feedback-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

/* 校验出错行闪烁高亮 */
.row-error-flash {
  animation: row-flash 0.5s ease-in-out 3;
}
@keyframes row-flash {
  0%,
  100% {
    background-color: transparent;
  }
  50% {
    background-color: rgba(255, 59, 48, 0.12);
  }
}

/* ===== AI 巡店实时对话面板 ===== */
.ai-inspect-panel {
  border-radius: 12px;
  overflow: hidden;
  background: #1d1d1f;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.28),
    0 2px 8px rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
}
.ai-inspect-panel__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: linear-gradient(180deg, #2c2c2e 0%, #252527 100%);
  user-select: none;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.ai-inspect-panel__title {
  font-size: 12px;
  font-weight: 600;
  color: #d1d1d6;
  letter-spacing: 0.04em;
}
.ai-inspect-panel__count {
  font-size: 11px;
  color: #636366;
  font-variant-numeric: tabular-nums;
  margin-right: 6px;
}
.ai-inspect-panel__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: #8e8e93;
  cursor: pointer;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}
.ai-inspect-panel__btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}
.ai-inspect-panel__body-wrap {
  overflow: hidden;
}
.ai-inspect-panel__body {
  height: 240px;
  overflow-y: auto;
  padding: 12px 14px;
  font-family: 'SF Mono', ui-monospace, Menlo, Monaco, 'Cascadia Code', 'Roboto Mono', monospace;
  font-size: 12px;
  line-height: 1.7;
  background:
    radial-gradient(ellipse 60% 40% at 80% 0%, rgba(0, 122, 255, 0.05), transparent), #1d1d1f;
}
.ai-inspect-panel__empty {
  color: #48484a;
  font-size: 12px;
}
.ai-inspect-panel__prompt {
  color: #34c759;
  margin-right: 6px;
}

.log-live {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: #34c759;
  margin-left: 8px;
  font-weight: 500;
}
.log-live__pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34c759;
  animation: log-pulse 1.2s ease-in-out infinite;
}
@keyframes log-pulse {
  0%,
  100% {
    opacity: 1;
    box-shadow: 0 0 0 0 rgba(52, 199, 89, 0.5);
  }
  50% {
    opacity: 0.6;
    box-shadow: 0 0 0 4px rgba(52, 199, 89, 0);
  }
}
.log-idle {
  font-size: 10px;
  color: #636366;
  margin-left: 8px;
}

.log-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  animation: log-in 0.25s cubic-bezier(0.25, 0.1, 0.25, 1) both;
  padding: 1px 0;
}
@keyframes log-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.log-line__time {
  color: #48484a;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.log-line__level {
  flex-shrink: 0;
  font-weight: 700;
  font-size: 10px;
  letter-spacing: 0.08em;
  padding: 1px 6px;
  border-radius: 4px;
  white-space: pre;
}
.log-line__msg {
  word-break: break-all;
}
.log-line--info .log-line__level {
  color: #8e8e93;
  background: rgba(142, 142, 147, 0.12);
}
.log-line--info .log-line__msg {
  color: #aeaeb2;
}
.log-line--req .log-line__level {
  color: #ff9f0a;
  background: rgba(255, 159, 10, 0.12);
}
.log-line--req .log-line__msg {
  color: #ffd60a;
}
.log-line--ok .log-line__level {
  color: #30d158;
  background: rgba(48, 209, 88, 0.12);
}
.log-line--ok .log-line__msg {
  color: #6ee7a0;
}
.log-line--err .log-line__level {
  color: #ff453a;
  background: rgba(255, 69, 58, 0.14);
}
.log-line--err .log-line__msg {
  color: #ff6961;
}
.log-line--cursor {
  margin-top: 2px;
}
.log-cursor {
  display: inline-block;
  width: 7px;
  height: 14px;
  background: #34c759;
  animation: cursor-blink 0.9s steps(1) infinite;
  vertical-align: middle;
}

.ai-inspect-stream {
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: #1d1d1f;
  padding: 12px 14px;
}
.ai-inspect-stream__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.ai-inspect-stream__header span {
  color: #d1d1d6;
}
.ai-inspect-stream__badge {
  font-size: 10px;
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.ai-inspect-stream__text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.75;
  color: #aeaeb2;
  max-height: 200px;
  overflow-y: auto;
}
.streaming-cursor {
  display: inline-block;
  width: 2px;
  height: 14px;
  background: #007aff;
  animation: cursor-blink 0.9s steps(1) infinite;
  vertical-align: text-bottom;
  margin-left: 1px;
  border-radius: 1px;
}
@keyframes cursor-blink {
  50% {
    opacity: 0;
  }
}

.log-line__row {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  cursor: default;
}
.log-line--has-detail .log-line__row {
  cursor: pointer;
}
.log-line--has-detail .log-line__row:hover {
  background: rgba(255, 255, 255, 0.04);
  border-radius: 3px;
  margin: 0 -4px;
  padding: 0 4px;
}
.log-line__toggle {
  color: #8e8e93;
  font-size: 11px;
  flex-shrink: 0;
  margin-left: 4px;
}
.log-line__detail {
  margin: 4px 0 0 0;
  padding: 8px 10px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 4px;
  color: #a1a1a6;
  font-size: 11px;
  line-height: 1.5;
  max-height: 240px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

@media (max-width: 768px) {
  .ai-inspect-panel__body {
    max-height: 200px;
    font-size: 11px;
    padding: 10px 12px;
  }
  .ai-inspect-panel__bar {
    padding: 8px 12px;
  }
  .ai-inspect-panel__title {
    font-size: 11px;
  }
  .log-line {
    gap: 6px;
  }
  .log-line__time {
    font-size: 10.5px;
  }
  .log-line__level {
    font-size: 9px;
    padding: 1px 4px;
  }
  .ai-inspect-stream__text {
    max-height: 160px;
    font-size: 12px;
  }
}
</style>
