package convert

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

const (
	ANTHROPIC2OPENAI    = "anthropic->openai"
	ANTHROPIC2GEMINI    = "anthropic->gemini"
	ANTHROPIC2ANTHROPIC = "anthropic->anthropic"
	GEMINI2ANTHROPIC    = "gemini->anthropic"
	GEMINI2OPENAI       = "gemini->openai"
	GEMINI2GEMINI       = "gemini->gemini"
	OPENAI2ANTHROPIC    = "openai->anthropic"
	OPENAI2GEMINI       = "openai->gemini"
	OPENAI2OPENAI       = "openai->openai"
)

var globalRegistry *Registry
var once sync.Once

// GetRegistry 获取全局转换器注册表
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			converters: make(map[string]Converter),
		}
	})
	return globalRegistry
}

// Register 注册转换器
func (r *Registry) Register(converter Converter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := converter.Name()
	if _, exists := r.converters[name]; exists {
		return fmt.Errorf("转换器已存在: %s", name)
	}

	r.converters[name] = converter
	slog.Info("注册转换器", "name", name)

	return nil
}

// Get 获取转换器
func (r *Registry) Get(name string) (Converter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	converter, exists := r.converters[name]
	if !exists {
		return nil, fmt.Errorf("转换器不存在: %s", name)
	}

	return converter, nil
}

// List 列出所有转换器
func (r *Registry) List() []Converter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	converters := make([]Converter, 0, len(r.converters))
	for _, converter := range r.converters {
		converters = append(converters, converter)
	}

	return converters
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.converters))
	for _, converter := range r.converters {
		names = append(names, converter.Name())
	}

	slices.Sort(names)
	return names
}

// Register 注册转换器到全局注册表
func Register(converter Converter) error {
	return GetRegistry().Register(converter)
}

// Get 从全局注册表获取转换器
func Get(name string) (Converter, error) {
	return GetRegistry().Get(name)
}

// List 列出全局注册表中的所有转换器
func List() []Converter {
	return GetRegistry().List()
}
