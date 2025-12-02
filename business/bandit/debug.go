package bandit

import (
	"context"
	"math/rand"
	"time"

	"myGreenMarket/domain"
)

// DebugRecommend returns a debug view of recommendations with context & features.
func (s *BanditService) DebugRecommend(
	ctx context.Context,
	userID uint,
	slot string,
	limit int,
	ctxMap map[string]any,
) ([]domain.DebugRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	// 1) load offline candidates (same as Recommend)
	offlineRows, limit, err := s.loadCandidates(ctx, slot, limit)
	if err != nil {
		return nil, err
	}
	if len(offlineRows) == 0 {
		return []domain.DebugRecommendation{}, nil
	}

	// 2) config + segment + variant for this user & slot
	cfg, seg, variant := s.loadConfigForUser(ctx, userID, slot) // :contentReference[oaicite:1]{index=1}

	// 3) build base context (time, dow, segment, variant, platform)
	now := time.Now()
	platform := ""
	if ctxMap != nil {
		if p, ok := ctxMap["platform"].(string); ok {
			platform = p
		}
	}
	baseCtx := buildBaseContext(now, platform, seg, variant)

	// enrich with user_tier & campaign_id from DB (same as Recommend) :contentReference[oaicite:2]{index=2}
	if s.userCtxRepo != nil {
		if uc, err := s.userCtxRepo.GetUserContext(ctx, userID); err == nil {
			if uc.Tier != "" {
				baseCtx["user_tier"] = uc.Tier
			}
			if uc.CampaignID != "" {
				baseCtx["campaign_id"] = uc.CampaignID
			}
		}
	}

	// fullCtx = base + request-provided ctx (page_name, device_type, etc.)
	fullCtx := mergeContext(baseCtx, ctxMap)

	// 4) load global + user states (read-only for debug) :contentReference[oaicite:3]{index=3}
	globalKey := stateGlobalKey(slot, seg)
	userKey := stateUserKey(slot, seg, userID)

	globalState, err := s.stateRepo.GetState(ctx, globalKey)
	if err != nil {
		return nil, err
	}
	if globalState == nil {
		globalState = newDefaultState()
	}

	userState, err := s.stateRepo.GetState(ctx, userKey)
	if err != nil {
		return nil, err
	}
	if userState == nil {
		userState = newDefaultState()
	}

	// 5) normalize offline score
	maxScore := 0.0
	for _, row := range offlineRows {
		if row.Score > maxScore {
			maxScore = row.Score
		}
	}
	if maxScore == 0 {
		maxScore = 1
	}

	wGlobal := cfg.WGlobal
	wUser := cfg.WUser
	if wGlobal == 0 && wUser == 0 {
		wGlobal = 0.7
		wUser = 0.3
	}

	type scored struct {
		rec   domain.DebugRecommendation
		score float64
	}

	scoredList := make([]scored, 0, len(offlineRows))

	for _, row := range offlineRows {
		pid := row.ProductID

		// eligibility filter (stock, hub, etc.) :contentReference[oaicite:4]{index=4}
		if s.eligChecker != nil {
			ok, err := s.eligChecker.IsEligible(ctx, userID, pid, slot)
			if err != nil || !ok {
				continue
			}
		}

		// GLOBAL arm
		gArm, ok := globalState.Arms[pid]
		if !ok {
			gArm = newArmState()
		}

		// USER arm
		uArm, ok := userState.Arms[pid]
		if !ok {
			uArm = newArmState()
		}

		// feature vector for this impression
		x := buildFeatureVector(userID, slot, pid, cfg, seg, fullCtx)

		// copy fixed array into slice for JSON
		fv := make([]float64, len(x))
		copy(fv, x[:])

		// global stats
		gAInv, err := invert4x4(gArm.A)
		if err != nil {
			gArm = newArmState()
			gAInv, _ = invert4x4(gArm.A)
		}
		gTheta := matVecMul(gAInv, gArm.B)

		// user stats
		uAInv, err := invert4x4(uArm.A)
		if err != nil {
			uArm = newArmState()
			uAInv, _ = invert4x4(uArm.A)
		}
		uTheta := matVecMul(uAInv, uArm.B)

		offlineNorm := row.Score / maxScore

		var gBandit, uBandit float64

		switch variant {
		case VariantOfflineOnly:
			gBandit = 0.0
			uBandit = 0.0
		case VariantThompson:
			gBandit = thompsonScore(gTheta, x, gAInv)
			uBandit = thompsonScore(uTheta, x, uAInv)
		case VariantUCB:
			fallthrough
		default:
			gBandit = ucbScore(gTheta, x, gAInv, cfg.Alpha)
			uBandit = ucbScore(uTheta, x, uAInv, cfg.Alpha)
		}

		banditScore := wGlobal*gBandit + wUser*uBandit
		final := cfg.WBandit*banditScore + cfg.WOffline*offlineNorm

		if variant != VariantOfflineOnly && cfg.ExploreNoise > 0 {
			final += cfg.ExploreNoise * rand.Float64()
		}

		rec := domain.DebugRecommendation{
			ProductID:  pid,
			FinalScore: final,
			Segment:    seg,
			Variant:    variant,
			Context:    fullCtx,
			Features:   fv,
		}

		scoredList = append(scoredList, scored{
			rec:   rec,
			score: final,
		})
	}

	// 6) sort top-N by score (simple selection, same as scoreCandidates) :contentReference[oaicite:5]{index=5}
	if len(scoredList) < limit {
		limit = len(scoredList)
	}
	for i := 0; i < limit; i++ {
		maxIdx := i
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[maxIdx].score {
				maxIdx = j
			}
		}
		scoredList[i], scoredList[maxIdx] = scoredList[maxIdx], scoredList[i]
	}

	out := make([]domain.DebugRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, scoredList[i].rec)
	}

	return out, nil
}
