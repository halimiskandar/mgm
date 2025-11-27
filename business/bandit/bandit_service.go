package bandit

import (
	"context"
	"fmt"
	"math"
	"myGreenMarket/domain"
	"time"
)

const (
	// How much we trust bandit vs offline model.
	// Larger wBandit => more exploration & personalization.
	// Larger wOffline => more stable, closer to offline ranking.
	wBandit  = 0.7
	wOffline = 0.3
)

// ---- Repository interfaces ----
// Offline model output from mock_recommendations
type OfflineRecommendationRepository interface {
	GetBySlot(ctx context.Context, slot string, limit int) ([]domain.MockRecommendation, error)
}

// Write feedback events (raw log)
type BanditRepository interface {
	SaveEvent(ctx context.Context, event domain.BanditEvent) error
}

// Read product candidates
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
}

func NewBanditService(
	banditRepo BanditRepository,
	productRepo ProductRepository,
	stateRepo BanditStateRepository,
	offlineRepo OfflineRecommendationRepository,
) *BanditService {
	return &BanditService{
		banditRepo:  banditRepo,
		productRepo: productRepo,
		stateRepo:   stateRepo,
	}
}

// Recommend N products for a user & slot using LinUCB.
// on top of offline recommendations from mock_recommendations.
func (s *BanditService) Recommend(
	ctx context.Context,
	userID uint,
	slot string,
	limit int,
) ([]domain.BanditRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}

	//offline model output = candidate pool
	candidateLimit := limit * 3
	if candidateLimit < limit {
		candidateLimit = limit
	}

	offlineRows, err := s.offlineRepo.GetBySlot(ctx, slot, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to load offline recommendations: %w", err)
	}
	if len(offlineRows) == 0 {
		return []domain.BanditRecommendation{}, nil
	}
	if len(offlineRows) < limit {
		limit = len(offlineRows)
	}

	//load bandit state for this slot
	state, err := s.stateRepo.GetState(ctx, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to get bandit state: %w", err)
	}

	// Fallback: no bandit state yet → just offline
	if state == nil {
		recs := make([]domain.BanditRecommendation, 0, limit)
		for i := 0; i < limit; i++ {
			row := offlineRows[i]
			recs = append(recs, domain.BanditRecommendation{
				ProductID: row.ProductID,
				Score:     row.Score,
			})
		}
		return recs, nil
	}

	// normalize offline scores to [0,1]
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
		id    uint64
		score float64
	}

	scores := make([]scored, 0, len(offlineRows))

	for _, row := range offlineRows {
		pid := row.ProductID

		arm, ok := state.Arms[pid]
		if !ok {
			arm = newArmState()
			state.Arms[pid] = arm
		}

		x := buildFeatureVector(userID, slot, pid)

		AInv, err := invert4x4(arm.A)
		if err != nil {
			arm = newArmState()
			state.Arms[pid] = arm
			AInv, _ = invert4x4(arm.A)
		}

		theta := matVecMul(AInv, arm.B)
		mean := dot(theta, x)

		tmp := matVecMul(AInv, x)
		uncertainty := math.Sqrt(dot(x, tmp))

		ucb := mean + state.Alpha*uncertainty
		offlineNorm := row.Score / maxScore

		final := wBandit*ucb + wOffline*offlineNorm

		scores = append(scores, scored{
			id:    pid,
			score: final,
		})
	}

	//top-N by final score
	if len(scores) < limit {
		limit = len(scores)
	}

	for i := 0; i < limit; i++ {
		maxIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[maxIdx].score {
				maxIdx = j
			}
		}
		scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
	}

	recs := make([]domain.BanditRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		recs = append(recs, domain.BanditRecommendation{
			ProductID: scores[i].id,
			Score:     scores[i].score,
		})
	}

	return recs, nil
}

// Map event type → numeric reward
func eventTypeToReward(evType string) (float64, error) {
	switch evType {
	case "impression":
		return 0.0, nil
	case "click":
		return 0.3, nil
	case "atc":
		return 0.7, nil
	case "order":
		return 1.0, nil
	default:
		return 0, fmt.Errorf("unknown event type: %s", evType)
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

	// convert to reward
	reward, err := eventTypeToReward(event.EventType)
	if err != nil {
		return err
	}

	// load state for this slot
	state, err := s.stateRepo.GetState(ctx, event.Slot)
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

	// rebuild feature vector (OK approximation)
	x := buildFeatureVector(event.UserID, event.Slot, event.ProductID)

	// LinUCB update: A += x x^T, b += r x
	addOuter(&arm.A, x)
	addScaled(&arm.B, x, reward)
	arm.Count++

	// persist updated state + raw event log
	if err := s.stateRepo.SaveState(ctx, event.Slot, state); err != nil {
		return fmt.Errorf("failed to save bandit state: %w", err)
	}

	if err := s.banditRepo.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save bandit event: %w", err)
	}

	return nil
}

// ---- Feature vector + math helpers ----
func buildFeatureVector(userID uint, slot string, productID uint64) [linUCBFeatureDim]float64 {
	hour := time.Now().Hour()

	return [linUCBFeatureDim]float64{
		1.0,                           // bias
		float64(hour) / 24.0,          // time of day
		float64(userID%1000) / 1000.0, // hashed user
		float64(productID%1000) / 1000.0,
	}
}

// y = A * x
func matVecMul(A [linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) [linUCBFeatureDim]float64 {
	var y [linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		sum := 0.0
		for j := 0; j < linUCBFeatureDim; j++ {
			sum += A[i][j] * x[j]
		}
		y[i] = sum
	}
	return y
}

func dot(a, b [linUCBFeatureDim]float64) float64 {
	sum := 0.0
	for i := 0; i < linUCBFeatureDim; i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// A := A + x x^T
func addOuter(A *[linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) {
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			(*A)[i][j] += x[i] * x[j]
		}
	}
}

// b := b + r x
func addScaled(b *[linUCBFeatureDim]float64, x [linUCBFeatureDim]float64, r float64) {
	for i := 0; i < linUCBFeatureDim; i++ {
		(*b)[i] += r * x[i]
	}
}

// Invert 4x4 matrix using Gauss-Jordan.
func invert4x4(A [linUCBFeatureDim][linUCBFeatureDim]float64) ([linUCBFeatureDim][linUCBFeatureDim]float64, error) {
	// augment A | I
	var aug [linUCBFeatureDim][2 * linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			aug[i][j] = A[i][j]
		}
		aug[i][linUCBFeatureDim+i] = 1.0
	}

	for col := 0; col < linUCBFeatureDim; col++ {
		pivot := aug[col][col]
		if math.Abs(pivot) < 1e-9 {
			return [linUCBFeatureDim][linUCBFeatureDim]float64{}, fmt.Errorf("matrix is singular")
		}
		// normalize pivot row
		for j := 0; j < 2*linUCBFeatureDim; j++ {
			aug[col][j] /= pivot
		}
		// eliminate other rows
		for i := 0; i < linUCBFeatureDim; i++ {
			if i == col {
				continue
			}
			factor := aug[i][col]
			for j := 0; j < 2*linUCBFeatureDim; j++ {
				aug[i][j] -= factor * aug[col][j]
			}
		}
	}

	var inv [linUCBFeatureDim][linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			inv[i][j] = aug[i][linUCBFeatureDim+j]
		}
	}
	return inv, nil
}

// DebugRecommend returns detailed score components for inspection.
func (s *BanditService) DebugRecommend(
	ctx context.Context,
	userID uint,
	slot string,
	limit int,
) ([]domain.DebugRecommendation, error) {

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}

	// 1) offline candidates
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

	// 2) state
	state, err := s.stateRepo.GetState(ctx, slot)
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

	// If no state yet → show offline-only with bandit fields = 0
	if state == nil {
		for _, row := range offlineRows {
			offlineNorm := row.Score / maxScore
			final := wOffline * offlineNorm // bandit part is 0

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
		for _, row := range offlineRows {
			pid := row.ProductID

			arm, ok := state.Arms[pid]
			if !ok {
				arm = newArmState()
				state.Arms[pid] = arm
			}

			x := buildFeatureVector(userID, slot, pid)

			AInv, err := invert4x4(arm.A)
			if err != nil {
				arm = newArmState()
				state.Arms[pid] = arm
				AInv, _ = invert4x4(arm.A)
			}

			theta := matVecMul(AInv, arm.B)
			mean := dot(theta, x)

			tmp := matVecMul(AInv, x)
			uncertainty := math.Sqrt(dot(x, tmp))

			ucb := mean + state.Alpha*uncertainty
			offlineNorm := row.Score / maxScore
			final := wBandit*ucb + wOffline*offlineNorm

			result = append(result, scored{
				rec: domain.DebugRecommendation{
					ProductID:         pid,
					OfflineScore:      row.Score,
					OfflineNormalized: offlineNorm,
					BanditMean:        mean,
					BanditUncertainty: uncertainty,
					BanditUCB:         ucb,
					FinalScore:        final,
				},
				score: final,
			})
		}
	}

	// top-N by final score
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
