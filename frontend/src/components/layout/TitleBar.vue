<script setup lang="ts">
import {ref} from 'vue'
import {useTheme} from '../../composables/useTheme'
import {WindowMinimise, WindowToggleMaximise} from '../../../wailsjs/runtime'
import CloseConfirmDialog from '../dialog/CloseConfirmDialog.vue'

// 定义事件
const emit = defineEmits<{
  minimize: []
  maximize: [isMaximized: boolean]
  close: []
  themeChange: []
}>()

// 使用主题管理
const {currentTheme, themes, changeTheme, loading} = useTheme()

// 状态
const isMaximized = ref(false)
const closeDialogRef = ref()

// 最小化窗口
const minimizeWindow = () => {
  WindowMinimise()
  emit('minimize')
}

// 最大化/还原窗口
const maximizeWindow = () => {
  WindowToggleMaximise()
  isMaximized.value = !isMaximized.value
  emit('maximize', isMaximized.value)
}

// 关闭窗口
const closeWindow = () => {
  closeDialogRef.value?.open()
  emit('close')
}

// 切换主题
const handleThemeChange = async (theme: string) => {
  const result = await changeTheme(theme)
  emit('themeChange')

  if (!result.success) {
    console.error('主题切换失败:', result.message)
  }
}
</script>

<template>
  <!-- 顶部标题栏 -->
  <header class="h-9 bg-base-300 flex items-center justify-between"
          style="--wails-draggable: drag; padding-left: 20px;">
    <h1 class="text-base font-bold" style="line-height: 24px">Chameleon</h1>

    <div class="flex items-center gap-1" style="--wails-draggable: no-drag">
      <!-- 主题切换按钮 -->
      <button
          onclick="theme_selector.showModal()"
          class="btn btn-ghost btn-circle"
          style="width: 36px; height: 36px; min-height: 36px"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"/>
        </svg>
      </button>

      <!-- 最小化按钮 -->
      <button
          class="btn btn-ghost btn-circle"
          @click="minimizeWindow"
          style="width: 36px; height: 36px; min-height: 36px"
      >
        <svg style="width: 15px; height: 15px" viewBox="0 0 15 15" fill="currentColor">
          <rect y="7" width="15" height="1.5"/>
        </svg>
      </button>

      <!-- 最大化/还原按钮 -->
      <button
          class="btn btn-ghost btn-circle"
          @click="maximizeWindow"
          style="width: 36px; height: 36px; min-height: 36px"
      >
        <svg v-if="!isMaximized" style="width: 15px; height: 15px" viewBox="0 0 15 15" fill="none"
             stroke="currentColor" stroke-width="1.2">
          <rect x="3" y="3" width="9" height="9"/>
        </svg>
        <svg v-else style="width: 15px; height: 15px" viewBox="0 0 15 15" fill="none" stroke="currentColor"
             stroke-width="1.2">
          <polyline points="5,2 12,2 12,9"/>
          <rect x="3" y="4" width="7" height="7"/>
        </svg>
      </button>

      <!-- 关闭按钮 -->
      <button
          class="btn btn-ghost btn-circle hover:btn-error"
          @click="closeWindow"
          style="width: 36px; height: 36px; min-height: 36px"
      >
        <svg style="width: 15px; height: 15px" viewBox="0 0 15 15" fill="none" stroke="currentColor"
             stroke-width="1.5">
          <path d="M2 2L13 13M13 2L2 13"/>
        </svg>
      </button>
    </div>
  </header>

  <!-- 主题选择弹窗 -->
  <dialog id="theme_selector" class="modal">
    <div class="modal-box max-w-3xl">
      <form method="dialog">
        <button class="btn btn-sm btn-circle absolute right-2 top-2">✕</button>
      </form>
      <h3 class="font-bold justify-self-center text-lg mb-6">选择主题</h3>

      <div class="grid grid-cols-4 gap-4 max-h-96 overflow-y-auto p-1">
        <div
            v-for="theme in themes"
            :key="theme"
            @click="handleThemeChange(theme)"
            class="outline-2 outline-offset-2 overflow-hidden rounded-lg text-left cursor-pointer"
            :class="{ 'outline-primary': currentTheme === theme, 'outline-transparent': currentTheme !== theme }"
            :data-theme="theme"
        >
          <div class="bg-base-100 text-base-content w-full font-sans">
            <div class="grid grid-cols-5 grid-rows-3">
              <div class="bg-base-200 col-start-1 row-span-2 row-start-1"/>
              <div class="bg-base-300 col-start-1 row-start-3"/>
              <div class="bg-base-100 col-span-4 col-start-2 row-span-3 row-start-1 flex flex-col gap-1 p-2">
                <div class="font-bold">{{ theme }}</div>
                <div class="flex flex-wrap gap-1">
                  <div class="bg-primary flex aspect-square w-5 items-center justify-center rounded lg:w-6">
                    <div class="text-primary-content text-sm font-bold">A</div>
                  </div>
                  <div class="bg-secondary flex aspect-square w-5 items-center justify-center rounded lg:w-6">
                    <div class="text-secondary-content text-sm font-bold">A</div>
                  </div>
                  <div class="bg-accent flex aspect-square w-5 items-center justify-center rounded lg:w-6">
                    <div class="text-accent-content text-sm font-bold">A</div>
                  </div>
                  <div class="bg-neutral flex aspect-square w-5 items-center justify-center rounded lg:w-6">
                    <div class="text-neutral-content text-sm font-bold">A</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>

  <!-- 关闭确认对话框 -->
  <CloseConfirmDialog ref="closeDialogRef" />
</template>
