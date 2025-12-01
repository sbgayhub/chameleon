package statistics

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Statistics 统计数据
type Statistics struct {
	ChannelName  string    `json:"channel_name"`  // 渠道名称
	RequestCount uint64    `json:"request_count"` // 请求次数
	SuccessCount uint64    `json:"success_count"` // 成功次数
	FailureCount uint64    `json:"failure_count"` // 失败次数
	InputToken   uint64    `json:"input_token"`   // 输入（请求）token数
	OutputToken  uint64    `json:"output_token"`  // 输出（响应）token数
	LastUsed     time.Time `json:"last_used"`     // 最后使用时间
}

// DailyStats 每日统计
type DailyStats struct {
	Date         string `json:"date"`          // 日期 YYYY-MM-DD
	RequestCount uint64 `json:"request_count"` // 请求次数
	SuccessCount uint64 `json:"success_count"` // 成功次数
	FailureCount uint64 `json:"failure_count"` // 失败次数
	InputToken   uint64 `json:"input_token"`   // 输入token数
	OutputToken  uint64 `json:"output_token"`  // 输出token数
}

// Manager 统计管理器
type Manager struct {
	dataPath    string
	dailyPath   string
	data        map[string]*Statistics // key: channelGroup/channelName
	dailyStats  map[string]*DailyStats // key: date
	currentDate string
	mutex       sync.RWMutex
}

var (
	manager *Manager
	once    sync.Once
)

// NewManager 创建统计管理器
func NewManager(dataDir string) *Manager {
	once.Do(func() {
		manager = &Manager{
			dataPath:    dataDir + "/stats.json",
			dailyPath:   dataDir + "/daily.json",
			data:        make(map[string]*Statistics),
			dailyStats:  make(map[string]*DailyStats),
			currentDate: time.Now().Format("2006-01-02"),
		}
		// 启动时加载数据
		if err := manager.Load(); err != nil {
			slog.Warn("加载统计数据失败，使用空数据", "error", err)
		}
		if err := manager.LoadDaily(); err != nil {
			slog.Warn("加载每日统计失败，使用空数据", "error", err)
		}
	})

	return manager
}

// Load 加载统计数据
func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 如果文件不存在，返回nil不算错误
	if _, err := os.Stat(m.dataPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(m.dataPath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &m.data)
}

// LoadDaily 加载每日统计
func (m *Manager) LoadDaily() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, err := os.Stat(m.dailyPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(m.dailyPath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &m.dailyStats)
}

// Save 保存统计数据
func (m *Manager) Save() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataPath, data, 0644)
}

// SaveDaily 保存每日统计
func (m *Manager) SaveDaily() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := json.MarshalIndent(m.dailyStats, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dailyPath, data, 0644)
}

// UpdateStatistics 更新统计数据
func (m *Manager) UpdateStatistics(channelName string, inputTokens, outputTokens uint64, success bool) {
	m.mutex.Lock()

	stats, exists := m.data[channelName]
	if !exists {
		stats = &Statistics{
			ChannelName: channelName,
		}
		m.data[channelName] = stats
	}

	stats.RequestCount++
	stats.InputToken += inputTokens
	stats.OutputToken += outputTokens
	stats.LastUsed = time.Now()

	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	// 更新每日统计
	today := time.Now().Format("2006-01-02")
	if today != m.currentDate {
		m.currentDate = today
	}

	dailyStats, exists := m.dailyStats[today]
	if !exists {
		dailyStats = &DailyStats{Date: today}
		m.dailyStats[today] = dailyStats
	}

	dailyStats.RequestCount++
	dailyStats.InputToken += inputTokens
	dailyStats.OutputToken += outputTokens
	if success {
		dailyStats.SuccessCount++
	} else {
		dailyStats.FailureCount++
	}

	m.mutex.Unlock()

	// 在锁外保存数据
	_ = m.Save()
	_ = m.SaveDaily()
}

// GetAllStatistics 获取所有统计数据
func (m *Manager) GetAllStatistics() map[string]*Statistics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 返回副本
	result := make(map[string]*Statistics)
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

// ResetAllStatistics 重置所有统计数据
func (m *Manager) ResetAllStatistics() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string]*Statistics)
	m.dailyStats = make(map[string]*DailyStats)
	_ = m.Save()
	_ = m.SaveDaily()
}

// GetDailyStatistics 获取每日统计
func (m *Manager) GetDailyStatistics() map[string]*DailyStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*DailyStats)
	for k, v := range m.dailyStats {
		result[k] = v
	}
	return result
}

// GetTotalRequests 获取总请求数
func (m *Manager) GetTotalRequests() int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var total int64
	for _, stats := range m.data {
		total += int64(stats.RequestCount)
	}
	return total
}

// GetTotalStatistics 获取总统计数据
func (m *Manager) GetTotalStatistics() *Statistics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	totalStats := &Statistics{
		ChannelName: "total",
	}

	for _, stats := range m.data {
		totalStats.RequestCount += stats.RequestCount
		totalStats.SuccessCount += stats.SuccessCount
		totalStats.FailureCount += stats.FailureCount
		totalStats.InputToken += stats.InputToken
		totalStats.OutputToken += stats.OutputToken

		if stats.LastUsed.After(totalStats.LastUsed) {
			totalStats.LastUsed = stats.LastUsed
		}
	}

	return totalStats
}

func UpdateStatistics(name string, success bool, input, output uint64) {
	manager.UpdateStatistics(name, input, output, success)
}
