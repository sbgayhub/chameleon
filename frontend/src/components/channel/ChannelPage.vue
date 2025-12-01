<style scoped>
/* 隐藏滚动条但保持滚动功能 */
.scrollbar-hide {
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
}

.scrollbar-hide::-webkit-scrollbar {
  display: none; /* Chrome, Safari and Opera */
}

.scrollbar-hide::-webkit-scrollbar-track {
  background: transparent;
}

.scrollbar-hide::-webkit-scrollbar-thumb {
  background: transparent;
}

/* 修复 tooltip 被父元素遮盖的问题 */
.tooltip {
  z-index: 1000 !important;
}

.tooltip::before {
  z-index: 1000 !important;
  white-space: pre-line;
  max-width: 300px;
}

.tooltip::after {
  z-index: 1000 !important;
}

/* 确保渠道组和渠道item的 overflow 不会遮挡 tooltip */
.card {
  overflow: visible;
}

/* 为操作按钮设置适中的 z-index，确保在普通内容之上但在模态框之下 */
.btn-circle {
  position: relative;
  z-index: 5;
}

/* 确保模态框显示在所有内容之上 */
.modal {
  z-index: 9999 !important;
}

.modal-box {
  z-index: 9999 !important;
}

.modal-backdrop {
  z-index: 9998 !important;
}

/* 确保父容器不会裁剪 tooltip */
.space-y-4 {
  overflow: visible;
}

.space-y-2 {
  overflow: visible;
}

/* 确保滚动容器不会遮挡 tooltip */
.overflow-auto {
  overflow: visible !important;
}

/* 为 tooltip 添加全局样式支持 */
:global(.tooltip) {
  z-index: 9999 !important;
}

:global(.tooltip::before) {
  z-index: 9999 !important;
}

:global(.tooltip::after) {
  z-index: 9999 !important;
}

/* 拖拽样式 */
.dragging {
  opacity: 0.5;
}

.drag-over {
  border-width: 2px;
  border-style: dashed;
}

</style>

<script setup lang="ts">
import {onMounted, ref} from 'vue'
import {
  AddChannel,
  AddGroup,
  DeleteChannel,
  DeleteGroup,
  List,
  SaveToFile,
  UpdateChannel,
  UpdateGroup,
  UpdateGroupPriority,
  UpdateChannelPriority,
  TestChannel,
  FetchModels
} from '../../../wailsjs/go/channel/Manager'
import {GetAllStatistics} from '../../../wailsjs/go/statistics/Manager'
import {channel} from '../../../wailsjs/go/models'
import ConfirmDialog from '../common/ConfirmDialog.vue'

const channelGroups = ref<channel.Group[]>([])
const loading = ref(false)
const error = ref('')
const channelStats = ref<Record<string, any>>({})

const showGroupModal = ref(false)
const showChannelModal = ref(false)
const editingGroup = ref<channel.Group | null>(null)
const editingChannel = ref<channel.Channel | null>(null)
const selectedGroupIndex = ref<number | null>(null)
const activeGroupIndex = ref<number | null>(null)
const isEditingChannel = ref(false) // 跟踪当前是否为编辑模式
const showApiKey = ref(false) // 控制API Key的显示/隐藏
const channelModels = ref<string[]>([]) // 渠道模型列表
const fetchingModels = ref(false) // 获取模型列表状态
const testModel = ref('') // 测试用的模型
const showTestResultModal = ref(false) // 测试结果模态框
const testResult = ref('') // 测试结果
const testSuccess = ref(false) // 测试是否成功

// 拖拽状态
const draggedGroupIndex = ref<number | null>(null)
const draggedChannelName = ref<string | null>(null)

// 确认对话框
const confirmDialog = ref<InstanceType<typeof ConfirmDialog> | null>(null)
const confirmAction = ref<(() => void) | null>(null)

// 供应商选项
const providers = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' }
]

// 获取排序后的渠道列表
const getSortedChannels = (channels: Record<string, channel.Channel> | undefined) => {
  if (!channels) return []
  return Object.entries(channels).sort(([, a], [, b]) => (a.priority || 0) - (b.priority || 0))
}

// 加载统计数据
const loadStatistics = async () => {
  try {
    const stats = await GetAllStatistics()
    channelStats.value = stats || {}
  } catch (err) {
    console.error('加载统计数据失败:', err)
  }
}

// 获取渠道统计信息
const getChannelStats = (channelName: string) => {
  return channelStats.value[channelName]
}

// 格式化统计信息为tooltip文本
const formatStatsTooltip = (channelName: string) => {
  const stats = getChannelStats(channelName)
  if (!stats) return '📊 暂无统计数据'

  const successRate = stats.request_count > 0
    ? ((stats.success_count / stats.request_count) * 100).toFixed(1)
    : '0.0'

  return `📨 请求: ${stats.request_count} | ✅ 成功: ${stats.success_count} | ❌ 失败: ${stats.failure_count}
📈 成功率: ${successRate}% | 📥 输入: ${stats.input_token} | 📤 输出: ${stats.output_token}`
}

// 数据加载函数
const loadChannelGroups = async () => {
  loading.value = true
  error.value = ''
  try {
    const groups = await List()
    // 处理null值情况
    if (!groups || groups.length === 0) {
      channelGroups.value = []
      return
    }

    // 前端再次按优先级排序
    channelGroups.value = groups.sort((a, b) => (a.priority || 0) - (b.priority || 0))

    // 加载统计数据
    await loadStatistics()
  } catch (err) {
    error.value = `加载渠道组失败: ${err}`
    console.error('加载渠道组失败:', err)
    channelGroups.value = [] // 确保出错时设置为空数组
  } finally {
    loading.value = false
  }
}

const openGroupModal = (group?: channel.Group, index?: number) => {
  if (group && index !== undefined) {
    editingGroup.value = channel.Group.createFrom(group)
    selectedGroupIndex.value = index
  } else {
    // 计算新渠道组的优先级（现有最高优先级+1）
    const maxPriority = channelGroups.value.length > 0 ? Math.max(...channelGroups.value.map(g => g.priority || 1)) + 1 : 1

    editingGroup.value = channel.Group.createFrom({
      endpoint: '',
      enabled: true,
      channels: {},
      priority: maxPriority,
      provider: 'openai' // 默认供应商
    })
    selectedGroupIndex.value = null
  }
  showGroupModal.value = true
}

const saveGroup = async () => {
  if (!editingGroup.value) return

  loading.value = true
  error.value = ''

  try {
    // 直接使用编辑中的组数据
    let groupData = channel.Group.createFrom(editingGroup.value)

    // 计算新渠道组的优先级（现有最高优先级+1）
    groupData.priority = selectedGroupIndex.value !== null ?
        channelGroups.value[selectedGroupIndex.value].priority || 1 :
        (channelGroups.value.length > 0 ? Math.max(...channelGroups.value.map(g => g.priority || 1)) + 1 : 1)

    if (selectedGroupIndex.value !== null) {
      // 更新现有渠道组
      await UpdateGroup(groupData)
    } else {
      // 添加新渠道组
      await AddGroup(groupData)
    }

    // 保存到文件
    await SaveToFile()

    showGroupModal.value = false
    await loadChannelGroups() // 重新加载数据
  } catch (err) {
    error.value = `保存渠道组失败: ${err}`
    console.error('保存渠道组失败:', err)
  } finally {
    loading.value = false
  }
}

const deleteGroup = (index: number) => {
  confirmAction.value = async () => {
    loading.value = true
    error.value = ''

    try {
      const group = channelGroups.value[index]
      await DeleteGroup(group.endpoint || '')
      await SaveToFile()
      await loadChannelGroups()
    } catch (err) {
      error.value = `删除渠道组失败: ${err}`
      console.error('删除渠道组失败:', err)
    } finally {
      loading.value = false
    }
  }
  confirmDialog.value?.open()
}

const openChannelModal = (groupIndex: number, ch?: channel.Channel, channelName?: string) => {
  selectedGroupIndex.value = groupIndex

  if (ch && channelName !== undefined) {
    editingChannel.value = channel.Channel.createFrom(ch)
    isEditingChannel.value = true // 设置为编辑模式
  } else {
    // 计算该渠道组中现有渠道的最高优先级
    const group = channelGroups.value[groupIndex]
    const existingChannels = Object.values(group?.channels || {})
    const maxPriority = existingChannels.length > 0 ?
        Math.max(...existingChannels.map(p => p.priority || 1)) + 1 : 1

    editingChannel.value = channel.Channel.createFrom({
      name: '',
      url: '',
      api_key: '',
      enabled: true,
      priority: maxPriority,
      model_mapping: {},
      provider: group ? group.provider : 'openai'
    })
    isEditingChannel.value = false // 设置为新建模式
  }

  showApiKey.value = false // 重置API Key显示状态
  showChannelModal.value = true
}

// 关闭渠道模态框
const closeChannelModal = () => {
  showChannelModal.value = false
  showApiKey.value = false // 重置API Key显示状态
  editingChannel.value = null
  isEditingChannel.value = false
  channelModels.value = []
  testModel.value = ''
}

// 获取模型列表
const fetchModels = async () => {
  if (!editingChannel.value || selectedGroupIndex.value === null) return

  fetchingModels.value = true
  try {
    const group = channelGroups.value[selectedGroupIndex.value]
    channelModels.value = await FetchModels(group.endpoint || '', editingChannel.value.name || '')
  } catch (err) {
    error.value = `获取模型列表失败: ${err}`
  } finally {
    fetchingModels.value = false
  }
}

// 添加模型映射
const addModelMapping = () => {
  if (editingChannel.value) {
    if (!editingChannel.value.model_mapping) {
      editingChannel.value.model_mapping = {}
    }
    editingChannel.value.model_mapping[''] = ''
  }
}

// 删除模型映射
const removeModelMapping = (pattern: string) => {
  if (editingChannel.value && editingChannel.value.model_mapping) {
    delete editingChannel.value.model_mapping[pattern]
  }
}

// 更新模型映射
const updateModelMapping = (oldPattern: string, newPattern: string, target?: string) => {
  if (editingChannel.value && editingChannel.value.model_mapping) {
    const current = editingChannel.value.model_mapping[oldPattern]
    if (target !== undefined) {
      // 更新target值
      delete editingChannel.value.model_mapping[oldPattern]
      editingChannel.value.model_mapping[newPattern] = target
    } else {
      // 更新pattern值
      editingChannel.value.model_mapping[newPattern] = current
      if (newPattern !== oldPattern) {
        delete editingChannel.value.model_mapping[oldPattern]
      }
    }
  }
}

// 添加预设映射
const addPresetMapping = (pattern: string, target: string) => {
  if (editingChannel.value) {
    if (!editingChannel.value.model_mapping) {
      editingChannel.value.model_mapping = {}
    }
    // 检查是否已存在相同的映射
    if (!(pattern in editingChannel.value.model_mapping)) {
      editingChannel.value.model_mapping[pattern] = target
    }
  }
}

const saveChannel = async () => {
  if (!editingChannel.value || selectedGroupIndex.value === null) return

  loading.value = true
  error.value = ''

  try {
    const group = channelGroups.value[selectedGroupIndex.value]

    // 直接使用model_mapping，无需转换
    const modelMapping = editingChannel.value.model_mapping || {}

    const channelData = channel.Channel.createFrom({
      name: editingChannel.value.name,
      enabled: editingChannel.value.enabled,
      priority: editingChannel.value.priority,
      url: editingChannel.value.url,
      api_key: editingChannel.value.api_key,
      model_mapping: modelMapping,
      status: 0,
      provider: editingChannel.value.provider || 'openai'
    })

    // 检查是编辑现有渠道还是添加新渠道
    const existingChannelNames = Object.keys(group.channels || {})
    const channelExists = existingChannelNames.includes(editingChannel.value!.name || '')

    if (channelExists) {
      // 更新现有渠道
      await UpdateChannel(group.endpoint || '', channelData)
    } else {
      // 添加新渠道
      await AddChannel(group.endpoint || '', channelData)
    }

    closeChannelModal()
    await loadChannelGroups() // 重新加载数据
  } catch (err) {
    error.value = `保存渠道失败: ${err}`
    console.error('保存渠道失败:', err)
  } finally {
    loading.value = false
  }
}

const deleteChannel = (groupIndex: number, channelName: string) => {
  confirmAction.value = async () => {
    loading.value = true
    error.value = ''

    try {
      const group = channelGroups.value[groupIndex]
      await DeleteChannel(group.endpoint || '', channelName)
      await SaveToFile()
      await loadChannelGroups()
    } catch (err) {
      error.value = `删除渠道失败: ${err}`
      console.error('删除渠道失败:', err)
    } finally {
      loading.value = false
    }
  }
  confirmDialog.value?.open()
}

// 测试渠道
const testChannel = async (groupIndex: number, channelName: string) => {
  const ch = channelGroups.value[groupIndex].channels?.[channelName]
  if (!ch) return
  ch.status = 4 // 临时测试状态

  try {
    const group = channelGroups.value[groupIndex]
    const result = await TestChannel(group.endpoint || '', channelName)
    testResult.value = result
    testSuccess.value = true
    ch.status = 1 // 测试成功
  } catch (err) {
    testResult.value = String(err)
    testSuccess.value = false
    ch.status = 3 // 测试失败
  } finally {
    showTestResultModal.value = true
  }
}

// 设置活跃渠道组
const setActiveGroup = (index: number) => {
  activeGroupIndex.value = index
}

// 获取渠道组样式
const getGroupCardClass = (index: number) => {
  const baseClass = "card border-2 p-3 cursor-move hover:shadow-md transition-all"
  const activeClass = activeGroupIndex.value === index ? " border-primary bg-primary/5" : " bg-base-200 border-primary/10"
  return baseClass + activeClass
}


// 切换渠道组启用状态
const toggleGroupStatus = async (groupIndex: number) => {
  try {
    const group = channelGroups.value[groupIndex]
    if (!group) return

    // 创建新的组数据
    const groupData = channel.Group.createFrom({
      endpoint: group.endpoint,
      enabled: group.enabled,
      priority: group.priority,
      lb_strategy: group.lb_strategy || 0,
      channels: {},
      provider: group.provider || 'openai'
    })

    // 转换渠道数据
    const channelsMap: Record<string, channel.Channel> = {}
    if (group.channels) {
      Object.entries(group.channels).forEach(([channelName, p]) => {
        const modelMapping: Record<string, string> = {}
        if (p.model_mapping) {
          Object.entries(p.model_mapping).forEach(([pattern, target]) => {
            if (pattern && target) {
              modelMapping[pattern] = target
            }
          })
        }

        const channelData = channel.Channel.createFrom({
          name: p.name,
          enabled: p.enabled,
          priority: p.priority,
          url: p.url,
          api_key: p.api_key,
          model_mapping: modelMapping,
          status: p.status ?? 0,
          provider: p.provider || 'openai'
        })
        channelsMap[channelName] = channelData
      })
    }
    groupData.channels = channelsMap

    // 更新渠道组
    await UpdateGroup(groupData)
  } catch (err) {
    console.error('切换渠道组状态失败:', err)
    error.value = '切换渠道组状态失败，请重试'
    // 回滚状态
    if (channelGroups.value[groupIndex]) {
      channelGroups.value[groupIndex].enabled = !channelGroups.value[groupIndex].enabled
    }
  }
}

// 切换渠道启用状态
const toggleChannelStatus = async (groupIndex: number, channelName: string) => {
  try {
    const group = channelGroups.value[groupIndex]
    if (!group || !group.channels || !group.channels[channelName]) return

    const channelData = group.channels[channelName]

    // 创建新的渠道数据
    const newChannelData = channel.Channel.createFrom({
      name: channelData.name,
      enabled: channelData.enabled,
      priority: channelData.priority,
      url: channelData.url,
      api_key: channelData.api_key,
      model_mapping: channelData.model_mapping || {},
      status: channelData.status ?? 0,
      provider: channelData.provider || 'openai'
    })

    // 更新渠道
    await UpdateChannel(group.endpoint || '', newChannelData)

    // 持久化配置
    await SaveToFile()
  } catch (err) {
    console.error('切换渠道状态失败:', err)
    error.value = '切换渠道状态失败，请重试'
    // 回滚状态
    if (channelGroups.value[groupIndex]?.channels?.[channelName]) {
      channelGroups.value[groupIndex].channels[channelName].enabled = !channelGroups.value[groupIndex].channels[channelName].enabled
    }
  }
}

// 获取渠道状态样式
const getChannelStatusClass = (status?: number, enabled?: boolean) => {
  // 首先检查是否禁用
  if (enabled === false) {
    return 'bg-base-300 opacity-60 border-base-300'
  }

  // 根据状态返回颜色
  switch (status) {
    case 1: // STATUS_NORMAL - 正常/可用（绿色）
      return 'bg-success/20 border-success/50'
    case 2: // STATUS_ERROR - 异常（黄色）
      return 'bg-warning/20 border-warning/50'
    case 3: // STATUS_NOT_AVAILABLE - 不可用（红色）
      return 'bg-error/20 border-error/50'
    case 4: // 测试中（黄色+动画）
      return 'bg-warning/20 border-warning/50 animate-pulse'
    default:
      return 'bg-base-200 border-base-300'
  }
}

// 渠道组拖拽处理
const onGroupDragStart = (e: DragEvent, index: number) => {
  draggedGroupIndex.value = index
  const target = e.target as HTMLElement
  target.classList.add('dragging')
}

const onGroupDragEnd = (e: DragEvent) => {
  const target = e.target as HTMLElement
  target.classList.remove('dragging')
}

const onGroupDragOver = (e: DragEvent) => {
  e.preventDefault()
  const target = e.currentTarget as HTMLElement
  target.classList.add('drag-over')
}

const onGroupDragLeave = (e: DragEvent) => {
  const target = e.currentTarget as HTMLElement
  target.classList.remove('drag-over')
}

const onGroupDrop = async (e: DragEvent, targetIndex: number) => {
  e.preventDefault()
  const target = e.currentTarget as HTMLElement
  target.classList.remove('drag-over')

  if (draggedGroupIndex.value === null || draggedGroupIndex.value === targetIndex) {
    draggedGroupIndex.value = null
    return
  }

  const sourceIndex = draggedGroupIndex.value
  const groups = [...channelGroups.value]

  // 交换位置
  const [movedGroup] = groups.splice(sourceIndex, 1)
  groups.splice(targetIndex, 0, movedGroup)

  // 重新分配优先级并更新后端
  for (let i = 0; i < groups.length; i++) {
    groups[i].priority = i + 1
    await UpdateGroupPriority(groups[i].endpoint || '', groups[i].priority || 1)
  }

  // 更新本地状态
  channelGroups.value = groups

  await SaveToFile()

  draggedGroupIndex.value = null
}

// 渠道拖拽处理
const onChannelDragStart = (e: DragEvent, channelName: string) => {
  draggedChannelName.value = channelName
  const target = e.target as HTMLElement
  target.classList.add('dragging')
}

const onChannelDragEnd = (e: DragEvent) => {
  const target = e.target as HTMLElement
  target.classList.remove('dragging')
}

const onChannelDragOver = (e: DragEvent) => {
  e.preventDefault()
  const target = e.currentTarget as HTMLElement
  target.classList.add('drag-over')
}

const onChannelDragLeave = (e: DragEvent) => {
  const target = e.currentTarget as HTMLElement
  target.classList.remove('drag-over')
}

const onChannelDrop = async (e: DragEvent, targetChannelName: string) => {
  e.preventDefault()
  const target = e.currentTarget as HTMLElement
  target.classList.remove('drag-over')

  if (draggedChannelName.value === null || draggedChannelName.value === targetChannelName || activeGroupIndex.value === null) {
    draggedChannelName.value = null
    return
  }

  const group = channelGroups.value[activeGroupIndex.value]
  // 使用排序后的渠道列表（与显示顺序一致）
  const channelEntries = getSortedChannels(group.channels)

  const sourceIdx = channelEntries.findIndex(([name]) => name === draggedChannelName.value)
  const targetIdx = channelEntries.findIndex(([name]) => name === targetChannelName)

  if (sourceIdx === -1 || targetIdx === -1) {
    draggedChannelName.value = null
    return
  }

  // 交换位置
  const [movedEntry] = channelEntries.splice(sourceIdx, 1)
  channelEntries.splice(targetIdx, 0, movedEntry)

  // 重新分配优先级并更新后端
  const newChannels: Record<string, channel.Channel> = {}
  for (let i = 0; i < channelEntries.length; i++) {
    const [name, ch] = channelEntries[i]
    ch.priority = i + 1
    newChannels[name] = ch
    await UpdateChannelPriority(group.endpoint || '', name, ch.priority || 1)
  }

  // 更新本地状态
  group.channels = newChannels

  await SaveToFile()

  draggedChannelName.value = null
}

// 组件挂载时加载数据
onMounted(() => {
  loadChannelGroups()
  setActiveGroup(0)
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 顶部工具栏 -->
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold">渠道管理</h2>
      <button @click="openGroupModal()" class="btn btn-primary">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 mr-2" fill="none" viewBox="0 0 24 24"
             stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
        </svg>
        新建渠道组
      </button>
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
    <div v-if="loading" class="flex justify-center items-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 两列布局 -->
    <div v-else class="flex-1 grid grid-cols-2 gap-6 min-h-0 ">
      <!-- 渠道组列 -->
      <div class="space-y-4">
        <h3 class="text-xl font-semibold mb-4">渠道组</h3>
        <div class="space-y-4 overflow-auto max-h-[calc(100vh-200px)] scrollbar-hide">
          <div class="space-y-2 ">
            <div
                v-for="(group, groupIndex) in channelGroups"
                :key="groupIndex"
                :class="getGroupCardClass(groupIndex)"
                @click="setActiveGroup(groupIndex)"
                draggable="true"
                @dragstart="onGroupDragStart($event, groupIndex)"
                @dragend="onGroupDragEnd"
                @dragover="onGroupDragOver"
                @dragleave="onGroupDragLeave"
                @drop="onGroupDrop($event, groupIndex)"
            >
              <!-- 可拖拽的标题栏 -->
              <div class="flex justify-between items-center h-full min-w-0">

                <div class="flex items-center gap-3 min-w-0 flex-1">
                  <div class="min-w-0 flex-1">
                    <div class="font-semibold truncate">{{ group.endpoint }}</div>
                    <div class="text-sm text-base-content/70">
                      {{ providers.find(p => p.value === group.provider)?.label || group.provider }}
                       •
                      {{
                        group.lb_strategy === 1 ? '优先级' :
                            group.lb_strategy === 2 ? '轮询' :
                                group.lb_strategy === 3 ? '加权轮询' : '随机'
                      }}
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-2 flex-shrink-0 ml-2">
                  <!-- 启用/禁用开关 -->
                  <label class="cursor-pointer">
                    <input type="checkbox" v-model="group.enabled" @change="toggleGroupStatus(groupIndex)" class="toggle toggle-primary"/>
                  </label>
                  <!-- 操作按钮 -->
                  <div class="tooltip tooltip-bottom" data-tip="创建渠道">
                    <button @click="openChannelModal(groupIndex)" class="btn btn-circle btn-sm btn-primary">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                           stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                      </svg>
                    </button>
                  </div>
                  <div class="tooltip tooltip-bottom" data-tip="编辑渠道组">
                    <button @click="openGroupModal(group, groupIndex)" class="btn btn-circle btn-sm btn-ghost">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                           stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                      </svg>
                    </button>
                  </div>
                  <div class="tooltip tooltip-bottom" data-tip="删除渠道组">
                    <button @click="deleteGroup(groupIndex)" class="btn btn-circle btn-sm btn-ghost text-error">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                           stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="channelGroups.length === 0" class="text-center py-12 text-base-content/50">
            暂无渠道组，点击上方按钮创建
          </div>
        </div>
      </div>

      <!-- 渠道列表列 -->
      <div class="space-y-4">
        <h3 class="text-xl font-semibold mb-4">
            渠道列表
            <span v-if="activeGroupIndex !== null && channelGroups[activeGroupIndex]" class="text-sm text-base-content/70 ml-2">
              ({{
                channelGroups[activeGroupIndex]?.endpoint || ''
              }} - {{ Object.keys(channelGroups[activeGroupIndex]?.channels || {}).length || 0 }} 个渠道)
            </span>
          </h3>
        <div class="space-y-4 overflow-auto max-h-[calc(100vh-200px)] scrollbar-hide">
          <!-- 只显示选中渠道组的渠道 -->
          <div v-if="activeGroupIndex !== null && channelGroups[activeGroupIndex]" class="space-y-2">
            <div class="space-y-2">
              <div
                  v-for="[channelName, channel] in getSortedChannels(channelGroups[activeGroupIndex].channels)"
                  :key="channelName"
                  class="card border-2 p-3 hover:shadow-md transition-all cursor-move tooltip tooltip-left"
                  :class="getChannelStatusClass(channel.status, channel.enabled)"
                  :data-tip="formatStatsTooltip(channelName as string)"
                  draggable="true"
                  @dragstart="onChannelDragStart($event, channelName as string)"
                  @dragend="onChannelDragEnd"
                  @dragover="onChannelDragOver"
                  @dragleave="onChannelDragLeave"
                  @drop="onChannelDrop($event, channelName as string)"
              >
                <div class="flex justify-between items-center min-w-0">
                  <div class="flex items-center gap-3 min-w-0 flex-1">
                    <div class="min-w-0 flex-1">
                      <div class="font-semibold truncate">{{ providers.find(p => p.value === channel.provider)?.label || channel.provider }} • {{ channel.name }}</div>
                      <div class="text-sm truncate text-base-content/70">
                        {{ channel.url }}
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 flex-shrink-0 ml-2">
                    <!-- 启用/禁用开关 -->
                    <label class="cursor-pointer">
                      <input type="checkbox" v-model="channel.enabled" :disabled="channel.status === 3"
                             @change="toggleChannelStatus(activeGroupIndex, channelName)" class="toggle toggle-secondary"/>
                    </label>
                    <!-- 操作按钮 -->
                    <div class="tooltip tooltip-bottom" data-tip="测试渠道">
                      <button @click="testChannel(activeGroupIndex, channelName)" class="btn btn-circle btn-sm btn-ghost"
                              :disabled="channel.status === 4">
                        <svg v-if="channel.status !== 4" xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none"
                             viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <div v-else class="loading loading-spinner loading-xs"></div>
                      </button>
                    </div>
                    <div class="tooltip tooltip-bottom" data-tip="编辑渠道">
                      <button @click="openChannelModal(activeGroupIndex, channel, channelName)"
                              class="btn btn-circle btn-sm btn-ghost">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                             stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                        </svg>
                      </button>
                    </div>
                    <div class="tooltip tooltip-bottom" data-tip="删除渠道">
                      <button @click="deleteChannel(activeGroupIndex, channelName)"
                              class="btn btn-circle btn-sm btn-ghost text-error">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                             stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 空状态提示 -->
          <div v-else class="text-center py-12 text-base-content/50">
            <div v-if="activeGroupIndex === null">
              请先从左侧选择一个渠道组
            </div>
            <div v-else>
              该渠道组暂无渠道，点击上方按钮创建
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 渠道组编辑模态框 -->
    <dialog :open="showGroupModal" class="modal">
      <div class="modal-box w-96 max-w-lg">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-bold text-lg">{{ selectedGroupIndex !== null ? '编辑' : '新建' }}渠道组</h3>
          <label class="cursor-pointer" v-if="editingGroup">
            <input v-model="editingGroup.enabled" type="checkbox" class="toggle toggle-primary"/>
          </label>
        </div>

        <fieldset class="fieldset"  v-if="editingGroup">
          <legend class="fieldset-legend">端点地址</legend>
          <input v-model="editingGroup.endpoint" type="url" class="input input-bordered" placeholder="请输入要被代理的地址" list="endpoints"/>
          <datalist id="endpoints">
            <option value="api.openai.com"></option>
            <option value="api.anthropic.com"></option>
            <option value="api.openai.com"></option>
            <option value="api.openai.com"></option>
            <option value="api.openai.com"></option>
            <option value="api.openai.com"></option>
          </datalist>
        </fieldset>
        <fieldset class="fieldset"  v-if="editingGroup">
          <legend class="fieldset-legend">供应商类型</legend>
          <select v-model="editingGroup.provider" class="select select-bordered">
            <option v-for="provider in providers" :key="provider.value" :value="provider.value">
              {{ provider.label }}
            </option>
          </select>
        </fieldset>
        <fieldset class="fieldset"  v-if="editingGroup">
          <legend class="fieldset-legend">负载均衡策略</legend>
          <select v-model="editingGroup.lb_strategy" class="select select-bordered">
            <option :value="1">优先级</option>
            <option :value="2">轮询</option>
            <option :value="3">加权轮询</option>
            <option :value="4">随机</option>
          </select>
        </fieldset>


        <div class="modal-action">
          <button @click="showGroupModal = false" class="btn">取消</button>
          <button @click="saveGroup" class="btn btn-primary">保存</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showGroupModal = false"></div>
    </dialog>

    <!-- 渠道编辑模态框 -->
    <dialog :open="showChannelModal" class="modal">
      <div class="modal-box w-11/12 max-w-2xl">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-bold text-lg">{{ isEditingChannel ? '编辑' : '新建' }}渠道</h3>
          <div class="flex items-center gap-2">
            <button @click="fetchModels" class="btn btn-sm btn-outline" :disabled="fetchingModels || !editingChannel?.name">
              <span v-if="!fetchingModels">获取模型列表</span>
              <span v-else class="loading loading-spinner loading-xs"></span>
            </button>
            <label class="cursor-pointer" v-if="editingChannel">
              <input v-model="editingChannel.enabled" type="checkbox" class="toggle toggle-primary"/>
            </label>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6" v-if="editingChannel">
          <!-- 左列：基本信息 -->
          <fieldset class="fieldset">
            <legend class="fieldset-legend">基本信息</legend>

            <label class="fieldset-label">名称</label>
            <input v-model="editingChannel.name" type="text" class="input input-bordered" placeholder="渠道名称"/>

            <label class="fieldset-label">端点</label>
            <input v-model="editingChannel.url" type="text" class="input input-bordered"
                   placeholder="https://api.anthropic.com"/>

            <label class="fieldset-label">API Key</label>
            <label class="input">
              <input
                  v-model="editingChannel.api_key"
                  :type="showApiKey ? 'text' : 'password'"
                  placeholder="sk-..."
                  />
              <button @click="showApiKey = !showApiKey">
                <svg v-if="showApiKey" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
              </button>
            </label>

            <label class="fieldset-label">优先级</label>
            <input v-model.number="editingChannel.priority" type="number" class="input input-bordered" min="1" max="255"
                   placeholder="1-255"/>

            <label class="fieldset-label">供应商类型</label>
            <select v-model="editingChannel.provider" class="select select-bordered">
              <option v-for="provider in providers" :key="provider.value" :value="provider.value">
                {{ provider.label }}
              </option>
            </select>

            <label class="fieldset-label">测试模型</label>
            <select v-model="testModel" class="select select-bordered" :disabled="channelModels.length === 0">
              <option value="">{{ channelModels.length === 0 ? '请先获取模型列表' : '选择测试模型' }}</option>
              <option v-for="model in channelModels" :key="model" :value="model">{{ model }}</option>
            </select>
          </fieldset>

          <!-- 右列：模型映射 -->
          <fieldset class="fieldset">
            <legend class="fieldset-legend">模型映射</legend>

            <label class="fieldset-label">将请求的模型名称映射到目标服务的模型名称</label>

            <div class="space-y-2 max-h-64 overflow-y-auto">
              <div v-for="(mapping, index) in Object.entries(editingChannel.model_mapping || {})" :key="index" class="flex gap-2">
                <input :value="mapping[0]" @input="updateModelMapping(mapping[0], ($event.target as HTMLInputElement).value)" type="text" class="input-sm input outline-0 flex-1"
                       placeholder="匹配模式 (如: gpt-4)"/>
                <input v-if="channelModels.length === 0" :value="mapping[1]" @input="updateModelMapping(mapping[0], mapping[0], ($event.target as HTMLInputElement).value)" type="text" class="input-sm input outline-0 flex-1"
                       placeholder="目标模型 (如: claude-3-sonnet)"/>
                <select v-else :value="mapping[1]" @change="updateModelMapping(mapping[0], mapping[0], ($event.target as HTMLSelectElement).value)" class="select select-sm flex-1">
                  <option value="">选择目标模型</option>
                  <option v-for="model in channelModels" :key="model" :value="model">{{ model }}</option>
                </select>
                <button @click="removeModelMapping(mapping[0])" class="btn btn-sm btn-ghost btn-square">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                       stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                  </svg>
                </button>
              </div>
            </div>

            <button @click="addModelMapping" class="btn btn-sm btn-outline w-full mt-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24"
                   stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
              添加映射
            </button>

          </fieldset>
        </div>

        <div class="modal-action">
          <button @click="closeChannelModal" class="btn">取消</button>
          <button @click="saveChannel" class="btn btn-primary">保存</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="closeChannelModal"></div>
    </dialog>

    <!-- 确认对话框 -->
    <ConfirmDialog
      ref="confirmDialog"
      title="确认删除"
      message="确定要删除吗？此操作不可撤销。"
      confirm-text="删除"
      cancel-text="取消"
      @confirm="confirmAction?.()"
    />

    <!-- 测试结果模态框 -->
    <dialog :open="showTestResultModal" class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">
          {{ testSuccess ? '✅ 测试成功' : '❌ 测试失败' }}
        </h3>
        <div class="bg-base-200 p-4 rounded-lg max-h-96 overflow-auto">
          <pre class="text-sm whitespace-pre-wrap">{{ testResult }}</pre>
        </div>
        <div class="modal-action">
          <button @click="showTestResultModal = false" class="btn">关闭</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showTestResultModal = false"></div>
    </dialog>
  </div>
</template>
