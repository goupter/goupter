package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/response"
)

const (
	AccessTokenTTL  = 24 * time.Hour
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// Config token configuration
type Config struct {
	SecretKey       string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// DefaultConfig returns default token configuration
func DefaultConfig() *Config {
	return &Config{
		SecretKey:       "secretkey",
		Issuer:          "goupter",
		AccessTokenTTL:  AccessTokenTTL,
		RefreshTokenTTL: RefreshTokenTTL,
	}
}

// NewConfigFromMap creates Config from a map (supports both secret_key and secretkey)
func NewConfigFromMap(m map[string]any) *Config {
	cfg := DefaultConfig()

	// Support both secret_key and secretkey
	if v, ok := m["secret_key"].(string); ok && v != "" {
		cfg.SecretKey = v
	} else if v, ok := m["secretkey"].(string); ok && v != "" {
		cfg.SecretKey = v
	}

	// Support both issuer formats
	if v, ok := m["issuer"].(string); ok && v != "" {
		cfg.Issuer = v
	}

	// Support access_token_ttl / accessTokenTtl
	if v, ok := m["access_token_ttl"].(string); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.AccessTokenTTL = d
		}
	} else if v, ok := m["accessTokenTtl"].(string); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.AccessTokenTTL = d
		}
	}

	// Support refresh_token_ttl / refreshTokenTtl
	if v, ok := m["refresh_token_ttl"].(string); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RefreshTokenTTL = d
		}
	} else if v, ok := m["refreshTokenTtl"].(string); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RefreshTokenTTL = d
		}
	}

	return cfg
}

// Claims custom JWT claims (compatible with ag-go MyClaims)
type Claims struct {
	UserId int64 `json:"user"` // Same as ag-go: json tag is "user"
	jwt.RegisteredClaims
}

// UserInfo cached user information (compatible with ag-go User struct)
type UserInfo struct {
	UserId   int64  `json:"userId"`
	UserAuth string `json:"auths,omitempty"`
	UserType int    `json:"userType,omitempty"`
	TeamType int    `json:"teamType,omitempty"`
	DataAuth int    `json:"dataAuth,omitempty"`
	IsEnable int    `json:"isEnable,omitempty"`

	AccountId           int64  `json:"accountId"`
	AccountCatalogId    int64  `json:"accountCatalogId,omitempty"`
	ParentAccountId     int64  `json:"parentAccountId,omitempty"`
	VipLevel            int    `json:"vipLevel,omitempty"`
	CheckStatus         int    `json:"checkStatus,omitempty"`
	AccountCatalogLevel int    `json:"accountCatalogLevel,omitempty"`
	Tag                 int    `json:"tag,omitempty"`
	AccountType         int    `json:"accountType,omitempty"`
	CanCreateSub        bool   `json:"canCreateSub,omitempty"`
	Province            string `json:"province,omitempty"`
	County              string `json:"county,omitempty"`
	City                string `json:"city,omitempty"`

	Groups           map[int64]int `json:"groups"`
	AllSubAccountIds []int64       `json:"allSubAccountIds"`

	// Extended fields for http-api
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"`
	Status   int8   `json:"status,omitempty"`
}

// Manager token manager
type Manager struct {
	config *Config
	cache  cache.Cache
}

// NewManager creates a new token manager
func NewManager(cfg *Config, c cache.Cache) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Manager{
		config: cfg,
		cache:  c,
	}
}

// CreateAccessToken creates an access token (compatible with ag-go)
func (m *Manager) CreateAccessToken(userId int64, subject string) (string, error) {
	claims := &Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// CreateRefreshToken creates a refresh token (compatible with ag-go)
func (m *Manager) CreateRefreshToken(userId int64) (string, error) {
	claims := &Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ParseToken parses and validates a token
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// RedisSaveUserInfo saves user info to Redis (compatible with ag-go)
// Key format: token:{userId} (same as ag-go)
// No TTL - permanent storage (same as ag-go)
func (m *Manager) RedisSaveUserInfo(user *UserInfo) error {
	key := fmt.Sprintf("token:%d", user.UserId)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// No TTL - permanent storage like ag-go
	return m.cache.Set(context.Background(), key, data, 0)
}

// RedisGetUserInfo gets user info from Redis (compatible with ag-go)
// Key format: token:{userId}
func (m *Manager) RedisGetUserInfo(userId int64) *UserInfo {
	key := fmt.Sprintf("token:%d", userId)
	var data []byte
	if err := m.cache.Get(context.Background(), key, &data); err != nil {
		return nil
	}
	var user UserInfo
	if err := json.Unmarshal(data, &user); err != nil {
		return nil
	}
	return &user
}

// RedisDeleteUserInfo deletes user info from Redis
func (m *Manager) RedisDeleteUserInfo(userId int64) error {
	key := fmt.Sprintf("token:%d", userId)
	return m.cache.Delete(context.Background(), key)
}

// Context key for user info
const ContextKeyUser = "user" // Same as ag-go: c.Set("user", user)

// JWTAuth middleware for JWT authentication (compatible with ag-go JWTAuth)
func (m *Manager) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		h := c.Request.Header.Get("Authorization")
		var tokenString = ""
		if len(h) > 7 {
			tokenString = h[7:]
		}
		if tokenString == "" {
			response.Error(c, errors.New(errors.CodeUnauthorized, "missing token"))
			c.Abort()
			return
		}

		// Parse JWT token
		claims, err := m.ParseToken(tokenString)
		if err != nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}

		// Get user info from Redis (same as ag-go: redis_get_user_info)
		user := m.RedisGetUserInfo(claims.UserId)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "user session not found"))
			c.Abort()
			return
		}

		// Set user to context (same as ag-go: c.Set("user", user))
		c.Set(ContextKeyUser, user)
		c.Next()
	}
}

// GetUser gets user info from gin context (compatible with ag-go get_user)
func GetUser(c *gin.Context) *UserInfo {
	u, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil
	}
	user, ok := u.(*UserInfo)
	if !ok {
		return nil
	}
	return user
}
