package rest

import (
	"context"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/AMFarhan21/fres"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type (
	BanditHandler struct {
		validate      *validator.Validate
		banditService BanditService
	}

	BanditService interface {
		Recommend(ctx context.Context, userID uint, slot string, limit int) ([]domain.BanditRecommendation, error)
		DebugRecommend(ctx context.Context, userID uint, slot string, limit int) ([]domain.DebugRecommendation, error)
		LogFeedback(ctx context.Context, event domain.BanditEvent) error
	}

	RecommendQuery struct {
		Slot     string `query:"slot" validate:"required"`
		N        int    `query:"n"`
		Platform string `query:"platform"`
	}

	FeedbackRequest struct {
		Slot      string  `json:"slot" validate:"required"`
		ProductID uint64  `json:"product_id" validate:"required"`
		EventType string  `json:"event_type" validate:"required,oneof=impression click atc order"`
		Value     float64 `json:"value"`
	}
)

func NewBanditHandler(svc BanditService) *BanditHandler {
	return &BanditHandler{
		validate:      validator.New(),
		banditService: svc,
	}
}

func (h *BanditHandler) Recommend(c echo.Context) error {
	start := time.Now()
	uidVal := c.Get("user_id")
	userID, ok := uidVal.(uint)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "unauthorized"})
	}

	var q RecommendQuery
	if err := c.Bind(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if err := h.validate.Struct(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if q.N <= 0 {
		q.N = 10
	}

	recs, err := h.banditService.Recommend(c.Request().Context(), userID, q.Slot, q.N)
	elapsed := time.Since(start).Seconds()
	metrics.BanditRecommendLatency.Observe(elapsed)
	metrics.BanditRecommendRequests.Inc()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(recs))
}

func (h *BanditHandler) Feedback(c echo.Context) error {
	uidVal := c.Get("user_id")
	userID, ok := uidVal.(uint)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "unauthorized"})
	}

	var req FeedbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	// LogFeedback will compute cfg, seg, variant using loadConfigForUser.

	event := domain.BanditEvent{
		UserID:    userID,
		Slot:      req.Slot,
		ProductID: req.ProductID,
		EventType: req.EventType,
		Value:     req.Value, // business value
	}

	if err := h.banditService.LogFeedback(c.Request().Context(), event); err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(nil))
}

// GET /api/v1/recommendations/debug?slot=home_row1&n=10
func (h *BanditHandler) DebugRecommend(c echo.Context) error {
	uidVal := c.Get("user_id")
	userID, ok := uidVal.(uint)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "unauthorized"})
	}

	var q RecommendQuery
	if err := c.Bind(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if err := h.validate.Struct(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if q.N <= 0 {
		q.N = 10
	}

	recs, err := h.banditService.DebugRecommend(c.Request().Context(), userID, q.Slot, q.N)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(recs))
}

// GET /api/v1/recommendations/debug?user_id=123&slot=home_row1&limit=5
func (h *BanditHandler) GetDebugRecommendations(c echo.Context) error {
	ctx := c.Request().Context()

	userIDStr := c.QueryParam("user_id")
	slot := c.QueryParam("slot")
	limitStr := c.QueryParam("limit")

	if userIDStr == "" || slot == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "user_id and slot are required",
		})
	}

	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid user_id",
		})
	}
	userID := uint(userID64)

	limit := 10
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	recs, err := h.banditService.DebugRecommend(ctx, userID, slot, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user_id": userID,
		"slot":    slot,
		"limit":   limit,
		"data":    recs,
	})
}

type BanditFeedbackRequest struct {
	UserID    uint   `json:"user_id"`
	Slot      string `json:"slot"`
	ProductID uint64 `json:"product_id"`
	EventType string `json:"event_type"` // "impression" | "click" | "atc" | "order"

	Value float64 `json:"value"`
}

func (h *BanditHandler) BanditFeedback(c echo.Context) error {
	var req BanditFeedbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	businessValue := req.Value
	if businessValue == 0 && req.EventType == "order" {
		businessValue = 1.0
	}

	ev := domain.BanditEvent{
		UserID:    req.UserID,
		Slot:      req.Slot,
		ProductID: req.ProductID,
		EventType: req.EventType,
		Value:     businessValue,
	}

	if err := h.banditService.LogFeedback(c.Request().Context(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, echo.Map{"status": "ok"})
}
