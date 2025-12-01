<script setup lang="ts">
import { ref } from 'vue'
import HomePage from '../home/HomePage.vue'
import ChannelPage from '../channel/ChannelPage.vue'
import LogsPage from '../LogsPage.vue'
import SettingsPage from '../settings/SettingsPage.vue'
import StatsPage from '../stats/StatsPage.vue'

// 定义属性
const props = defineProps<{
  currentPage: string
  proxyStatus: any
  proxyConfig: any
}>()

// 定义事件
const emit = defineEmits<{
  startProxy: []
  stopProxy: []
  installCertificate: []
}>()

// 处理代理操作
const handleStartProxy = () => {
  emit('startProxy')
}

const handleStopProxy = () => {
  emit('stopProxy')
}

// 处理证书安装
const handleInstallCertificate = () => {
  emit('installCertificate')
}
</script>

<template>
  <main class="flex-1 overflow-auto p-8">
    <!-- 主页 -->
    <HomePage
      v-if="currentPage === 'home'"
      :proxyStatus="proxyStatus"
      :proxyConfig="proxyConfig"
      @start-proxy="handleStartProxy"
      @stop-proxy="handleStopProxy"
      @install-certificate="handleInstallCertificate"
    />

    <!-- 代理页面 -->
    <ChannelPage v-else-if="currentPage === 'channel'" />

    <!-- 日志页面 -->
    <LogsPage v-else-if="currentPage === 'logs'" />

    <!-- 统计页面 -->
    <StatsPage v-else-if="currentPage === 'stats'" />

    <!-- 设置页面 -->
    <SettingsPage v-else-if="currentPage === 'settings'" />
  </main>
</template>