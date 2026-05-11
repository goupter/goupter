package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/examples/http-api/model"
	"github.com/goupter/goupter/examples/http-api/pkg/token"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userModel    *model.UsersModel
	tokenManager *token.Manager
}

func NewAuthHandler(userModel *model.UsersModel, tokenManager *token.Manager) *AuthHandler {
	return &AuthHandler{
		userModel:    userModel,
		tokenManager: tokenManager,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Username     string `json:"username"`
	Role         string `json:"role"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
		return
	}

	user, err := h.userModel.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		response.Error(c, errors.New(errors.CodeUnauthorized, "invalid credentials"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Error(c, errors.New(errors.CodeUnauthorized, "invalid credentials"))
		return
	}

	// Create tokens
	accessToken, err := h.tokenManager.CreateAccessToken(user.ID, user.Username)
	if err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to generate token"))
		return
	}

	refreshToken, err := h.tokenManager.CreateRefreshToken(user.ID)
	if err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to generate refresh token"))
		return
	}

	// Save user info to cache (compatible with ag-go)
	userInfo := &token.UserInfo{
		UserId:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Status:   user.Status,
	}
	if err := h.tokenManager.RedisSaveUserInfo(userInfo); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to save user session"))
		return
	}

	response.Success(c, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     user.Username,
		Role:         user.Role,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context, user *token.UserInfo) {
	if err := h.tokenManager.RedisDeleteUserInfo(user.UserId); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to logout"))
		return
	}
	response.Success(c, gin.H{"message": "logged out"})
}

// Me returns current user info
func (h *AuthHandler) Me(c *gin.Context, user *token.UserInfo) {
	response.Success(c, gin.H{
		"user_id":  user.UserId,
		"username": user.Username,
		"role":     user.Role,
	})
}

// RefreshToken refreshes access token using refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	claims, err := h.tokenManager.ParseToken(tokenString)
	if err != nil {
		response.Error(c, errors.New(errors.CodeUnauthorized, "invalid refresh token"))
		return
	}

	// Get user from database to ensure user still exists
	user, err := h.userModel.FindOne(c.Request.Context(), map[string]any{"id": claims.UserId})
	if err != nil {
		response.Error(c, errors.New(errors.CodeUnauthorized, "user not found"))
		return
	}

	// Create new access token
	accessToken, err := h.tokenManager.CreateAccessToken(user.ID, user.Username)
	if err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to generate token"))
		return
	}

	// Update user info in cache (compatible with ag-go)
	userInfo := &token.UserInfo{
		UserId:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Status:   user.Status,
	}
	if err := h.tokenManager.RedisSaveUserInfo(userInfo); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to save user session"))
		return
	}

	response.Success(c, gin.H{
		"access_token": accessToken,
	})
}
