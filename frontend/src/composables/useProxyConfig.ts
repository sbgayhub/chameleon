import {onMounted, ref} from 'vue'
import {GetConfig, UpdateConfig} from '../../wailsjs/go/config/Manager'
import {Install} from '../../wailsjs/go/certificate/CertManager'

// 渠道配置接口
export interface ProxyConfig {
    Mode: string
    Port: number
    CertInstalled: boolean
}

// 渠道代理配置管理 Hook
export function useProxyConfig() {
    // 渠道配置状态
    const proxyConfig = ref<ProxyConfig>({
        Mode: 'http',
        Port: 9527,
        CertInstalled: false
    })

    const loading = ref(false)

    // 获取渠道配置
    const fetchProxyConfig = async () => {
        try {
            loading.value = true
            console.log('useProxyConfig: Getting full config...')
            const config = await GetConfig()
            console.log('useProxyConfig: Got full config:', config)

            proxyConfig.value = {
                Mode: config.Proxy?.Mode || 'http',
                Port: config.Proxy?.Port || 9527,
                CertInstalled: config.Proxy?.CertInstalled || false
            }
        } catch (error) {
            console.error('useProxyConfig: 获取代理配置失败:', error)
        } finally {
            loading.value = false
        }
    }

    // 安装CA证书
    const installCertificate = async (): Promise<{ success: boolean; message: string }> => {
        try {
            console.log('useProxyConfig: Installing certificate...')
            const success = await Install()
            console.log('useProxyConfig: Certificate install result:', success)

            if (success) {
                proxyConfig.value.CertInstalled = true
                await updateConfig(proxyConfig.value)

                location.reload()

                // // 安装成功后，重新获取配置
                // await fetchProxyConfig()
                return {success: true, message: 'CA证书安装成功'}
            } else {
                return {success: false, message: 'CA证书安装失败，请检查系统权限'}
            }
        } catch (error) {
            console.error('useProxyConfig: 安装证书失败:', error)
            return {success: false, message: `安装证书失败: ${error}`}
        }
    }

    // 更新代理配置
    const updateConfig = async (config: ProxyConfig): Promise<{ success: boolean; message: string }> => {
        try {
            console.log('useProxyConfig: Updating full config...')

            // 获取当前完整配置
            const fullConfig = await GetConfig()

            // 更新代理配置部分
            if (!fullConfig.Proxy) {
                fullConfig.Proxy = {
                    Mode: config.Mode,
                    Port: config.Port,
                    CertInstalled: config.CertInstalled
                }
            } else {
                fullConfig.Proxy.Mode = config.Mode
                fullConfig.Proxy.Port = config.Port
                fullConfig.Proxy.CertInstalled = config.CertInstalled
            }

            // 保存完整配置
            await UpdateConfig(fullConfig)

            // 更新本地状态
            proxyConfig.value = {...config}

            return {success: true, message: '配置更新成功'}
        } catch (error) {
            console.error('useProxyConfig: 更新配置失败:', error)
            return {success: false, message: `更新配置失败: ${error}`}
        }
    }

    // 组件挂载时获取配置
    onMounted(() => {
        fetchProxyConfig().then(() => {})
    })

    return {
        proxyConfig,
        loading,
        fetchProxyConfig,
        installCertificate,
        updateConfig
    }
}