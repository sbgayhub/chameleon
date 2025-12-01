<script setup lang="ts">
import {onMounted, ref} from 'vue'
import {
  GetConfig,
  UpdateGeneralConfig,
  UpdateLogConfig,
  UpdateProxyConfig,
  UpdateUIConfig
} from '../../../wailsjs/go/config/Manager'
import {Uninstall} from "../../../wailsjs/go/certificate/CertManager"
import {config} from '../../../wailsjs/go/models'
import {BrowserOpenURL} from '../../../wailsjs/runtime/runtime'
import ConfirmDialog from '../common/ConfirmDialog.vue'

const configData = ref<config.Config | null>(null)
const loading = ref(false)
const error = ref('')
const success = ref('')
const confirmDialog = ref<InstanceType<typeof ConfirmDialog> | null>(null)

// 加载配置
const loadConfig = async () => {
  loading.value = true
  error.value = ''
  try {
    configData.value = await GetConfig()
  } catch (err) {
    error.value = `加载配置失败: ${err}`
  } finally {
    loading.value = false
  }
}

// 保存配置
const saveConfig = async () => {
  if (!configData.value) return

  loading.value = true
  error.value = ''
  success.value = ''

  try {
    await UpdateProxyConfig(configData.value.Proxy!)
    await UpdateGeneralConfig(configData.value.General!)
    await UpdateUIConfig(configData.value.UI!)
    await UpdateLogConfig(configData.value.Log!)
    success.value = '配置保存成功'
    setTimeout(() => success.value = '', 3000)
  } catch (err) {
    error.value = `保存配置失败: ${err}`
  } finally {
    loading.value = false
  }
}

// 卸载证书
const uninstallCert = () => {
  confirmDialog.value?.open()
}

const handleUninstallConfirm = async () => {
  if (!configData.value) return
  // todo 处理证书卸载逻辑
  await Uninstall().then(res => {
    if (res) {
      success.value = "证书卸载成功"
      configData.value!.Proxy!.CertInstalled = false
      saveConfig()
    } else {
      error.value = "证书卸载失败"
      return
    }
  })
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
}

// 清理 Host 文件（待实现）
const cleanHost = () => {
  alert('此功能正在开发中')
}

onMounted(() => {
  loadConfig()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 标题 -->
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold">设置</h2>
      <button @click="saveConfig" class="btn btn-primary gap-2" :disabled="loading">
        <svg v-if="!loading" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
             stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
        </svg>
        <span v-if="loading" class="loading loading-spinner loading-sm"></span>
        保存配置
      </button>
    </div>

    <!-- 成功提示 -->
    <div v-if="success" class="alert alert-success mb-4">
      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      <span>{{ success }}</span>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="alert alert-error mb-4">
      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      <span>{{ error }}</span>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading && !configData" class="flex justify-center items-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 配置表单 -->
    <div v-else-if="configData" class="flex-1 overflow-auto space-y-6 pb-6">
      <!-- 代理配置 -->
      <section class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                 stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
            </svg>
            代理配置
          </h3>

          <div class="space-y-4">
            <div class="form-control">
              <label class="label mr-2">
                <span class="label-text">代理模式</span>
              </label>
              <select v-model="configData.Proxy!.Mode" class="select select-sm select-bordered float-right w-40">
                <option value="http">HTTP 代理</option>
                <option value="host">HOST 劫持</option>
              </select>
            </div>

            <div v-if="configData.Proxy!.Mode === 'http'" class="form-control">
              <label class="label mr-2">
                <span class="label-text">HTTP 端口</span>
              </label>
              <input
                  v-model.number="configData.Proxy!.Port"
                  type="number"
                  min="1"
                  max="65535"
                  class="input input-sm float-right w-40"
                  placeholder="9527"
              />
            </div>

            <div class="flex gap-3 float-right">
              <button
                  @click="uninstallCert"
                  class="btn btn-outline btn-error gap-2"
                  :disabled="!configData.Proxy!.CertInstalled"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                     stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                卸载证书
              </button>

              <button
                  @click="cleanHost"
                  class="btn btn-outline gap-2"
                  disabled
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                     stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                清理 Host 文件（开发中）
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- 通用配置 -->
      <section class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                 stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
            通用配置
          </h3>

          <div class="space-y-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">开机自启</span>
              </label>
              <input
                  v-model="configData.General!.AutoStart"
                  type="checkbox"
                  class="toggle toggle-primary float-right"
              />
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text">最小化启动</span>
              </label>
              <input
                  v-model="configData.General!.StartMinimized"
                  type="checkbox"
                  class="toggle toggle-primary float-right"
              />
            </div>

<!--            <div class="form-control">-->
<!--              <label class="label">-->
<!--                <span class="label-text">以管理员权限启动</span>-->
<!--              </label>-->
<!--              <input-->
<!--                  v-model="configData.General!.StartAsAdmin"-->
<!--                  type="checkbox"-->
<!--                  class="toggle toggle-primary float-right"-->
<!--              />-->
<!--            </div>-->

            <div class="form-control">
              <label class="label">
                <span class="label-text">语言</span>
              </label>
              <select v-model="configData.UI!.Language" class="select select-sm select-bordered float-right w-40">
                <option value="zh-CN">简体中文</option>
                <option value="en-US">English</option>
              </select>
            </div>
          </div>
        </div>
      </section>

      <!-- 日志配置 -->
      <section class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                 stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            日志配置
          </h3>

          <div class="space-y-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">日志等级</span>
              </label>
              <select v-model="configData.Log!.Level" class="select select-sm select-bordered float-right w-40">
                <option value="debug">Debug</option>
                <option value="info">Info</option>
                <option value="warn">Warn</option>
                <option value="error">Error</option>
              </select>
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text">保存到文件</span>
              </label>
              <input
                  v-model="configData.Log!.File"
                  type="checkbox"
                  class="toggle toggle-primary float-right"
              />
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text">输出到控制台</span>
              </label>
              <input
                  v-model="configData.Log!.Console"
                  type="checkbox"
                  class="toggle toggle-primary float-right"
              />
            </div>
          </div>
        </div>
      </section>

      <!-- 关于 -->
      <section class="card bg-base-200 shadow-lg">
        <div class="card-body">
          <h3 class="card-title text-2xl mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                 stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            关于
          </h3>

          <div class="space-y-4">
            <div class="flex items-center justify-between gap-4">
              <div class="flex items-center gap-4">
                <div class="avatar">
                  <div class="w-16 rounded-lg">
                    <img src="../../assets/images/appicon.png" alt="Chameleon" />
                  </div>
                </div>
                <div>
                  <h4 class="text-xl font-bold">Chameleon</h4>
                  <p class="text-sm text-base-content/60">版本 v1.0.0</p>
                </div>
              </div>
              <button @click="BrowserOpenURL('https://github.com/sbgayhub')" class="btn btn-outline gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                  <path
                      d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
                GitHub
              </button>
            </div>

            <div class="divider my-2"></div>

            <div class="space-y-2 text-sm">
              <p class="text-base-content/80">
                LLM API 桌面代理工具，专注于 API 地址替换、参数修改、格式转换等功能。
              </p>
              <p class="text-base-content/60">
                © 2025 Chameleon. All rights reserved.
              </p>
            </div>

          </div>
        </div>
      </section>
    </div>

    <!-- 确认对话框 -->
    <ConfirmDialog
        ref="confirmDialog"
        title="卸载证书"
        message="确定要卸载证书吗？卸载后 Chameleon 将无法正常工作。"
        confirm-text="卸载"
        cancel-text="取消"
        @confirm="handleUninstallConfirm"
    />
  </div>
</template>
