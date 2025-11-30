package bandit

import (
	"context"
	"fmt"
	"math"
	"time"

	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
)

// DebugRecommend returns detailed score components for inspection.
func (s *BanditService) DebugRecommend(
	ctx context.Context,
	userID uint,
	slot string,
	limit int,
	ctxMap map[string]any,
) ([]domain.DebugRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}

	// 1) figure out variant + config + segment (same as Recommend) -----
	cfg, seg, variant := s.loadConfigForUser(ctx, userID, slot)
	// Build the same base context as Recommend/LogFeedback.
	now := time.Now()
	platform := ""
	if ctxMap != nil {
		if p, ok := ctxMap["platform"].(string); ok {
			platform = p
		}
	}
	baseCtx := buildBaseContext(now, platform, seg, variant)
	fullCtx := mergeContext(baseCtx, ctxMap)
	slotKey := stateSlotKey(slot, seg)

	// trace logging
	tid := TraceIDFromContext(ctx)
	logger.Debug("bandit_debug_recommend",
		"trace_id", tid,
		"user_id", userID,
		"slot", slot,
		"segment", seg,
		"variant", variant,
		"limit", limit,
	)

	//2) offline candidates -----
	candidateLimit := limit * 3
	if candidateLimit < limit {
		candidateLimit = limit
	}

	offlineRows, err := s.offlineRepo.GetBySlot(ctx, slot, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to load offline recommendations: %w", err)
	}
	if len(offlineRows) == 0 {
		return []domain.DebugRecommendation{}, nil
	}
	if len(offlineRows) < limit {
		limit = len(offlineRows)
	}

	// 3) fetch bandit state for this slot+segment -----
	state, err := s.stateRepo.GetState(ctx, slotKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bandit state: %w", err)
	}

	// normalize offline scores
	maxScore := 0.0
	for _, row := range offlineRows {
		if row.Score > maxScore {
			maxScore = row.Score
		}
	}
	if maxScore == 0 {
		maxScore = 1
	}

	type scored struct {
		rec   domain.DebugRecommendation
		score float64
	}

	result := make([]scored, 0, len(offlineRows))

	// 4) no state yet → offline only, bandit fields = 0 -----
	if state == nil {
		for _, row := range offlineRows {
			offlineNorm := row.Score / maxScore
			final := cfg.WOffline * offlineNorm

			result = append(result, scored{
				rec: domain.DebugRecommendation{
					ProductID:         row.ProductID,
					OfflineScore:      row.Score,
					OfflineNormalized: offlineNorm,
					BanditMean:        0,
					BanditUncertainty: 0,
					BanditUCB:         0,
					FinalScore:        final,
				},
				score: final,
			})
		}
	} else {
		// 5) full LinUCB scoring with debug info -----
		for _, row := range offlineRows {
			pid := row.ProductID

			arm, ok := state.Arms[pid]
			wasNew := false
			if !ok {
				arm = newArmState()
				state.Arms[pid] = arm
				wasNew = true
			}

			// feature vector
			x := buildFeatureVector(userID, slot, pid, cfg, seg, fullCtx)

			// A^-1
			AInv, err := invert4x4(arm.A)
			if err != nil {
				arm = newArmState()
				state.Arms[pid] = arm
				AInv, _ = invert4x4(arm.A)
			}

			theta := matVecMul(AInv, arm.B)
			fv := buildFeatureVector(userID, slot, pid, cfg, seg, fullCtx)
			// UCB components for debug fields
			mean := dot(theta, x)
			tmp := matVecMul(AInv, x)
			uncertainty := math.Sqrt(dot(x, tmp))
			ucb := mean + cfg.Alpha*uncertainty

			offlineNorm := row.Score / maxScore

			// === ALGO-LEVEL A/B SWITCH ===
			var banditScore float64

			switch variant {
			case VariantOfflineOnly:
				// pure offline baseline: ignore bandit, just use offline score
				banditScore = 0.0

			case VariantThompson:
				// Thompson Sampling
				banditScore = thompsonScore(theta, x, AInv)

			case VariantUCB:
				fallthrough
			default:
				// UCB (current behavior)
				banditScore = ucbScore(theta, x, AInv, cfg.Alpha)
			}

			// combine offline + bandit
			final := cfg.WBandit*banditScore + cfg.WOffline*offlineNorm

			// === COLD START BOOST only when bandit is active ===
			if variant != VariantOfflineOnly && wasNew {
				final += 0.25
			}

			// NOTE: no exploration noise in debug output

			result = append(result, scored{
				rec: domain.DebugRecommendation{
					ProductID:         pid,
					OfflineScore:      row.Score,
					OfflineNormalized: offlineNorm,
					BanditMean:        mean,
					BanditUncertainty: uncertainty,
					BanditUCB:         ucb,
					FinalScore:        final,

					// 🔹 here we use fv
					Features: featuresToSlice(fv),
					Context:  fullCtx,
					Segment:  seg,
					Variant:  variant,
				},
				score: final,
			})
		}
	}

	// 6) top-N selection by final score -----
	if len(result) < limit {
		limit = len(result)
	}

	for i := 0; i < limit; i++ {
		maxIdx := i
		for j := i + 1; j < len(result); j++ {
			if result[j].score > result[maxIdx].score {
				maxIdx = j
			}
		}
		result[i], result[maxIdx] = result[maxIdx], result[i]
	}

	out := make([]domain.DebugRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, result[i].rec)
	}

	return out, nil
}
func featuresToSlice(fv [linUCBFeatureDim]float64) []float64 {
	out := make([]float64, linUCBFeatureDim)
	for i := 0; i < linUCBFeatureDim; i++ {
		out[i] = fv[i]
	}
	return out
}
