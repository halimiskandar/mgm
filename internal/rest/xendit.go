package rest

import (
	"log"
	"myGreenMarket/domain"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AMFarhan21/fres"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type (
	WebhookController struct {
		paymentService PaymentsService
		validate       *validator.Validate
	}

	WebhookRequest struct {
		ID                 string    `json:"id"`
		Items              []Item    `json:"items"`
		Amount             int64     `json:"amount"`
		Status             string    `json:"status"`
		Created            time.Time `json:"created"`
		IsHigh             bool      `json:"is_high"`
		Updated            time.Time `json:"updated"`
		UserID             string    `json:"user_id"`
		Currency           string    `json:"currency"`
		Description        string    `json:"description"`
		ExternalID         string    `json:"external_id"`
		MerchantName       string    `json:"merchant_name"`
		PaymentMethod      string    `json:"payment_method"`
		PaymentChannel     string    `json:"payment_channel"`
		PaymentDestination string    `json:"payment_destination"`
		FailureRedirectURL string    `json:"failure_redirect_url"`
		SuccessRedirectURL string    `json:"success_redirect_url"`
		Metadata           Meta      `json:"metadata"`
	}

	Meta struct {
		Purpose string `json:"purpose"`
	}

	Item struct {
		Purpose  string `json:"purpose"`
		Name     string `json:"name"`
		Price    int64  `json:"price"`
		Category string `json:"category"`
		Quantity int64  `json:"quantity"`
	}
)

func NewWebhookController(paymentService PaymentsService) *WebhookController {
	return &WebhookController{
		paymentService: paymentService,
		validate:       validator.New(),
	}
}

func (ctrl WebhookController) HandleWebhook(c echo.Context) error {
	var request WebhookRequest

	if err := c.Bind(&request); err != nil {
		log.Println("Failed to bind webhook request:", err)
		return c.JSON(http.StatusBadRequest, fres.Response.StatusBadRequest("Invalid request"))
	}

	log.Print("Received webhook from Xendit:", request)

	externalID := strings.Split(request.ExternalID, "|")

	paymentId, _ := strconv.Atoi(externalID[0])
	userId, _ := strconv.Atoi(externalID[1])
	productId, _ := strconv.Atoi(externalID[2])
	purpose := externalID[3]

	err := ctrl.paymentService.UpdatePayment(domain.Payments{
		ID: paymentId,
	}, userId, productId, request, purpose)
	if err != nil {
		log.Println("Failed to update payment status:", err.Error())
		return c.JSON(http.StatusInternalServerError, fres.Response.StatusInternalServerError(http.StatusInternalServerError))
	}

	log.Print(request)
	return c.JSON(http.StatusOK, fres.Response.StatusOK(http.StatusOK))
}
