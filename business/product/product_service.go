package product

import (
	"context"
	"errors"
	"fmt"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
)

// ProductRepository contract interface
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	FindByID(ctx context.Context, id uint64) (domain.Product, error)
	FindAll(ctx context.Context) ([]domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uint64) error
}

type productService struct {
	productRepo ProductRepository
}

func NewProductService(productRepo ProductRepository) *productService {
	return &productService{
		productRepo: productRepo,
	}
}

func (s *productService) GetAllProducts(ctx context.Context) ([]domain.Product, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when get all product")
		return nil, fmt.Errorf("context error: %w", err)
	}

	product, err := s.productRepo.FindAll(ctx)
	if err != nil {
		logger.Error("Failed to find all product", err)
		return nil, err
	}

	return product, nil
}

func (s *productService) GetProductByID(ctx context.Context, id uint) (*domain.Product, error) {
	if id == 0 {
		logger.Error("invalid product id")
		return nil, errors.New("invalid product id")
	}

	if err := ctx.Err(); err != nil {
		logger.Error("context error when create product")
		return nil, fmt.Errorf("context error: %w", err)
	}

	product, err := s.productRepo.FindByID(ctx, uint64(id))
	if err != nil {
		logger.Error("failed to find product by id", err.Error())
		return nil, err
	}

	return &product, nil
}

func (s *productService) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when create product")
		return nil, fmt.Errorf("context error: %w", err)
	}

	// Validation
	if product.ProductName == "" {
		logger.Error("Invalid product data: product name is required")
		return nil, errors.New("product name is required")
	}

	if product.ProductCategory == "" {
		logger.Error("Invalid product data: product category is required")
		return nil, errors.New("product category is required")
	}

	if product.Unit == "" {
		logger.Error("Invalid product data: unit is required")
		return nil, errors.New("unit is required")
	}

	if product.NormalPrice <= 0 {
		logger.Error("Invalid product data: normal price must be greater than 0")
		return nil, errors.New("normal price must be greater than 0")
	}

	if product.Quantity < 0 {
		logger.Error("Invalid product data: quantity cannot be negative")
		return nil, errors.New("quantity cannot be negative")
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		logger.Error("failed to create new product", err)
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	logger.Info("product created successfully")

	return product, nil
}

func (s *productService) UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when updating product")
		return nil, fmt.Errorf("context error: %w", err)
	}

	if product.ID == 0 {
		logger.Error("Invalid product data: ID is required")
		return nil, errors.New("product ID is required")
	}

	// Validation
	if product.ProductName == "" {
		logger.Error("Invalid product data: product name is required")
		return nil, errors.New("product name is required")
	}

	if product.NormalPrice <= 0 {
		logger.Error("Invalid product data: normal price must be greater than 0")
		return nil, errors.New("normal price must be greater than 0")
	}

	if product.Quantity < 0 {
		logger.Error("Invalid product data: quantity cannot be negative")
		return nil, errors.New("quantity cannot be negative")
	}

	// Verify product exists
	_, err := s.productRepo.FindByID(ctx, product.ID)
	if err != nil {
		logger.Error("product not found", err)
		return nil, errors.New("product not found")
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		logger.Error("failed to update product", err)
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Get updated product from database
	updatedProduct, err := s.productRepo.FindByID(ctx, product.ID)
	if err != nil {
		logger.Error("failed to fetch updated product", err)
		return nil, fmt.Errorf("failed to fetch updated product: %w", err)
	}

	logger.Info("product updated success")

	return &updatedProduct, nil
}

func (s *productService) DeleteProduct(ctx context.Context, id uint64) error {
	if id == 0 {
		logger.Error("Invalid product id when deleting product")
		return errors.New("invalid product id")
	}

	if err := ctx.Err(); err != nil {
		logger.Error("context error when deleting product")
		return fmt.Errorf("context error: %w", err)
	}

	// Verify product exists
	_, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("product not found", err)
		return errors.New("product not found")
	}

	if err := s.productRepo.Delete(ctx, id); err != nil {
		logger.Error("failed to delete product", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	logger.Info("product deleted success")

	return nil
}
