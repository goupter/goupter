package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// cacheItem 缓存项
type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// isExpired 检查是否过期
func (item *cacheItem) isExpired() bool {
	if item.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(item.expiresAt)
}

// memoryCache 内存缓存实现
type memoryCache struct {
	data    map[string]*cacheItem
	mu      sync.RWMutex
	cleaner *time.Ticker
	stop    chan struct{}
}

// MemoryOption 内存缓存选项
type MemoryOption func(*memoryCache)

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(interval time.Duration) MemoryOption {
	return func(c *memoryCache) {
		if c.cleaner != nil {
			c.cleaner.Stop()
		}
		c.cleaner = time.NewTicker(interval)
		go c.cleanup()
	}
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(opts ...MemoryOption) Cache {
	c := &memoryCache{
		data:    make(map[string]*cacheItem),
		cleaner: time.NewTicker(time.Minute),
		stop:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	go c.cleanup()

	return c
}

// cleanup 清理过期缓存
func (c *memoryCache) cleanup() {
	for {
		select {
		case <-c.cleaner.C:
			c.mu.Lock()
			for key, item := range c.data {
				if item.isExpired() {
					delete(c.data, key)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			c.cleaner.Stop()
			return
		}
	}
}

func (c *memoryCache) Get(ctx context.Context, key string, value interface{}) error {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		return &NotFoundError{Key: key}
	}

	return json.Unmarshal(item.value, value)
}

func (c *memoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	item := &cacheItem{
		value: data,
	}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.data[key] = item
	c.mu.Unlock()

	return nil
}

func (c *memoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		return false, nil
	}
	return true, nil
}

func (c *memoryCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(keys))

	c.mu.RLock()
	for _, key := range keys {
		if item, ok := c.data[key]; ok && !item.isExpired() {
			var value interface{}
			if err := json.Unmarshal(item.value, &value); err == nil {
				result[key] = value
			}
		}
	}
	c.mu.RUnlock()

	return result, nil
}

func (c *memoryCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		item := &cacheItem{
			value: data,
		}
		if ttl > 0 {
			item.expiresAt = time.Now().Add(ttl)
		}
		c.data[key] = item
	}

	return nil
}

func (c *memoryCache) MDelete(ctx context.Context, keys []string) error {
	c.mu.Lock()
	for _, key := range keys {
		delete(c.data, key)
	}
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.IncrBy(ctx, key, 1)
}

func (c *memoryCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var current int64
	if item, ok := c.data[key]; ok && !item.isExpired() {
		json.Unmarshal(item.value, &current)
	}

	current += value
	data, _ := json.Marshal(current)
	c.data[key] = &cacheItem{value: data}

	return current, nil
}

func (c *memoryCache) Decr(ctx context.Context, key string) (int64, error) {
	return c.DecrBy(ctx, key, 1)
}

func (c *memoryCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.IncrBy(ctx, key, -value)
}

func (c *memoryCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.data[key]; ok && !item.isExpired() {
		return false, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	item := &cacheItem{value: data}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}
	c.data[key] = item

	return true, nil
}

func (c *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.data[key]; ok {
		item.expiresAt = time.Now().Add(ttl)
	}
	return nil
}

func (c *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return -2, nil // key不存在
	}
	if item.expiresAt.IsZero() {
		return -1, nil // 无过期时间
	}
	if item.isExpired() {
		return -2, nil
	}

	return time.Until(item.expiresAt), nil
}

func (c *memoryCache) Close() error {
	close(c.stop)
	return nil
}

func (c *memoryCache) Ping(ctx context.Context) error {
	return nil
}

// NotFoundError 缓存未找到错误
type NotFoundError struct {
	Key string
}

func (e *NotFoundError) Error() string {
	return "cache: key not found: " + e.Key
}
