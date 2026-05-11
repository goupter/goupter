package main

import (
	"fmt"
	"log"

	"github.com/goupter/goupter/examples/http-api/model"
	"github.com/goupter/goupter/examples/http-api/pkg/token"
	"github.com/goupter/goupter/pkg/app"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/config"
	pkglog "github.com/goupter/goupter/pkg/log"
	"github.com/goupter/goupter/pkg/server"
	"github.com/goupter/goupter/pkg/server/middleware"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load(
		config.WithConfigFile("config"),
		config.WithConfigPaths("./cmd/api/config", "."),
	)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize logger
	logger, err := pkglog.NewZapLogger(&pkglog.LoggerConfig{
		Level:  pkglog.ParseLevel(cfg.Log.Level),
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	pkglog.SetDefault(logger)

	// 3. Initialize database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 4. Initialize cache (Redis)
	redisCache, err := cache.NewRedisCache(
		cache.WithRedisConfig(&cfg.Redis),
	)
	if err != nil {
		log.Fatalf("Failed to connect Redis: %v", err)
	}

	// 5. Initialize token manager (supports both secret_key and secretkey)
	tokenConfig := token.NewConfigFromMap(cfg.Auth.Config)
	tokenManager := token.NewManager(tokenConfig, redisCache)

	// 6. Initialize models
	userModel := model.NewUsersModel(db)
	articleModel := model.NewArticlesModel(db)

	// 7. Create HTTP server with middleware
	httpServer := server.NewHTTPServer(
		server.WithHTTPConfig(&cfg.Server.HTTP),
		server.WithHTTPLogger(logger),
		server.WithMiddleware(
			middleware.Recovery(logger),
			middleware.Logger(logger),
			middleware.Security(),
			middleware.CORS(),
		),
	)

	// 8. Create health manager
	healthManager := app.NewHealthManager(app.DefaultHealthConfig())
	healthManager.RegisterReadiness(app.NewPingChecker("cache", redisCache.Ping))

	// 9. Create application
	application := app.New(
		app.WithName(cfg.App.Name),
		app.WithConfig(cfg),
		app.WithLogger(logger),
		app.WithHTTPServer(httpServer),
		app.WithDatabase(db),
		app.WithCache(redisCache),
		app.WithHealthManager(healthManager),
		app.WithMetricsConfig(app.DefaultMetricsConfig()),
		app.WithPProfConfig(app.DefaultPProfConfig()),
		app.BeforeStart(func() error {
			logger.Info("Application starting...")
			return nil
		}),
		app.AfterStart(func() error {
			logger.Info("Application started",
				pkglog.String("name", cfg.App.Name),
				pkglog.String("addr", cfg.HTTPAddr()),
			)
			return nil
		}),
		app.BeforeStop(func() error {
			logger.Info("Application stopping...")
			return nil
		}),
	)

	// 10. Register routes
	registerRoutes(application, userModel, articleModel, tokenManager, redisCache)

	// 11. Run application
	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
