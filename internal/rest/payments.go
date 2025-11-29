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
	PaymentsHandler struct {
		validate        *validator.Validate
		paymentsService PaymentsService
	}

	PaymentsService interface {
		CreatePayment(data domain.Payments, isWallet bool, user_id uint) (domain.PaymentWithLink, error)
		GetAllPayments(user_id int) ([]domain.Payments, error)
		GetPayment(payment_id, user_id int) (domain.Payments, error)
		ReceivePaymentWebhook(request WebhookRequest) error
		DeletePayment(payment_id int) error
		TopUp(user_id uint, amount float64) (domain.TopUp, error)
	}

	PaymentsInput struct {
		OrderID  int   `json:"order_id" validate:"required"`
		IsWallet *bool `json:"is_wallet" validate:"required"`
	}

	TopUpInput struct {
		Amount float64 `json:"amount" validate:"required"`
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
		UserID:  int(user_id),
		OrderID: &request.OrderID,
	}, *request.IsWallet, user_id)
	if err != nil {
		logger.Error("Failed to create order items", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, fres.Response.StatusCreated(payment))
}

func (h *PaymentsHandler) GetPaymentsByID(c echo.Context) error {
	id := c.Param("id")
	payment_id, _ := strconv.Atoi(id)

	user_id := c.Get("user_id").(uint)

	payment, err := h.paymentsService.GetPayment(payment_id, int(user_id))
	if err != nil {
		logger.Error("Failed to get payment by id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(payment))
}

func (h *PaymentsHandler) GetAllPayments(c echo.Context) error {
	user_id := c.Get("user_id").(uint)

	payments, err := h.paymentsService.GetAllPayments(int(user_id))
	if err != nil {
		logger.Error("Failed to get all payments", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(payments))
}

func (h *PaymentsHandler) TopUp(c echo.Context) error {
	user_id := c.Get("user_id").(uint)

	var request TopUpInput

	if err := c.Bind(&request); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validate.Struct(&request); err != nil {
		logger.Error("Failed to validation create payment validation", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	res, err := h.paymentsService.TopUp(user_id, request.Amount)
	if err != nil {
		logger.Error("internal server error on TopUp: ", err)
		return c.JSON(http.StatusBadRequest, ResponseError{err.Error()})
	}

	return c.JSON(http.StatusCreated, fres.Response.StatusCreated(res))
}

func (h *PaymentsHandler) PaidResponse(c echo.Context) error {
	return c.JSON(http.StatusOK, fres.Response.StatusOK("Your payment was successfull!"))
}
