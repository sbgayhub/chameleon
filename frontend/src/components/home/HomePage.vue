<script setup lang="ts">
import { ref } from 'vue'
import CertificateInstallCard from './CertificateInstallCard.vue'

import {useProxyConfig} from '../../composables/useProxyConfig'

// 代理状态接口
interface ProxyState {
  isRunning: boolean
  startTime: Date | null
  uptime: string
  port: number
  activeConnections: number
  totalRequests: number
}

// 代理配置接口
interface ProxyConfig {
  Mode: string
  Port: number
  CertInstalled: boolean
}

// 定义属性
const props = defineProps<{
  proxyStatus: ProxyState
  proxyConfig: ProxyConfig
}>()

// 定义事件
const emit = defineEmits<{
  startProxy: []
  stopProxy: []
  installCertificate: []
}>()

// 启动代理
const startProxy = () => {
  console.log('HomePage: startProxy clicked')
  emit('startProxy')
}

// 停止代理
const stopProxy = () => {
  console.log('HomePage: stopProxy clicked')
  emit('stopProxy')
}

// 处理证书安装
const handleInstallCertificate = () => {
  console.log('HomePage: installCertificate clicked')
  emit('installCertificate')
}
</script>

<template>
  <div class="max-w-4xl mx-auto">
    <h2 class="text-3xl font-bold mb-6">欢迎使用 Chameleon</h2>

    <!-- 条件显示：证书安装卡片 或 代理服务状态卡片 -->
    <CertificateInstallCard
      v-if="!proxyConfig.CertInstalled"
      @install-certificate="handleInstallCertificate"
    />

    <!-- 代理服务状态卡片 -->
    <div v-else class="card bg-base-200 shadow-xl">
      <div class="card-body">
        <h3 class="card-title">代理服务状态</h3>

        <!-- 状态指示器 -->
        <div class="flex items-center gap-2 mb-4">
          <div
            class="w-4 h-4 rounded-full transition-colors"
            :class="proxyStatus.isRunning ? 'bg-success' : 'bg-error'"
          ></div>
          <span
            class="text-lg transition-colors"
            :class="proxyStatus.isRunning ? 'text-success' : 'text-error'"
          >
            代理服务器{{ proxyStatus.isRunning ? '运行中' : '已停止' }}
          </span>
        </div>

        <!-- 统计信息 -->
        <div v-if="proxyStatus.isRunning" class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div class="stat">
            <div class="stat-title">运行时长</div>
            <div class="stat-value text-lg font-mono">{{ proxyStatus.uptime }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">监听端口</div>
            <div class="stat-value text-lg">:{{ proxyStatus.port }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">活跃连接</div>
            <div class="stat-value text-lg">{{ proxyStatus.activeConnections }}</div>
          </div>
        </div>

        <!-- 描述信息 -->
        <p v-else class="text-base-content/70 mb-4">
          点击下方按钮启动代理服务，开始使用 Chameleon 的代理功能。
        </p>

        <!-- 控制按钮 -->
        <div class="card-actions justify-end">
          <button
            v-if="!proxyStatus.isRunning"
            @click="emit('startProxy')"
            class="btn btn-primary"
          >
            启动服务
          </button>
          <button
            v-else
            @click="emit('stopProxy')"
            class="btn btn-error"
          >
            停止服务
          </button>
        </div>
      </div>
    </div>

    <!-- 快速操作卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
      <!-- 渠道配置 -->
      <div class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h4 class="card-title text-lg">渠道配置</h4>
          <p class="text-base-content/70 mb-4">配置渠道组和渠道服务器</p>
          <div class="card-actions">
            <button class="btn btn-soft btn-sm">前往配置</button>
          </div>
        </div>
      </div>

      <!-- 查看日志 -->
      <div class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h4 class="card-title text-lg">查看日志</h4>
          <p class="text-base-content/70 mb-4">实时查看系统运行日志</p>
          <div class="card-actions">
            <button class="btn btn-soft btn-sm">查看日志</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 使用说明 -->
    <div class="card bg-base-200 shadow-lg mt-6">
      <div class="card-body">
        <h4 class="card-title text-lg">快速开始</h4>
        <div class="space-y-2">
          <div class="flex items-start gap-2">
            <span class="text-primary font-bold">1.</span>
            <p class="text-base-content/80">点击"启动服务"按钮启动代理服务器</p>
          </div>
          <div class="flex items-start gap-2">
            <span class="text-primary font-bold">2.</span>
            <p class="text-base-content/80">前往"渠道"页配置渠道组和渠道服务器</p>
          </div>
          <div class="flex items-start gap-2">
            <span class="text-primary font-bold">3.</span>
            <p class="text-base-content/80">在客户端应用中配置代理为 localhost:8080</p>
          </div>
          <div class="flex items-start gap-2">
            <span class="text-primary font-bold">4.</span>
            <p class="text-base-content/80">在"统计"页面查看使用情况</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>