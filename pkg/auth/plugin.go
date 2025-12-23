package auth

import (
	"fmt"
	"sync"
)

// Plugin 鉴权插件接口
type Plugin interface {
	Name() string
	Init(config map[string]interface{}) error
	Authenticator() Authenticator
}

var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

// Register 注册插件
func Register(plugin Plugin) {
	mu.Lock()
	defer mu.Unlock()

	name := plugin.Name()
	if name == "" {
		panic("auth: plugin name cannot be empty")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("auth: plugin %s already registered", name))
	}
	registry[name] = plugin
}

// Get 获取插件
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	plugin, ok := registry[name]
	return plugin, ok
}

// NewAuthenticator 根据插件名创建鉴权器
func NewAuthenticator(name string, config map[string]interface{}) (Authenticator, error) {
	plugin, ok := Get(name)
	if !ok {
		return nil, fmt.Errorf("auth plugin %s not found", name)
	}

	if err := plugin.Init(config); err != nil {
		return nil, fmt.Errorf("init auth plugin %s failed: %w", name, err)
	}

	return plugin.Authenticator(), nil
}
