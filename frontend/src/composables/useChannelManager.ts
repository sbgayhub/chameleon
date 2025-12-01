import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { StartProxy, StopProxy, GetProxyStatus } from '../../wailsjs/go/application/App'

// 渠道状态接口
export interface ChannelState {
  isRunning: boolean
  startTime: Date | null
  uptime: string
  port: number
  activeConnections: number
  totalRequests: number
  mode: string
}

// 渠道管理 Hook
export function useChannelManager() {
  // 渠道状态
  const channelStatus = ref<ChannelState>({
    isRunning: false,
    startTime: null,
    uptime: '00:00:00',
    port: 8080,
    activeConnections: 0,
    totalRequests: 0,
    mode: 'http'
  })

  let statusUpdateInterval: number | null = null

  // 更新渠道状态
  const updateChannelStatus = async () => {
    try {
      // console.log('useChannelManager: Getting channel status...')
      const status = await GetProxyStatus()
      // console.log('useChannelManager: Got channel status:', status)

      channelStatus.value = {
        isRunning: status.isRunning,
        startTime: status.startTime ? new Date(status.startTime * 1000) : null,
        uptime: formatUptime(status.uptime),
        port: status.port,
        activeConnections: status.activeConnections,
        totalRequests: status.totalRequests,
        mode: status.mode
      }
    } catch (error) {
      console.error('useChannelManager: 获取渠道状态失败:', error)
    }
  }

  // 格式化运行时间
  const formatUptime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  // 启动代理
  const startProxy = async () => {
    try {
      console.log('useChannelManager: Starting proxy...')
      await StartProxy()
      console.log('useChannelManager: StartProxy returned successfully')

      // 更新状态
      await updateChannelStatus()

      return { success: true, message: '代理服务已启动' }
    } catch (error) {
      console.error('useChannelManager: 启动代理失败:', error)
      return { success: false, message: `启动代理失败: ${error}` }
    }
  }

  // 停止代理
  const stopProxy = async () => {
    try {
      await StopProxy()

      // 更新状态
      await updateChannelStatus()

      return { success: true, message: '代理服务已停止' }
    } catch (error) {
      console.error('停止代理失败:', error)
      return { success: false, message: `停止代理失败: ${error}` }
    }
  }

  // 切换代理状态
  const toggleProxy = async () => {
    if (channelStatus.value.isRunning) {
      return await stopProxy()
    } else {
      return await startProxy()
    }
  }

  // 开始状态监控
  const startStatusMonitoring = () => {
    // 立即更新一次状态
    updateChannelStatus()

    // 每3秒更新一次状态
    statusUpdateInterval = window.setInterval(updateChannelStatus, 3000)
  }

  // 停止状态监控
  const stopStatusMonitoring = () => {
    if (statusUpdateInterval) {
      clearInterval(statusUpdateInterval)
      statusUpdateInterval = null
    }
  }

  // 组件挂载时开始监控
  onMounted(() => {
    startStatusMonitoring()
  })

  // 组件卸载时停止监控
  onUnmounted(() => {
    stopStatusMonitoring()
  })

  return {
    channelStatus,
    startProxy,
    stopProxy,
    toggleProxy,
    updateChannelStatus
  }
}