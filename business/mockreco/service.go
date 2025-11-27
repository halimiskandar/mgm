package mockreco

import (
	"context"
	"fmt"
	"myGreenMarket/domain"
)

type MockRecommendationRepository interface {
	GetBySlot(ctx context.Context, slot string, limit int) ([]domain.MockRecommendation, error)
}

type Service struct {
	repo MockRecommendationRepository
}

func NewService(repo MockRecommendationRepository) *Service {
	return &Service{repo: repo}
}

// Returns standardized BanditRecommendation
func (s *Service) GetRecommendations(
	ctx context.Context,
	slot string,
	limit int,
) ([]domain.BanditRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.repo.GetBySlot(ctx, slot, limit)
	if err != nil {
		return nil, err
	}

	recs := make([]domain.BanditRecommendation, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, domain.BanditRecommendation{
			ProductID: r.ProductID,
			Score:     r.Score,
		})
	}

	return recs, nil
}
