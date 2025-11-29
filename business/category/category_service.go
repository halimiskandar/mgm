package category

import (
	"context"
	"errors"
	"fmt"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
)

// CategoryRepository contract interface
type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	FindByID(ctx context.Context, id uint64) (domain.Category, error)
	FindAll(ctx context.Context) ([]domain.Category, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id uint64) error
}

type categoryService struct {
	categoryRepo CategoryRepository
}

func NewCategoryService(categoryRepo CategoryRepository) *categoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
	}
}

func (s *categoryService) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when get all categories")
		return nil, fmt.Errorf("context error: %w", err)
	}

	categories, err := s.categoryRepo.FindAll(ctx)
	if err != nil {
		logger.Error("Failed to find all categories", err)
		return nil, err
	}

	return categories, nil
}

func (s *categoryService) GetCategoryByID(ctx context.Context, id uint64) (domain.Category, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when get category by id")
		return domain.Category{}, fmt.Errorf("context error: %w", err)
	}

	if id == 0 {
		logger.Error("Invalid category id")
		return domain.Category{}, errors.New("invalid category id")
	}

	category, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("Failed to find category", err)
		return domain.Category{}, err
	}

	return category, nil
}

func (s *categoryService) CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when create category")
		return nil, fmt.Errorf("context error: %w", err)
	}

	// Validation
	if category.ProductCategory == "" {
		logger.Error("Invalid category data: product category is required")
		return nil, errors.New("product category is required")
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		logger.Error("failed to create new category", err)
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	logger.Info("category created successfully")

	return category, nil
}

func (s *categoryService) UpdateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	if err := ctx.Err(); err != nil {
		logger.Error("context error when updating category")
		return nil, fmt.Errorf("context error: %w", err)
	}

	if category.CategoryID == 0 {
		logger.Error("Invalid category data: ID is required")
		return nil, errors.New("category ID is required")
	}

	// Validation
	if category.ProductCategory == "" {
		logger.Error("Invalid category data: product category is required")
		return nil, errors.New("product category is required")
	}

	// Verify category exists
	_, err := s.categoryRepo.FindByID(ctx, category.CategoryID)
	if err != nil {
		logger.Error("category not found", err)
		return nil, errors.New("category not found")
	}

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		logger.Error("failed to update category", err)
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Get updated category from database
	updatedCategory, err := s.categoryRepo.FindByID(ctx, category.CategoryID)
	if err != nil {
		logger.Error("failed to fetch updated category", err)
		return nil, fmt.Errorf("failed to fetch updated category: %w", err)
	}

	logger.Info("category updated successfully")

	return &updatedCategory, nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, id uint64) error {
	if id == 0 {
		logger.Error("Invalid category id when deleting category")
		return errors.New("invalid category id")
	}

	if err := ctx.Err(); err != nil {
		logger.Error("context error when deleting category")
		return fmt.Errorf("context error: %w", err)
	}

	// Verify category exists
	_, err := s.categoryRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("category not found", err)
		return errors.New("category not found")
	}

	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		logger.Error("failed to delete category", err)
		return fmt.Errorf("failed to delete category: %w", err)
	}

	logger.Info("category deleted successfully")

	return nil
}
