<style scoped>
/* 隐藏滚动条但保持滚动功能 */
.scrollbar-hide {
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
}

.scrollbar-hide::-webkit-scrollbar {
  display: none; /* Chrome, Safari and Opera */
}

.scrollbar-hide::-webkit-scrollbar-track {
  background: transparent;
}

.scrollbar-hide::-webkit-scrollbar-thumb {
  background: transparent;
}
</style>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { GetAllStatistics, GetTotalStatistics, ResetAllStatistics } from '../../../wailsjs/go/statistics/Manager'
import { statistics } from '../../../wailsjs/go/models'
import ConfirmDialog from '../common/ConfirmDialog.vue'

const allStats = ref<Record<string, statistics.Statistics>>({})
const totalStats = ref<statistics.Statistics | null>(null)
const dailyStats = ref<Record<string, any>>({})
const loading = ref(false)
const confirmDialog = ref<InstanceType<typeof ConfirmDialog> | null>(null)

// 分页状态
const currentPage = ref(1)
const pageSize = 10

// 加载统计数据
const loadStats = async () => {
  loading.value = true
  try {
    allStats.value = await GetAllStatistics()
    totalStats.value = await GetTotalStatistics()
    const dailyData = await (window as any).go.statistics.Manager.GetDailyStatistics()
    dailyStats.value = dailyData || {}
  } catch (err) {
    console.error('加载统计数据失败:', err)
  } finally {
    loading.value = false
  }
}

// 计算成功率
const getSuccessRate = (stats: statistics.Statistics) => {
  if (stats.request_count === 0) return 0
  return ((stats.success_count / stats.request_count) * 100).toFixed(1)
}

// 格式化数字
const formatNumber = (num: number) => {
  return num.toLocaleString()
}

// 格式化 Token（K/M/G）
const formatToken = (num: number) => {
  if (num >= 1000000000) {
    return (num / 1000000000).toFixed(1) + 'G'
  } else if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return '从未使用'
  return new Date(time).toLocaleString('zh-CN')
}

// 重置统计
const resetStats = () => {
  confirmDialog.value?.open()
}

const handleResetConfirm = async () => {
  try {
    await ResetAllStatistics()
    await loadStats()
  } catch (err) {
    console.error('重置统计失败:', err)
  }
}

// 渠道列表
const channelList = computed(() => {
  return Object.entries(allStats.value)
})

// 每日统计列表（按日期倒序）
const dailyList = computed(() => {
  return Object.entries(dailyStats.value)
    .sort(([a], [b]) => b.localeCompare(a))
})

// 分页后的每日统计
const paginatedDailyList = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  const end = start + pageSize
  return dailyList.value.slice(start, end)
})

// 总页数
const totalPages = computed(() => {
  return Math.ceil(dailyList.value.length / pageSize)
})

onMounted(() => {
  loadStats()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 标题 -->
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold">使用统计</h2>
      <div class="flex gap-3">
        <button @click="loadStats" class="btn btn-primary gap-2" :disabled="loading">
          <svg v-if="!loading" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span v-if="loading" class="loading loading-spinner loading-sm"></span>
          刷新
        </button>
        <button @click="resetStats" class="btn btn-outline btn-error gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          重置统计
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading && !totalStats" class="flex justify-center items-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 统计内容 -->
    <div v-else class="flex-1 overflow-auto space-y-6 pb-6 scrollbar-hide">
      <!-- 总览卡片 -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div class="stat bg-base-200 rounded-lg shadow-lg">
          <div class="stat-figure text-primary">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
            </svg>
          </div>
          <div class="stat-title">总请求数</div>
          <div class="stat-value text-primary">{{ formatNumber(totalStats?.request_count || 0) }}</div>
        </div>

        <div class="stat bg-base-200 rounded-lg shadow-lg">
          <div class="stat-figure text-success">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="stat-title">成功率</div>
          <div class="stat-value text-success">{{ totalStats ? getSuccessRate(totalStats) : 0 }}%</div>
        </div>

        <div class="stat bg-base-200 rounded-lg shadow-lg">
          <div class="stat-figure text-info">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </div>
          <div class="stat-title">输入 Token</div>
          <div class="stat-value text-info text-2xl">{{ formatToken(totalStats?.input_token || 0) }}</div>
        </div>

        <div class="stat bg-base-200 rounded-lg shadow-lg">
          <div class="stat-figure text-secondary">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
            </svg>
          </div>
          <div class="stat-title">输出 Token</div>
          <div class="stat-value text-secondary text-2xl">{{ formatToken(totalStats?.output_token || 0) }}</div>
        </div>
      </div>

      <!-- 渠道统计表格 -->
      <div class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            渠道统计
          </h3>

          <div v-if="channelList.length === 0" class="text-center py-12 text-base-content/60">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            <p>暂无统计数据</p>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="table">
              <thead>
                <tr>
                  <th>渠道</th>
                  <th>请求次数</th>
                  <th>成功次数</th>
                  <th>失败次数</th>
                  <th>成功率</th>
                  <th>输入Token</th>
                  <th>输出Token</th>
                  <th>最后使用</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="[name, stats] in channelList" :key="name" class="hover">
                  <td class="font-semibold">{{ name }}</td>
                  <td>{{ formatNumber(stats.request_count) }}</td>
                  <td class="text-success">{{ formatNumber(stats.success_count) }}</td>
                  <td class="text-error">{{ formatNumber(stats.failure_count) }}</td>
                  <td>
                    <div class="badge badge-success">{{ getSuccessRate(stats) }}%</div>
                  </td>
                  <td>{{ formatToken(stats.input_token) }}</td>
                  <td>{{ formatToken(stats.output_token) }}</td>
                  <td class="text-sm">{{ formatTime(stats.last_used) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- 每日统计 -->
      <div class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            每日统计
          </h3>

          <div v-if="dailyList.length === 0" class="text-center py-12 text-base-content/60">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p>暂无每日统计数据</p>
          </div>

          <div v-else>
            <div class="overflow-x-auto">
              <table class="table">
                <thead>
                  <tr>
                    <th>日期</th>
                    <th>请求次数</th>
                    <th>成功次数</th>
                    <th>失败次数</th>
                    <th>成功率</th>
                    <th>输入Token</th>
                    <th>输出Token</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="[date, stats] in paginatedDailyList" :key="date" class="hover">
                    <td class="font-semibold">{{ date }}</td>
                    <td>{{ formatNumber(stats.request_count) }}</td>
                    <td class="text-success">{{ formatNumber(stats.success_count) }}</td>
                    <td class="text-error">{{ formatNumber(stats.failure_count) }}</td>
                    <td>
                      <div class="badge badge-success">{{ stats.request_count > 0 ? ((stats.success_count / stats.request_count) * 100).toFixed(1) : 0 }}%</div>
                    </td>
                    <td>{{ formatToken(stats.input_token) }}</td>
                    <td>{{ formatToken(stats.output_token) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <!-- 分页 -->
            <div v-if="totalPages > 1" class="flex justify-center mt-4">
              <div class="join">
                <button
                  class="join-item btn btn-sm"
                  :disabled="currentPage === 1"
                  @click="currentPage--"
                >«</button>
                <button class="join-item btn btn-sm">第 {{ currentPage }} / {{ totalPages }} 页</button>
                <button
                  class="join-item btn btn-sm"
                  :disabled="currentPage === totalPages"
                  @click="currentPage++"
                >»</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 确认对话框 -->
    <ConfirmDialog
      ref="confirmDialog"
      title="重置统计"
      message="确定要重置所有统计数据吗？此操作不可撤销。"
      confirm-text="重置"
      cancel-text="取消"
      @confirm="handleResetConfirm"
    />
  </div>
</template>