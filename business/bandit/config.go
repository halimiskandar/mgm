package bandit

import (
	"context"
	"myGreenMarket/domain"
)

type Config struct {
	NumSegments  int
	NumVariants  int
	WBandit      float64
	WOffline     float64
	ExploreNoise float64
	Alpha        float64

	// how much monetary value influences the reward
	ValueWeight float64

	// per-state arm cap (both global & user states)
	MaxArmsPerState int

	//how much global vs user bandit scores matter
	WGlobal float64
	WUser   float64

	// business-context rewards per event type
	RewardImpression float64
	RewardClick      float64
	RewardATC        float64
	RewardOrder      float64

	Features FeatureFlags
}

type VariantConfig struct {
	PctOfflineOnly int
	PctUCB         int
	PctThompson    int
}

const (
	defaultWBandit          = 0.7
	defaultWOffline         = 0.3
	defaultExploreNoise     = 0.05
	defaultAlpha            = 1.0
	defaultWGlobal          = 0.7
	defaultWUser            = 0.3
	defaultValueWeight      = 0.0001
	defaultRewardImpression = 0.0
	defaultRewardClick      = 1.0
	defaultRewardATC        = 3.0
	defaultRewardOrder      = 5.0
	defaultNumSegments      = 3
	defaultNumVariants      = 3
	defaultMaxArmsPerState  = 300
)

func DefaultConfig() Config {
	return Config{
		NumSegments:  defaultNumSegments,
		NumVariants:  defaultNumVariants,
		WBandit:      defaultWBandit,
		WOffline:     defaultWOffline,
		ExploreNoise: defaultExploreNoise,
		Alpha:        defaultAlpha,

		WGlobal: defaultWGlobal,
		WUser:   defaultWUser,

		ValueWeight:      defaultValueWeight,
		MaxArmsPerState:  defaultMaxArmsPerState,
		RewardImpression: defaultRewardImpression,
		RewardClick:      defaultRewardClick,
		RewardATC:        defaultRewardATC,
		RewardOrder:      defaultRewardOrder,

		Features: FeatureFlags{
			UseBias:        true,
			UseTimeBucket:  true,
			UseDowBucket:   true,
			UseSlotHash:    true,
			UseSegment:     true,
			UseProductHash: true,
			UseUserHash:    false,
		},
	}
}

// read per-slot/per-variant bandit config from DB.
type ConfigRepository interface {
	GetConfig(ctx context.Context, slot string, variant int) (domain.BanditConfig, bool, error)
	UpsertConfig(ctx context.Context, cfg domain.BanditConfig) error
}

// read user segment from DB (if exists).
type SegmentRepository interface {
	GetSegment(ctx context.Context, userID uint) (int, bool, error)
	UpsertSegment(ctx context.Context, userID uint, segment int) error
}
