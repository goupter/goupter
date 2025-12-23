package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 认证相关错误
var (
	ErrMissingToken      = errors.New("missing authentication token")
	ErrInvalidToken      = errors.New("invalid authentication token")
	ErrTokenExpired      = errors.New("token has expired")
	ErrInsufficientScope = errors.New("insufficient scope")
)

// AuthContextKey 认证上下文键
type AuthContextKey struct{}

// AuthInfo 认证信息
type AuthInfo struct {
	UserID    string
	Username  string
	Roles     []string
	Scopes    []string
	Extra     map[string]interface{}
	ExpiresAt time.Time
}

// HasRole 检查是否有指定角色
func (a *AuthInfo) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole 检查是否有任意一个角色
func (a *AuthInfo) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if a.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles 检查是否有所有角色
func (a *AuthInfo) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !a.HasRole(role) {
			return false
		}
	}
	return true
}

// HasScope 检查是否有指定权限
func (a *AuthInfo) HasScope(scope string) bool {
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsExpired 检查是否过期
func (a *AuthInfo) IsExpired() bool {
	return !a.ExpiresAt.IsZero() && time.Now().After(a.ExpiresAt)
}

// GetAuthInfo 从Gin上下文获取认证信息
func GetAuthInfo(c *gin.Context) (*AuthInfo, bool) {
	if info, exists := c.Get("auth_info"); exists {
		if authInfo, ok := info.(*AuthInfo); ok {
			return authInfo, true
		}
	}
	return nil, false
}

// GetAuthInfoFromContext 从标准上下文获取认证信息
func GetAuthInfoFromContext(ctx context.Context) (*AuthInfo, bool) {
	if info := ctx.Value(AuthContextKey{}); info != nil {
		if authInfo, ok := info.(*AuthInfo); ok {
			return authInfo, true
		}
	}
	return nil, false
}

// SetAuthInfo 设置认证信息到上下文
func SetAuthInfo(c *gin.Context, info *AuthInfo) {
	c.Set("auth_info", info)
	ctx := context.WithValue(c.Request.Context(), AuthContextKey{}, info)
	c.Request = c.Request.WithContext(ctx)
}

// TokenValidator Token验证器接口
type TokenValidator interface {
	Validate(token string) (*AuthInfo, error)
}

// TokenValidatorFunc 函数类型的Token验证器
type TokenValidatorFunc func(token string) (*AuthInfo, error)

func (f TokenValidatorFunc) Validate(token string) (*AuthInfo, error) {
	return f(token)
}

// BearerAuthConfig Bearer认证配置
type BearerAuthConfig struct {
	Validator    TokenValidator
	Realm        string
	SkipPaths    []string
	ErrorHandler func(*gin.Context, error)
}

// BearerAuth Bearer Token认证中间件
func BearerAuth(validator TokenValidator) gin.HandlerFunc {
	return BearerAuthWithConfig(BearerAuthConfig{
		Validator: validator,
		Realm:     "Authorization Required",
	})
}

// BearerAuthWithConfig 带配置的Bearer认证中间件
func BearerAuthWithConfig(cfg BearerAuthConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			handleAuthError(c, cfg, ErrMissingToken)
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			handleAuthError(c, cfg, ErrInvalidToken)
			return
		}

		token := strings.TrimPrefix(auth, prefix)
		if token == "" {
			handleAuthError(c, cfg, ErrInvalidToken)
			return
		}

		authInfo, err := cfg.Validator.Validate(token)
		if err != nil {
			handleAuthError(c, cfg, err)
			return
		}

		if authInfo.IsExpired() {
			handleAuthError(c, cfg, ErrTokenExpired)
			return
		}

		SetAuthInfo(c, authInfo)
		c.Next()
	}
}

func handleAuthError(c *gin.Context, cfg BearerAuthConfig, err error) {
	if cfg.ErrorHandler != nil {
		cfg.ErrorHandler(c, err)
		return
	}

	realm := cfg.Realm
	if realm == "" {
		realm = "Authorization Required"
	}

	c.Header("WWW-Authenticate", `Bearer realm="`+realm+`"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    http.StatusUnauthorized,
		"message": err.Error(),
	})
}

// RequireRoles 要求指定角色（需要所有角色）
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authInfo, ok := GetAuthInfo(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			})
			return
		}

		if !authInfo.HasAllRoles(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequireAnyRole 要求任意一个角色
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authInfo, ok := GetAuthInfo(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			})
			return
		}

		if !authInfo.HasAnyRole(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequireScopes 要求指定权限范围
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authInfo, ok := GetAuthInfo(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authentication required",
			})
			return
		}

		for _, scope := range scopes {
			if !authInfo.HasScope(scope) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    http.StatusForbidden,
					"message": ErrInsufficientScope.Error(),
				})
				return
			}
		}

		c.Next()
	}
}
