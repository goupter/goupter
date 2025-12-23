package cache

import (
	"context"
	"time"
)

// multiLevelCache 多级缓存
type multiLevelCache struct {
	levels []Cache
}

// MultiLevelOption 多级缓存选项
type MultiLevelOption func(*multiLevelCache)

// WithLevels 设置缓存层级
func WithLevels(levels ...Cache) MultiLevelOption {
	return func(c *multiLevelCache) {
		c.levels = levels
	}
}

// NewMultiLevelCache 创建多级缓存
// 通常配置为: L1 = 内存缓存, L2 = Redis缓存
func NewMultiLevelCache(opts ...MultiLevelOption) Cache {
	c := &multiLevelCache{
		levels: make([]Cache, 0),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewDefaultMultiLevelCache 创建默认多级缓存 (内存 + Redis)
func NewDefaultMultiLevelCache(redisCache Cache) Cache {
	return NewMultiLevelCache(
		WithLevels(
			NewMemoryCache(WithCleanupInterval(time.Minute)),
			redisCache,
		),
	)
}

func (c *multiLevelCache) Get(ctx context.Context, key string, value interface{}) error {
	// 从L1到Ln依次查找
	for i, level := range c.levels {
		if err := level.Get(ctx, key, value); err == nil {
			// 如果从非L1缓存获取到，回填L1
			if i > 0 {
				// 获取TTL并回填上层缓存
				ttl, _ := level.TTL(ctx, key)
				if ttl > 0 {
					for j := 0; j < i; j++ {
						_ = c.levels[j].Set(ctx, key, value, ttl)
					}
				}
			}
			return nil
		}
	}
	return &NotFoundError{Key: key}
}

func (c *multiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 写入所有层级
	var lastErr error
	for _, level := range c.levels {
		if err := level.Set(ctx, key, value, ttl); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) Delete(ctx context.Context, key string) error {
	// 从所有层级删除
	var lastErr error
	for _, level := range c.levels {
		if err := level.Delete(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	for _, level := range c.levels {
		if exists, err := level.Exists(ctx, key); err == nil && exists {
			return true, nil
		}
	}
	return false, nil
}

func (c *multiLevelCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(keys))
	remainingKeys := keys

	for _, level := range c.levels {
		if len(remainingKeys) == 0 {
			break
		}

		values, err := level.MGet(ctx, remainingKeys)
		if err != nil {
			continue
		}

		// 收集找到的值
		var notFoundKeys []string
		for _, key := range remainingKeys {
			if v, ok := values[key]; ok {
				result[key] = v
			} else {
				notFoundKeys = append(notFoundKeys, key)
			}
		}
		remainingKeys = notFoundKeys
	}

	return result, nil
}

func (c *multiLevelCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	var lastErr error
	for _, level := range c.levels {
		if err := level.MSet(ctx, items, ttl); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) MDelete(ctx context.Context, keys []string) error {
	var lastErr error
	for _, level := range c.levels {
		if err := level.MDelete(ctx, keys); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) Incr(ctx context.Context, key string) (int64, error) {
	// 只在最后一级执行原子操作
	if len(c.levels) > 0 {
		return c.levels[len(c.levels)-1].Incr(ctx, key)
	}
	return 0, nil
}

func (c *multiLevelCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if len(c.levels) > 0 {
		return c.levels[len(c.levels)-1].IncrBy(ctx, key, value)
	}
	return 0, nil
}

func (c *multiLevelCache) Decr(ctx context.Context, key string) (int64, error) {
	if len(c.levels) > 0 {
		return c.levels[len(c.levels)-1].Decr(ctx, key)
	}
	return 0, nil
}

func (c *multiLevelCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	if len(c.levels) > 0 {
		return c.levels[len(c.levels)-1].DecrBy(ctx, key, value)
	}
	return 0, nil
}

func (c *multiLevelCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	// 只在最后一级执行原子操作
	if len(c.levels) > 0 {
		result, err := c.levels[len(c.levels)-1].SetNX(ctx, key, value, ttl)
		if err != nil {
			return false, err
		}
		// 如果设置成功，同步到其他层级
		if result {
			for i := 0; i < len(c.levels)-1; i++ {
				_ = c.levels[i].Set(ctx, key, value, ttl)
			}
		}
		return result, nil
	}
	return false, nil
}

func (c *multiLevelCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	var lastErr error
	for _, level := range c.levels {
		if err := level.Expire(ctx, key, ttl); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	// 从最后一级获取TTL
	if len(c.levels) > 0 {
		return c.levels[len(c.levels)-1].TTL(ctx, key)
	}
	return -2, nil
}

func (c *multiLevelCache) Close() error {
	var lastErr error
	for _, level := range c.levels {
		if err := level.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *multiLevelCache) Ping(ctx context.Context) error {
	// 检查所有层级
	for _, level := range c.levels {
		if err := level.Ping(ctx); err != nil {
			return err
		}
	}
	return nil
}
