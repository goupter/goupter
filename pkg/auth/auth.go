package auth

import (
	"context"
	"time"
)

// Authenticator 鉴权器接口
type Authenticator interface {
	// Authenticate 验证令牌
	Authenticate(ctx context.Context, token string) (*Claims, error)
	// GenerateToken 生成令牌
	GenerateToken(claims *Claims) (string, error)
	// RefreshToken 刷新令牌
	RefreshToken(token string) (string, error)
	// RevokeToken 撤销令牌
	RevokeToken(ctx context.Context, token string) error
}

// Claims 认证信息
type Claims struct {
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Roles     []string               `json:"roles"`
	Extra     map[string]interface{} `json:"extra"`
	IssuedAt  time.Time              `json:"issued_at"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// IsExpired 检查令牌是否过期
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// HasRole 检查是否拥有指定角色
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AuthError 鉴权错误
type AuthError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// 预定义鉴权错误
var (
	ErrTokenInvalid = &AuthError{Code: 10001, Message: "token invalid"}
	ErrTokenExpired = &AuthError{Code: 10002, Message: "token expired"}
	ErrTokenRevoked = &AuthError{Code: 10003, Message: "token revoked"}
)
