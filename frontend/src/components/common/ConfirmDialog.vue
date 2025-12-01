<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  title: string
  message: string
  confirmText?: string
  cancelText?: string
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const show = ref(false)

const open = () => {
  show.value = true
}

const close = () => {
  show.value = false
}

const handleConfirm = () => {
  emit('confirm')
  close()
}

const handleCancel = () => {
  emit('cancel')
  close()
}

defineExpose({ open, close })
</script>

<template>
  <dialog :open="show" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">{{ title }}</h3>
        <span>{{ message }}</span>
      <div class="modal-action">
        <button @click="handleCancel" class="btn">{{ cancelText || '取消' }}</button>
        <button @click="handleConfirm" class="btn btn-primary">{{ confirmText || '确定' }}</button>
      </div>
    </div>
    <div class="modal-backdrop" @click="handleCancel"></div>
  </dialog>
</template>
