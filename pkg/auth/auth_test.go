package auth

import (
	"testing"
	"time"
)

func TestClaims_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"expired", time.Now().Add(-1 * time.Hour), true},
		{"not expired", time.Now().Add(1 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{ExpiresAt: tt.expiresAt}
			if got := claims.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaims_HasRole(t *testing.T) {
	claims := &Claims{Roles: []string{"admin", "user"}}

	if !claims.HasRole("admin") {
		t.Error("HasRole(admin) should return true")
	}
	if claims.HasRole("guest") {
		t.Error("HasRole(guest) should return false")
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{Code: 10001, Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Error() = %s, want test error", err.Error())
	}
}

func TestPredefinedErrors(t *testing.T) {
	if ErrTokenInvalid.Code != 10001 {
		t.Errorf("ErrTokenInvalid.Code = %d, want 10001", ErrTokenInvalid.Code)
	}
	if ErrTokenExpired.Code != 10002 {
		t.Errorf("ErrTokenExpired.Code = %d, want 10002", ErrTokenExpired.Code)
	}
	if ErrTokenRevoked.Code != 10003 {
		t.Errorf("ErrTokenRevoked.Code = %d, want 10003", ErrTokenRevoked.Code)
	}
}
