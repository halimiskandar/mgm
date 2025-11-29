package postgres

import (
	"context"
	"errors"
	"fmt"
	"myGreenMarket/domain"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	DB *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{
		DB: db,
	}
}

func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	if err := r.DB.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id uint64) (domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return domain.Category{}, fmt.Errorf("context error: %w", err)
	}

	var category domain.Category

	err := r.DB.WithContext(ctx).Where("category_id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Category{}, errors.New("category not found")
		}
		return domain.Category{}, fmt.Errorf("failed to find category: %w", err)
	}

	return category, nil
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	var categories []domain.Category
	err := r.DB.WithContext(ctx).Find(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find categories: %w", err)
	}

	return categories, nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// Update product_category field
	updateData := map[string]interface{}{
		"product_category": category.ProductCategory,
	}

	result := r.DB.WithContext(ctx).Model(&domain.Category{}).Where("category_id = ?", category.CategoryID).Updates(updateData)
	if result.Error != nil {
		return fmt.Errorf("failed to update category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("category not found")
	}

	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id uint64) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	result := r.DB.WithContext(ctx).Where("category_id = ?", id).Delete(&domain.Category{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("category not found")
	}

	return nil
}
