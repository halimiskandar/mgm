package rest

import (
	"context"
	"myGreenMarket/domain"
	"net/http"
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
		Slot      string `json:"slot" validate:"required"`
		ProductID uint64 `json:"product_id" validate:"required"`
		EventType string `json:"event_type" validate:"required,oneof=impression click atc order"`
	}
)

func NewBanditHandler(svc BanditService) *BanditHandler {
	return &BanditHandler{
		validate:      validator.New(),
		banditService: svc,
	}
}

func (h *BanditHandler) Recommend(c echo.Context) error {
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
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(recs))
}

func (h *BanditHandler) Feedback(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var req FeedbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	event := domain.BanditEvent{
		UserID:    userID,
		Slot:      req.Slot,
		ProductID: req.ProductID,
		EventType: req.EventType,
		CreatedAt: time.Now(),
	}

	if err := h.banditService.LogFeedback(c.Request().Context(), event); err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, fres.Response.StatusCreated("feedback recorded"))
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
