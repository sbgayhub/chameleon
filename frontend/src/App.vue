<script setup lang="ts">
import {onMounted, ref} from 'vue'
import AppLayout from './components/layout/AppLayout.vue'
import AppRouter from './components/layout/AppRouter.vue'
import {useChannelManager} from './composables/useChannelManager'
import {useProxyConfig} from './composables/useProxyConfig'
import {useToast} from './composables/useToast'

// 页面状态
const currentPage = ref('home')

// 使用渠道管理
const {channelStatus, startProxy, stopProxy, toggleProxy} = useChannelManager()

// 使用渠道配置管理
const {proxyConfig, installCertificate} = useProxyConfig()

// 使用 Toast 服务
const {toastRef, showSuccess, showError, showInfo} = useToast()

// 应用布局引用
const appLayoutRef = ref()

// 导航处理
const handleNavigate = (page: string) => {
  currentPage.value = page
}

// 代理切换处理
const handleToggleProxy = async () => {
  const result = await toggleProxy()

  // 显示 Toast 通知
  if (result.success) {
    if (channelStatus.value.isRunning) {
      showSuccess('代理启动成功', `代理服务已在端口 :${channelStatus.value.port} 启动`)
    } else {
      showInfo('代理已停止', '代理服务已安全停止')
    }
  } else {
    showError('代理操作失败', result.message)
  }
}

// 启动代理处理
const handleStartProxy = async () => {
  console.log('App: handleStartProxy called')
  const result = await startProxy()
  console.log('App: startProxy result:', result)

  if (result.success) {
    showSuccess('代理启动成功', `代理服务已在端口 :${channelStatus.value.port} 启动`)
  } else {
    showError('代理启动失败', result.message)
  }
}

// 停止代理处理
const handleStopProxy = async () => {
  const result = await stopProxy()

  if (result.success) {
    showInfo('代理已停止', '代理服务已安全停止')
  } else {
    showError('代理停止失败', result.message)
  }
}

// 证书安装处理
const handleInstallCertificate = async () => {
  const result = await installCertificate()

  if (result.success) {
    showSuccess('证书安装成功', result.message)
  } else {
    showError('证书安装失败', result.message)
  }
}

onMounted(() => {
  console.log('App mounted')
})
</script>

<template>
  <AppLayout
      ref="appLayoutRef"
      :current-page="currentPage"
      :proxy-status="channelStatus"
      @navigate="handleNavigate"
      @toggle-proxy="handleToggleProxy"
      @start-proxy="handleStartProxy"
      @stop-proxy="handleStopProxy"
  >
    <AppRouter
        :current-page="currentPage"
        :proxy-status="channelStatus"
        :proxy-config="proxyConfig"
        @start-proxy="handleStartProxy"
        @stop-proxy="handleStopProxy"
        @install-certificate="handleInstallCertificate"
    />
  </AppLayout>
</template>