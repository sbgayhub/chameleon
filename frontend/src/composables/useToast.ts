import { ref } from 'vue'

export interface ToastMessage {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  title: string
  message: string
  duration?: number
}

// Toast Hook
export function useToast() {
  const toastRef = ref()

  // 显示 Toast 消息
  const showToast = (type: ToastMessage['type'], title: string, message: string, duration = 3000) => {
    // 延迟确保组件已挂载
    setTimeout(() => {
      if (toastRef.value) {
        toastRef.value.showToast(type, title, message, duration)
      } else {
        console.warn('Toast ref not available, using fallback')
        console.log(`[${type.toUpperCase()}] ${title}: ${message}`)
      }
    }, 100)
  }

  // 便捷方法
  const showSuccess = (title: string, message: string, duration?: number) => {
    showToast('success', title, message, duration)
  }

  const showError = (title: string, message: string, duration?: number) => {
    showToast('error', title, message, duration)
  }

  const showInfo = (title: string, message: string, duration?: number) => {
    showToast('info', title, message, duration)
  }

  const showWarning = (title: string, message: string, duration?: number) => {
    showToast('warning', title, message, duration)
  }

  return {
    toastRef,
    showToast,
    showSuccess,
    showError,
    showInfo,
    showWarning
  }
}