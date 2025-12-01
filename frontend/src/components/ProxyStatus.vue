<script setup lang="ts">
import {onUnmounted, watch} from 'vue'

// 代理状态接口
interface ProxyState {
  isRunning: boolean
  startTime: Date | null
  uptime: string
  port: number
  activeConnections: number
  totalRequests: number
}

// 定义属性
const props = defineProps<{
  status: ProxyState
}>()

// 定义事件
const emit = defineEmits<{
  toggleProxy: []
}>()

// 计算运行时长
let uptimeInterval: ReturnType<typeof setInterval> | null = null

const calculateUptime = () => {
  if (!props.status.isRunning || !props.status.startTime) {
    props.status.uptime = '00:00:00'
    return
  }

  const now = new Date()
  const diff = now.getTime() - props.status.startTime.getTime()

  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)

  // 格式化为两位数显示
  const formattedHours = hours.toString().padStart(2, '0')
  const formattedMinutes = minutes.toString().padStart(2, '0')
  const formattedSeconds = seconds.toString().padStart(2, '0')

  props.status.uptime = `${formattedHours}:${formattedMinutes}:${formattedSeconds}`
}

// 处理开关点击事件
const handleToggleChange = () => {
  emit('toggleProxy')
}

// 启动计时器
const startUptimeTimer = () => {
  if (uptimeInterval) {
    clearInterval(uptimeInterval)
  }
  uptimeInterval = setInterval(calculateUptime, 1000)
  // 立即计算一次
  calculateUptime()
}

// 停止计时器
const stopUptimeTimer = () => {
  if (uptimeInterval) {
    clearInterval(uptimeInterval)
    uptimeInterval = null
  }
  // 清空时长显示
  props.status.uptime = '00:00:00'
}

// 监听状态变化，自动启停计时器
watch(() => props.status.isRunning, (newValue) => {
  if (newValue) {
    startUptimeTimer()
  } else {
    stopUptimeTimer()
  }
})

// 组件卸载时清理定时器
onUnmounted(() => {
  stopUptimeTimer()
})

// 暴露方法给父组件
defineExpose({
  startUptimeTimer,
  stopUptimeTimer
})
</script>

<template>
  <div class="flex flex-col items-center gap-2">
    <!-- 代理启动开关 -->
    <div class="flex flex-col items-center gap-1">
      <input
          type="checkbox"
          :checked="status.isRunning"
          class="toggle toggle-success toggle-lg"
          @change="handleToggleChange"
      />
    </div>

    <!-- 运行时长 -->
    <div class="text-center">
      <div class="text-xs font-mono text-base-content/80">{{ status.uptime || '00:00:00' }}</div>
    </div>
  </div>
</template>