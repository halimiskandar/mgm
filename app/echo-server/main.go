package main

import (
	"context"
	"fmt"
	"log"
	"myGreenMarket/app/echo-server/router"
	"myGreenMarket/business/bandit"
	"myGreenMarket/business/category"
	"myGreenMarket/business/mockreco"
	"myGreenMarket/business/orders"
	"myGreenMarket/business/payments"
	"myGreenMarket/business/product"
	userService "myGreenMarket/business/user"
	"myGreenMarket/internal/middleware"
	"myGreenMarket/internal/repository/notification"
	redisRepo "myGreenMarket/internal/repository/redis"
	"myGreenMarket/pkg/database/redis"
	"myGreenMarket/pkg/metrics"

	psqlRepo "myGreenMarket/internal/repository/postgres"
	"myGreenMarket/internal/repository/xendit"
	"myGreenMarket/internal/rest"
	"myGreenMarket/pkg/config"
	"myGreenMarket/pkg/database"
	"myGreenMarket/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	metrics.Init()

	logger.Init(cfg.App.Environment)
	logger.Info("Starting MyGreenMarket", "version", cfg.App.Version)

	db, err := database.InitPostgres(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	logger.Info("Database connected successfully")

	// init redis
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.CloseRedisClient(redisClient)
	logger.Info("Successfully connected to Redis")

	// Init notification from mailjet
	mailjetEmail := notification.NewMailjetRepository(
		notification.MailjetConfig{
			MailjetBaseURL:           cfg.Mailjet.MailjetBaseUrl,
			MailjetBasicAuthUsername: cfg.Mailjet.MailjetBasicAuthUsername,
			MailjetBasicAuthPassword: cfg.Mailjet.MailjetBasicAuthPassword,
			MailjetSenderEmail:       cfg.Mailjet.MailjetSenderEmail,
			MailjetSenderName:        cfg.Mailjet.MailjetSenderName,
		},
	)

	xenditRepo := xendit.NewXenditRepository(
		xendit.XenditConfig{
			XenditApi:          cfg.Xendit.XenditSecretKey,
			XenditUrl:          cfg.Xendit.XenditUrl,
			SuccessRedirectUrl: cfg.Xendit.RedirectUrl,
			FailureRedirectUrl: cfg.Xendit.RedirectUrl,
		},
	)

	// Init validate
	validate := validator.New()

	// Init repo
	userRepo := psqlRepo.NewUserRepository(db)
	tokenRepo := redisRepo.NewTokenRepository(redisClient)
	ordersRepo := psqlRepo.NewOrdersRepository(db)
	productsRepo := psqlRepo.NewProductRepository(db)
	paymentsRepo := psqlRepo.NewPaymentsRepository(db)
	banditRepo := psqlRepo.NewBanditRepository(db)
	mockRecoRepo := psqlRepo.NewMockRecommendationRepository(db)
	cfgRepo := psqlRepo.NewBanditConfigRepository(db)
	segmentRepo := psqlRepo.NewUserSegmentRepository(db)
	categoryRepo := psqlRepo.NewCategoryRepository(db)

	// Init service
	userService := userService.NewUserService(userRepo, tokenRepo, validate, mailjetEmail, cfg.App.AppEmailVerificationKey, cfg.App.AppDeploymentUrl)
	ordersService := orders.NewOrdersService(ordersRepo, productsRepo)
	paymentsService := payments.NewPaymentsService(paymentsRepo, xenditRepo, userRepo, ordersRepo, productsRepo)
	productService := product.NewProductService(productsRepo)
	categoryService := category.NewCategoryService(categoryRepo)

	eligChecker := bandit.NoopEligibilityChecker{}
	defaultCfg := bandit.DefaultConfig()
	banditService := bandit.NewBanditService(
		banditRepo,   // BanditRepository (events + state)
		productsRepo, // ProductRepository
		banditRepo,   // BanditStateRepository (state)
		eligChecker,  // EligibilityChecker
		mockRecoRepo, // OfflineRecommendationRepository
		cfgRepo,      // ConfigRepository
		segmentRepo,  // SegmentRepository
		defaultCfg,   // base Config
	)
	mockRecoService := mockreco.NewService(mockRecoRepo)

	// Init handler
	userHandler := rest.NewUserHandler(userService)
	productHandler := rest.NewProductHandler(productService)
	ordersHandler := rest.NewOrdersHandler(ordersService)
	paymentsHandler := rest.NewPaymentsHandler(paymentsService)
	webhookHandler := rest.NewWebhookHandler(paymentsService, cfg.Xendit.XenditWebhookVerificationToken)
	banditHandler := rest.NewBanditHandler(banditService)
	mockRecoHandler := rest.NewMockRecommendationHandler(mockRecoService)
	banditAdminHandler := rest.NewBanditAdminHandler(cfgRepo, segmentRepo)
	categoryHandler := rest.NewCategoryHandler(categoryService)

	// Init echo
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			traceID := req.Header.Get("X-Request-Id")
			if traceID == "" {
				traceID = uuid.NewString()
			}

			ctx = context.WithValue(ctx, bandit.TraceIDKey, traceID)
			c.SetRequest(req.WithContext(ctx))

			// optionally add to response header
			c.Response().Header().Set("X-Request-Id", traceID)

			return next(c)
		}
	})
	e.HideBanner = true
	e.HidePort = true

	// HTTP error handler
	e.HTTPErrorHandler = middleware.ErrorHandler

	// Global middleware
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Auth middleware
	authRequired := middleware.AuthMiddleware()
	adminOnly := middleware.AdminOnly()
	selfOrAdmin := middleware.SelfOrAdmin()

	// Redis-based auth middleware (untuk route yang memerlukan validasi Redis)
	authWithRedis := middleware.AuthMiddlewareWithRedis(userService)

	// Setup routes
	api := e.Group("/api/v1")
	router.SetupUserRoutes(api, userHandler, authWithRedis, selfOrAdmin, adminOnly)
	router.SetupProductRoutes(api, productHandler, authRequired, adminOnly)
	router.SetOrdersRoutes(api, ordersHandler)
	router.SetPaymentsRoutes(api, paymentsHandler)
	router.SetWebhookHandler(api, webhookHandler)
	router.SetBanditRoutes(api, banditHandler)
	router.SetBanditAdminRoutes(api, banditAdminHandler)
	router.SetMockRecommendationRoutes(api, mockRecoHandler)
	router.SetupCategoryRoutes(api, categoryHandler)
	router.SetPaymentsRoutes(api, paymentsHandler)
	router.SetWebhookHandler(api, webhookHandler)

	// Goroutine server
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		logger.Info("Server starting", "address", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
	}

	logger.Info("Server stopped")
}
