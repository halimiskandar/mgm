package router

import (
	"myGreenMarket/internal/middleware"
	"myGreenMarket/internal/rest"

	"github.com/labstack/echo/v4"
)

func SetupUserRoutes(api *echo.Group, handler *rest.UserHandler, ordersHandler *rest.OrdersHandler, paymentsHandler *rest.PaymentsHandler) {
	users := api.Group("/users")

	users.GET("/email-verification/:code", handler.VerifyEmail)
	users.POST("/register", handler.Register)
	users.POST("/login", handler.Login)

	orders := api.Group("/orders", middleware.AuthMiddleware())
	orders.POST("", ordersHandler.CreateOrderItem)

	payments := api.Group("/payments", middleware.AuthMiddleware())
	payments.POST("", paymentsHandler.CreatePayment)

	// api.POST("/webhook/handler", webhookHandler.HandleWebhook)

}
