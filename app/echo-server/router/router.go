package router

import (
	"myGreenMarket/internal/middleware"
	"myGreenMarket/internal/rest"

	"github.com/labstack/echo/v4"
)

func SetupUserRoutes(api *echo.Group, handler *rest.UserHandler, authRequired echo.MiddlewareFunc, adminOnly echo.MiddlewareFunc) {
	users := api.Group("/users")

	users.GET("/email-verification/:code", handler.VerifyEmail)
	users.POST("/register", handler.Register)
	users.POST("/login", handler.Login)

	users.PUT("/:id", handler.UpdateUser, authRequired)
	users.GET("", handler.GetAllUsers, authRequired, adminOnly)
	users.GET("/:id", handler.GetUserByID, authRequired, adminOnly)
	users.DELETE("/:id", handler.DeleteUser, authRequired, adminOnly)
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

func SetWebhookHandler(api *echo.Group, webhookHandler *rest.WebhookHandler) {
	webhook := api.Group("/webhook")
	webhook.POST("/handler", webhookHandler.HandleWebhook)
}

func SetBanditRoutes(api *echo.Group, handler *rest.BanditHandler) {
	reco := api.Group("/recommendations", middleware.AuthMiddleware())
	reco.GET("", handler.Recommend)
	reco.GET("/debug", handler.DebugRecommend)
	reco.POST("/feedback", handler.Feedback)
}

func SetMockRecommendationRoutes(api *echo.Group, h *rest.MockRecommendationHandler) {
	grp := api.Group("/mock-recommendations")
	grp.GET("", h.Get)
}

func SetBanditAdminRoutes(api *echo.Group, handler *rest.BanditAdminHandler) {

	admin := api.Group("/admin/bandit", middleware.AuthMiddleware())

	admin.GET("/config", handler.GetConfig)
	admin.PUT("/config", handler.UpsertConfig)
	admin.GET("/segment", handler.GetSegment)
	admin.PUT("/segment", handler.UpsertSegment)
}

func SetupCategoryRoutes(api *echo.Group, handler *rest.CategoryHandler) {
	categories := api.Group("/categories")

	categories.GET("", handler.GetAllCategories)
	categories.GET("/:id", handler.GetCategoryByID)
	categories.POST("", handler.CreateCategory)
	categories.PUT("/:id", handler.UpdateCategory)
	categories.DELETE("/:id", handler.DeleteCategory)
}
