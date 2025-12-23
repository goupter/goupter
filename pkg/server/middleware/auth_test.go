package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthInfo_HasRole(t *testing.T) {
	info := &AuthInfo{
		Roles: []string{"admin", "user"},
	}

	if !info.HasRole("admin") {
		t.Error("应该有admin角色")
	}
	if !info.HasRole("user") {
		t.Error("应该有user角色")
	}
	if info.HasRole("guest") {
		t.Error("不应该有guest角色")
	}
}

func TestAuthInfo_HasAnyRole(t *testing.T) {
	info := &AuthInfo{
		Roles: []string{"admin"},
	}

	if !info.HasAnyRole("admin", "user") {
		t.Error("应该匹配admin")
	}
	if info.HasAnyRole("user", "guest") {
		t.Error("不应该匹配任何角色")
	}
}

func TestAuthInfo_HasAllRoles(t *testing.T) {
	info := &AuthInfo{
		Roles: []string{"admin", "user", "editor"},
	}

	if !info.HasAllRoles("admin", "user") {
		t.Error("应该有admin和user")
	}
	if info.HasAllRoles("admin", "guest") {
		t.Error("没有guest角色")
	}
}

func TestAuthInfo_HasScope(t *testing.T) {
	info := &AuthInfo{
		Scopes: []string{"read", "write"},
	}

	if !info.HasScope("read") {
		t.Error("应该有read权限")
	}
	if info.HasScope("delete") {
		t.Error("不应该有delete权限")
	}
}

func TestAuthInfo_IsExpired(t *testing.T) {
	info := &AuthInfo{}
	if info.IsExpired() {
		t.Error("未设置过期时间不应该过期")
	}

	info.ExpiresAt = time.Now().Add(time.Hour)
	if info.IsExpired() {
		t.Error("未来时间不应该过期")
	}

	info.ExpiresAt = time.Now().Add(-time.Hour)
	if !info.IsExpired() {
		t.Error("过去时间应该过期")
	}
}

func TestBearerAuth_Success(t *testing.T) {
	validator := TokenValidatorFunc(func(token string) (*AuthInfo, error) {
		if token == "valid_token" {
			return &AuthInfo{
				UserID:   "user123",
				Username: "testuser",
			}, nil
		}
		return nil, ErrInvalidToken
	})

	router := gin.New()
	router.Use(BearerAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		info, _ := GetAuthInfo(c)
		c.JSON(http.StatusOK, gin.H{"user": info.Username})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}
}

func TestBearerAuth_MissingToken(t *testing.T) {
	validator := TokenValidatorFunc(func(token string) (*AuthInfo, error) {
		return &AuthInfo{}, nil
	})

	router := gin.New()
	router.Use(BearerAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望401，实际: %d", w.Code)
	}
}

func TestBearerAuth_InvalidToken(t *testing.T) {
	validator := TokenValidatorFunc(func(token string) (*AuthInfo, error) {
		return nil, ErrInvalidToken
	})

	router := gin.New()
	router.Use(BearerAuth(validator))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望401，实际: %d", w.Code)
	}
}

func TestBearerAuth_SkipPaths(t *testing.T) {
	validator := TokenValidatorFunc(func(token string) (*AuthInfo, error) {
		return nil, ErrInvalidToken
	})

	cfg := BearerAuthConfig{
		Validator: validator,
		SkipPaths: []string{"/health", "/public"},
	}

	router := gin.New()
	router.Use(BearerAuthWithConfig(cfg))
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "Protected")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health应该跳过认证，状态码: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("/protected应该需要认证，状态码: %d", w.Code)
	}
}

func TestRequireRoles(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		SetAuthInfo(c, &AuthInfo{
			UserID: "user1",
			Roles:  []string{"user", "editor"},
		})
		c.Next()
	})

	router.GET("/admin", RequireRoles("admin"), func(c *gin.Context) {
		c.String(http.StatusOK, "Admin")
	})
	router.GET("/user", RequireRoles("user"), func(c *gin.Context) {
		c.String(http.StatusOK, "User")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("/admin应该返回403，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/user", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/user应该返回200，实际: %d", w.Code)
	}
}

func TestRequireAnyRole(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		SetAuthInfo(c, &AuthInfo{
			UserID: "user1",
			Roles:  []string{"editor"},
		})
		c.Next()
	})

	router.GET("/test", RequireAnyRole("admin", "editor"), func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200（有editor角色），实际: %d", w.Code)
	}
}

func TestRequireScopes(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		SetAuthInfo(c, &AuthInfo{
			UserID: "user1",
			Scopes: []string{"read", "write"},
		})
		c.Next()
	})

	router.GET("/read", RequireScopes("read"), func(c *gin.Context) {
		c.String(http.StatusOK, "Read")
	})
	router.GET("/delete", RequireScopes("delete"), func(c *gin.Context) {
		c.String(http.StatusOK, "Delete")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/read", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/read应该返回200，实际: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/delete", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("/delete应该返回403，实际: %d", w.Code)
	}
}
