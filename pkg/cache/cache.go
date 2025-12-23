package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string, value interface{}) error
	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查key是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// MGet 批量获取
	MGet(ctx context.Context, keys []string) (map[string]interface{}, error)
	// MSet 批量设置
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	// MDelete 批量删除
	MDelete(ctx context.Context, keys []string) error

	// Incr 自增
	Incr(ctx context.Context, key string) (int64, error)
	// IncrBy 自增指定值
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	// Decr 自减
	Decr(ctx context.Context, key string) (int64, error)
	// DecrBy 自减指定值
	DecrBy(ctx context.Context, key string, value int64) (int64, error)

	// SetNX 仅当key不存在时设置
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)

	// Expire 设置过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// TTL 获取过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Close 关闭连接
	Close() error

	// Ping 检查连接
	Ping(ctx context.Context) error
}
