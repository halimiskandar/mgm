package router

import (
	"myGreenMarket/internal/middleware"
	"myGreenMarket/internal/rest"

	"github.com/labstack/echo/v4"
)

func SetupUserRoutes(api *echo.Group, handler *rest.UserHandler) {
	users := api.Group("/users")

	users.GET("/email-verification/:code", handler.VerifyEmail)
	users.POST("/register", handler.Register)
	users.POST("/login", handler.Login)
}

func SetupProductRoutes(api *echo.Group, handler *rest.ProductHandler, authRequired echo.MiddlewareFunc, adminOnly echo.MiddlewareFunc) {
	products := api.Group("/products")

	products.GET("", handler.GetAllProducts, authRequired)
	products.GET("/:id", handler.GetProductByID, authRequired)
	products.POST("", handler.CreateProduct, authRequired, adminOnly)
	products.PUT("/:id", handler.UpdateProduct, authRequired, adminOnly)
	products.DELETE("/:id", handler.DeleteProduct, authRequired, adminOnly)

}

func SetOrdersRoutes(api *echo.Group, ordersHandler *rest.OrdersHandler) {
	orders := api.Group("/orders", middleware.AuthMiddleware())
	orders.POST("", ordersHandler.CreateOrderItem)
	orders.GET("", ordersHandler.GetAllOrders)
	orders.GET("/:id", ordersHandler.GetOrderByID)
	orders.PUT("/:id", ordersHandler.UpdateOrder)
	orders.DELETE("/:id", ordersHandler.DeleteOrder)

}

func SetPaymentsRoutes(api *echo.Group, paymentsHandler *rest.PaymentsHandler) {
	payments := api.Group("/payments", middleware.AuthMiddleware())
	payments.POST("", paymentsHandler.CreatePayment)
	payments.POST("/topup", paymentsHandler.TopUp)
	payments.GET("/:id", paymentsHandler.GetPaymentsByID)
	payments.GET("", paymentsHandler.GetAllPayments)
	api.GET("/paid", paymentsHandler.PaidResponse)
}

func SetWebhookHandler(api *echo.Group, webhookHandler *rest.WebhookController) {
	webhook := api.Group("/webhook")
	webhook.POST("/handler", webhookHandler.HandleWebhook)
}
