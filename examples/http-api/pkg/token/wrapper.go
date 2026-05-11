package token

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/response"
)

// WrapUser wraps a handler that requires user info
func WrapUser(f func(*gin.Context, *UserInfo)) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			return
		}
		f(c, user)
	}
}

// WrapJSON wraps a handler that requires JSON body
func WrapJSON[T any](f func(*gin.Context, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T
		if err := c.ShouldBindJSON(&param); err != nil {
			response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
			return
		}
		f(c, &param)
	}
}

// WrapJSONUser wraps a handler that requires both user info and JSON body
func WrapJSONUser[T any](f func(*gin.Context, *UserInfo, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			return
		}
		var param T
		if err := c.ShouldBindJSON(&param); err != nil {
			response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
			return
		}
		f(c, user, &param)
	}
}

// WrapUserPaging wraps a handler that requires user info and pagination
func WrapUserPaging(f func(*gin.Context, *UserInfo, int, int)) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
		f(c, user, page, pageSize)
	}
}

// WrapRole wraps a middleware that checks user role
func WrapRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}
		for _, role := range roles {
			if user.Role == role {
				c.Next()
				return
			}
		}
		response.Error(c, errors.New(errors.CodeForbidden, "permission denied"))
		c.Abort()
	}
}

// WrapPermission wraps a middleware that checks user permission (via UserAuth field, compatible with ag-go)
func WrapPermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}
		// Check if user has "ALL" permission or specific permission (using UserAuth like ag-go)
		if user.UserAuth == "ALL" || contains(user.UserAuth, permission) {
			c.Next()
			return
		}
		response.Error(c, errors.New(errors.CodeForbidden, "permission denied"))
		c.Abort()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
