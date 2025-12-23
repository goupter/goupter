package auth

import (
	"context"
	"testing"
)

type mockAuthenticator struct{}

func (m *mockAuthenticator) Authenticate(ctx context.Context, token string) (*Claims, error) {
	return &Claims{UserID: "test"}, nil
}

func (m *mockAuthenticator) GenerateToken(claims *Claims) (string, error) {
	return "test-token", nil
}

func (m *mockAuthenticator) RefreshToken(token string) (string, error) {
	return "new-token", nil
}

func (m *mockAuthenticator) RevokeToken(ctx context.Context, token string) error {
	return nil
}

type mockPlugin struct {
	name string
	auth Authenticator
}

func (p *mockPlugin) Name() string                            { return p.name }
func (p *mockPlugin) Init(config map[string]interface{}) error { return nil }
func (p *mockPlugin) Authenticator() Authenticator             { return p.auth }

// cleanup helper for tests
func unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(registry, name)
}

func TestRegister(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin-1", auth: &mockAuthenticator{}}
	defer unregister("test-plugin-1")

	Register(plugin)

	got, ok := Get("test-plugin-1")
	if !ok {
		t.Fatal("Plugin should be registered")
	}
	if got.Name() != "test-plugin-1" {
		t.Errorf("Name() = %s, want test-plugin-1", got.Name())
	}
}

func TestGet(t *testing.T) {
	plugin := &mockPlugin{name: "test-get-1", auth: &mockAuthenticator{}}
	defer unregister("test-get-1")

	Register(plugin)

	got, ok := Get("test-get-1")
	if !ok {
		t.Fatal("Plugin should exist")
	}
	if got == nil {
		t.Error("Plugin should not be nil")
	}

	_, ok = Get("non-existent")
	if ok {
		t.Error("Plugin should not exist")
	}
}

func TestNewAuthenticator(t *testing.T) {
	plugin := &mockPlugin{name: "test-new-auth", auth: &mockAuthenticator{}}
	defer unregister("test-new-auth")

	Register(plugin)

	auth, err := NewAuthenticator("test-new-auth", nil)
	if err != nil {
		t.Errorf("NewAuthenticator() error = %v", err)
	}
	if auth == nil {
		t.Error("NewAuthenticator should return authenticator")
	}

	_, err = NewAuthenticator("definitely-not-exist", nil)
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}
