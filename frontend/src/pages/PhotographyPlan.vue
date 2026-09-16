<template>
  <div class="page-main photography-page">
    <PageHeader
      :title="isWeatherPage ? '天气' : '计划'"
      :subtitle="
        isWeatherPage
          ? '查询未来 ' + selectedSourceMaxDays + ' 天的朝霞与晚霞条件'
          : '管理已保存的个人拍摄安排'
      "
    >
      <template #actions>
        <a-button
          v-if="isWeatherPage"
          size="mini"
          @click="router.push({ name: 'PhotographyPlans' })"
        >
          <template #icon><IconCalendar :size="13" /></template>
          我的计划
        </a-button>
        <a-button v-else size="mini" @click="loadPlans">
          <template #icon><IconRefresh :size="13" /></template>
          刷新计划
        </a-button>
        <a-button
          v-if="isWeatherPage"
          v-perm="'photography_plan:create:write'"
          type="primary"
          size="mini"
          :disabled="!forecast || !selectedDay"
          @click="openCreatePlan"
        >
          <template #icon><IconPlus :size="13" /></template>
          保存当前查询
        </a-button>
        <a-button
          v-else
          type="primary"
          size="mini"
          @click="router.push({ name: 'PhotographyWeather' })"
        >
          <template #icon><IconSun :size="13" /></template>
          查询天气
        </a-button>
      </template>
    </PageHeader>

    <div class="px-4 md:px-6 lg:px-8 flex-1 pb-8">
      <a-alert type="info" show-icon class="mb-4">
        适合度基于云量、降水概率和天气状况估算，仅作为拍摄决策参考，不保证实际出现霞彩。
      </a-alert>

      <template v-if="isWeatherPage">
        <a-card :bordered="false" class="mb-4 shadow-sm">
          <template #title>
            <div class="flex items-center gap-2">
              <IconSearch class="text-[#007AFF]" />
              <span>查询朝霞 / 晚霞</span>
            </div>
          </template>
          <template #extra>
            <span class="text-[12px] text-[#86868B]"
              >天气数据：{{
                forecast?.source_name || selectedWeatherSource?.name || 'Open-Meteo'
              }}</span
            >
          </template>

          <a-alert v-if="forecast?.fallback_reason" type="warning" show-icon class="mb-4">
            {{ forecast.fallback_reason }}
          </a-alert>
          <a-form :model="queryForm" layout="vertical">
            <a-row :gutter="16">
              <a-col :xs="24" :md="24" :lg="24">
                <a-form-item content-class="flex items-center gap-2" label="拍摄地点" required>
                  <div class="flex w-full items-center gap-2">
                    <el-select
                      v-model="selectedLocationKey"
                      class="w-full"
                      popper-class="photography-location-select"
                      filterable
                      remote
                      reserve-keyword
                      clearable
                      :remote-method="remoteSearch"
                      :loading="searchLoading"
                      placeholder="搜索城市或地址，例如：杭州西湖"
                      @change="handleLocationChange"
                      @clear="clearLocationSearch"
                    >
                      <el-option
                        v-for="item in locationOptions"
                        :key="locationOptionKey(item)"
                        :label="item.display_name || item.name"
                        :value="locationOptionKey(item)"
                      >
                        <div class="flex flex-col gap-0.5">
                          <div class="flex items-center justify-between gap-3">
                            <span class="truncate">{{ item.display_name || item.name }}</span>
                            <span class="shrink-0 text-[11px] text-[#86868B]">
                              {{ item.latitude.toFixed(4) }}, {{ item.longitude.toFixed(4) }} ·
                              {{ item.timezone || '自动时区' }}
                            </span>
                          </div>
                          <span class="truncate text-[11px] text-[#86868B]">
                            {{ item.detail || '无详细地址' }}
                            <template v-if="item.adcode || item.city_code">
                              · {{ item.adcode || item.city_code }}
                            </template>
                          </span>
                        </div>
                      </el-option>
                    </el-select>
                    <a-tooltip content="使用当前位置">
                      <a-button size="small" :loading="locating" @click="useCurrentLocation()">
                        <template #icon>
                          <IconLocation />
                        </template>
                      </a-button>
                    </a-tooltip>
                  </div>
                  <div v-if="queryForm.location_name" class="mt-1 text-[11px] text-[#86868B]">
                    已选：{{ queryForm.location_name }}
                    <span v-if="locationTimezone"> · {{ locationTimezone }}</span>
                  </div>
                  <div
                    v-if="
                      queryForm.location_name &&
                      queryForm.latitude !== undefined &&
                      queryForm.longitude !== undefined
                    "
                    class="text-[11px] text-[#86868B]"
                  >
                    坐标：{{ queryForm.latitude.toFixed(4) }}, {{ queryForm.longitude.toFixed(4) }}
                  </div>
                </a-form-item>
              </a-col>
              <a-col :xs="12" :md="5" :lg="4">
                <a-form-item label="纬度" required>
                  <a-input-number
                    v-model="queryForm.latitude"
                    @change="locationTimezone = ''"
                    :min="-90"
                    :max="90"
                    :precision="6"
                    :step="0.0001"
                    placeholder="纬度"
                    style="width: 100%"
                  />
                </a-form-item>
              </a-col>
              <a-col :xs="12" :md="5" :lg="4">
                <a-form-item label="经度" required>
                  <a-input-number
                    v-model="queryForm.longitude"
                    @change="locationTimezone = ''"
                    :min="-180"
                    :max="180"
                    :precision="6"
                    :step="0.0001"
                    placeholder="经度"
                    style="width: 100%"
                  />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8" :lg="6">
                <a-form-item label="天气数据源 / 模型" required>
                  <a-select
                    v-model="weatherSource"
                    :loading="weatherSourcesLoading"
                    placeholder="选择数据源"
                    style="width: 100%"
                  >
                    <a-option
                      v-for="source in weatherSources"
                      :key="source.id"
                      :value="source.id"
                      :disabled="!source.available"
                    >
                      <div class="flex flex-col gap-2" style="line-height: 1; padding: 8px 0">
                        <span>
                          <span>{{ source.name }}</span>
                          <span v-if="!source.available">（未配置）</span>
                        </span>
                        <span class="text-[11px] text-[#86868B]">{{ source.description }}</span>
                      </div>
                    </a-option>
                  </a-select>
                  <template #extra>
                    <div
                      v-if="selectedWeatherSource?.description"
                      class="mt-1 text-[11px] text-[#86868B]"
                    >
                      {{ selectedWeatherSource.description }}
                    </div>
                  </template>
                </a-form-item>
              </a-col>
              <a-col :xs="12" :md="5" :lg="4">
                <a-form-item label="查询日期" required>
                  <a-date-picker
                    v-model="queryForm.date"
                    value-format="YYYY-MM-DD"
                    :disabled-date="disabledDate"
                    style="width: 100%"
                  />
                </a-form-item>
              </a-col>
              <a-col :xs="12" :md="5" :lg="4">
                <a-form-item label="计划时段" required>
                  <a-select v-model="queryForm.session" style="width: 100%">
                    <a-option value="both">朝霞 + 晚霞</a-option>
                    <a-option value="sunrise">只看朝霞</a-option>
                    <a-option value="sunset">只看晚霞</a-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <div class="flex flex-wrap items-center gap-2">
              <a-button type="primary" :loading="forecastLoading" @click="runForecast">
                <template #icon><IconSearch :size="14" /></template>
                查询天气条件
              </a-button>
              <a-button @click="resetQuery">重置</a-button>
              <span v-if="forecast" class="text-[12px] text-[#86868B]">
                {{ forecast.timezone || '自动时区' }} · 已加载 {{ forecast.days.length }} 天
              </span>
            </div>
          </a-form>
        </a-card>

        <a-spin :loading="forecastLoading" tip="正在查询天气数据..." class="w-full">
          <template v-if="forecast && forecast.days.length">
            <a-card :bordered="false" class="mb-4 shadow-sm">
              <template #title>
                <div class="flex items-center gap-2">
                  <IconCalendar class="text-[#007AFF]" />
                  <span>未来 {{ forecast.days.length }} 天概览</span>
                </div>
              </template>
              <template #extra>
                <span class="text-[12px] text-[#86868B]">点击日期查看详情</span>
              </template>

              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8">
                <button
                  v-for="day in forecast.days"
                  :key="day.date"
                  type="button"
                  :class="[
                    'rounded-xl border p-3 text-left transition-all',
                    selectedDate === day.date
                      ? 'border-[#007AFF] bg-[#007AFF]/[0.06] shadow-sm'
                      : 'border-[#E5E5EA] bg-white hover:border-[#A7C7FF] hover:bg-[#F8FBFF]',
                  ]"
                  @click="selectedDate = day.date"
                >
                  <div class="mb-2 flex items-center justify-between gap-2">
                    <span class="text-[13px] font-semibold text-[#1D1D1F]">
                      {{ formatDateLabel(day.date) }}
                    </span>
                    <span v-if="day.date === todayIso" class="text-[10px] text-[#007AFF]"
                      >今天</span
                    >
                  </div>
                  <div class="mb-2 flex items-center gap-2">
                    <span class="text-[24px] leading-none">{{
                      weatherIcon(day.weather_code)
                    }}</span>
                    <div class="min-w-0">
                      <div class="truncate text-[12px] font-medium text-[#1D1D1F]">
                        {{ day.weather_text || weatherLabel(day.weather_code) }}
                      </div>
                      <div class="text-[11px] text-[#86868B]">
                        {{ formatTemperatureRange(day) }} · 湿度
                        {{ formatPercent(day.relative_humidity) }}
                      </div>
                    </div>
                  </div>
                  <div class="space-y-1.5 text-[11px] text-[#5D5D63]">
                    <div class="flex items-center justify-between gap-2">
                      <span class="flex items-center gap-1"><IconSun /> 朝霞</span>
                      <a-tag :color="scoreColor(day.sunrise_level)" size="small">
                        {{ day.sunrise_score }} · {{ scoreLabel(day.sunrise_level) }}
                      </a-tag>
                    </div>
                    <div class="flex items-center justify-between gap-2">
                      <span class="flex items-center gap-1"><IconMoon /> 晚霞</span>
                      <a-tag :color="scoreColor(day.sunset_level)" size="small">
                        {{ day.sunset_score }} · {{ scoreLabel(day.sunset_level) }}
                      </a-tag>
                    </div>
                  </div>
                </button>
              </div>
            </a-card>

            <a-row :gutter="16" class="mb-4">
              <a-col :xs="24" :lg="15">
                <a-card v-if="selectedDay" :bordered="false" class="h-full shadow-sm">
                  <template #title>
                    <div class="flex flex-wrap items-center gap-2">
                      <span>{{ formatFullDate(selectedDay.date) }}</span>
                      <a-tag color="arcoblue" size="small">{{
                        queryForm.location_name || '自定义坐标'
                      }}</a-tag>
                    </div>
                  </template>
                  <template #extra>
                    <span class="text-[12px] text-[#86868B]">{{ forecast.timezone }}</span>
                  </template>

                  <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                    <div class="rounded-2xl bg-[#FFF8E8] p-4">
                      <div class="mb-3 flex items-center justify-between">
                        <div class="flex items-center gap-2 text-[#A46B00]">
                          <IconSun :size="20" />
                          <span class="font-semibold">朝霞窗口</span>
                        </div>
                        <a-tag :color="scoreColor(selectedDay.sunrise_level)">
                          {{ scoreLabel(selectedDay.sunrise_level) }}
                          {{ selectedDay.sunrise_score }} 分
                        </a-tag>
                      </div>
                      <div class="mb-3 text-[32px] font-bold tracking-tight text-[#5C4300]">
                        {{ formatSunTime(selectedDay.sunrise) }}
                      </div>
                      <ul class="space-y-1 text-[12px] leading-relaxed text-[#765D20]">
                        <li v-for="reason in selectedDay.sunrise_reasons" :key="reason">
                          · {{ reason }}
                        </li>
                      </ul>
                    </div>
                    <div class="rounded-2xl bg-[#EEF4FF] p-4">
                      <div class="mb-3 flex items-center justify-between">
                        <div class="flex items-center gap-2 text-[#315EAC]">
                          <IconMoon :size="20" />
                          <span class="font-semibold">晚霞窗口</span>
                        </div>
                        <a-tag :color="scoreColor(selectedDay.sunset_level)">
                          {{ scoreLabel(selectedDay.sunset_level) }}
                          {{ selectedDay.sunset_score }} 分
                        </a-tag>
                      </div>
                      <div class="mb-3 text-[32px] font-bold tracking-tight text-[#23477F]">
                        {{ formatSunTime(selectedDay.sunset) }}
                      </div>
                      <ul class="space-y-1 text-[12px] leading-relaxed text-[#4B6590]">
                        <li v-for="reason in selectedDay.sunset_reasons" :key="reason">
                          · {{ reason }}
                        </li>
                      </ul>
                    </div>
                  </div>

                  <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">天气</div>
                      <div
                        class="mt-1 flex items-center gap-1 text-[16px] font-semibold text-[#1D1D1F]"
                      >
                        <span>{{ weatherIcon(selectedDay.weather_code) }}</span>
                        <span class="truncate">{{
                          selectedDay.weather_text || weatherLabel(selectedDay.weather_code)
                        }}</span>
                      </div>
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">温度</div>
                      <div class="mt-1 text-[17px] font-semibold text-[#1D1D1F]">
                        {{ formatTemperatureRange(selectedDay) }}
                      </div>
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">平均湿度</div>
                      <div class="mt-1 text-[18px] font-semibold text-[#1D1D1F]">
                        {{ formatPercent(selectedDay.relative_humidity) }}
                      </div>
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">平均云量</div>
                      <div class="mt-1 text-[18px] font-semibold text-[#1D1D1F]">
                        {{ formatPercent(selectedDay.cloud_cover) }}
                      </div>
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">降水概率</div>
                      <div class="mt-1 text-[18px] font-semibold text-[#1D1D1F]">
                        {{ formatPercent(selectedDay.precipitation_probability) }}
                      </div>
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] p-3">
                      <div class="text-[11px] text-[#86868B]">坐标</div>
                      <div class="mt-1 text-[13px] font-semibold text-[#1D1D1F]">
                        {{ queryForm.latitude?.toFixed(4) }}, {{ queryForm.longitude?.toFixed(4) }}
                      </div>
                    </div>
                  </div>

                  <div class="mt-4 rounded-2xl border border-[#E5E5EA] bg-[#FBFBFC] p-4">
                    <div class="mb-3 flex items-center justify-between gap-3">
                      <div>
                        <div class="text-[14px] font-semibold text-[#1D1D1F]">24 小时天气</div>
                        <div class="mt-1 text-[11px] text-[#86868B]">
                          当地时间 00:00–23:00 · 温度（°C）
                        </div>
                      </div>
                      <span class="text-[11px] text-[#86868B]"
                        >{{ selectedDay.hours?.length || 0 }} 个时段</span
                      >
                    </div>
                    <template v-if="selectedDay.hours?.length">
                      <div class="overflow-x-auto pb-1">
                        <div class="flex min-w-[720px] gap-1">
                          <div
                            v-for="hour in selectedDay.hours"
                            :key="hour.time"
                            class="flex w-7 shrink-0 flex-col items-center gap-1 text-[10px] text-[#86868B]"
                            :title="hour.weather_text || weatherLabel(hour.weather_code)"
                          >
                            <span>{{ formatHour(hour.time) }}</span>
                            <span class="text-[18px] leading-none">{{
                              weatherIcon(hour.weather_code)
                            }}</span>
                          </div>
                        </div>
                      </div>
                      <div class="mt-2 overflow-x-auto">
                        <div ref="hourlyChartRef" class="h-64 min-w-[720px] w-full"></div>
                      </div>
                    </template>
                    <div
                      v-else
                      class="flex h-32 items-center justify-center text-[12px] text-[#86868B]"
                    >
                      暂无 24 小时天气数据，请刷新天气
                    </div>
                  </div>
                </a-card>
              </a-col>
              <a-col :xs="24" :lg="9">
                <a-card :bordered="false" class="h-full shadow-sm">
                  <template #title>
                    <div class="flex items-center gap-2">
                      <IconInfoCircle class="text-[#FF9500]" />
                      <span>拍摄建议</span>
                    </div>
                  </template>
                  <div class="space-y-3 text-[13px] leading-relaxed text-[#5D5D63]">
                    <p>
                      朝霞和晚霞适合度是基于短时天气条件的估算，实际效果还会受到地形、视野、空气质量和云层高度影响。
                    </p>
                    <div class="rounded-xl bg-[#FFF8E8] px-3 py-2.5 text-[#765D20]">
                      建议在评分达到“良”以上时优先安排，并根据降水概率准备防雨装备。
                    </div>
                    <div class="rounded-xl bg-[#F5F5F7] px-3 py-2.5">
                      当前地点：{{ queryForm.location_name || '手动坐标' }}<br />
                      当地时区：{{ forecast.timezone || '自动时区' }}
                    </div>
                    <a-button
                      v-perm="'photography_plan:create:write'"
                      type="primary"
                      long
                      :disabled="!selectedDay"
                      @click="openCreatePlan"
                    >
                      <template #icon><IconPlus /></template>
                      保存为摄影计划
                    </a-button>
                  </div>
                </a-card>
              </a-col>
            </a-row>
          </template>
          <a-empty v-else description="搜索一个地点并查询天气条件" class="my-16" />
        </a-spin>
      </template>

      <a-card v-if="!isWeatherPage" :bordered="false" class="shadow-sm">
        <template #title>
          <div class="flex items-center gap-2">
            <IconCalendar class="text-[#007AFF]" />
            <span>我的摄影计划</span>
            <a-tag size="small" color="arcoblue">{{ plansTotal }}</a-tag>
          </div>
        </template>
        <a-spin :loading="plansLoading" tip="加载中...">
          <a-table
            :columns="planColumns"
            :data="plans"
            :bordered="false"
            :hoverable="true"
            :pagination="false"
          >
            <template #name="{ record }">
              <div class="min-w-0">
                <div class="truncate text-[13px] font-medium text-[#1D1D1F]">{{ record.name }}</div>
                <div v-if="record.note" class="truncate text-[11px] text-[#86868B]">
                  {{ record.note }}
                </div>
              </div>
            </template>
            <template #date="{ record }">
              <div class="flex flex-col gap-0.5">
                <span class="text-[13px] text-[#1D1D1F]">{{ record.date }}</span>
                <span class="text-[11px] text-[#86868B]">{{ sessionLabel(record.session) }}</span>
              </div>
            </template>
            <template #location="{ record }">
              <div class="flex flex-col gap-0.5">
                <span class="truncate text-[13px] text-[#1D1D1F]">{{ record.location_name }}</span>
                <span class="text-[11px] text-[#86868B]">{{ record.timezone || '自动时区' }}</span>
              </div>
            </template>
            <template #score="{ record }">
              <div v-if="planDay(record)" class="flex flex-wrap gap-1">
                <a-tag
                  v-if="record.session !== 'sunset'"
                  :color="scoreColor(planDay(record)?.sunrise_level)"
                  size="small"
                >
                  朝 {{ planDay(record)?.sunrise_score }}
                </a-tag>
                <a-tag
                  v-if="record.session !== 'sunrise'"
                  :color="scoreColor(planDay(record)?.sunset_level)"
                  size="small"
                >
                  晚 {{ planDay(record)?.sunset_score }}
                </a-tag>
              </div>
              <span v-else class="text-[12px] text-[#C7C7CC]">暂无</span>
            </template>
            <template #synced="{ record }">
              <div class="flex flex-col gap-0.5">
                <span class="text-[12px] text-[#5D5D63]">{{
                  formatDateTime(record.last_synced_at)
                }}</span>
                <span v-if="record.sync_error" class="text-[11px] text-[#FF3B30]"
                  >数据可能已过期</span
                >
              </div>
            </template>
            <template #actions="{ record }">
              <a-space :size="2">
                <a-tooltip content="刷新天气">
                  <a-button
                    v-perm="'photography_plan:update:write'"
                    type="text"
                    size="small"
                    :loading="busyPlanId === record.id"
                    @click="refreshPlan(record)"
                  >
                    <template #icon><IconRefresh /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="编辑">
                  <a-button
                    v-perm="'photography_plan:update:write'"
                    type="text"
                    size="small"
                    @click="openEditPlan(record)"
                  >
                    <template #icon><IconEdit /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除">
                  <a-button
                    v-perm="'photography_plan:delete:write'"
                    type="text"
                    size="small"
                    @click="removePlan(record)"
                  >
                    <template #icon><IconDelete /></template>
                  </a-button>
                </a-tooltip>
              </a-space>
            </template>
          </a-table>
          <a-empty
            v-if="!plansLoading && plans.length === 0"
            description="还没有保存的摄影计划"
            class="py-8"
          />
          <div v-if="plansTotal > pageSize" class="mt-4 flex justify-end">
            <a-pagination
              :current="plansPage"
              :page-size="pageSize"
              :total="plansTotal"
              show-total
              show-jumper
              @change="handlePageChange"
            />
          </div>
        </a-spin>
      </a-card>
    </div>

    <a-modal
      v-model:visible="saveModalVisible"
      :title="editingPlanId ? '编辑摄影计划' : '保存摄影计划'"
      :ok-loading="savingPlan"
      ok-text="保存"
      cancel-text="取消"
      @ok="handleSavePlan"
    >
      <a-form :model="saveForm" layout="vertical">
        <a-form-item label="计划名称" required>
          <a-input v-model="saveForm.name" maxlength="200" placeholder="例如：西湖秋日晨雾" />
        </a-form-item>
        <a-form-item label="拍摄时段" required>
          <a-radio-group v-model="saveForm.session" type="button">
            <a-radio value="both">朝霞 + 晚霞</a-radio>
            <a-radio value="sunrise">只看朝霞</a-radio>
            <a-radio value="sunset">只看晚霞</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea
            v-model="saveForm.note"
            :auto-size="{ minRows: 3, maxRows: 6 }"
            maxlength="1000"
            placeholder="记录机位、镜头、交通或器材准备"
          />
        </a-form-item>
        <div class="rounded-xl bg-[#F5F5F7] px-3 py-2.5 text-[12px] leading-relaxed text-[#5D5D63]">
          {{ queryForm.location_name || '手动坐标' }} · {{ queryForm.date }} ·
          {{ queryForm.latitude?.toFixed(4) }}, {{ queryForm.longitude?.toFixed(4) }}<br />
          数据源：{{ selectedWeatherSource?.name || '自动匹配' }}
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconCalendar,
  IconDelete,
  IconEdit,
  IconInfoCircle,
  IconLocation,
  IconMoon,
  IconPlus,
  IconRefresh,
  IconSearch,
  IconSun,
} from '@arco-design/web-vue/es/icon'
import PageHeader from '@/components/layout/PageHeader.vue'
import { useRoute, useRouter } from 'vue-router'
import api, { WEATHER_API_TIMEOUT, getApiErrorDetail } from '@/utils/api'
import { countryCodeForRegion } from '@/utils/time'
import { useLocationStore } from '@/stores/location'
import { useRegionStore } from '@/stores/region'
import {
  amapTipToLocation,
  ensureAmapJS,
  searchLocationsByAmapJS,
  type PhotographyMapConfig,
} from '@/utils/amapSearch'
import type {
  PaginatedPhotographyPlans,
  PhotographyAssessmentLevel,
  PhotographyForecast,
  PhotographyForecastDay,
  PhotographyLocation,
  PhotographyWeatherSource,
  PhotographyPlan,
  PhotographyPlanPayload,
  PhotographySession,
} from './PhotographyPlan.types'

const pageSize = 10
const route = useRoute()
const router = useRouter()
const locationStore = useLocationStore()
const { selectedTz } = useRegionStore()
const isWeatherPage = computed(() => route.name === 'PhotographyWeather')
const defaultCountryCode = computed(() => countryCodeForRegion(selectedTz.value))
const locationTimezone = ref('')
const todayIso = computed(() => formatDateInTimezone(new Date(), locationTimezone.value))
const selectedSourceMaxDays = computed(() => selectedWeatherSource.value?.max_days || 16)
const maxDateIso = computed(() =>
  formatIsoDate(addDays(new Date(todayIso.value + 'T00:00:00'), selectedSourceMaxDays.value - 1)),
)

const queryForm = reactive<{
  date: string
  session: PhotographySession
  location_name: string
  latitude: number | undefined
  longitude: number | undefined
}>({
  date: todayIso.value,
  session: 'both',
  location_name: '',
  latitude: undefined,
  longitude: undefined,
})

const searchResults = ref<PhotographyLocation[]>([])
const mapConfig = ref<PhotographyMapConfig | null>(null)
const selectedLocation = ref<PhotographyLocation | null>(null)
const selectedLocationKey = ref('')
const searchLoading = ref(false)
const locationOptions = computed(() => {
  const list = [...searchResults.value]
  const current = selectedLocation.value
  if (current && !list.some((item) => locationOptionKey(item) === selectedLocationKey.value)) {
    list.unshift(current)
  }
  return list
})
const forecast = ref<PhotographyForecast | null>(null)
const weatherSources = ref<PhotographyWeatherSource[]>([])
const weatherSourcesLoading = ref(false)
const weatherSource = ref('open-meteo-best-match')
const selectedWeatherSource = computed(() =>
  weatherSources.value.find((source) => source.id === weatherSource.value),
)
const selectedDate = ref(todayIso.value)
const forecastLoading = ref(false)
const locating = computed(() => locationStore.isLocating)
const plansLoading = ref(false)
const plans = ref<PhotographyPlan[]>([])
const plansTotal = ref(0)
const plansPage = ref(1)
const busyPlanId = ref<string | null>(null)
const saveModalVisible = ref(false)
const savingPlan = ref(false)
const editingPlanId = ref<string | null>(null)
const saveForm = reactive<{ name: string; session: PhotographySession; note: string }>({
  name: '',
  session: 'both',
  note: '',
})

let locationTimer: number | undefined
let locationRequestId = 0

const selectedDay = computed<PhotographyForecastDay | null>(() => {
  return (
    forecast.value?.days.find((day) => day.date === selectedDate.value) ??
    forecast.value?.days[0] ??
    null
  )
})

const hourlyChartRef = ref<HTMLDivElement | null>(null)
let hourlyChart: echarts.ECharts | null = null

const planColumns = [
  { title: '计划', dataIndex: 'name', slotName: 'name', width: 220 },
  { title: '日期 / 时段', dataIndex: 'date', slotName: 'date', width: 135 },
  { title: '地点', dataIndex: 'location_name', slotName: 'location', width: 230 },
  { title: '适合度', dataIndex: 'score', slotName: 'score', width: 150 },
  { title: '天气数据', dataIndex: 'last_synced_at', slotName: 'synced', width: 160 },
  { title: '操作', slotName: 'actions', align: 'right' as const, width: 130 },
]

watch(weatherSource, () => {
  ensureDateWithinLocationRange()
})

watch(
  [selectedDay, isWeatherPage],
  () => {
    void renderHourlyChart()
  },
  { flush: 'post' },
)

// el-select 的 option value 必须是稳定字符串，用它反查完整地点对象。
function locationOptionKey(item: PhotographyLocation) {
  return `${item.latitude},${item.longitude},${item.display_name || item.name}`
}

function remoteSearch(keyword: string) {
  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  searchResults.value = []
  const query = keyword.trim()
  if (query.length < 2) {
    searchLoading.value = false
    return
  }
  const requestId = ++locationRequestId
  searchLoading.value = true
  locationTimer = window.setTimeout(async () => {
    try {
      const list = await fetchLocations(query)
      if (requestId !== locationRequestId) return
      searchResults.value = list
    } catch (error) {
      if (requestId === locationRequestId) {
        searchResults.value = []
        Message.error(getApiErrorDetail(error) || '地址搜索失败，请检查地图服务配置')
      }
    } finally {
      if (requestId === locationRequestId) searchLoading.value = false
    }
  }, 350)
}

async function loadMapConfig(): Promise<PhotographyMapConfig | null> {
  if (mapConfig.value) return mapConfig.value
  try {
    const response = await api.get<PhotographyMapConfig>('/photography-tools/map-config', {
      params: { country_code: defaultCountryCode.value },
    })
    mapConfig.value = response.data || null
  } catch {
    mapConfig.value = null
  }
  return mapConfig.value
}

// 中国大陆直接调用高德 JS API 2.0 的 AutoComplete，其他地区走服务端 Google 地理编码。
async function fetchLocations(query: string): Promise<PhotographyLocation[]> {
  const config = await loadMapConfig()
  if (config?.provider === 'google') {
    const response = await api.get<PhotographyLocation[]>('/photography-plans/geocode', {
      params: { query, country_code: defaultCountryCode.value },
    })
    return response.data || []
  }
  await ensureAmapJS(config)
  const tips = await searchLocationsByAmapJS(query)
  return tips.map((tip) => amapTipToLocation(tip))
}

function handleLocationChange(key: string) {
  const item = locationOptions.value.find((entry) => locationOptionKey(entry) === key)
  if (item) selectLocation(item)
}

watch(selectedTz, () => {
  locationRequestId++
  mapConfig.value = null
  searchResults.value = []
  searchLoading.value = false
})

function selectLocation(location: PhotographyLocation) {
  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  locationRequestId++
  selectedLocation.value = location
  selectedLocationKey.value = locationOptionKey(location)
  queryForm.location_name = location.display_name || location.name
  queryForm.latitude = location.latitude
  queryForm.longitude = location.longitude
  locationTimezone.value = location.timezone || ''
  ensureDateWithinLocationRange()
  searchResults.value = []
  searchLoading.value = false
  void fillLocationTimezone(location)
}

// 高德 JS API 搜索结果没有时区，按经纬度向 Open-Meteo 补一次，用于展示与日期校验。
async function fillLocationTimezone(location: PhotographyLocation) {
  if (location.timezone) return
  try {
    const response = await api.get<{ timezone: string }>('/photography-tools/timezone', {
      params: { latitude: location.latitude, longitude: location.longitude },
    })
    const timezone = response.data?.timezone || ''
    if (!timezone || selectedLocationKey.value !== locationOptionKey(location)) return
    selectedLocation.value = { ...location, timezone }
    locationTimezone.value = timezone
    ensureDateWithinLocationRange()
  } catch {
    // 时区只是辅助信息，查询结果里仍会带回地点时区，静默忽略。
  }
}

function clearLocationSearch() {
  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  locationRequestId++
  selectedLocation.value = null
  selectedLocationKey.value = ''
  searchResults.value = []
  searchLoading.value = false
  queryForm.location_name = ''
  queryForm.latitude = undefined
  queryForm.longitude = undefined
  locationTimezone.value = ''
}

async function useCurrentLocation(force = true) {
  const location = await locationStore.locate(force)
  if (!location) {
    Message.error(locationStore.error || '获取当前位置失败，请搜索城市或手动填写经纬度')
    return
  }

  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  locationRequestId++
  selectedLocation.value = null
  selectedLocationKey.value = ''
  searchResults.value = []
  searchLoading.value = false
  queryForm.location_name = '当前位置'
  queryForm.latitude = Number(location.latitude.toFixed(6))
  queryForm.longitude = Number(location.longitude.toFixed(6))
  locationTimezone.value = ''
  queryForm.date = formatIsoDate(new Date())
  selectedDate.value = queryForm.date
  runForecast()
}

function disabledDate(date?: Date): boolean {
  if (!date) return false
  const value = formatIsoDate(date)
  return value < todayIso.value || value > maxDateIso.value
}

function ensureDateWithinLocationRange() {
  if (queryForm.date < todayIso.value || queryForm.date > maxDateIso.value) {
    queryForm.date = todayIso.value
    selectedDate.value = todayIso.value
  }
}

function validateQuery(): boolean {
  if (!queryForm.location_name.trim()) {
    Message.warning('请先选择或填写拍摄地点')
    return false
  }
  if (queryForm.latitude === undefined || queryForm.longitude === undefined) {
    Message.warning('请填写有效的经纬度')
    return false
  }
  if (disabledDate(new Date(queryForm.date + 'T00:00:00'))) {
    Message.warning('查询日期仅支持所选数据源今天起 ' + selectedSourceMaxDays.value + ' 天内')
    return false
  }
  return true
}

async function runForecast() {
  if (!validateQuery()) return
  forecastLoading.value = true
  try {
    const response = await api.get<PhotographyForecast>('/photography-plans/forecast', {
      params: {
        latitude: queryForm.latitude,
        longitude: queryForm.longitude,
        date: queryForm.date,
        source: weatherSource.value,
      },
      timeout: WEATHER_API_TIMEOUT,
    })
    forecast.value = response.data
    weatherSource.value = response.data.source || weatherSource.value
    locationTimezone.value = response.data.timezone || locationTimezone.value
    ensureDateWithinLocationRange()
    selectedDate.value = queryForm.date
    Message.success('天气条件查询完成')
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '天气服务暂时不可用，请稍后重试')
  } finally {
    forecastLoading.value = false
  }
}

function resetQuery() {
  clearLocationSearch()
  locationTimezone.value = ''
  queryForm.date = todayIso.value
  queryForm.session = 'both'
  queryForm.location_name = ''
  queryForm.latitude = undefined
  queryForm.longitude = undefined
  forecast.value = null
  selectedDate.value = todayIso.value
}

async function loadWeatherSources() {
  weatherSourcesLoading.value = true
  try {
    const response = await api.get<PhotographyWeatherSource[]>('/photography-plans/weather-sources')
    weatherSources.value = response.data || []
  } catch (error) {
    Message.warning(getApiErrorDetail(error) || '天气数据源加载失败，将使用默认数据源')
  } finally {
    weatherSourcesLoading.value = false
  }
}

async function loadPlans() {
  plansLoading.value = true
  try {
    const response = await api.get<PaginatedPhotographyPlans>('/photography-plans/', {
      params: { page: plansPage.value, page_size: pageSize },
    })
    plans.value = response.data.items || []
    plansTotal.value = response.data.total || 0
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '加载摄影计划失败')
  } finally {
    plansLoading.value = false
  }
}

function handlePageChange(page: number) {
  plansPage.value = page
  loadPlans()
}

function openCreatePlan() {
  if (!forecast.value || !selectedDay.value) {
    Message.warning('请先查询天气条件')
    return
  }
  editingPlanId.value = null
  saveForm.name = formatDateLabel(selectedDay.value.date) + ' ' + sessionLabel(queryForm.session)
  saveForm.session = queryForm.session
  saveForm.note = ''
  saveModalVisible.value = true
}

function openEditPlan(plan: PhotographyPlan) {
  editingPlanId.value = plan.id
  queryForm.date = plan.date
  queryForm.session = plan.session
  queryForm.location_name = plan.location_name
  queryForm.latitude = plan.latitude
  queryForm.longitude = plan.longitude
  weatherSource.value = plan.weather_source || 'open-meteo-best-match'
  locationTimezone.value = plan.timezone || ''
  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  locationRequestId++
  selectedLocation.value = {
    name: plan.location_name,
    display_name: plan.location_name,
    detail: '',
    country: '',
    country_code: '',
    admin1: '',
    city_code: '',
    adcode: '',
    latitude: plan.latitude,
    longitude: plan.longitude,
    timezone: plan.timezone,
  }
  selectedLocationKey.value = locationOptionKey(selectedLocation.value)
  searchResults.value = []
  searchLoading.value = false
  selectedDate.value = plan.date
  forecast.value = {
    latitude: plan.latitude,
    longitude: plan.longitude,
    timezone: plan.timezone,
    source: plan.weather_source || 'open-meteo-best-match',
    source_name: plan.weather_source_name || '自动匹配（推荐）',
    days: plan.forecast_snapshot?.days || [],
  }
  saveForm.name = plan.name
  saveForm.session = plan.session
  saveForm.note = plan.note || ''
  saveModalVisible.value = true
}

async function handleSavePlan() {
  if (!saveForm.name.trim()) {
    Message.warning('请填写计划名称')
    return
  }
  if (!validateQuery()) return
  const payload: PhotographyPlanPayload = {
    name: saveForm.name.trim(),
    date: queryForm.date,
    session: saveForm.session,
    location_name: queryForm.location_name.trim(),
    latitude: queryForm.latitude as number,
    longitude: queryForm.longitude as number,
    weather_source: weatherSource.value,
    note: saveForm.note.trim(),
  }
  savingPlan.value = true
  try {
    if (editingPlanId.value) {
      await api.put<PhotographyPlan>('/photography-plans/' + editingPlanId.value, payload, {
        timeout: WEATHER_API_TIMEOUT,
      })
      Message.success('摄影计划已更新')
    } else {
      await api.post<PhotographyPlan>('/photography-plans/', payload, {
        timeout: WEATHER_API_TIMEOUT,
      })
      Message.success('摄影计划已保存')
    }
    saveModalVisible.value = false
    await loadPlans()
  } catch (error) {
    Message.error(getApiErrorDetail(error) || '保存摄影计划失败')
  } finally {
    savingPlan.value = false
  }
}

async function refreshPlan(plan: PhotographyPlan) {
  busyPlanId.value = plan.id
  try {
    const response = await api.post<PhotographyPlan>(
      '/photography-plans/' + plan.id + '/refresh',
      undefined,
      { timeout: WEATHER_API_TIMEOUT },
    )
    const index = plans.value.findIndex((item) => item.id === plan.id)
    if (index !== -1) plans.value[index] = response.data
    Message.success('天气数据已刷新')
  } catch (error) {
    const detail = getApiErrorDetail(error) || '刷新天气失败，数据可能已过期'
    const index = plans.value.findIndex((item) => item.id === plan.id)
    if (index !== -1 && !plans.value[index].sync_error) {
      plans.value[index] = { ...plans.value[index], sync_error: detail }
    }
    Message.error(detail)
    await loadPlans()
  } finally {
    busyPlanId.value = null
  }
}

function removePlan(plan: PhotographyPlan) {
  Modal.warning({
    title: '确认删除摄影计划',
    content: '删除后不可恢复，确定要删除“' + plan.name + '”吗？',
    hideCancel: false,
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      try {
        await api.delete('/photography-plans/' + plan.id)
        plans.value = plans.value.filter((item) => item.id !== plan.id)
        plansTotal.value -= 1
        Message.success('摄影计划已删除')
      } catch (error) {
        Message.error(getApiErrorDetail(error) || '删除摄影计划失败')
      }
    },
  })
}

function planDay(plan: PhotographyPlan): PhotographyForecastDay | null {
  return (
    plan.forecast_snapshot?.days?.find((day) => day.date === plan.date) ||
    plan.forecast_snapshot?.days?.[0] ||
    null
  )
}

function formatDateInTimezone(date: Date, timezone: string): string {
  if (!timezone) return formatIsoDate(date)
  try {
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).formatToParts(date)
    const values = Object.fromEntries(parts.map((part) => [part.type, part.value]))
    return values.year + '-' + values.month + '-' + values.day
  } catch {
    return formatIsoDate(date)
  }
}

function formatIsoDate(date: Date): string {
  const pad = (value: number) => (value < 10 ? '0' + value : String(value))
  return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate())
}

function addDays(date: Date, days: number): Date {
  const result = new Date(date)
  result.setDate(result.getDate() + days)
  return result
}

function formatDateLabel(value: string): string {
  const date = new Date(value + 'T00:00:00')
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    weekday: 'short',
  }).format(date)
}

function formatFullDate(value: string): string {
  const date = new Date(value + 'T00:00:00')
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  }).format(date)
}

function formatSunTime(value: string): string {
  return value ? value.slice(11, 16) : '--:--'
}

function formatPercent(value: number | null): string {
  return value === null || value === undefined ? '--' : Math.round(value) + '%'
}

function formatDateTime(value: string | null): string {
  if (!value) return '未同步'
  return value.replace('T', ' ').slice(0, 16)
}

function sessionLabel(value: PhotographySession): string {
  if (value === 'sunrise') return '朝霞'
  if (value === 'sunset') return '晚霞'
  return '朝霞 + 晚霞'
}

function scoreLabel(level: PhotographyAssessmentLevel | undefined): string {
  if (level === 'excellent') return '优'
  if (level === 'good') return '良'
  if (level === 'fair') return '一般'
  return '不建议'
}

function scoreColor(level: PhotographyAssessmentLevel | undefined): string {
  if (level === 'excellent') return 'green'
  if (level === 'good') return 'arcoblue'
  if (level === 'fair') return 'orange'
  return 'red'
}

function formatTemperature(value: number | null | undefined): string {
  return value === null || value === undefined ? '--' : Math.round(value) + '°'
}

function formatTemperatureRange(day: PhotographyForecastDay): string {
  if (day.temperature_min == null && day.temperature_max == null) return '--'
  return (
    '最低 ' +
    formatTemperature(day.temperature_min) +
    ' / 最高 ' +
    formatTemperature(day.temperature_max)
  )
}

function formatHour(value: string): string {
  const timeIndex = value.indexOf('T')
  if (timeIndex === -1) return value.slice(-5)
  return value.slice(timeIndex + 1, timeIndex + 6)
}

function weatherIcon(code: number | null | undefined): string {
  if (code === null || code === undefined) return '—'
  if (code === 0) return '☀️'
  if (code === 1 || code === 2) return '🌤️'
  if (code === 3) return '☁️'
  if (code === 45 || code === 48) return '🌫️'
  if (code >= 51 && code <= 67) return '🌧️'
  if (code >= 71 && code <= 77) return '🌨️'
  if (code >= 80 && code <= 86) return '🌦️'
  if (code >= 95) return '⛈️'
  return '🌥️'
}

async function renderHourlyChart() {
  await nextTick()
  const hours = selectedDay.value?.hours || []
  if (!isWeatherPage.value || !hourlyChartRef.value || hours.length === 0) {
    if (hourlyChart) {
      hourlyChart.dispose()
      hourlyChart = null
    }
    return
  }

  if (!hourlyChart) {
    hourlyChart = echarts.init(hourlyChartRef.value)
  }
  hourlyChart.setOption(
    {
      animation: false,
      grid: { left: 12, right: 18, top: 20, bottom: 24, containLabel: true },
      tooltip: { trigger: 'axis' },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: hours.map((hour) => formatHour(hour.time)),
        axisLabel: { interval: 2, color: '#86868B' },
        axisLine: { lineStyle: { color: '#D8D8DE' } },
      },
      yAxis: {
        type: 'value',
        name: '°C',
        nameTextStyle: { color: '#86868B' },
        axisLabel: { color: '#86868B', formatter: '{value}°' },
        splitLine: { lineStyle: { color: '#EDEDF0' } },
      },
      series: [
        {
          type: 'line',
          smooth: true,
          connectNulls: true,
          symbol: 'circle',
          symbolSize: 7,
          data: hours.map((hour) => hour.temperature),
          lineStyle: { width: 3, color: '#007AFF' },
          itemStyle: { color: '#007AFF', borderColor: '#FFFFFF', borderWidth: 2 },
          areaStyle: { color: 'rgba(0, 122, 255, 0.10)' },
        },
      ],
    },
    true,
  )
}

function handleHourlyChartResize() {
  hourlyChart?.resize()
}

function weatherLabel(code: number | null): string {
  if (code === null || code === undefined) return '暂无数据'
  if (code === 0) return '晴朗'
  if (code === 1 || code === 2) return '少云'
  if (code === 3) return '阴天'
  if (code === 45 || code === 48) return '雾'
  if (code >= 51 && code <= 67) return '降雨'
  if (code >= 71 && code <= 77) return '降雪'
  if (code >= 80 && code <= 86) return '阵雨或阵雪'
  if (code >= 95) return '雷暴'
  return '多云'
}

onMounted(() => {
  loadWeatherSources()
  loadPlans()
  window.addEventListener('resize', handleHourlyChartResize)
})

onBeforeUnmount(() => {
  if (locationTimer !== undefined) window.clearTimeout(locationTimer)
  window.removeEventListener('resize', handleHourlyChartResize)
  hourlyChart?.dispose()
  hourlyChart = null
})
</script>

<style scoped lang="scss">
.photography-page :deep(.arco-card-header) {
  padding-bottom: 14px;
}

.photography-page :deep(.arco-form-item) {
  margin-bottom: 14px;
}

@media (max-width: 768px) {
  .photography-page :deep(.arco-table-th),
  .photography-page :deep(.arco-table-td) {
    padding-left: 8px;
    padding-right: 8px;
  }
}
</style>
