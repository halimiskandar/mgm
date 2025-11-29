package rest

import (
	"context"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type CategoryService interface {
	GetAllCategories(ctx context.Context) ([]domain.Category, error)
	GetCategoryByID(ctx context.Context, id uint64) (domain.Category, error)
	CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id uint64) error
}

type CategoryHandler struct {
	categoryService CategoryService
	validator       *validator.Validate
	timeout         time.Duration
}

func NewCategoryHandler(categoryService CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		validator:       validator.New(),
		timeout:         10 * time.Second,
	}
}

type CreateCategoryRequest struct {
	ProductCategory string `json:"product_category" validate:"required"`
}

type UpdateCategoryRequest struct {
	ProductCategory string `json:"product_category" validate:"required"`
}

func (h *CategoryHandler) GetAllCategories(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	categories, err := h.categoryService.GetAllCategories(ctx)
	if err != nil {
		logger.Error("Failed to find all categories", err)
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "successfully get all categories",
		"categories": categories,
	})
}

func (h *CategoryHandler) GetCategoryByID(c echo.Context) error {
	categoryIDStr := c.Param("id")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		logger.Error("Invalid category id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid category id"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	category, err := h.categoryService.GetCategoryByID(ctx, categoryID)
	if err != nil {
		logger.Error("Failed to find category", err)
		// Check if category not found
		if err.Error() == "category not found" || err.Error() == "invalid category id" {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "successfully get category",
		"category": category,
	})
}

func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest

	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&req); err != nil {
		logger.Error("Failed to validate category request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	category := &domain.Category{
		ProductCategory: req.ProductCategory,
	}

	newCategory, err := h.categoryService.CreateCategory(ctx, category)
	if err != nil {
		logger.Error("Failed to create category", err)
		// Check if it's a validation error
		if err.Error() == "product category is required" {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":  "category successfully created",
		"category": newCategory,
	})
}

func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	categoryIDStr := c.Param("id")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		logger.Error("Invalid category id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid category id"})
	}

	var req UpdateCategoryRequest
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&req); err != nil {
		logger.Error("Failed to validate category request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	category := &domain.Category{
		CategoryID:      categoryID,
		ProductCategory: req.ProductCategory,
	}

	updatedCategory, err := h.categoryService.UpdateCategory(ctx, category)
	if err != nil {
		logger.Error("Failed to update category", err)
		// Check if category not found
		if err.Error() == "category not found" {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		// Check if it's a validation error
		if err.Error() == "category ID is required" || err.Error() == "product category is required" {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "successfully update category",
		"category": updatedCategory,
	})
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	categoryIDStr := c.Param("id")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		logger.Error("Invalid category id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid category id"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	err = h.categoryService.DeleteCategory(ctx, categoryID)
	if err != nil {
		logger.Error("Failed to delete category", err)
		// Check if category not found
		if err.Error() == "category not found" || err.Error() == "invalid category id" {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":     "category successfully deleted",
		"category_id": categoryID,
	})
}
