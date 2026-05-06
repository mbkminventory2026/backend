// @title           Permatatex IT Inventory API
// @version         1.0
// @description     API documentation for Permatatex IT Inventory System backend.
// @termsOfService  https://swagger.io/terms/
// @contact.name    API Support
// @contact.email   support@permatatex.local
// @license.name    MIT
// @host            localhost:8080
// @BasePath        /
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "permatatex-inventory/docs"
	"permatatex-inventory/internal/config"
	httpdelivery "permatatex-inventory/internal/delivery/http"
	"permatatex-inventory/internal/entity"
	turnstilegateway "permatatex-inventory/internal/gateway/turnstile"
	"permatatex-inventory/internal/usecase"
)

const (
	serverReadTimeout  = 10 * time.Second
	serverWriteTimeout = 15 * time.Second
	serverIdleTimeout  = 60 * time.Second
	shutdownTimeout    = 10 * time.Second
)

func main() {
	logger := newJSONLogger()
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	dbPool, err := config.NewPGXPool(cfg)
	if err != nil {
		logger.Error("failed to initialize database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info(
		"database pool initialized",
		slog.Int("max_conns", int(cfg.DBMaxConns)),
		slog.Int("min_conns", int(cfg.DBMinConns)),
	)

	if err := httpdelivery.SetupValidator(); err != nil {
		logger.Error("failed to setup validator", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	// 1. Gateways
	turnstileGateway, err := turnstilegateway.NewTurnstileGateway(cfg.TurnstileSecret)
	if err != nil {
		logger.Error("failed to initialize turnstile gateway", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	// 2. Repository/Queries
	queries := entity.New(dbPool)

	// 3. Usecases
	turnstileUseCase, err := usecase.NewTurnstileUseCase(turnstileGateway)
	if err != nil {
		logger.Error("failed to initialize turnstile usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	authUseCase := usecase.NewAuthUseCase(queries, turnstileUseCase, cfg.JWTSecret)
	userUseCase, err := usecase.NewUserUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize user usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	masterDataUseCase, err := usecase.NewMasterDataUseCase(queries)
	if err != nil {
		logger.Error("failed to initialize master data usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	// 4. Handlers
	authHandler, err := httpdelivery.NewAuthHandler(authUseCase)
	if err != nil {
		logger.Error("failed to initialize auth handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	userHandler, err := httpdelivery.NewUserHandler(userUseCase)
	if err != nil {
		logger.Error("failed to initialize user handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	masterDataHandler, err := httpdelivery.NewMasterDataHandler(masterDataUseCase)
	if err != nil {
		logger.Error("failed to initialize master data handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	healthHandler := httpdelivery.NewHealthHandler(dbPool)

	// 5. Routes
	docs.SwaggerInfo.Host = "localhost:" + cfg.ServerPort
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}

	router := gin.New()
	router.Use(httpdelivery.ErrorHandlerMiddleware())
	router.Use(corsMiddleware(cfg.CORSAllowOrigin))

	healthHandler.RegisterRoutes(router)

	authMiddleware := httpdelivery.AuthMiddleware(cfg.JWTSecret)

	authHandler.RegisterRoutes(
		router,
		authMiddleware,
		httpdelivery.NewLoginRateLimitMiddleware(
			cfg.LoginRateLimitMaxAttempts,
			time.Duration(cfg.LoginRateLimitWindowSec)*time.Second,
		),
	)

	userHandler.RegisterRoutes(router, authMiddleware)
	masterDataHandler.RegisterRoutes(router, authMiddleware)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server := &stdhttp.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("web server starting", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErrCh:
		logger.Error("server stopped unexpectedly", slog.String("error", err.Error()))
		dbPool.Close()
		logger.Info("database pool closed")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("force close failed", slog.String("error", closeErr.Error()))
		}
		dbPool.Close()
		logger.Info("database pool closed")
		os.Exit(1)
	}

	dbPool.Close()
	logger.Info("database pool closed")
	logger.Info("server shutdown complete")
}

func newJSONLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler)
}

func corsMiddleware(allowOrigin string) gin.HandlerFunc {
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", allowOrigin)
		headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization")
		headers.Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == stdhttp.MethodOptions {
			c.AbortWithStatus(stdhttp.StatusNoContent)
			return
		}

		c.Next()
	}
}
