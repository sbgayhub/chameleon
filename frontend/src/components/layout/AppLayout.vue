<script setup lang="ts">
import { ref } from 'vue'
import TitleBar from './TitleBar.vue'
import NavigationBar from './NavigationBar.vue'
import ToastNotification from './ToastNotification.vue'

// 定义属性
const props = defineProps<{
  currentPage: string
  proxyStatus: any
}>()

// 定义事件
const emit = defineEmits<{
  navigate: [page: string]
  toggleProxy: []
  startProxy: []
  stopProxy: []
}>()

// Toast 引用
const toastRef = ref()

// 标题栏事件处理
const handleTitleBarEvent = (event: string, ...args: any[]) => {
  console.log('TitleBar event:', event, args)
}

// 导航事件处理
const handleNavigate = (page: string) => {
  emit('navigate', page)
}

// 代理切换事件处理
const handleToggleProxy = () => {
  emit('toggleProxy')
}

// 暴露 Toast 引用给父组件
defineExpose({
  toastRef
})
</script>

<template>
  <div class="flex flex-col h-screen">
    <!-- 顶部标题栏 -->
    <TitleBar
      @minimize="handleTitleBarEvent('minimize')"
      @maximize="handleTitleBarEvent('maximize', $event)"
      @close="handleTitleBarEvent('close')"
      @themeChange="handleTitleBarEvent('themeChange')"
    />

    <!-- 主内容区 -->
    <div class="flex flex-1 overflow-hidden">
      <!-- 侧边导航栏 -->
      <NavigationBar
        :current-page="currentPage"
        :proxy-status="proxyStatus"
        @navigate="handleNavigate"
        @toggle-proxy="handleToggleProxy"
      />

      <!-- 页面内容 -->
      <slot :toast-ref="toastRef" />
    </div>

    <!-- Toast 通知组件 -->
    <ToastNotification ref="toastRef" />
  </div>
</template>