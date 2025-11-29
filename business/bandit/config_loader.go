package bandit

import (
	"context"
	"fmt"
	"hash/fnv"
)

// main entry point used by Recommend / LogFeedback / DebugRecommend
func (s *BanditService) loadConfigForUser(
	ctx context.Context,
	userID uint,
	slot string,
) (Config, int, int) {
	// 1) base config for slot, variant 0
	baseCfg := s.loadConfig(ctx, slot, 0)

	// 2) stable variant assignment per (user, slot)
	variant := s.assignVariant(userID, slot, baseCfg)

	// 3) load variant-specific config (override base)
	cfg := s.loadConfig(ctx, slot, variant)

	// 4) derive segment (from repo or hash)
	seg := s.userSegment(ctx, userID, cfg)

	return cfg, seg, variant
}

// read config for a given (slot, variant) from repo, falling back to defaultCfg
func (s *BanditService) loadConfig(
	ctx context.Context,
	slot string,
	variant int,
) Config {
	if s.cfgRepo == nil {
		return s.defaultCfg
	}

	dbCfg, ok, err := s.cfgRepo.GetConfig(ctx, slot, variant)
	if err != nil || !ok {
		return s.defaultCfg
	}

	// start from defaults to keep sane fallbacks for any missing fields
	cfg := s.defaultCfg

	// copy fields from DB config
	cfg.NumSegments = dbCfg.NumSegments
	cfg.NumVariants = dbCfg.NumVariants

	cfg.WBandit = dbCfg.WBandit
	cfg.WOffline = dbCfg.WOffline
	cfg.ExploreNoise = dbCfg.ExploreNoise
	cfg.Alpha = dbCfg.Alpha

	cfg.ValueWeight = dbCfg.ValueWeight

	cfg.RewardImpression = dbCfg.RewardImpression
	cfg.RewardClick = dbCfg.RewardClick
	cfg.RewardATC = dbCfg.RewardATC
	cfg.RewardOrder = dbCfg.RewardOrder

	// feature flags
	cfg.Features = FeatureFlags{
		UseBias:        dbCfg.Features.UseBias,
		UseTimeBucket:  dbCfg.Features.UseTimeBucket,
		UseDowBucket:   dbCfg.Features.UseDowBucket,
		UseSlotHash:    dbCfg.Features.UseSlotHash,
		UseSegment:     dbCfg.Features.UseSegment,
		UseProductHash: dbCfg.Features.UseProductHash,
		UseUserHash:    dbCfg.Features.UseUserHash,
	}

	return cfg
}

// userSegment either uses a stored segment or hashes userID into [0, NumSegments)
func (s *BanditService) userSegment(ctx context.Context, userID uint, cfg Config) int {
	if s.segmentRepo != nil {
		if seg, ok, err := s.segmentRepo.GetSegment(ctx, userID); err == nil && ok {
			if cfg.NumSegments > 0 {
				return seg % cfg.NumSegments
			}
			return seg
		}
	}

	if cfg.NumSegments <= 0 {
		cfg.NumSegments = defaultNumSegments
	}

	return int(userID % uint(cfg.NumSegments))
}

// assignVariant hashes (user, slot) into [0, NumVariants)
func (s *BanditService) assignVariant(userID uint, slot string, cfg Config) int {
	if cfg.NumVariants <= 1 {
		return 0
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", userID, slot)))
	v := h.Sum32()

	return int(v % uint32(cfg.NumVariants))
}
