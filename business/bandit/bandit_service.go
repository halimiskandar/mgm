package bandit

import (
	"context"
	"fmt"
	"math/rand"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"

	"strconv"
	"time"

	"gorm.io/datatypes"
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

// ---- context helpers ----

// buildBaseContext builds the standard context used for both recommendation & feedback.
func buildBaseContext(now time.Time, platform string, segment, variant int) map[string]any {
	return map[string]any{
		"time_bucket": computeTimeBucket(now),
		"dow":         int(now.Weekday()), // 0=Sunday
		"platform":    platform,
		"segment":     segment,
		"variant":     variant,
		"event_time":  now.Format(time.RFC3339),
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

type OfflineRecommendationRepository interface {
	GetBySlot(ctx context.Context, slot string, limit int) ([]domain.MockRecommendation, error)
}

type BanditRepository interface {
	SaveEvent(ctx context.Context, event domain.BanditEvent) error
}

type ProductRepository interface {
	FindAll(ctx context.Context) ([]domain.Product, error)
}

type BanditStateRepository interface {
	GetState(ctx context.Context, key string) (*LinUCBState, error)
	SaveState(ctx context.Context, key string, state *LinUCBState) error
}

type UserContext struct {
	Tier       string
	CampaignID string
}

type UserContextRepository interface {
	GetUserContext(ctx context.Context, userID uint) (UserContext, error)
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
	userCtxRepo UserContextRepository
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
	userCtxRepo UserContextRepository,
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
		userCtxRepo: userCtxRepo,
		defaultCfg:  defaultCfg,
	}
}

//  Feedback / learning

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

	// 1) derive cfg + segment + variant
	cfg, seg, variant := s.loadConfigForUser(ctx, event.UserID, event.Slot)

	now := time.Now()

	// convert event.Context (JSONMap) into plain map[string]any for merging
	eventCtxMap := map[string]any{}
	if event.Context != nil {
		for k, v := range event.Context {
			eventCtxMap[k] = v
		}
	}

	platform := ""
	if p, ok := eventCtxMap["platform"].(string); ok {
		platform = p
	}

	baseCtx := buildBaseContext(now, platform, seg, variant)

	// enrich with user_tier & campaign_id from DB
	if s.userCtxRepo != nil {
		if uc, err := s.userCtxRepo.GetUserContext(ctx, event.UserID); err == nil {
			if uc.Tier != "" {
				baseCtx["user_tier"] = uc.Tier
			}
			if uc.CampaignID != "" {
				baseCtx["campaign_id"] = uc.CampaignID
			}
		}
	}

	// merged feedback context = base + client-provided context
	mergedCtx := mergeContext(baseCtx, eventCtxMap)

	// write back into event.Context as JSONMap for DB persistence
	event.Context = datatypes.JSONMap(mergedCtx)

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

	// 3) load global + user states
	globalKey := stateGlobalKey(event.Slot, seg)
	userKey := stateUserKey(event.Slot, seg, event.UserID)

	globalState, err := s.stateRepo.GetState(ctx, globalKey)
	if err != nil {
		return fmt.Errorf("load global state: %w", err)
	}
	if globalState == nil {
		globalState = newDefaultState()
	}

	userState, err := s.stateRepo.GetState(ctx, userKey)
	if err != nil {
		return fmt.Errorf("load user state: %w", err)
	}
	if userState == nil {
		userState = newDefaultState()
	}

	pid := event.ProductID

	// GLOBAL arm
	gArm, ok := globalState.Arms[pid]
	if !ok {
		gArm = newArmState()
		globalState.Arms[pid] = gArm
	}

	// USER arm
	uArm, ok := userState.Arms[pid]
	if !ok {
		uArm = newArmState()
		userState.Arms[pid] = uArm
	}

	// feature vector using merged event.Context
	x := buildFeatureVector(event.UserID, event.Slot, event.ProductID, cfg, seg, mergedCtx)

	// Apply decay then update both arms
	applyDecay(gArm)
	applyDecay(uArm)

	addOuter(&gArm.A, x)
	addScaled(&gArm.B, x, reward)
	gArm.Count++
	gArm.LastUpdated = time.Now()

	addOuter(&uArm.A, x)
	addScaled(&uArm.B, x, reward)
	uArm.Count++
	uArm.LastUpdated = time.Now()

	maxArms := cfg.MaxArmsPerState
	capArms(globalState, maxArms)
	capArms(userState, maxArms)

	// 4) persist updated states + raw event log
	if err := s.stateRepo.SaveState(ctx, globalKey, globalState); err != nil {
		return fmt.Errorf("failed to save global bandit state: %w", err)
	}
	if err := s.stateRepo.SaveState(ctx, userKey, userState); err != nil {
		return fmt.Errorf("failed to save user bandit state: %w", err)
	}

	if err := s.banditRepo.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save bandit event: %w", err)
	}

	// increment Prometheus counter AFTER we successfully process the event
	segLabel := strconv.Itoa(seg)
	varLabel := strconv.Itoa(variant)

	BanditFeedbackEventsTotal.
		WithLabelValues(event.Slot, event.EventType, segLabel, varLabel).
		Inc()

	return nil
}

//  Recommendation / serving

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

	// 1) load offline candidates
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
	fullCtx := mergeContext(baseCtx, reqCtx)

	// 3) load global + user states
	globalKey := stateGlobalKey(slot, seg)
	userKey := stateUserKey(slot, seg, userID)

	globalState, err := s.stateRepo.GetState(ctx, globalKey)
	if err != nil {
		return nil, fmt.Errorf("load global state: %w", err)
	}
	if globalState == nil {
		globalState = newDefaultState()
	}

	userState, err := s.stateRepo.GetState(ctx, userKey)
	if err != nil {
		return nil, fmt.Errorf("load user state: %w", err)
	}
	if userState == nil {
		userState = newDefaultState()
	}

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

	// 4) score candidates with global + user state
	recs := s.scoreCandidates(
		ctx,
		userID,
		slot,
		offlineRows,
		globalState,
		userState,
		cfg,
		seg,
		variant,
		limit,
		fullCtx,
	)

	// after scoring, save both states
	if err := s.stateRepo.SaveState(ctx, globalKey, globalState); err != nil {
		return nil, fmt.Errorf("save global state: %w", err)
	}
	if err := s.stateRepo.SaveState(ctx, userKey, userState); err != nil {
		return nil, fmt.Errorf("save user state: %w", err)
	}

	return recs, nil
}

// ---- Scoring ----

// scoreCandidates combines offline score + (global + user) bandit UCB into final scores.
func (s *BanditService) scoreCandidates(
	ctx context.Context,
	userID uint,
	slot string,
	offlineRows []domain.MockRecommendation,
	globalState *LinUCBState,
	userState *LinUCBState,
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

	wGlobal := cfg.WGlobal
	wUser := cfg.WUser
	if wGlobal == 0 && wUser == 0 {
		wGlobal = 0.7
		wUser = 0.3
	}

	for _, row := range offlineRows {
		pid := row.ProductID

		// eligibility filter (stock, hub, etc.)
		if s.eligChecker != nil {
			ok, err := s.eligChecker.IsEligible(ctx, userID, pid, slot)
			if err != nil || !ok {
				continue
			}
		}

		// GLOBAL arm (read-only for scoring)
		gArm, ok := globalState.Arms[pid]
		if !ok {
			gArm = newArmState()
		}

		// USER arm (read-only for scoring)
		uArm, ok := userState.Arms[pid]
		if !ok {
			uArm = newArmState()
		}

		// feature vector for this impression
		x := buildFeatureVector(userID, slot, pid, cfg, segment, ctxMap)

		// global A^-1 / theta
		gAInv, err := invert4x4(gArm.A)
		if err != nil {
			gArm = newArmState()
			gAInv, _ = invert4x4(gArm.A)
		}
		gTheta := matVecMul(gAInv, gArm.B)

		// user A^-1 / theta
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
			// pure offline; no bandit contribution
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
