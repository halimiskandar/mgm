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

type ProductService interface {
	GetAllProducts(ctx context.Context) ([]domain.Product, error)
	GetProductByID(ctx context.Context, id uint) (*domain.Product, error)
	CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id uint64) error
}

type ProductHandler struct {
	productService ProductService
	validator      *validator.Validate
	timeout        time.Duration
}

func NewProductHandler(productService ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		validator:      validator.New(),
		timeout:        10 * time.Second,
	}
}

type CreateProductRequest struct {
	ProductID       uint64  `json:"product_id"`
	ProductSKUID    uint64  `json:"product_skuid"`
	IsGreenTag      bool    `json:"is_green_tag"`
	ProductName     string  `json:"product_name" validate:"required"`
	ProductCategory string  `json:"product_category" validate:"required"`
	Unit            string  `json:"unit" validate:"required"`
	NormalPrice     float64 `json:"normal_price" validate:"required,gt=0"`
	SalePrice       float64 `json:"sale_price" validate:"gte=0"`
	Discount        float64 `json:"discount" validate:"gte=0,lte=100"`
	Quantity        float64 `json:"quantity" validate:"required,gte=0"`
}

type UpdateProductRequest struct {
	ProductID       uint64  `json:"product_id"`
	ProductSKUID    uint64  `json:"product_skuid"`
	IsGreenTag      bool    `json:"is_green_tag"`
	ProductName     string  `json:"product_name" validate:"required"`
	ProductCategory string  `json:"product_category" validate:"required"`
	Unit            string  `json:"unit" validate:"required"`
	NormalPrice     float64 `json:"normal_price" validate:"required,gt=0"`
	SalePrice       float64 `json:"sale_price" validate:"gte=0"`
	Discount        float64 `json:"discount" validate:"gte=0,lte=100"`
	Quantity        float64 `json:"quantity" validate:"required,gte=0"`
}

func (h *ProductHandler) GetAllProducts(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	products, err := h.productService.GetAllProducts(ctx)
	if err != nil {
		logger.Error("Failed to find all Product", err)
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "successfully get all products",
		"products": products,
	})
}

func (h *ProductHandler) GetProductByID(c echo.Context) error {
	productIdStr := c.Param("id")

	productId, err := strconv.ParseUint(productIdStr, 10, 64)
	if err != nil {
		logger.Error("Invalid venue id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	product, err := h.productService.GetProductByID(ctx, uint(productId))
	if err != nil {
		if err.Error() == "product not found" {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "successfully find product by id",
		"product": product,
	})
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req CreateProductRequest

	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&req); err != nil {
		logger.Error("Failed to validate product request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	product := &domain.Product{
		ProductID:       req.ProductID,
		ProductSKUID:    req.ProductSKUID,
		IsGreenTag:      req.IsGreenTag,
		ProductName:     req.ProductName,
		ProductCategory: req.ProductCategory,
		Unit:            req.Unit,
		NormalPrice:     req.NormalPrice,
		SalePrice:       req.SalePrice,
		Discount:        req.Discount,
		Quantity:        req.Quantity,
	}

	newProduct, err := h.productService.CreateProduct(ctx, product)
	if err != nil {
		logger.Error("Failed to create Product", err)
		// Check if it's a validation error
		if err.Error() == "product name is required" ||
			err.Error() == "product category is required" ||
			err.Error() == "unit is required" ||
			err.Error() == "normal price must be greater than 0" ||
			err.Error() == "quantity cannot be negative" {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Product successfully created",
		"product": newProduct,
	})
}

func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	ProductIdStr := c.Param("id")

	ProductId, err := strconv.ParseUint(ProductIdStr, 10, 64)
	if err != nil {
		logger.Error("Invalid Product id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	var req UpdateProductRequest
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&req); err != nil {
		logger.Error("Failed to validate product request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	product := &domain.Product{
		ID:              ProductId,
		ProductID:       req.ProductID,
		ProductSKUID:    req.ProductSKUID,
		IsGreenTag:      req.IsGreenTag,
		ProductName:     req.ProductName,
		ProductCategory: req.ProductCategory,
		Unit:            req.Unit,
		NormalPrice:     req.NormalPrice,
		SalePrice:       req.SalePrice,
		Discount:        req.Discount,
		Quantity:        req.Quantity,
	}

	updateProduct, err := h.productService.UpdateProduct(ctx, product)
	if err != nil {
		logger.Error("Failed to update Product", err)
		// Check if product not found
		if err.Error() == "product not found" {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		// Check if it's a validation error
		if err.Error() == "product ID is required" ||
			err.Error() == "product name is required" ||
			err.Error() == "normal price must be greater than 0" ||
			err.Error() == "quantity cannot be negative" {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "successfully update product",
		"product": updateProduct,
	})
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	ProductIdStr := c.Param("id")

	ProductId, err := strconv.ParseUint(ProductIdStr, 10, 64)
	if err != nil {
		logger.Error("Invalid Product id", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	err = h.productService.DeleteProduct(ctx, ProductId)
	if err != nil {
		logger.Error("Failed to delete Product", err)
		// Check if product not found
		if err.Error() == "product not found" || err.Error() == "invalid product id" {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "product successfully deleted",
		"product_id": ProductId,
	})
}
