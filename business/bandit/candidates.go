package bandit

import (
	"context"
	"fmt"
	"myGreenMarket/domain"
)

// loadCandidates loads products from repo and adjusts limit safely.
func (s *BanditService) loadCandidates(
	ctx context.Context,
	slot string,
	limit int,
) ([]domain.MockRecommendation, int, error) {

	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("context error: %w", err)
	}

	if s.offlineRepo != nil {
		candidateLimit := limit * 3
		if candidateLimit < limit {
			candidateLimit = limit
		}

		rows, err := s.offlineRepo.GetBySlot(ctx, slot, candidateLimit)
		if err != nil {
			return nil, 0, fmt.Errorf("load offline recommendations: %w", err)
		}
		if len(rows) == 0 {
			return []domain.MockRecommendation{}, 0, nil
		}
		if len(rows) < limit {
			limit = len(rows)
		}

		return rows, limit, nil
	}

	products, err := s.productRepo.FindAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("load products: %w", err)
	}
	if len(products) == 0 {
		return []domain.MockRecommendation{}, 0, nil
	}
	if len(products) < limit {
		limit = len(products)
	}

	rows := make([]domain.MockRecommendation, 0, len(products))
	for _, p := range products {
		rows = append(rows, domain.MockRecommendation{
			ProductID: uint64(p.ID),
			Score:     1.0,
		})
	}

	return rows, limit, nil
}
