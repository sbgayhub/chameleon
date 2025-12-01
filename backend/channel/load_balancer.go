package channel

import (
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/gookit/goutil/errorx"
)

// LoadBalancer 负载均衡接口
type LoadBalancer interface {
	Next(channels []*Channel) (*Channel, error)
}

// CreateLoadBalancer 创建负载均衡器
func CreateLoadBalancer(strategy LBStrategy) (LoadBalancer, error) {
	switch strategy {
	case LB_PRIORITY:
		return &PriorityBalancer{}, nil
	case LB_ROUND:
		return &RoundBalancer{}, nil
	case LB_WEIGHTED_ROUND:
		return &WeightRoundBalancer{}, nil
	case LB_RANDOM:
		return &RandomBalancer{}, nil
	default:
		return nil, errorx.Ef("不支持的负载均衡策略: %d", strategy)
	}
}

// RoundBalancer 轮询负载均衡器
type RoundBalancer struct {
	current int
	mu      sync.Mutex
}

func (r *RoundBalancer) Next(channels []*Channel) (*Channel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(channels) == 0 {
		return nil, fmt.Errorf("渠道列表为空")
	}

	// 轮询选择（传入的channels已经是过滤后的可用渠道）
	channel := channels[r.current%len(channels)]
	r.current++

	slog.Debug("轮询选择渠道", "channel", channel.Name, "index", r.current-1)

	return channel, nil
}

// WeightRoundBalancer 加权轮询负载均衡器
type WeightRoundBalancer struct {
	weights map[string]int
	current map[string]int
	mu      sync.Mutex
}

func (w *WeightRoundBalancer) Next(channels []*Channel) (*Channel, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(channels) == 0 {
		return nil, fmt.Errorf("渠道列表为空")
	}

	// 初始化权重
	if w.weights == nil {
		w.weights = make(map[string]int)
		w.current = make(map[string]int)
		for _, channel := range channels {
			// 使用优先级作为权重，如果没有设置则默认为1
			weight := int(channel.Priority)
			if weight == 0 {
				weight = 1
			}
			w.weights[channel.Name] = weight
			w.current[channel.Name] = weight
		}
	}

	// 加权轮询选择
	for _, channel := range channels {
		if w.current[channel.Name] > 0 {
			w.current[channel.Name]--
			slog.Debug("加权轮询选择渠道", "channel", channel.Name, "weight", w.weights[channel.Name], "current", w.current[channel.Name])
			return channel, nil
		}
	}

	// 所有权重都用完，重新开始
	for _, channel := range channels {
		w.current[channel.Name] = w.weights[channel.Name]
	}

	// 重新选择第一个
	firstChannel := channels[0]
	w.current[firstChannel.Name]--
	slog.Debug("加权轮询重新开始", "channel", firstChannel.Name)

	return firstChannel, nil
}

// PriorityBalancer 优先级负载均衡器
type PriorityBalancer struct{}

func (p PriorityBalancer) Next(channels []*Channel) (*Channel, error) {
	if len(channels) == 0 {
		return nil, fmt.Errorf("渠道列表为空")
	}

	// 按优先级排序，选择最高优先级的渠道（传入的channels已经是过滤后的可用渠道）
	var selectedChannel *Channel
	highestPriority := uint8(255)

	for _, channel := range channels {
		if channel.Priority < highestPriority {
			highestPriority = channel.Priority
			selectedChannel = channel
		}
	}

	slog.Debug("优先级选择渠道", "channel", selectedChannel.Name, "priority", selectedChannel.Priority)

	return selectedChannel, nil
}

// RandomBalancer 随机负载均衡器
type RandomBalancer struct {
	once sync.Once
}

func (r *RandomBalancer) Next(channels []*Channel) (*Channel, error) {
	r.once.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})

	if len(channels) == 0 {
		return nil, fmt.Errorf("渠道列表为空")
	}

	// 随机选择（传入的channels���经是过滤后的可用渠道）
	index := rand.Intn(len(channels))
	channel := channels[index]

	slog.Debug("随机选择渠道", "channel", channel.Name, "index", index)

	return channel, nil
}
