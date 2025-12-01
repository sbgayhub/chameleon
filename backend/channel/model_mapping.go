package channel

import (
	"strings"
	"sync"
)

// ModelMapper 模型映射器
type ModelMapper struct {
	rules []ModelMappingRule
	mu    sync.RWMutex
}

// ModelMappingRule 模型映射规则
type ModelMappingRule struct {
	Pattern string   `json:"pattern"` // 匹配模式
	Target  string   `json:"target"`  // 目标模型
	Type    RuleType `json:"type"`    // 规则类型
}

// RuleType 规则类型
type RuleType int

const (
	ExactMatch    RuleType = iota // 精确匹配
	WildcardMatch                 // 通配符匹配
	AllMatch                      // 全通配符
)

// NewModelMapper 创建模型映射器
func NewModelMapper() *ModelMapper {
	return &ModelMapper{
		rules: make([]ModelMappingRule, 0),
	}
}

// AddRule 添加映射规则
func (m *ModelMapper) AddRule(pattern, target string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ruleType := m.getRuleType(pattern)
	rule := ModelMappingRule{
		Pattern: pattern,
		Target:  target,
		Type:    ruleType,
	}

	// 按优先级插入：精确匹配 > 通配符匹配 > 全通配符
	inserted := false
	for i, existingRule := range m.rules {
		if ruleType < existingRule.Type {
			m.rules = append(m.rules[:i], append([]ModelMappingRule{rule}, m.rules[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		m.rules = append(m.rules, rule)
	}
}

// getRuleType 确定规则类型
func (m *ModelMapper) getRuleType(pattern string) RuleType {
	if pattern == "*" {
		return AllMatch
	}
	if strings.Contains(pattern, "*") {
		return WildcardMatch
	}
	return ExactMatch
}

// MapModel 映射模型名称
func (m *ModelMapper) MapModel(model string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, rule := range m.rules {
		if m.matchRule(rule, model) {
			return rule.Target
		}
	}

	// 没有匹配的规则，返回原模型名
	return model
}

// matchRule 检查模型是否匹配规则
func (m *ModelMapper) matchRule(rule ModelMappingRule, model string) bool {
	switch rule.Type {
	case ExactMatch:
		return model == rule.Pattern
	case WildcardMatch:
		return m.wildcardMatch(rule.Pattern, model)
	case AllMatch:
		return true
	default:
		return false
	}
}

// wildcardMatch 通配符匹配
func (m *ModelMapper) wildcardMatch(pattern, model string) bool {
	// 简单的通配符实现，支持 * 在开头、结尾或中间
	if !strings.Contains(pattern, "*") {
		return pattern == model
	}

	// 将模式分割为非通配符部分
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return true
	}

	// 检查开头
	if parts[0] != "" && !strings.HasPrefix(model, parts[0]) {
		return false
	}

	// 检查结尾
	if parts[len(parts)-1] != "" && !strings.HasSuffix(model, parts[len(parts)-1]) {
		return false
	}

	// 检查中间部分
	currentIndex := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		if part == "" {
			continue // 连续的 ** 被当作一个 *
		}

		index := strings.Index(model[currentIndex:], part)
		if index == -1 {
			return false
		}
		currentIndex += index + len(part)
	}

	return true
}

// SetRules 设置映射规则（替换所有现有规则）
func (m *ModelMapper) SetRules(rules []ModelMappingRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules = make([]ModelMappingRule, len(rules))
	copy(m.rules, rules)
}

// GetRules 获取所有映射规则
func (m *ModelMapper) GetRules() []ModelMappingRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]ModelMappingRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// ClearRules 清空所有规则
func (m *ModelMapper) ClearRules() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules = make([]ModelMappingRule, 0)
}

// DeleteRule 删除指定的规则
func (m *ModelMapper) DeleteRule(pattern string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rule := range m.rules {
		if rule.Pattern == pattern {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return true
		}
	}
	return false
}
