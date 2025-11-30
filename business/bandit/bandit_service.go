package bandit

import (
	"context"
	"fmt"
	"math/rand"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
	"time"
)

type FeatureFlags struct {
	UseBias        bool
	UseTimeBucket  bool
	UseDowBucket   bool
	UseSlotHash    bool
	UseSegment     bool
	UseProductHash bool
	UseUserHash    bool
}

const (
	VariantUCB         = 0
	VariantThompson    = 1
	VariantOfflineOnly = 2
)

// buildBaseContext builds the standard context used for both recommendation & feedback.
// `platform` can be "android", "ios", "web", etc.
func buildBaseContext(now time.Time, platform string, segment, variant int) map[string]any {
	return map[string]any{
		"time_bucket": computeTimeBucket(now), // you implement this (morning/afternoon/evening)
		"dow":         int(now.Weekday()),     // 0=Sunday, 1=Monday, ...
		"platform":    platform,
		"segment":     segment,
		"variant":     variant,
		"event_time":  now.Format(time.RFC3339),
	}
}
func computeTimeBucket(t time.Time) string {
	h := t.Hour()
	switch {
	case h < 6:
		return "night"
	case h < 12:
		return "morning"
	case h < 18:
		return "afternoon"
	default:
		return "evening"
	}
}

// mergeContext merges multiple maps into a new one.

func mergeContext(maps ...map[string]any) map[string]any {
	out := make(map[string]any)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ---- Repository interfaces ----

// Offline model output from mock_recommendations
type OfflineRecommendationRepository interface {
	GetBySlot(ctx context.Context, slot string, limit int) ([]domain.MockRecommendation, error)
}

// Write feedback events (raw log)
type BanditRepository interface {
	SaveEvent(ctx context.Context, event domain.BanditEvent) error
}

// Read product candidates (kept for future use if needed)
type ProductRepository interface {
	FindAll(ctx context.Context) ([]domain.Product, error)
}

// Persist LinUCB state per slot
type BanditStateRepository interface {
	GetState(ctx context.Context, slot string) (*LinUCBState, error)
	SaveState(ctx context.Context, slot string, state *LinUCBState) error
}

// ---- Usecase / Service ----

type BanditService struct {
	banditRepo  BanditRepository
	productRepo ProductRepository
	stateRepo   BanditStateRepository
	offlineRepo OfflineRecommendationRepository
	eligChecker EligibilityChecker
	cfgRepo     ConfigRepository
	segmentRepo SegmentRepository
	defaultCfg  Config
}

func NewBanditService(
	banditRepo BanditRepository,
	productRepo ProductRepository,
	stateRepo BanditStateRepository,
	eligChecker EligibilityChecker,
	offlineRepo OfflineRecommendationRepository,
	cfgRepo ConfigRepository,
	segmentRepo SegmentRepository,
	defaultCfg Config,
) *BanditService {
	return &BanditService{
		banditRepo:  banditRepo,
		productRepo: productRepo,
		stateRepo:   stateRepo,
		offlineRepo: offlineRepo,
		eligChecker: eligChecker,
		cfgRepo:     cfgRepo,
		segmentRepo: segmentRepo,
		defaultCfg:  defaultCfg,
	}
}

func (s *BanditService) LogFeedback(
	ctx context.Context,
	event domain.BanditEvent,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	// 1) derive cfg + segment + variant using the same A/B engine as Recommend.
	cfg, seg, variant := s.loadConfigForUser(ctx, event.UserID, event.Slot)
	now := time.Now()
	platform := ""
	if event.Context != nil {
		if p, ok := event.Context["platform"].(string); ok {
			platform = p
		}
	}
	baseCtx := buildBaseContext(now, platform, seg, variant)

	// merge existing event.Context (from client) with baseCtx
	event.Context = mergeContext(baseCtx, event.Context)

	// 2) compute reward using config-aware business rules
	reward, err := cfg.RewardForEvent(event)
	if err != nil {
		return err
	}

	// keep variant info in event for later analysis
	event.Variant = variant

	tid := TraceIDFromContext(ctx)
	logger.Debug("bandit_feedback",
		"trace_id", tid,
		"user_id", event.UserID,
		"slot", event.Slot,
		"product_id", event.ProductID,
		"event_type", event.EventType,
		"value", event.Value,
		"segment", seg,
		"variant", variant,
		"reward", reward,
	)

	event.Variant = variant

	// 2) slotKey for state
	slotKey := stateSlotKey(event.Slot, seg)

	// load state for this slot+segment
	state, err := s.stateRepo.GetState(ctx, slotKey)
	if err != nil {
		return fmt.Errorf("failed to get bandit state: %w", err)
	}
	if state == nil {
		state = newDefaultState()
	}

	// arm for this product
	pid := event.ProductID
	arm, ok := state.Arms[pid]
	if !ok {
		arm = newArmState()
		state.Arms[pid] = arm
	}

	// rebuild feature vector using cfg + segment
	x := buildFeatureVector(event.UserID, event.Slot, event.ProductID, cfg, seg, event.Context)

	// Apply decay so old behavior slowly fades
	applyDecay(arm)

	// LinUCB update: A += x x^T, b += r x
	addOuter(&arm.A, x)
	addScaled(&arm.B, x, reward)
	arm.Count++
	arm.LastUpdated = time.Now()

	capArms(state)
	// persist updated state + raw event log
	if err := s.stateRepo.SaveState(ctx, slotKey, state); err != nil {
		return fmt.Errorf("failed to save bandit state: %w", err)
	}

	if err := s.banditRepo.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save bandit event: %w", err)
	}

	return nil
}

// Recommend returns N products for a user & slot using LinUCB
// on top of offline recommendations from mock_recommendations.
func (s *BanditService) Recommend(
	ctx context.Context,
	userID uint,
	slot string,
	limit int,
	reqCtx map[string]any,
) ([]domain.BanditRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}

	// 1) load offline candidates (mock_recommendations or products)
	offlineRows, limit, err := s.loadCandidates(ctx, slot, limit)
	if err != nil {
		return nil, err
	}
	if len(offlineRows) == 0 {
		return []domain.BanditRecommendation{}, nil
	}

	// 2) config + segment + variant for this user & slot
	cfg, seg, variant := s.loadConfigForUser(ctx, userID, slot)

	// build base context (time, dow, segment, variant, platform)
	now := time.Now()
	platform := ""
	if reqCtx != nil {
		if p, ok := reqCtx["platform"].(string); ok {
			platform = p
		}
	}
	baseCtx := buildBaseContext(now, platform, seg, variant)

	// fullCtx = base + request-provided ctx (page_name, user_segment_override, etc.)
	fullCtx := mergeContext(baseCtx, reqCtx)

	slotKey := stateSlotKey(slot, seg)

	// trace logging
	tid := TraceIDFromContext(ctx)
	logger.Debug("bandit_recommend",
		"trace_id", tid,
		"user_id", userID,
		"slot", slot,
		"segment", seg,
		"variant", variant,
		"limit", limit,
		"candidate_count", len(offlineRows),
	)

	// 3) load / init bandit state for this (slot,segment)
	state, err := s.stateRepo.GetState(ctx, slotKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bandit state: %w", err)
	}
	if state == nil {
		state = newDefaultState()
	}

	// 4) RANDOM COLD START INJECTION (Option C – part B)
	// find 1 offline candidate that does not yet have an arm in state
	var coldProduct uint64
	for _, row := range offlineRows {
		if _, ok := state.Arms[row.ProductID]; !ok {
			coldProduct = row.ProductID
			break
		}
	}

	if coldProduct > 0 {
		offlineRows = append(offlineRows, domain.MockRecommendation{
			ProductID: coldProduct,
			Score:     0.0, // let bandit scoring handle it
		})
	}

	// 5) score candidates (Option C – part A cold-boost is inside)
	recs := s.scoreCandidates(ctx, userID, slot, offlineRows, state, cfg, seg, variant, limit, fullCtx)

	// 6) save updated state
	if err := s.stateRepo.SaveState(ctx, slotKey, state); err != nil {
		return nil, fmt.Errorf("failed to save bandit state: %w", err)
	}

	return recs, nil
}

// scoreCandidates combines offline score + bandit UCB into final scores.
// It is A/B-aware via the `variant` and tunable via `cfg`.
func (s *BanditService) scoreCandidates(
	ctx context.Context,
	userID uint,
	slot string,
	offlineRows []domain.MockRecommendation,
	state *LinUCBState,
	cfg Config,
	segment int,
	variant int,
	limit int,
	ctxMap map[string]any,
) []domain.BanditRecommendation {

	if len(offlineRows) == 0 || limit <= 0 {
		return []domain.BanditRecommendation{}
	}
	if limit > len(offlineRows) {
		limit = len(offlineRows)
	}

	// normalize offline score
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
		prodID uint64
		score  float64
	}

	scoredList := make([]scored, 0, len(offlineRows))

	for _, row := range offlineRows {
		pid := row.ProductID

		// eligibility filter (stock, hub, etc.)
		if s.eligChecker != nil {
			ok, err := s.eligChecker.IsEligible(ctx, userID, pid, slot)
			if err != nil || !ok {
				continue
			}
		}

		arm, ok := state.Arms[pid]
		if !ok {
			arm = newArmState()
			state.Arms[pid] = arm
		}

		// feature vector for this impression
		x := buildFeatureVector(userID, slot, pid, cfg, segment, ctxMap)

		// A^-1
		AInv, err := invert4x4(arm.A)
		if err != nil {
			arm = newArmState()
			state.Arms[pid] = arm
			AInv, _ = invert4x4(arm.A)
		}
		theta := matVecMul(AInv, arm.B)

		offlineNorm := row.Score / maxScore

		// === ALGO-LEVEL VARIANT SWITCH (live path) ===
		var banditScore float64
		switch variant {
		case VariantOfflineOnly:
			// pure offline; no bandit contribution
			banditScore = 0.0
		case VariantThompson:
			banditScore = thompsonScore(theta, x, AInv)
		case VariantUCB:
			fallthrough
		default:
			banditScore = ucbScore(theta, x, AInv, cfg.Alpha)
		}

		final := cfg.WBandit*banditScore + cfg.WOffline*offlineNorm

		// optional: exploration noise only for bandit variants
		if variant != VariantOfflineOnly && cfg.ExploreNoise > 0 {
			final += cfg.ExploreNoise * rand.Float64()
		}

		scoredList = append(scoredList, scored{
			prodID: pid,
			score:  final,
		})
	}

	// sort top-N by final score (simple selection)
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

	out := make([]domain.BanditRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, domain.BanditRecommendation{
			ProductID: scoredList[i].prodID,
			Score:     scoredList[i].score,
		})
	}

	return out
}
