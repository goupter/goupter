package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ConsulSource Consul配置源
type ConsulSource struct {
	client     *api.Client
	path       string
	watchStop  chan struct{}
	mu         sync.RWMutex
	lastIndex  uint64
	onChangeFn func([]byte)
}

// ConsulSourceOption Consul配置源选项
type ConsulSourceOption func(*ConsulSource)

// WithWatchCallback 设置配置变更回调
func WithWatchCallback(fn func([]byte)) ConsulSourceOption {
	return func(s *ConsulSource) {
		s.onChangeFn = fn
	}
}

// NewConsulSource 创建Consul配置源
func NewConsulSource(addr, path string, opts ...ConsulSourceOption) (*ConsulSource, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %w", err)
	}

	source := &ConsulSource{
		client:    client,
		path:      path,
		watchStop: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(source)
	}

	return source, nil
}

// NewConsulSourceWithConfig 使用配置创建Consul配置源
func NewConsulSourceWithConfig(cfg *ConsulConfig, opts ...ConsulSourceOption) (*ConsulSource, error) {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	if cfg.Token != "" {
		config.Token = cfg.Token
	}
	if cfg.Datacenter != "" {
		config.Datacenter = cfg.Datacenter
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建Consul客户端失败: %w", err)
	}

	source := &ConsulSource{
		client:    client,
		path:      cfg.ConfigPath,
		watchStop: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(source)
	}

	return source, nil
}

// Load 从Consul加载配置
func (s *ConsulSource) Load() ([]byte, error) {
	kv := s.client.KV()

	pair, meta, err := kv.Get(s.path, nil)
	if err != nil {
		return nil, fmt.Errorf("从Consul获取配置失败: %w", err)
	}

	if pair == nil {
		return nil, fmt.Errorf("配置路径 %s 不存在", s.path)
	}

	s.mu.Lock()
	s.lastIndex = meta.LastIndex
	s.mu.Unlock()

	return pair.Value, nil
}

// Watch 监听配置变更
func (s *ConsulSource) Watch(ctx context.Context) error {
	kv := s.client.KV()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.watchStop:
			return nil
		default:
			s.mu.RLock()
			lastIndex := s.lastIndex
			s.mu.RUnlock()

			opts := &api.QueryOptions{
				WaitIndex: lastIndex,
				WaitTime:  time.Minute,
			}

			pair, meta, err := kv.Get(s.path, opts)
			if err != nil {
				// 短暂等待后重试
				time.Sleep(time.Second)
				continue
			}

			// 检查是否有变更
			if meta.LastIndex > lastIndex {
				s.mu.Lock()
				s.lastIndex = meta.LastIndex
				s.mu.Unlock()

				if pair != nil && s.onChangeFn != nil {
					s.onChangeFn(pair.Value)
				}
			}
		}
	}
}

// StopWatch 停止监听
func (s *ConsulSource) StopWatch() {
	close(s.watchStop)
}

// Put 写入配置到Consul
func (s *ConsulSource) Put(data []byte) error {
	kv := s.client.KV()

	pair := &api.KVPair{
		Key:   s.path,
		Value: data,
	}

	_, err := kv.Put(pair, nil)
	if err != nil {
		return fmt.Errorf("写入配置到Consul失败: %w", err)
	}

	return nil
}

// Delete 删除Consul中的配置
func (s *ConsulSource) Delete() error {
	kv := s.client.KV()

	_, err := kv.Delete(s.path, nil)
	if err != nil {
		return fmt.Errorf("删除Consul配置失败: %w", err)
	}

	return nil
}

// Client 获取Consul客户端
func (s *ConsulSource) Client() *api.Client {
	return s.client
}

// Path 获取配置路径
func (s *ConsulSource) Path() string {
	return s.path
}

// ConsulConfigCenter Consul配置中心
type ConsulConfigCenter struct {
	source          *ConsulSource
	config          *Config
	mu              sync.RWMutex
	callbacks       []func(*Config)
	ctx             context.Context
	cancel          context.CancelFunc
	isWatching      bool
	format          string // yaml, json
	autoRefresh     bool
	refreshInterval time.Duration
}

// ConsulConfigCenterOption Consul配置中心选项
type ConsulConfigCenterOption func(*ConsulConfigCenter)

// WithFormat 设置配置格式
func WithFormat(format string) ConsulConfigCenterOption {
	return func(c *ConsulConfigCenter) {
		c.format = format
	}
}

// WithAutoRefresh 设置自动刷新
func WithAutoRefresh(interval time.Duration) ConsulConfigCenterOption {
	return func(c *ConsulConfigCenter) {
		c.autoRefresh = true
		c.refreshInterval = interval
	}
}

// NewConsulConfigCenter 创建Consul配置中心
func NewConsulConfigCenter(source *ConsulSource, opts ...ConsulConfigCenterOption) *ConsulConfigCenter {
	ctx, cancel := context.WithCancel(context.Background())

	c := &ConsulConfigCenter{
		source:          source,
		callbacks:       make([]func(*Config), 0),
		ctx:             ctx,
		cancel:          cancel,
		format:          "yaml",
		refreshInterval: time.Minute,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Load 从Consul加载配置
func (c *ConsulConfigCenter) Load() (*Config, error) {
	data, err := c.source.Load()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := c.parseConfig(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	c.mu.Lock()
	c.config = cfg
	c.mu.Unlock()

	// 更新全局配置
	configMu.Lock()
	globalConfig = cfg
	configMu.Unlock()

	return cfg, nil
}

// parseConfig 解析配置
func (c *ConsulConfigCenter) parseConfig(data []byte, cfg *Config) error {
	format := c.format
	if format == "" {
		if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
			format = "json"
		} else {
			format = "yaml"
		}
	}
	if format == "yml" {
		format = "yaml"
	}

	vp := viper.New()
	vp.SetConfigType(format)
	if err := vp.ReadConfig(bytes.NewReader(data)); err != nil {
		return err
	}
	return vp.Unmarshal(cfg)
}

// Get 获取当前配置
func (c *ConsulConfigCenter) Get() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Watch 开始监听配置变更
func (c *ConsulConfigCenter) Watch() error {
	c.mu.Lock()
	if c.isWatching {
		c.mu.Unlock()
		return nil
	}
	c.isWatching = true
	c.mu.Unlock()

	// 设置变更回调
	c.source.onChangeFn = func(data []byte) {
		cfg := &Config{}
		if err := c.parseConfig(data, cfg); err != nil {
			return
		}

		c.mu.Lock()
		oldConfig := c.config
		c.config = cfg
		c.mu.Unlock()

		// 更新全局配置
		configMu.Lock()
		globalConfig = cfg
		configMu.Unlock()

		// 调用回调（传入旧配置以便比较）
		_ = oldConfig // 可用于未来扩展

		c.mu.RLock()
		callbacks := make([]func(*Config), len(c.callbacks))
		copy(callbacks, c.callbacks)
		c.mu.RUnlock()

		for _, fn := range callbacks {
			fn(cfg)
		}
	}

	// 启动监听
	go func() {
		_ = c.source.Watch(c.ctx)
	}()

	// 如果启用自动刷新
	if c.autoRefresh {
		go c.autoRefreshLoop()
	}

	return nil
}

// autoRefreshLoop 自动刷新循环
func (c *ConsulConfigCenter) autoRefreshLoop() {
	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			_, _ = c.Load()
		}
	}
}

// Stop 停止监听
func (c *ConsulConfigCenter) Stop() {
	c.cancel()
	c.source.StopWatch()

	c.mu.Lock()
	c.isWatching = false
	c.mu.Unlock()
}

// OnChange 注册配置变更回调
func (c *ConsulConfigCenter) OnChange(fn func(*Config)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, fn)
}

// Save 保存配置到Consul
func (c *ConsulConfigCenter) Save(cfg *Config) error {
	var data []byte
	var err error

	switch c.format {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cfg)
	default:
		data, err = yaml.Marshal(cfg)
	}

	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return c.source.Put(data)
}

// GetKey 获取指定路径的配置值
func (c *ConsulConfigCenter) GetKey(key string) ([]byte, error) {
	kv := c.source.client.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return pair.Value, nil
}

// PutKey 设置指定路径的配置值
func (c *ConsulConfigCenter) PutKey(key string, value []byte) error {
	kv := c.source.client.KV()
	pair := &api.KVPair{
		Key:   key,
		Value: value,
	}
	_, err := kv.Put(pair, nil)
	return err
}

// DeleteKey 删除指定路径的配置
func (c *ConsulConfigCenter) DeleteKey(key string) error {
	kv := c.source.client.KV()
	_, err := kv.Delete(key, nil)
	return err
}

// List 列出指定前缀下的所有配置键
func (c *ConsulConfigCenter) List(prefix string) ([]string, error) {
	kv := c.source.client.KV()
	keys, _, err := kv.Keys(prefix, "", nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// IsWatching 是否正在监听
func (c *ConsulConfigCenter) IsWatching() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isWatching
}
