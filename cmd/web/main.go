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

	"permatatex-inventory/internal/config"
	httpdelivery "permatatex-inventory/internal/delivery/http"
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

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware(cfg.CORSAllowOrigin))

	healthHandler := httpdelivery.NewHealthHandler()
	healthHandler.RegisterRoutes(router)

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
