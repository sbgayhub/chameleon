package channel

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/gookit/goutil/errorx"
)

type Manager struct {
	dataPath string
	groups   map[string]*Group
	mu       sync.RWMutex
}

// NewManager 创建渠道管理器
func NewManager(dataPath string) *Manager {
	return &Manager{
		dataPath: dataPath,
		groups:   make(map[string]*Group),
	}
}

// LoadFromFile 从文件加载渠道配置
func (m *Manager) LoadFromFile() error {
	configPath := filepath.Join(m.dataPath, "channels.json")

	// 如果文件不存在，直接返回成功
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Info("渠道配置文件不存在，跳过加载", "path", configPath)
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取渠道配置文件失败: %w", err)
	}

	// 如果文件为空，直接返回成功
	if len(data) == 0 {
		slog.Info("渠道配置文件为空，跳过加载", "path", configPath)
		return nil
	}

	// 解析JSON配置
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := json.Unmarshal(data, &m.groups); err != nil {
		return fmt.Errorf("解析渠道配置文件失败: %w", err)
	}

	for _, group := range m.groups {
		if group.LoadBalancer, err = CreateLoadBalancer(group.LBStrategy); err != nil {
			slog.Error("负载均衡器创建失败", "group", group.Endpoint, "strategy", group.LBStrategy)
		}
		for _, channel := range group.Channels {
			channel.ConverterName = fmt.Sprintf("%s->%s", group.Provider, channel.Provider)
			channel.ModelMapper = NewModelMapper()
			for key, val := range channel.ModelMapping {
				channel.ModelMapper.AddRule(key, val)
			}
		}
	}
	slog.Info("成功加载渠道配置", "groups", len(m.groups), "path", configPath)
	return nil
}

// List 列出所有渠道组，按优先级排序
func (m *Manager) List() []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := slices.Collect(maps.Values(m.groups))
	slices.SortFunc(groups, func(a, b *Group) int {
		return int(a.Priority - b.Priority)
	})
	return groups
}

// UpdateGroupPriority 更新渠道组优先级
func (m *Manager) UpdateGroupPriority(endpoint string, priority uint8) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.groups[endpoint]; ok {
		g.Priority = priority
		slog.Info("更新渠道组优先级", "endpoint", endpoint, "priority", priority)
		return true
	} else {
		slog.Warn("更新渠道组优先级失败，渠道组不存在", "endpoint", endpoint)
		return false
	}
}

// UpdateChannelPriority 更新渠道优先级
func (m *Manager) UpdateChannelPriority(group, name string, priority uint8) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.groups[group]; ok {
		if ch, ok := g.Channels[name]; ok {
			ch.Priority = priority
			slog.Info("更新渠道优先级", "channel", name, "priority", priority)
			return true
		} else {
			slog.Warn("更新渠道优先级失败，渠道不存在", "channel", name)
			return false
		}
	} else {
		slog.Warn("更新渠道优先级失败，渠道组不存在", "endpoint", group)
		return false
	}
}

// AddGroup 添加渠道组
func (m *Manager) AddGroup(group *Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if group.Endpoint == "" {
		return fmt.Errorf("渠道组端点不能为空")
	}

	// 检查是否已存在
	if _, exists := m.groups[group.Endpoint]; exists {
		return fmt.Errorf("渠道组已存在: %s", group.Endpoint)
	}

	// 初始化渠道组的渠道映射
	if group.Channels == nil {
		group.Channels = make(map[string]*Channel)
	}

	// 创建负载均衡器
	lb, err := CreateLoadBalancer(group.LBStrategy)
	if err != nil {
		return fmt.Errorf("创建负载均衡器失败: %w", err)
	}
	group.LoadBalancer = lb

	m.groups[group.Endpoint] = group
	slog.Info("添加渠道组", "endpoint", group.Endpoint, "strategy", group.LBStrategy)

	return nil
}

// GetGroup 获取渠道组
func (m *Manager) GetGroup(endpoint string) (*Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[endpoint]
	if !exists {
		return nil, fmt.Errorf("渠道组不存在: %s", endpoint)
	}

	return group, nil
}

// UpdateGroup 更新渠道组
func (m *Manager) UpdateGroup(group *Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if group.Endpoint == "" {
		return fmt.Errorf("渠道组端点不能为空")
	}

	// 检查渠道组是否存在
	if _, exists := m.groups[group.Endpoint]; !exists {
		return fmt.Errorf("渠道组不存在: %s", group.Endpoint)
	}

	// 更新负载均衡器
	lb, err := CreateLoadBalancer(group.LBStrategy)
	if err != nil {
		return fmt.Errorf("创建负载均衡器失败: %w", err)
	}
	group.LoadBalancer = lb

	m.groups[group.Endpoint] = group
	slog.Info("更新渠道组", "endpoint", group.Endpoint)

	return nil
}

// DeleteGroup 删除渠道组
func (m *Manager) DeleteGroup(endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.groups[endpoint]; !exists {
		return fmt.Errorf("渠道组不存在: %s", endpoint)
	}

	delete(m.groups, endpoint)
	slog.Info("删除渠道组", "endpoint", endpoint)

	return nil
}

// AddChannel 添加渠道
func (m *Manager) AddChannel(groupEndpoint string, channel *Channel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupEndpoint]
	if !exists {
		return fmt.Errorf("渠道组不存在: %s", groupEndpoint)
	}

	if channel.Name == "" {
		return fmt.Errorf("渠道名称不能为空")
	}

	if channel.URL == "" {
		return fmt.Errorf("渠道端点不能为空")
	}

	// 检查是否已存在同名渠道
	if _, exists := group.Channels[channel.Name]; exists {
		return fmt.Errorf("渠道已存在: %s", channel.Name)
	}

	// 设置默认值
	if channel.Status == 0 {
		channel.Status = STATUS_NORMAL
	}

	channel.ConverterName = fmt.Sprintf("%s->%s", group.Provider, channel.Provider)
	channel.ModelMapper = NewModelMapper()
	for key, val := range channel.ModelMapping {
		channel.ModelMapper.AddRule(key, val)
	}

	group.Channels[channel.Name] = channel
	slog.Info("添加渠道", "group", groupEndpoint, "channel", channel.Name, "endpoint", channel.URL)

	return nil
}

// UpdateChannel 更新渠道
func (m *Manager) UpdateChannel(groupEndpoint string, channel *Channel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupEndpoint]
	if !exists {
		return fmt.Errorf("渠道组不存在: %s", groupEndpoint)
	}

	if _, exists := group.Channels[channel.Name]; !exists {
		return fmt.Errorf("渠道不存在: %s", channel.Name)
	}

	channel.ConverterName = fmt.Sprintf("%s->%s", group.Provider, channel.Provider)
	channel.ModelMapper = NewModelMapper()
	for key, val := range channel.ModelMapping {
		channel.ModelMapper.AddRule(key, val)
	}

	group.Channels[channel.Name] = channel
	slog.Info("更新渠道", "group", groupEndpoint, "channel", channel.Name)

	return nil
}

// DeleteChannel 删除渠道
func (m *Manager) DeleteChannel(groupEndpoint, channelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupEndpoint]
	if !exists {
		return fmt.Errorf("渠道组不存在: %s", groupEndpoint)
	}

	if _, exists := group.Channels[channelName]; !exists {
		return fmt.Errorf("渠道不存在: %s", channelName)
	}

	delete(group.Channels, channelName)
	slog.Info("删除渠道", "group", groupEndpoint, "channel", channelName)

	return nil
}

// GetChannel 获取渠道
func (m *Manager) GetChannel(groupEndpoint, channelName string) (*Channel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[groupEndpoint]
	if !exists {
		return nil, fmt.Errorf("渠道组不存在: %s", groupEndpoint)
	}

	channel, exists := group.Channels[channelName]
	if !exists {
		return nil, fmt.Errorf("渠道不存在: %s", channelName)
	}

	return channel, nil
}

// SetChannelStatus 设置渠道状态
func (m *Manager) SetChannelStatus(groupEndpoint, channelName string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupEndpoint]
	if !exists {
		return fmt.Errorf("渠道组不存在: %s", groupEndpoint)
	}

	channel, exists := group.Channels[channelName]
	if !exists {
		return fmt.Errorf("渠道不存在: %s", channelName)
	}

	channel.Status = status
	group.Channels[channelName] = channel

	slog.Debug("设置渠道状态", "group", groupEndpoint, "channel", channelName, "status", status)

	return nil
}

// SaveToFile 将渠道配置保存到JSON文件
func (m *Manager) SaveToFile() error {
	path := filepath.Join(m.dataPath, "channels.json")

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 序列化为JSON
	data, err := json.MarshalIndent(m.groups, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化渠道配置失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(m.dataPath, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入渠道配置文件失败: %w", err)
	}

	slog.Info("成功保存渠道配置", "groups", len(m.groups), "path", path)
	return nil
}

func (m *Manager) SelectChannel(endpoint string) (*Channel, error) {
	group, err := m.GetGroup(endpoint)
	if err != nil {
		return nil, err
	}

	if !group.Enabled {
		return nil, fmt.Errorf("渠道组未启用")
	}

	if group.LoadBalancer == nil {
		return nil, fmt.Errorf("负载均衡器未初始化")
	}

	// 筛选出可用的渠道
	channels := make([]*Channel, 0)
	for _, channel := range group.Channels {
		if channel.Enabled && channel.Status == STATUS_NORMAL {
			channels = append(channels, channel)
		}
	}

	return group.LoadBalancer.Next(channels)
}

func (m *Manager) fetchModels(node *Channel) error {
	switch node.Provider {
	case "anthropic":
		return fetchAnthropicModels(node)
	case "openai":
		return fetchOpenaiModels(node)
	case "gemini":
		return fetchGeminiModels(node)
	}
	return nil
}

// FetchModels 获取渠道的模型列表
func (m *Manager) FetchModels(groupEndpoint, channelName string) ([]string, error) {
	m.mu.RLock()
	group, ok := m.groups[groupEndpoint]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("渠道组不存在")
	}

	node, ok := group.Channels[channelName]
	if !ok {
		return nil, fmt.Errorf("渠道不存在")
	}

	if err := m.fetchModels(node); err != nil {
		return nil, err
	}

	return node.Models, nil
}

func (m *Manager) TestChannel(groupEndpoint, channelName string) (result string, err error) {
	// 从管理器中获取到渠道节点
	node := m.groups[groupEndpoint].Channels[channelName]
	// 如果模型列表为空且测试模型为空，那么先获取模型列表
	if node.TestModel == "" && len(node.Models) == 0 {
		err := m.fetchModels(node)
		if err != nil {
			return "", errorx.With(err, "获取模型列表失败")
		}
	}
	switch node.Provider {
	case "anthropic":
		result, err = testAnthropicChannel(node)
	case "openai":
		result, err = testOpenaiChannel(node)
	case "gemini":
		result, err = testGeminiChannel(node)
	}

	// 测试失败，设置渠道节点状态
	if err != nil {
		node.Status = STATUS_ERROR
		slog.Warn("测试渠道失败", "group", groupEndpoint, "channel", channelName, "error", err)
	} else {
		node.Status = STATUS_NORMAL
		slog.Info("测试渠道成功", "group", groupEndpoint, "channel", channelName)
	}
	_ = m.SaveToFile()

	return result, err
}
