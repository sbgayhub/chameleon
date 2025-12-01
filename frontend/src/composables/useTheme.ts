import { ref, onMounted } from 'vue'

// 完整配置接口
interface AppConfig {
  General: {
    AutoStart: boolean
    StartMinimized: boolean
    StartAsAdmin: boolean
  }
  UI: {
    Language: string
    Theme: string
    Width: number
    Height: number
  }
  Proxy: {
    Mode: string
    Port: number
    CertInstalled: boolean
  }
  Log: {
    Level: string
    File: boolean
    Console: boolean
  }
}

// 主题管理 Hook
export function useTheme() {
  // 当前主题
  const currentTheme = ref('light')

  // 主题列表
  const themes = [
    'light', 'dark', 'cupcake', 'bumblebee', 'emerald', 'corporate', 'synthwave', 'retro',
    'cyberpunk', 'valentine', 'halloween', 'garden', 'forest', 'aqua', 'lofi', 'pastel',
    'fantasy', 'wireframe', 'black', 'luxury', 'dracula', 'cmyk', 'autumn', 'business',
    'acid', 'lemonade', 'night', 'coffee', 'winter', 'dim', 'nord', 'sunset'
  ]

  const loading = ref(false)

  // 从配置文件加载主题
  const loadThemeFromConfig = async (): Promise<string> => {
    try {
      loading.value = true
      console.log('useTheme: Loading config...')

      // 动态导入避免循环依赖
      const { GetConfig } = await import('../../wailsjs/go/config/Manager')
      const config = await GetConfig()
      console.log('useTheme: Got config:', config)

      const theme = config.UI?.Theme || 'light'
      console.log('useTheme: Found theme:', theme)

      return theme
    } catch (error) {
      console.error('useTheme: 加载配置失败:', error)
      return 'light'
    } finally {
      loading.value = false
    }
  }

  // 保存主题到配置文件
  const saveThemeToConfig = async (theme: string): Promise<{ success: boolean; message: string }> => {
    try {
      console.log('useTheme: Saving theme:', theme)

      // 动态导入避免循环依赖
      const { GetConfig, UpdateConfig } = await import('../../wailsjs/go/config/Manager')
      const config = await GetConfig()

      // 更新主题配置
      if (!config.UI) {
        config.UI = {
          Language: 'zh-CN',
          Theme: theme,
          Width: 1200,
          Height: 800
        }
      } else {
        config.UI.Theme = theme
      }

      // 保存配置
      await UpdateConfig(config)

      console.log('useTheme: Theme saved successfully:', theme)
      return { success: true, message: '主题保存成功' }
    } catch (error) {
      console.error('useTheme: 保存主题失败:', error)
      return { success: false, message: `保存主题失败: ${error}` }
    }
  }

  // 应用主题
  const applyTheme = (theme: string) => {
    currentTheme.value = theme
    document.documentElement.setAttribute('data-theme', theme)
    console.log('useTheme: Applied theme:', theme)
  }

  // 切换主题
  const changeTheme = async (theme: string) => {
    // 先应用主题到UI
    applyTheme(theme)

    // 然后保存到配置
    const result = await saveThemeToConfig(theme)
    return result
  }

  // 初始化主题
  const initializeTheme = async () => {
    console.log('useTheme: Initializing theme...')
    const savedTheme = await loadThemeFromConfig()
    applyTheme(savedTheme)
    return savedTheme
  }

  // 组件挂载时初始化主题
  onMounted(async () => {
    await initializeTheme()
  })

  return {
    currentTheme,
    themes,
    loading,
    loadThemeFromConfig,
    saveThemeToConfig,
    applyTheme,
    changeTheme,
    initializeTheme
  }
}