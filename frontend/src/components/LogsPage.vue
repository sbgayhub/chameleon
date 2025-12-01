<script setup lang="ts">
import {computed, nextTick, onMounted, onUnmounted, ref} from 'vue'
import {ClearLogs, GetLogs, SearchLogs} from "../../wailsjs/go/application/App";
import ConfirmDialog from './common/ConfirmDialog.vue'


// 日志相关状态
const logs = ref<string[]>([])
const loading = ref(false)
const searchTerm = ref('')
const maxLines = ref(500)
const autoRefresh = ref(false)
const refreshInterval = ref<number>()
const confirmDialog = ref<InstanceType<typeof ConfirmDialog> | null>(null)

// 滚动位置相关状态
const logContainer = ref<HTMLElement>()
const scrollPosition = ref(0)
const isAtBottom = ref(true)

// 日志级别过滤器
const logLevels = ['all', 'dbg', 'inf', 'wrn', 'err']
const selectedLevel = ref('all')

// 加载日志数据
const loadLogs = async () => {
  loading.value = true
  try {
    const result = await GetLogs(maxLines.value)
    logs.value = result || []
    // 恢复滚动位置
    restoreScrollPosition()
  } catch (error) {
    console.error('加载日志失败:', error)
    logs.value = [`加载日志失败: ${error}`]
  } finally {
    loading.value = false
  }
}

// 搜索日志
const searchLogs = async () => {
  loading.value = true
  console.log("search log")
  try {
    const result = await SearchLogs(searchTerm.value, maxLines.value)
    logs.value = result || []
    // 恢复滚动位置
    restoreScrollPosition()
  } catch (error) {
    console.error('搜索日志失败:', error)
    logs.value = [`搜索日志失败: ${error}`]
  } finally {
    loading.value = false
  }
}

// 清空日志
const clearLogs = () => {
  confirmDialog.value?.open()
}

const handleClearConfirm = async () => {
  try {
    await ClearLogs()
    logs.value = []
    console.log('日志已清空')
  } catch (error) {
    console.error('清空日志失败:', error)
  }
}

// 检查是否在底部
const checkIfAtBottom = () => {
  if (!logContainer.value) return

  const container = logContainer.value
  const threshold = 50 // 50px 阈值，认为在底部
  isAtBottom.value = container.scrollHeight - container.scrollTop - container.clientHeight <= threshold
}

// 保存滚动位置
const saveScrollPosition = () => {
  if (!logContainer.value) return
  scrollPosition.value = logContainer.value.scrollTop
  checkIfAtBottom()
}

// 恢复滚动位置
const restoreScrollPosition = () => {
  if (!logContainer.value) return

  // 如果原本在底部，或者日志有新增内容且用户在底部，则滚动到新的底部
  if (isAtBottom.value) {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight
      }
    })
  } else {
    // 否则恢复原来的滚动位置
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = scrollPosition.value
      }
    })
  }
}

// 刷新日志
const refreshLogs = () => {
  // 保存当前滚动位置
  saveScrollPosition()

  if (searchTerm.value) {
    searchLogs()
  } else {
    loadLogs()
  }
}

// 切换自动刷新
const toggleAutoRefresh = () => {
  if (autoRefresh.value) {
    refreshInterval.value = setInterval(refreshLogs, 3000) as unknown as number // 每3秒刷新一次
  } else {
    if (refreshInterval.value) {
      clearInterval(refreshInterval.value)
    }
  }
}

// 过滤日志
const filteredLogs = computed(() => {
  if (selectedLevel.value === 'all') {
    return logs.value
  }

  const levelPattern = new RegExp(`\\b${selectedLevel.value}\\b`, 'i')
  return logs.value.filter(log =>
      levelPattern.test(log.split('\t')[0] || '') ||
      levelPattern.test(log.split(' ')[0] || '')
  )
})

// 格式化日志时间戳
const formatLogLine = (line: string) => {
  // 简单的日志高亮处理
  if (line.includes('ERR')) {
    return {text: line, class: 'text-error'}
  } else if (line.includes('WRN')) {
    return {text: line, class: 'text-warning'}
  } else if (line.includes('INF')) {
    return {text: line, class: 'text-info'}
  } else if (line.includes('DBG')) {
    return {text: line, class: 'text-base-content/60'}
  }
  return {text: line, class: ''}
}

// 页面加载时获取日志
onMounted(async () => {
  await loadLogs()

  // 等待 DOM 更新完成
  await nextTick()

  // 添加滚动事件监听
  if (logContainer.value) {
    logContainer.value.addEventListener('scroll', saveScrollPosition)
    // 初始化时检查是否在底部
    checkIfAtBottom()
    // 初始状态下滚动到底部
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
})

// 监听自动刷新状态变化
const onAutoRefreshChange = () => {
  toggleAutoRefresh()
}

// 组件卸载时清理定时器和事件监听器
onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }

  // 移除滚动事件监听器
  if (logContainer.value) {
    logContainer.value.removeEventListener('scroll', saveScrollPosition)
  }
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 页面标题和控制栏 -->
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-3xl font-bold">系统日志</h2>
      <div class="flex items-center gap-3">
        <div class="form-control">
          <label class="label cursor-pointer">
            <span class="label-text mr-2">自动刷新</span>
            <input
                type="checkbox"
                v-model="autoRefresh"
                @change="onAutoRefreshChange"
                class="checkbox checkbox-sm"
            />
          </label>
        </div>
        <button
            @click="refreshLogs"
            :disabled="loading"
            class="btn btn-primary btn-sm"
        >
          <span v-if="loading" class="loading loading-spinner loading-xs"></span>
          刷新
        </button>
        <button
            @click="clearLogs"
            class="btn btn-error btn-sm"
        >
          清空日志
        </button>
      </div>
    </div>

    <!-- 搜索和过滤栏 -->
    <div class="card bg-base-200 shadow-lg mb-2">
      <div class="card-body p-4">
        <div class="flex flex-wrap gap-4">
          <!-- 搜索框 -->
          <div class="form-control flex-2">
<!--            <label class="label mr-2">-->
<!--              <span class="label-text">搜索日志</span>-->
<!--            </label>-->
            <label class="input input-sm">
              <svg class="h-[1em] opacity-50" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                <g stroke-linejoin="round" stroke-linecap="round" stroke-width="2.5" fill="none" stroke="currentColor">
                  <circle cx="11" cy="11" r="8"></circle>
                  <path d="m21 21-4.3-4.3"></path>
                </g>
              </svg>
              <input type="search" class="" placeholder="输入关键词搜索"
                     @keyup.enter="searchLogs"
              />
            </label>
          </div>

          <!-- 日志级别过滤 -->
          <div class="form-control flex-1">
            <label class="label mr-2">
              <span class="label-text">日志级别</span>
            </label>
            <select v-model="selectedLevel" class="max-w-30 select select-bordered select-sm">
              <option v-for="level in logLevels" :key="level" :value="level"> {{ level.toUpperCase() }}</option>
            </select>
          </div>

          <!-- 显示行数 -->
          <div class="form-control flex-1">
            <label class="label mr-2">
              <span class="label-text">显示行数</span>
            </label>
            <select v-model.number="maxLines" @change="refreshLogs" class="max-w-30 select select-bordered select-sm">
              <option :value="100">100 行</option>
              <option :value="500">500 行</option>
              <option :value="1000">1000 行</option>
              <option :value="2000">2000 行</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- 日志显示区域 -->
    <div class="card bg-base-100 shadow-lg flex-1 overflow-hidden">
      <div class="card-body p-0 h-full">
        <div ref="logContainer" class="h-full overflow-auto">
          <div v-if="loading" class="flex items-center justify-center h-32">
            <span class="loading loading-spinner loading-lg"></span>
            <span class="ml-2">加载日志中...</span>
          </div>

          <div v-else-if="filteredLogs.length === 0" class="flex items-center justify-center h-32 text-base-content/60">
            <div class="text-center">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-2 opacity-50" fill="none"
                   viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
              </svg>
              <p>暂无日志记录</p>
            </div>
          </div>

          <div v-else class="font-mono text-sm">
            <div
                v-for="(log, index) in filteredLogs"
                :key="index"
                class="border-b border-base-200 hover:bg-base-200 px-4 py-2 transition-colors"
            >
              <div :class="formatLogLine(log).class">
                <pre class="whitespace-pre-wrap break-all">{{ log }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="mt-2 mb-[-20px] text-sm text-base-content/60">
      <div class="flex items-center justify-between">
        <span>共 {{ filteredLogs.length }} 条日志记录</span>
        <span v-if="autoRefresh" class="text-info">
          <span class="inline-block w-2 h-2 bg-info rounded-full mr-1 animate-pulse"></span>
          自动刷新中
        </span>
      </div>
    </div>

    <!-- 确认对话框 -->
    <ConfirmDialog
      ref="confirmDialog"
      title="清空日志"
      message="确定要清空所有日志吗？此操作不可撤销。"
      confirm-text="清空"
      cancel-text="取消"
      @confirm="handleClearConfirm"
    />
  </div>
</template>