<script setup lang="ts">
import { ref } from 'vue'
import ProxyStatus from '../ProxyStatus.vue'

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
  currentPage: string
  proxyStatus: ProxyState
}>()

// 定义事件
const emit = defineEmits<{
  navigate: [page: string]
  toggleProxy: []
}>()

// 导航到指定页面
const navigateTo = (page: string) => {
  emit('navigate', page)
}

// 处理代理切换
const handleToggleProxy = () => {
  emit('toggleProxy')
}
</script>

<template>
  <!-- 侧边导航栏 -->
  <aside class="w-20 bg-base-200 flex flex-col" style="padding: 24px 0">
    <!-- 上部导航按钮 -->
    <div class="flex flex-col items-center gap-1">
      <!-- 主页按钮 -->
      <button
          @click="navigateTo('home')"
          class="flex flex-col items-center justify-center gap-1 rounded-lg hover:bg-base-300 transition-colors"
          :class="{ 'bg-base-300': currentPage === 'home' }"
          style="width: 60px; height: 60px"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
        </svg>
        <span class="text-xs">主页</span>
      </button>

      <!-- 代理按钮 -->
      <button
          @click="navigateTo('channel')"
          class="flex flex-col items-center justify-center gap-1 rounded-lg hover:bg-base-300 transition-colors"
          :class="{ 'bg-base-300': currentPage === 'channel' }"
          style="width: 60px; height: 60px"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
        </svg>
        <span class="text-xs">渠道</span>
      </button>

      <!-- 统计按钮 -->
      <button
          @click="navigateTo('stats')"
          class="flex flex-col items-center justify-center gap-1 rounded-lg hover:bg-base-300 transition-colors"
          :class="{ 'bg-base-300': currentPage === 'stats' }"
          style="width: 60px; height: 60px"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
        </svg>
        <span class="text-xs">统计</span>
      </button>

      <!-- 日志按钮 -->
      <button
          @click="navigateTo('logs')"
          class="flex flex-col items-center justify-center gap-1 rounded-lg hover:bg-base-300 transition-colors"
          :class="{ 'bg-base-300': currentPage === 'logs' }"
          style="width: 60px; height: 60px"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
        </svg>
        <span class="text-xs">日志</span>
      </button>

      <!-- 设置按钮 -->
      <button
          @click="navigateTo('settings')"
          class="flex flex-col items-center justify-center gap-1 rounded-lg hover:bg-base-300 transition-colors"
          :class="{ 'bg-base-300': currentPage === 'settings' }"
          style="width: 60px; height: 60px"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
        </svg>
        <span class="text-xs">设置</span>
      </button>
    </div>

    <!-- 底部代理状态 -->
    <div class="mt-auto flex flex-col pt-2 ">
      <div class="divider h-1 m-2"></div>
      <ProxyStatus
        :status="proxyStatus"
        ref="proxyStatusRef"
        @toggle-proxy="handleToggleProxy"
      />
    </div>
  </aside>
</template>