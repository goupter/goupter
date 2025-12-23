package cache

import (
	"context"
	"testing"
	"time"
)

func TestNewMultiLevelCache(t *testing.T) {
	cache := NewMultiLevelCache()
	if cache == nil {
		t.Fatal("NewMultiLevelCache should return non-nil")
	}
	defer cache.Close()
}

func TestNewMultiLevelCache_WithLevels(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	if cache == nil {
		t.Fatal("NewMultiLevelCache with levels should return non-nil")
	}
	// Don't close cache here to avoid double close, levels will be closed by test cleanup
}

func TestNewDefaultMultiLevelCache(t *testing.T) {
	redis := NewMemoryCache() // Using memory as mock Redis

	cache := NewDefaultMultiLevelCache(redis)
	if cache == nil {
		t.Fatal("NewDefaultMultiLevelCache should return non-nil")
	}
	// Only close the multi-level cache, which closes all levels including redis
	defer cache.Close()
}

func TestMultiLevelCache_SetGet(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	// Only close the multi-level cache once
	defer cache.Close()

	err := cache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var result string
	err = cache.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}
}

func TestMultiLevelCache_GetFromL2_BackfillL1(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	// Set only in L2
	l2.Set(ctx, "key1", "value1", time.Minute)

	// Get should find it in L2 and backfill L1
	var result string
	err := cache.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}

	// Now L1 should also have the value
	var l1Result string
	err = l1.Get(ctx, "key1", &l1Result)
	if err != nil {
		t.Errorf("L1 should have value after backfill: %v", err)
	}
}

func TestMultiLevelCache_GetNotFound(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	var result string
	err := cache.Get(ctx, "nonexistent", &result)
	if err == nil {
		t.Fatal("Expected error for nonexistent key")
	}
	if !IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestMultiLevelCache_Delete(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Delete(ctx, "key1")

	// Both levels should not have the key
	exists1, _ := l1.Exists(ctx, "key1")
	exists2, _ := l2.Exists(ctx, "key1")

	if exists1 || exists2 {
		t.Error("Key should be deleted from all levels")
	}
}

func TestMultiLevelCache_Exists(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	// Set only in L2
	l2.Set(ctx, "key1", "value1", time.Minute)

	exists, err := cache.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist in L2")
	}
}

func TestMultiLevelCache_MGet(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	// Set key1 in L1, key2 in L2
	l1.Set(ctx, "key1", "value1", time.Minute)
	l2.Set(ctx, "key2", "value2", time.Minute)

	result, err := cache.MGet(ctx, []string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
}

func TestMultiLevelCache_MSet(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := cache.MSet(ctx, items, time.Minute)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Both levels should have the values
	exists1, _ := l1.Exists(ctx, "key1")
	exists2, _ := l2.Exists(ctx, "key1")

	if !exists1 || !exists2 {
		t.Error("Both levels should have the values")
	}
}

func TestMultiLevelCache_MDelete(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Set(ctx, "key2", "value2", time.Minute)

	err := cache.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}

	exists1, _ := l1.Exists(ctx, "key1")
	exists2, _ := l2.Exists(ctx, "key1")

	if exists1 || exists2 {
		t.Error("Keys should be deleted from all levels")
	}
}

func TestMultiLevelCache_Incr(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	val, err := cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}
}

func TestMultiLevelCache_IncrBy(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	val, err := cache.IncrBy(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 5 {
		t.Errorf("Expected 5, got %d", val)
	}
}

func TestMultiLevelCache_Decr(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	l2.Set(ctx, "counter", int64(10), time.Minute)

	val, err := cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}
	if val != 9 {
		t.Errorf("Expected 9, got %d", val)
	}
}

func TestMultiLevelCache_DecrBy(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	l2.Set(ctx, "counter", int64(20), time.Minute)

	val, err := cache.DecrBy(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("DecrBy failed: %v", err)
	}
	if val != 15 {
		t.Errorf("Expected 15, got %d", val)
	}
}

func TestMultiLevelCache_SetNX(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	ok, err := cache.SetNX(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if !ok {
		t.Error("First SetNX should succeed")
	}

	// Both levels should have the value
	exists1, _ := l1.Exists(ctx, "key1")
	exists2, _ := l2.Exists(ctx, "key1")

	if !exists1 || !exists2 {
		t.Error("Both levels should have the value after SetNX")
	}

	// Second SetNX should fail
	ok, err = cache.SetNX(ctx, "key1", "value2", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if ok {
		t.Error("Second SetNX should fail")
	}
}

func TestMultiLevelCache_Expire(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	cache.Set(ctx, "key1", "value1", time.Hour)

	err := cache.Expire(ctx, "key1", time.Minute)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}
}

func TestMultiLevelCache_TTL(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	cache.Set(ctx, "key1", "value1", time.Minute)

	ttl, err := cache.TTL(ctx, "key1")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}
}

func TestMultiLevelCache_Ping(t *testing.T) {
	l1 := NewMemoryCache()
	l2 := NewMemoryCache()
	ctx := context.Background()

	cache := NewMultiLevelCache(WithLevels(l1, l2))
	defer cache.Close()

	err := cache.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestMultiLevelCache_EmptyLevels(t *testing.T) {
	ctx := context.Background()

	cache := NewMultiLevelCache() // No levels
	defer cache.Close()

	// Operations on empty cache should return gracefully
	var result string
	err := cache.Get(ctx, "key1", &result)
	if err == nil {
		t.Fatal("Expected error for empty cache")
	}

	val, _ := cache.Incr(ctx, "counter")
	if val != 0 {
		t.Errorf("Expected 0 for empty cache, got %d", val)
	}

	ok, _ := cache.SetNX(ctx, "key1", "value1", time.Minute)
	if ok {
		t.Error("Expected false for empty cache")
	}

	ttl, _ := cache.TTL(ctx, "key1")
	if ttl != -2 {
		t.Errorf("Expected -2 for empty cache, got %v", ttl)
	}
}
