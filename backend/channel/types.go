package channel

import (
	"fmt"
)

type LBStrategy uint8
type Status uint8

const (
	LB_PRIORITY       = LBStrategy(1) // 优先级
	LB_ROUND          = LBStrategy(2) // 轮询
	LB_WEIGHTED_ROUND = LBStrategy(3) // 加权轮询
	LB_RANDOM         = LBStrategy(4) // 随机

	STATUS_NORMAL        = Status(1) // 正常
	STATUS_ERROR         = Status(2) // 异常
	STATUS_NOT_AVAILABLE = Status(3) // 不可用
)

type Channel struct {
	Name          string            `json:"name,omitempty"`          // 名称(ID)
	Enabled       bool              `json:"enabled,omitempty"`       // 启用状态
	Priority      uint8             `json:"priority,omitempty"`      // 优先级
	URL           string            `json:"url,omitempty"`           // 目标地址
	ApiKey        string            `json:"api_key,omitempty"`       // 目标apikey
	Provider      string            `json:"provider"`                // 渠道供应商类型
	ModelMapping  map[string]string `json:"model_mapping,omitempty"` // 模型映射
	Status        Status            `json:"status,omitempty"`        // 状态
	TestModel     string            `json:"test_model"`              // 用于测试的模型
	ConverterName string            `json:"-"`                       // 使用的转换器名称
	ModelMapper   *ModelMapper      `json:"-"`                       // 模型映射器（运行时使用）
	Models        []string          `json:"-"`                       // 渠道的模型列表
}

// Group 渠道组
type Group struct {
	Endpoint     string              `json:"endpoint,omitempty"`    // 渠道的端点地址(ID)
	Enabled      bool                `json:"enabled,omitempty"`     // 启用状态
	Priority     uint8               `json:"priority,omitempty"`    // 优先级（用于UI排序）
	LBStrategy   LBStrategy          `json:"lb_strategy,omitempty"` // 渠道组内的负载均衡策略
	Provider     string              `json:"provider"`              // 渠道组的供应商格式
	Channels     map[string]*Channel `json:"channels,omitempty"`    // 渠道 [Channel.Name:Channel]
	LoadBalancer LoadBalancer        `json:"-"`                     // 负载均衡器
}

// SelectChannel 根据负载均衡策略选择渠道
func (g *Group) SelectChannel() (*Channel, error) {
	if !g.Enabled {
		return nil, fmt.Errorf("渠道组未启用")
	}

	if g.LoadBalancer == nil {
		return nil, fmt.Errorf("负载均衡器未初始化")
	}

	channels := make([]*Channel, 0)
	for _, channel := range g.Channels {
		if channel.Enabled && channel.Status == STATUS_NORMAL {
			channels = append(channels, channel)
		}
	}

	return g.LoadBalancer.Next(channels)
}

func (g *Group) init() error {
	// 根据负载均衡策略创建负载均衡器
	if balancer, err := CreateLoadBalancer(g.LBStrategy); err != nil {
		return err
	} else {
		g.LoadBalancer = balancer
		return nil
	}
}

func (c *Channel) init() error {
	// 创建模型映射器
	c.ModelMapper = NewModelMapper()
	for pattern, target := range c.ModelMapping {
		c.ModelMapper.AddRule(pattern, target)
	}
	return nil
}
