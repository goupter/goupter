package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	if cache == nil {
		t.Fatal("NewMemoryCache should return non-nil cache")
	}
	defer cache.Close()
}

func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	// Test string value
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

	// Test struct value
	type User struct {
		Name string
		Age  int
	}
	user := User{Name: "test", Age: 25}
	err = cache.Set(ctx, "user1", user, time.Minute)
	if err != nil {
		t.Fatalf("Set struct failed: %v", err)
	}

	var userResult User
	err = cache.Get(ctx, "user1", &userResult)
	if err != nil {
		t.Fatalf("Get struct failed: %v", err)
	}
	if userResult.Name != "test" || userResult.Age != 25 {
		t.Errorf("Expected user {test, 25}, got %+v", userResult)
	}
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "nonexistent", &result)
	if err == nil {
		t.Fatal("Expected error for nonexistent key")
	}

	if !IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Delete(ctx, "key1")

	var result string
	err := cache.Get(ctx, "key1", &result)
	if err == nil {
		t.Fatal("Expected error after delete")
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Minute)

	exists, err := cache.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	exists, err = cache.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Minute)

	ttl, err := cache.TTL(ctx, "key1")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("Expected TTL between 0 and 1 minute, got %v", ttl)
	}

	// Test key without TTL
	cache.Set(ctx, "key2", "value2", 0)
	ttl, err = cache.TTL(ctx, "key2")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl != -1 {
		t.Errorf("Expected TTL -1 for key without expiry, got %v", ttl)
	}

	// Test nonexistent key
	ttl, err = cache.TTL(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl != -2 {
		t.Errorf("Expected TTL -2 for nonexistent key, got %v", ttl)
	}
}

func TestMemoryCache_Expire(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Hour)
	cache.Expire(ctx, "key1", time.Second)

	ttl, _ := cache.TTL(ctx, "key1")
	if ttl > time.Second {
		t.Errorf("Expected TTL <= 1 second, got %v", ttl)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", 50*time.Millisecond)

	// Should exist initially
	var result string
	err := cache.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	err = cache.Get(ctx, "key1", &result)
	if err == nil {
		t.Fatal("Expected error after expiration")
	}
}

func TestMemoryCache_MGet(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Set(ctx, "key2", "value2", time.Minute)

	result, err := cache.MGet(ctx, []string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
}

func TestMemoryCache_MSet(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := cache.MSet(ctx, items, time.Minute)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result string
	cache.Get(ctx, "key1", &result)
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}
}

func TestMemoryCache_MDelete(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", time.Minute)
	cache.Set(ctx, "key2", "value2", time.Minute)

	err := cache.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}

	exists, _ := cache.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should be deleted")
	}
}

func TestMemoryCache_Incr(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	val, err := cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	val, err = cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 2 {
		t.Errorf("Expected 2, got %d", val)
	}
}

func TestMemoryCache_IncrBy(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	val, err := cache.IncrBy(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 5 {
		t.Errorf("Expected 5, got %d", val)
	}

	val, err = cache.IncrBy(ctx, "counter", 10)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 15 {
		t.Errorf("Expected 15, got %d", val)
	}
}

func TestMemoryCache_Decr(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "counter", int64(10), time.Minute)

	val, err := cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}
	if val != 9 {
		t.Errorf("Expected 9, got %d", val)
	}
}

func TestMemoryCache_DecrBy(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "counter", int64(20), time.Minute)

	val, err := cache.DecrBy(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("DecrBy failed: %v", err)
	}
	if val != 15 {
		t.Errorf("Expected 15, got %d", val)
	}
}

func TestMemoryCache_SetNX(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	// First SetNX should succeed
	ok, err := cache.SetNX(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if !ok {
		t.Error("First SetNX should succeed")
	}

	// Second SetNX should fail
	ok, err = cache.SetNX(ctx, "key1", "value2", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if ok {
		t.Error("Second SetNX should fail")
	}

	// Value should be original
	var result string
	cache.Get(ctx, "key1", &result)
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}
}

func TestMemoryCache_Ping(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	err := cache.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			cache.Set(ctx, key, n, time.Minute)
			var result int
			cache.Get(ctx, key, &result)
		}(i)
	}
	wg.Wait()
}

func TestMemoryCache_WithCleanupInterval(t *testing.T) {
	cache := NewMemoryCache(WithCleanupInterval(50 * time.Millisecond))
	defer cache.Close()
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1", 30*time.Millisecond)

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	exists, _ := cache.Exists(ctx, "key1")
	if exists {
		t.Error("Expected key to be cleaned up")
	}
}
