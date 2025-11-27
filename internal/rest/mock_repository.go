package rest

import (
	"context"
	"net/http"

	"myGreenMarket/domain"

	"github.com/AMFarhan21/fres"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type MockRecommendationService interface {
	GetRecommendations(ctx context.Context, slot string, limit int) ([]domain.BanditRecommendation, error)
}

type MockRecommendationHandler struct {
	validate *validator.Validate
	service  MockRecommendationService
}

func NewMockRecommendationHandler(service MockRecommendationService) *MockRecommendationHandler {
	return &MockRecommendationHandler{
		validate: validator.New(),
		service:  service,
	}
}

type MockRecoQuery struct {
	Slot string `query:"slot" validate:"required"`
	N    int    `query:"n"`
}

func (h *MockRecommendationHandler) Get(c echo.Context) error {
	var q MockRecoQuery
	if err := c.Bind(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}
	if err := h.validate.Struct(&q); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if q.N <= 0 {
		q.N = 10
	}

	recs, err := h.service.GetRecommendations(c.Request().Context(), q.Slot, q.N)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, fres.Response.StatusOK(recs))
}
