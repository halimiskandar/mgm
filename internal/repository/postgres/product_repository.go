package postgres

import (
	"context"
	"errors"
	"fmt"
	"myGreenMarket/domain"

	"gorm.io/gorm"
)

type ProductRepository struct {
	DB *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{
		DB: db,
	}
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	if err := r.DB.WithContext(ctx).Create(product).Error; err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id uint64) (domain.Product, error) {
	if err := ctx.Err(); err != nil {
		return domain.Product{}, fmt.Errorf("context error: %w", err)
	}

	var product domain.Product

	err := r.DB.WithContext(ctx).First(&product, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Product{}, errors.New("product not found")
		}
		return domain.Product{}, fmt.Errorf("failed to find product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) FindAll(ctx context.Context) ([]domain.Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	var products []domain.Product
	err := r.DB.WithContext(ctx).Find(&products).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find Products: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// Cek apakah product exists
	var existingProduct domain.Product
	if err := r.DB.WithContext(ctx).First(&existingProduct, product.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("product not found")
		}
		return fmt.Errorf("failed to find product: %w", err)
	}

	// Update semua field yang bisa diubah
	updateData := map[string]interface{}{
		"product_id":       product.ProductID,
		"product_skuid":    product.ProductSKUID,
		"is_green_tag":     product.IsGreenTag,
		"product_name":     product.ProductName,
		"product_category": product.ProductCategory,
		"unit":             product.Unit,
		"normal_price":     product.NormalPrice,
		"sale_price":       product.SalePrice,
		"discount":         product.Discount,
		"quantity":         product.Quantity,
	}

	result := r.DB.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", product.ID).Updates(updateData)
	if result.Error != nil {
		return fmt.Errorf("failed to update product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("product not found or already deleted")
	}

	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uint64) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	result := r.DB.WithContext(ctx).Delete(&domain.Product{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("product not found or already deleted")
	}

	return nil
}
