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
	aigateway "permatatex-inventory/internal/gateway/ai"
	turnstilegateway "permatatex-inventory/internal/gateway/turnstile"
	"permatatex-inventory/internal/usecase"
	excel "permatatex-inventory/pkg/exporter/excel"
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

	aiGateway := aigateway.NewGateway("http://host.docker.internal:8000")

	// 2. Repository/Queries
	queries := entity.New(dbPool)

	// 3. Usecases
	turnstileUseCase, err := usecase.NewTurnstileUseCase(turnstileGateway)
	if err != nil {
		logger.Error("failed to initialize turnstile usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	auditLogUseCase, err := usecase.NewAuditLogUseCase(queries)
	if err != nil {
		logger.Error("failed to initialize audit log usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	authUseCase := usecase.NewAuthUseCase(queries, dbPool, turnstileUseCase, auditLogUseCase, cfg.JWTSecret)

	userUseCase, err := usecase.NewUserUseCase(queries, dbPool, auditLogUseCase)
	if err != nil {
		logger.Error("failed to initialize user usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	roleUseCase, err := usecase.NewRoleUseCase(queries, dbPool, auditLogUseCase)
	if err != nil {
		logger.Error("failed to initialize role usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	masterDataUseCase, err := usecase.NewMasterDataUseCase(queries, auditLogUseCase)
	if err != nil {
		logger.Error("failed to initialize master data usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	profilPerusahaanUseCase, err := usecase.NewProfilPerusahaanUseCase(queries)
	if err != nil {
		logger.Error("failed to initialize profil perusahaan usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	transactionDocumentUseCase, err := usecase.NewTransactionDocumentUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize transaction document usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	workOrderProductionUseCase, err := usecase.NewWorkOrderProductionUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize work order production usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	materialListUseCase, err := usecase.NewMaterialListUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize material list usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	timelineProduksiUseCase, err := usecase.NewTimelineProduksiUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize timeline produksi usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	markerPlanUseCase, err := usecase.NewMarkerPlanUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize marker plan usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	spreadingCuttingPlanUseCase, err := usecase.NewSpreadingCuttingPlanUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize spreading cutting plan usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	warehouseDeliveryUseCase, err := usecase.NewWarehouseDeliveryUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize warehouse delivery usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	dashboardUseCase, err := usecase.NewDashboardUseCase(queries, aiGateway)
	if err != nil {
		logger.Error("failed to initialize dashboard usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	approvalUseCase, err := usecase.NewApprovalUseCase(queries, dbPool)
	if err != nil {
		logger.Error("failed to initialize approval usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	reportUseCase, err := usecase.NewReportUseCase(queries)
	if err != nil {
		logger.Error("failed to initialize report usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	productionMasterUseCase, err := usecase.NewProductionMasterUseCase(queries)
	if err != nil {
		logger.Error("failed to initialize production master usecase", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	excelRenderer, err := excel.NewRenderer(cfg.ExportTemplateDir)
	if err != nil {
		logger.Error("failed to initialize excel renderer", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	excelExportUseCase, err := usecase.NewExcelExportUseCase(excelRenderer)
	if err != nil {
		logger.Error("failed to initialize excel export usecase", slog.String("error", err.Error()))
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

	roleHandler, err := httpdelivery.NewRoleHandler(roleUseCase)
	if err != nil {
		logger.Error("failed to initialize role handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	masterDataHandler, err := httpdelivery.NewMasterDataHandler(masterDataUseCase)
	if err != nil {
		logger.Error("failed to initialize master data handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	profilPerusahaanHandler, err := httpdelivery.NewProfilPerusahaanHandler(profilPerusahaanUseCase)
	if err != nil {
		logger.Error("failed to initialize profil perusahaan handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	transactionDocumentHandler, err := httpdelivery.NewTransactionDocumentHandler(transactionDocumentUseCase)
	if err != nil {
		logger.Error("failed to initialize transaction document handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	workOrderProductionHandler, err := httpdelivery.NewWorkOrderProductionHandler(workOrderProductionUseCase)
	if err != nil {
		logger.Error("failed to initialize work order production handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	materialListHandler, err := httpdelivery.NewMaterialListHandler(materialListUseCase)
	if err != nil {
		logger.Error("failed to initialize material list handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	timelineProduksiHandler, err := httpdelivery.NewTimelineProduksiHandler(timelineProduksiUseCase)
	if err != nil {
		logger.Error("failed to initialize timeline produksi handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	productionMasterHandler, err := httpdelivery.NewProductionMasterHandler(productionMasterUseCase)
	if err != nil {
		logger.Error("failed to initialize production master handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	markerPlanHandler, err := httpdelivery.NewMarkerPlanHandler(markerPlanUseCase)
	if err != nil {
		logger.Error("failed to initialize marker plan handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	spreadingCuttingPlanHandler, err := httpdelivery.NewSpreadingCuttingPlanHandler(spreadingCuttingPlanUseCase)
	if err != nil {
		logger.Error("failed to initialize spreading cutting plan handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	warehouseDeliveryHandler, err := httpdelivery.NewWarehouseDeliveryHandler(warehouseDeliveryUseCase)
	if err != nil {
		logger.Error("failed to initialize warehouse delivery handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	healthHandler := httpdelivery.NewHealthHandler(dbPool)

	dashboardHandler, err := httpdelivery.NewDashboardHandler(dashboardUseCase)
	if err != nil {
		logger.Error("failed to initialize dashboard handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	approvalHandler, err := httpdelivery.NewApprovalHandler(approvalUseCase)
	if err != nil {
		logger.Error("failed to initialize approval handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	reportHandler, err := httpdelivery.NewReportHandler(reportUseCase)
	if err != nil {
		logger.Error("failed to initialize report handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	excelExportHandler, err := httpdelivery.NewExcelExportHandler(excelExportUseCase)
	if err != nil {
		logger.Error("failed to initialize excel export handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	auditLogHandler, err := httpdelivery.NewAuditLogHandler(auditLogUseCase)
	if err != nil {
		logger.Error("failed to initialize audit log handler", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	activityLogService, err := usecase.NewActivityLogService(queries, logger)
	if err != nil {
		logger.Error("failed to initialize activity log service", slog.String("error", err.Error()))
		dbPool.Close()
		os.Exit(1)
	}

	// 5. Routes
	docs.SwaggerInfo.Host = "localhost:" + cfg.ServerPort
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}

	router := gin.New()
	router.Use(httpdelivery.ErrorHandlerMiddleware())
	router.Use(corsMiddleware(cfg.CORSAllowOrigin))
	router.Use(httpdelivery.ActivityLogMiddleware(activityLogService))

	// Serve uploaded files statically
	router.Static("/uploads", "./uploads")

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
	roleHandler.RegisterRoutes(router, authMiddleware)
	masterDataHandler.RegisterRoutes(router, authMiddleware)
	profilPerusahaanHandler.RegisterRoutes(router, authMiddleware)
	transactionDocumentHandler.RegisterRoutes(router, authMiddleware)
	workOrderProductionHandler.RegisterRoutes(router, authMiddleware)
	materialListHandler.RegisterRoutes(router, authMiddleware)
	timelineProduksiHandler.RegisterRoutes(router, authMiddleware)
	// Note: Register production master alongside other master data routes
	productionMasterGroup := router.Group("/api/v1/master").Use(authMiddleware)
	{
		productionMasterGroup.GET("/production-lines", productionMasterHandler.ListProductionLines)
		productionMasterGroup.GET("/production-lines/:id", productionMasterHandler.GetProductionLineByID)
		productionMasterGroup.POST("/production-lines", productionMasterHandler.CreateProductionLine)
		productionMasterGroup.PUT("/production-lines/:id", productionMasterHandler.UpdateProductionLine)
		productionMasterGroup.DELETE("/production-lines/:id", productionMasterHandler.DeleteProductionLine)

		productionMasterGroup.GET("/production-status-plans", productionMasterHandler.ListProductionStatusPlans)
		productionMasterGroup.GET("/production-status-plans/:id", productionMasterHandler.GetProductionStatusPlanByID)
		productionMasterGroup.POST("/production-status-plans", productionMasterHandler.CreateProductionStatusPlan)
		productionMasterGroup.PUT("/production-status-plans/:id", productionMasterHandler.UpdateProductionStatusPlan)
		productionMasterGroup.DELETE("/production-status-plans/:id", productionMasterHandler.DeleteProductionStatusPlan)
	}

	markerPlanHandler.RegisterRoutes(router, authMiddleware)
	spreadingCuttingPlanHandler.RegisterRoutes(router, authMiddleware)
	warehouseDeliveryHandler.RegisterRoutes(router, authMiddleware)
	approvalHandler.RegisterRoutes(router, authMiddleware)

	dashboardHandler.RegisterRoutes(router, authMiddleware)
	reportHandler.RegisterRoutes(router, authMiddleware)
	excelExportHandler.RegisterRoutes(router, authMiddleware)
	auditLogHandler.RegisterRoutes(router, authMiddleware)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server := &stdhttp.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
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

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		if logErr := activityLogService.Shutdown(shutdownCtx); logErr != nil {
			logger.Error("activity log service shutdown failed", slog.String("error", logErr.Error()))
		}
		cancelShutdown()

		dbPool.Close()
		logger.Info("database pool closed")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("force close failed", slog.String("error", closeErr.Error()))
		}
		if logErr := activityLogService.Shutdown(ctx); logErr != nil {
			logger.Error("activity log service shutdown failed", slog.String("error", logErr.Error()))
		}
		dbPool.Close()
		logger.Info("database pool closed")
		os.Exit(1)
	}

	if err := activityLogService.Shutdown(ctx); err != nil {
		logger.Error("activity log service shutdown failed", slog.String("error", err.Error()))
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
		headers.Set("Access-Control-Expose-Headers", "X-Total-Count")
		headers.Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == stdhttp.MethodOptions {
			c.AbortWithStatus(stdhttp.StatusNoContent)
			return
		}

		c.Next()
	}
}
