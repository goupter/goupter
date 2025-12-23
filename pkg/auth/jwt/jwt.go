package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/goupter/goupter/pkg/auth"
	"github.com/goupter/goupter/pkg/cache"
)

const PluginName = "jwt"

func init() {
	auth.Register(&Plugin{})
}

// Plugin JWT鉴权插件
type Plugin struct {
	config        *Config
	authenticator *Authenticator
}

// Config JWT配置
type Config struct {
	SecretKey        string        `json:"secret_key"`
	Issuer           string        `json:"issuer"`
	AccessTokenTTL   time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `json:"refresh_token_ttl"`
	BlacklistEnabled bool          `json:"blacklist_enabled"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		SecretKey:        "change-me-in-production",
		Issuer:           "goupter",
		AccessTokenTTL:   2 * time.Hour,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		BlacklistEnabled: false,
	}
}

func (p *Plugin) Name() string { return PluginName }

func (p *Plugin) Init(config map[string]interface{}) error {
	p.config = DefaultConfig()

	if v, ok := config["secret_key"].(string); ok {
		p.config.SecretKey = v
	}
	if v, ok := config["issuer"].(string); ok {
		p.config.Issuer = v
	}
	if v, ok := config["access_token_ttl"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			p.config.AccessTokenTTL = d
		}
	}
	if v, ok := config["refresh_token_ttl"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			p.config.RefreshTokenTTL = d
		}
	}
	if v, ok := config["blacklist_enabled"].(bool); ok {
		p.config.BlacklistEnabled = v
	}

	p.authenticator = NewAuthenticator(p.config)
	return nil
}

func (p *Plugin) Authenticator() auth.Authenticator {
	return p.authenticator
}

// Authenticator JWT鉴权器
type Authenticator struct {
	config    *Config
	blacklist cache.Cache
}

type Option func(*Authenticator)

// WithBlacklist 设置黑名单缓存
func WithBlacklist(c cache.Cache) Option {
	return func(a *Authenticator) { a.blacklist = c }
}

// NewAuthenticator 创建JWT鉴权器
func NewAuthenticator(config *Config, opts ...Option) *Authenticator {
	a := &Authenticator{config: config}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID   string                 `json:"user_id"`
	Username string                 `json:"username"`
	Roles    []string               `json:"roles"`
	Extra    map[string]interface{} `json:"extra"`
}

func (a *Authenticator) Authenticate(ctx context.Context, tokenString string) (*auth.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.SecretKey), nil
	})

	if err != nil {
		return nil, auth.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, auth.ErrTokenInvalid
	}

	if a.config.BlacklistEnabled && a.blacklist != nil {
		exists, _ := a.blacklist.Exists(ctx, "jwt:blacklist:"+tokenString)
		if exists {
			return nil, auth.ErrTokenRevoked
		}
	}

	return &auth.Claims{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Roles:     claims.Roles,
		Extra:     claims.Extra,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (a *Authenticator) GenerateToken(claims *auth.Claims) (string, error) {
	now := time.Now()
	expiresAt := now.Add(a.config.AccessTokenTTL)

	jwtClaims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID:   claims.UserID,
		Username: claims.Username,
		Roles:    claims.Roles,
		Extra:    claims.Extra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(a.config.SecretKey))
}

func (a *Authenticator) RefreshToken(refreshToken string) (string, error) {
	token, err := jwt.ParseWithClaims(refreshToken, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return "", auth.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return "", auth.ErrTokenInvalid
	}

	return a.GenerateToken(&auth.Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Roles:    claims.Roles,
		Extra:    claims.Extra,
	})
}

func (a *Authenticator) RevokeToken(ctx context.Context, tokenString string) error {
	if !a.config.BlacklistEnabled || a.blacklist == nil {
		return nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.SecretKey), nil
	})
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return auth.ErrTokenInvalid
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}

	return a.blacklist.Set(ctx, "jwt:blacklist:"+tokenString, true, ttl)
}
