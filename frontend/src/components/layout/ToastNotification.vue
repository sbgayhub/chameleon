<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'

interface ToastMessage {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  title: string
  message: string
  duration?: number
  show?: boolean
}

const messages = ref<ToastMessage[]>([])

// 显示 Toast 消息
const showToast = (type: ToastMessage['type'], title: string, message: string, duration = 3000) => {
  const id = Date.now().toString()
  const toast: ToastMessage = {
    id,
    type,
    title,
    message,
    duration,
    show: false
  }

  messages.value.push(toast)

  // 触发显示动画
  nextTick(() => {
    toast.show = true
  })

  // 自动隐藏
  setTimeout(() => {
    hideToast(id)
  }, duration)
}

// 隐藏 Toast 消息
const hideToast = (id: string) => {
  const toast = messages.value.find(msg => msg.id === id)
  if (toast) {
    toast.show = false
    // 延迟移除以完成动画
    setTimeout(() => {
      const index = messages.value.findIndex(msg => msg.id === id)
      if (index > -1) {
        messages.value.splice(index, 1)
      }
    }, 300)
  }
}

// 获取 alert 样式类
const getAlertClass = (type: ToastMessage['type']) => {
  const baseClass = 'alert shadow-lg'
  switch (type) {
    case 'success':
      return `${baseClass} alert-success`
    case 'error':
      return `${baseClass} alert-error`
    case 'warning':
      return `${baseClass} alert-warning`
    case 'info':
    default:
      return `${baseClass} alert-info`
  }
}

// 获取图标
const getIcon = (type: ToastMessage['type']) => {
  switch (type) {
    case 'success':
      return 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
    case 'error':
      return 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z'
    case 'warning':
      return 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z'
    case 'info':
    default:
      return 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
  }
}

// 暴露方法给父组件
defineExpose({
  showToast,
  hideToast
})


onMounted(() => {
  console.log('ToastNotification mounted')
})
</script>

<template>
  <!-- Toast 容器 - 固定在右下角 -->
  <div class="toast-container fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
    <div
      v-for="toast in messages"
      :key="toast.id"
      :class="[
        getAlertClass(toast.type),
        'transition-all duration-300 transform',
        toast.show ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0',
        'pointer-events-auto'
      ]"
      class="min-w-80 max-w-md"
    >
      <div class="flex items-start gap-3">
        <!-- 图标 -->
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="getIcon(toast.type)" />
        </svg>

        <!-- 内容 -->
        <div class="flex-1">
          <h3 class="font-bold">{{ toast.title }}</h3>
          <div class="text-sm">{{ toast.message }}</div>
        </div>

        <!-- 关闭按钮 -->
        <button
          @click="hideToast(toast.id)"
          class="btn btn-ghost btn-xs btn-circle"
        >
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast-container {
  animation: slideInRight 0.3s ease-out;
}

@keyframes slideInRight {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* 确保在最上层 */
.toast-container {
  z-index: 9999;
}
</style>