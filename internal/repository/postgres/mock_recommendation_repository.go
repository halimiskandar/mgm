package postgres

import (
	"context"
	"fmt"
	"myGreenMarket/domain"

	"gorm.io/gorm"
)

type MockRecommendationRepository struct {
	DB *gorm.DB
}

func NewMockRecommendationRepository(db *gorm.DB) *MockRecommendationRepository {
	return &MockRecommendationRepository{
		DB: db,
	}
}

// Get top-N recommendations for a slot ordered by score DESC
func (r *MockRecommendationRepository) GetBySlot(
	ctx context.Context,
	slot string,
	limit int,
) ([]domain.MockRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}

	var recs []domain.MockRecommendation
	if err := r.DB.WithContext(ctx).
		Where("slot = ?", slot).
		Order("score DESC").
		Limit(limit).
		Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("failed to query mock_recommendations: %w", err)
	}

	return recs, nil
}
