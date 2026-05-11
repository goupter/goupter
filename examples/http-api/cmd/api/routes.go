package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/examples/http-api/cmd/api/handler"
	"github.com/goupter/goupter/examples/http-api/model"
	"github.com/goupter/goupter/examples/http-api/pkg/token"
	"github.com/goupter/goupter/pkg/app"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/response"
)

func registerRoutes(a *app.App, userModel *model.UsersModel, articleModel *model.ArticlesModel, tokenManager *token.Manager, c cache.Cache) {
	s := a.HTTPServer()

	// Root endpoint
	s.GET("/", func(ctx *gin.Context) {
		cfg := a.Config()
		response.Success(ctx, gin.H{
			"service": cfg.App.Name,
			"version": cfg.App.Version,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Create handlers
	authHandler := handler.NewAuthHandler(userModel, tokenManager)
	articleHandler := handler.NewArticleHandler(articleModel, c)

	// API v1 group
	api := s.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/health", handler.Health)

		// Auth endpoints (public)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.RefreshToken)

		// Protected endpoints (require JWT)
		protected := api.Group("")
		protected.Use(tokenManager.JWTAuth())
		{
			// User endpoints
			protected.GET("/me", token.WrapUser(authHandler.Me))
			protected.POST("/auth/logout", token.WrapUser(authHandler.Logout))

			// Article endpoints (using wrapper functions)
			protected.POST("/articles", token.WrapJSONUser[handler.CreateArticleRequest](articleHandler.Create))
			protected.GET("/articles", token.WrapUserPaging(articleHandler.List))
			protected.GET("/articles/:id", articleHandler.Get)
			protected.PUT("/articles/:id", token.WrapJSONUser[handler.UpdateArticleRequest](articleHandler.Update))
			protected.DELETE("/articles/:id", token.WrapUser(articleHandler.Delete))
		}

		// Admin endpoints (require admin role)
		admin := api.Group("/admin")
		admin.Use(tokenManager.JWTAuth())
		admin.Use(token.WrapRole("admin"))
		{
			admin.GET("/stats", func(ctx *gin.Context) {
				response.Success(ctx, gin.H{
					"message": "Admin only endpoint",
					"stats":   "This would show admin statistics",
				})
			})
		}
	}
}
