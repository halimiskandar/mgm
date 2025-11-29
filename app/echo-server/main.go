package main

import (
	"context"
	"fmt"
	"log"
	"myGreenMarket/app/echo-server/router"
	"myGreenMarket/business/category"
	"myGreenMarket/business/orders"
	"myGreenMarket/business/payments"
	userService "myGreenMarket/business/user"
	"myGreenMarket/internal/middleware"
	"myGreenMarket/internal/repository/notification"
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
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.App.Environment)
	logger.Info("Starting MyGreenMarket", "version", cfg.App.Version)

	db, err := database.InitPostgres(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}

	logger.Info("Database connected successfully")

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
	ordersRepo := psqlRepo.NewOrdersRepository(db)
	productsRepo := psqlRepo.NewProductRepository(db)
	paymentsRepo := psqlRepo.NewPaymentsRepository(db)
	categoryRepo := psqlRepo.NewCategoryRepository(db)

	// Init service
	userService := userService.NewUserService(userRepo, validate, mailjetEmail, cfg.App.AppEmailVerificationKey, cfg.App.AppDeploymentUrl)
	ordersService := orders.NewOrdersService(ordersRepo, productsRepo)
	paymentsService := payments.NewPaymentsService(paymentsRepo, xenditRepo, userRepo, ordersRepo, productsRepo)
	categoryService := category.NewCategoryService(categoryRepo)

	// Init handler
	userHandler := rest.NewUserHandler(userService)
	ordersHandler := rest.NewOrdersHandler(ordersService)
	paymentsHandler := rest.NewPaymentsHandler(paymentsService)
	webhookHandler := rest.NewWebhookController(paymentsService)
	categoryHandler := rest.NewCategoryHandler(categoryService)

	// Init echo
	e := echo.New()
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

	// Auth middleware
	// authRequired := middleware.AuthMiddleware()
	// Setup routes
	api := e.Group("/api/v1")
	router.SetupUserRoutes(api, userHandler)
	router.SetOrdersRoutes(api, ordersHandler)
	router.SetPaymentsRoutes(api, paymentsHandler)
	router.SetWebhookHandler(api, webhookHandler)
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
