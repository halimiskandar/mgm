package rest

import (
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
	"net/http"
	"strconv"

	"github.com/AMFarhan21/fres"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type (
	OrdersHandler struct {
		validate      *validator.Validate
		ordersService OrdersService
	}

	OrdersService interface {
		CreateOrder(data domain.Orders) (domain.Orders, error)
		GetAllOrders(user_id int) ([]domain.Orders, error)
		GetOrder(order_id, user_id int) (domain.Orders, error)
		GetOrderStatus(status string, user_id int) (domain.Orders, error)
		UpdateOrder(data domain.Orders) error
		DeleteOrder(order_id, user_id int) error
	}

	OrdersInput struct {
		ProductID int `json:"product_id" validate:"required"`
		Quantity  int `json:"quantity" validate:"required"`
	}

	UpdateInput struct {
		Quantity int `json:"quantity" validate:"required"`
	}
)

func NewOrdersHandler(ordersService OrdersService) *OrdersHandler {
	return &OrdersHandler{
		validate:      validator.New(),
		ordersService: ordersService,
	}
}

func (h *OrdersHandler) CreateOrderItem(c echo.Context) error {
	user_id := c.Get("user_id").(uint)

	var request OrdersInput

	if err := c.Bind(&request); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validate.Struct(&request); err != nil {
		logger.Error("Failed to validation order items validation", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	orderItem, err := h.ordersService.CreateOrder(domain.Orders{
		UserID:    int(user_id),
		ProductID: request.ProductID,
		Quantity:  request.Quantity,
	})
	if err != nil {
		logger.Error("Failed to create order items", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, fres.Response.StatusCreated(orderItem))
}

func (h *OrdersHandler) GetAllOrders(c echo.Context) error {
	user_id := c.Get("user_id").(uint)
	orders, err := h.ordersService.GetAllOrders(int(user_id))
	if err != nil {
		logger.Error("Failed to get all orders", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(orders))
}

func (h *OrdersHandler) GetOrderByID(c echo.Context) error {
	id := c.Param("id")
	order_id, _ := strconv.Atoi(id)

	user_id := c.Get("user_id").(uint)
	orders, err := h.ordersService.GetOrder(order_id, int(user_id))
	if err != nil {
		logger.Error("Failed to get order by id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(orders))
}

func (h *OrdersHandler) UpdateOrder(c echo.Context) error {
	id := c.Param("id")
	order_id, _ := strconv.Atoi(id)

	user_id := c.Get("user_id").(uint)

	var request UpdateInput

	if err := c.Bind(&request); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validate.Struct(&request); err != nil {
		logger.Error("Failed to validation order items validation", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	err := h.ordersService.UpdateOrder(domain.Orders{
		ID:       order_id,
		UserID:   int(user_id),
		Quantity: request.Quantity,
	})
	if err != nil {
		logger.Error("Failed to update order", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK("Order updated successfully"))
}

func (h *OrdersHandler) DeleteOrder(c echo.Context) error {
	id := c.Param("id")
	order_id, _ := strconv.Atoi(id)
	user_id := c.Get("user_id").(uint)
	err := h.ordersService.DeleteOrder(order_id, int(user_id))
	if err != nil {
		logger.Error("Failed to delete order", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK("Order deleted successfully"))
}
