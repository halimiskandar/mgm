package rest

import (
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
	"net/http"

	"github.com/AMFarhan21/fres"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type (
	PaymentsHandler struct {
		validate        *validator.Validate
		paymentsService PaymentsService
	}

	PaymentsService interface {
		CreatePayment(data domain.Payments, isWallet bool, user_id uint) (domain.PaymentWithLink, error)
		GetAllPayments() ([]domain.Payments, error)
		GetPayment(payment_id int) (domain.Payments, error)
		UpdatePayment(data domain.Payments, user_id, productId int, request WebhookRequest, purpose string) error
		DeletePayment(payment_id int) error
	}

	PaymentsInput struct {
		OrderID  int   `json:"order_id" validate:"required"`
		IsWallet *bool `json:"is_wallet" validate:"required"`
	}
)

func NewPaymentsHandler(paymentsService PaymentsService) *PaymentsHandler {
	return &PaymentsHandler{
		validate:        validator.New(),
		paymentsService: paymentsService,
	}
}

func (h *PaymentsHandler) CreatePayment(c echo.Context) error {
	user_id := c.Get("user_id").(uint)

	var request PaymentsInput

	if err := c.Bind(&request); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validate.Struct(&request); err != nil {
		logger.Error("Failed to validation create payment validation", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	payment, err := h.paymentsService.CreatePayment(domain.Payments{
		OrderID: request.OrderID,
	}, *request.IsWallet, user_id)
	if err != nil {
		logger.Error("Failed to create order items", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, fres.Response.StatusCreated(payment))
}
