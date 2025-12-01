<script setup lang="ts">
import { ref } from 'vue'
import { HandleWindowClose, MinimizeToTray, QuitApp, SaveCloseAction } from '../../../wailsjs/go/application/App'

const showDialog = ref(false)
const rememberChoice = ref(false)

// 打开对话框
const open = async () => {
  const action = await HandleWindowClose()

  if (action === 'minimize') {
    MinimizeToTray()
  } else if (action === 'exit') {
    QuitApp()
  } else {
    showDialog.value = true
  }
}

// 最小化到托盘
const handleMinimize = async () => {
  if (rememberChoice.value) {
    await SaveCloseAction('minimize')
  }
  showDialog.value = false
  MinimizeToTray()
}

// 退出应用
const handleExit = async () => {
  if (rememberChoice.value) {
    await SaveCloseAction('exit')
  }
  showDialog.value = false
  QuitApp()
}

// 取消
const handleCancel = () => {
  showDialog.value = false
  rememberChoice.value = false
}

// 暴露方法给父组件
defineExpose({
  open
})
</script>

<template>
  <dialog :open="showDialog" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">关闭应用</h3>
      <p class="py-4">您想要最小化到托盘还是退出应用？</p>

      <div class="form-control mb-4">
        <label class="label cursor-pointer justify-start gap-2">
          <input type="checkbox" v-model="rememberChoice" class="checkbox checkbox-sm" />
          <span class="label-text">记住我的选择</span>
        </label>
      </div>

      <div class="modal-action">
        <button @click="handleCancel" class="btn btn-ghost">取消</button>
        <button @click="handleMinimize" class="btn btn-primary">最小化到托盘</button>
        <button @click="handleExit" class="btn btn-error">退出应用</button>
      </div>
    </div>
  </dialog>
</template>
