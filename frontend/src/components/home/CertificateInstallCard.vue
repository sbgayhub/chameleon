<script setup lang="ts">
import {ref} from 'vue'

// 定义事件
const emit = defineEmits<{
  installCertificate: []
}>()

// 安装状态
const isInstalling = ref(false)

// 处理证书安装
const handleInstallCertificate = async () => {


  try {
    // await Install()
    emit('installCertificate')
  } finally {
    setTimeout(() => {
      // 触发 Alt+Tab 切换窗口的操作
      if (typeof window !== 'undefined' && window.KeyboardEvent) {
        const altTabEvent = new KeyboardEvent('keydown', {
          key: 'Tab',
          code: 'Tab',
          keyCode: 9,
          which: 9,
          altKey: true,
          bubbles: true,
          cancelable: true
        });
        document.dispatchEvent(altTabEvent);
      }
    }, 500)
    // // 给用户一些反馈，假设安装过程需要时间
    // setTimeout(() => {
    //   isInstalling.value = false
    // }, 1000)
  }
}
</script>

<template>
  <div class="card bg-warning/10 border-warning/20 shadow-xl">
    <div class="card-body">
      <div class="flex items-start gap-3 mb-4">
        <div class="w-8 h-8 rounded-full bg-warning/20 flex items-center justify-center flex-shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-warning" fill="none" viewBox="0 0 24 24"
               stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z"/>
          </svg>
        </div>
        <div class="flex-1">
          <h3 class="card-title text-lg mb-2">需要安装CA证书</h3>
          <p class="text-sm text-base-content/80 mb-1">
            Chameleon运行依赖CA证书，需要安装Chameleon的CA证书到系统信任存储。
          </p>
          <p class="text-sm text-base-content/70">
            这将允许Chameleon拦截和解密HTTPS流量，以便进行API转换和代理。
          </p>
        </div>
      </div>

      <!-- 证书说明 -->
      <div class="bg-base-200/50 rounded-lg p-3 mb-4">
        <h4 class="text-sm font-semibold mb-2 flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-info" fill="none" viewBox="0 0 24 24"
               stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          为何需要证书？
        </h4>
        <ul class="text-xs text-base-content/70 space-y-1">
          <li>• 代理需要证书来建立HTTPS连接</li>
          <li>• 证书用于加密客户端与Chameleon之间的通信</li>
          <li>• CA证书用于为https请求动态签发tls证书</li>
          <li>• 证书仅存储在本地，不会上传到任何服务器</li>
        </ul>
      </div>

      <!-- 操作按钮 -->
      <div class="card-actions justify-end gap-2">
        <button
            @click="handleInstallCertificate"
            :disabled="isInstalling"
            class="btn btn-warning btn-sm"
        >
          <span v-if="isInstalling" class="loading loading-spinner loading-xs mr-2"></span>
          <svg v-if="!isInstalling" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" fill="none"
               viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
          </svg>
          {{ isInstalling ? '安装中...' : '安装CA证书' }}
        </button>
      </div>
    </div>
  </div>
</template>