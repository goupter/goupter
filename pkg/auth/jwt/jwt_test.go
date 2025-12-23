package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/goupter/goupter/pkg/auth"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SecretKey == "" {
		t.Error("SecretKey should have default value")
	}
	if cfg.AccessTokenTTL != 2*time.Hour {
		t.Errorf("AccessTokenTTL = %v, want 2h", cfg.AccessTokenTTL)
	}
}

func TestPlugin_Init(t *testing.T) {
	p := &Plugin{}
	config := map[string]interface{}{
		"secret_key":        "test-secret-key",
		"issuer":            "test-issuer",
		"access_token_ttl":  "1h",
		"blacklist_enabled": true,
	}

	err := p.Init(config)
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}
	if p.config.SecretKey != "test-secret-key" {
		t.Errorf("SecretKey = %s, want test-secret-key", p.config.SecretKey)
	}
}

func TestAuthenticator_GenerateAndAuthenticate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SecretKey = "test-secret"
	a := NewAuthenticator(cfg)

	claims := &auth.Claims{
		UserID:   "user-123",
		Username: "testuser",
		Roles:    []string{"admin"},
	}

	token, err := a.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	ctx := context.Background()
	result, err := a.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if result.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", result.UserID)
	}
}

func TestAuthenticator_InvalidToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SecretKey = "test-secret"
	a := NewAuthenticator(cfg)

	ctx := context.Background()
	_, err := a.Authenticate(ctx, "invalid-token")
	if err != auth.ErrTokenInvalid {
		t.Errorf("Authenticate(invalid) error = %v, want ErrTokenInvalid", err)
	}
}

func TestAuthenticator_RefreshToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SecretKey = "test-secret"
	a := NewAuthenticator(cfg)

	claims := &auth.Claims{UserID: "user-123", Username: "testuser"}
	token, _ := a.GenerateToken(claims)

	newToken, err := a.RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if newToken == "" {
		t.Error("New token should not be empty")
	}
}

func TestAuthenticator_RevokeToken_WithBlacklist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SecretKey = "test-secret"
	cfg.BlacklistEnabled = true

	mockCache := &mockCache{data: make(map[string]interface{})}
	a := NewAuthenticator(cfg, WithBlacklist(mockCache))

	token, _ := a.GenerateToken(&auth.Claims{UserID: "test"})

	ctx := context.Background()
	err := a.RevokeToken(ctx, token)
	if err != nil {
		t.Errorf("RevokeToken error = %v", err)
	}

	_, err = a.Authenticate(ctx, token)
	if err != auth.ErrTokenRevoked {
		t.Errorf("Authenticate revoked token error = %v, want ErrTokenRevoked", err)
	}
}

// mockCache 模拟缓存
type mockCache struct {
	data map[string]interface{}
}

func (c *mockCache) Get(ctx context.Context, key string, value interface{}) error { return nil }
func (c *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.data[key] = value
	return nil
}
func (c *mockCache) Delete(ctx context.Context, key string) error { delete(c.data, key); return nil }
func (c *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := c.data[key]
	return ok, nil
}
func (c *mockCache) Expire(ctx context.Context, key string, ttl time.Duration) error { return nil }
func (c *mockCache) TTL(ctx context.Context, key string) (time.Duration, error)      { return 0, nil }
func (c *mockCache) Incr(ctx context.Context, key string) (int64, error)             { return 0, nil }
func (c *mockCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}
func (c *mockCache) Decr(ctx context.Context, key string) (int64, error) { return 0, nil }
func (c *mockCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}
func (c *mockCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	return nil, nil
}
func (c *mockCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	return nil
}
func (c *mockCache) MDelete(ctx context.Context, keys []string) error { return nil }
func (c *mockCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return true, nil
}
func (c *mockCache) Close() error                    { return nil }
func (c *mockCache) Ping(ctx context.Context) error { return nil }
