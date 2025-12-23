package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/goupter/goupter/pkg/config"
	"github.com/redis/go-redis/v9"
)

// redisCache Redis缓存实现
type redisCache struct {
	client *redis.Client
	config *config.RedisConfig
}

// RedisOption Redis选项
type RedisOption func(*redisCache)

// WithRedisConfig 设置Redis配置
func WithRedisConfig(cfg *config.RedisConfig) RedisOption {
	return func(c *redisCache) {
		c.config = cfg
	}
}

// NewRedisCache 创建Redis缓存
func NewRedisCache(opts ...RedisOption) (Cache, error) {
	c := &redisCache{
		config: &config.RedisConfig{
			Host:         "localhost",
			Port:         6379,
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		Password:     c.config.Password,
		DB:           c.config.DB,
		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,
		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return c, nil
}

// Client 获取Redis客户端
func (c *redisCache) Client() *redis.Client {
	return c.client
}

func (c *redisCache) Get(ctx context.Context, key string, value interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return &NotFoundError{Key: key}
		}
		return err
	}
	return json.Unmarshal(data, value)
}

func (c *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *redisCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{}, len(keys))
	for i, key := range keys {
		if values[i] != nil {
			result[key] = values[i]
		}
	}
	return result, nil
}

func (c *redisCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *redisCache) MDelete(ctx context.Context, keys []string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

func (c *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

func (c *redisCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.DecrBy(ctx, key, value).Result()
}

func (c *redisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return c.client.SetNX(ctx, key, data, ttl).Result()
}

func (c *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

func (c *redisCache) Close() error {
	return c.client.Close()
}

func (c *redisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
