package postgres

import (
	"context"
	"encoding/json"
	"myGreenMarket/business/bandit"
	"myGreenMarket/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BanditConfigRepository struct {
	DB *gorm.DB
}

var _ bandit.ConfigRepository = (*BanditConfigRepository)(nil)

func NewBanditConfigRepository(db *gorm.DB) *BanditConfigRepository {
	return &BanditConfigRepository{DB: db}
}

func (r *BanditConfigRepository) GetConfig(ctx context.Context, slot string, variant int) (domain.BanditConfig, bool, error) {
	var cfg domain.BanditConfig

	err := r.DB.WithContext(ctx).
		Where("slot = ? AND variant = ?", slot, variant).
		First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return domain.BanditConfig{}, false, nil
	}
	if err != nil {
		return domain.BanditConfig{}, false, err
	}

	if len(cfg.FeaturesRaw) > 0 {
		_ = json.Unmarshal(cfg.FeaturesRaw, &cfg.Features)
	}
	return cfg, true, nil
}

func (r *BanditConfigRepository) UpsertConfig(ctx context.Context, cfg domain.BanditConfig) error {
	// if Features struct is set but FeaturesRaw is empty, serialize it
	if len(cfg.FeaturesRaw) == 0 && (cfg.Features != (domain.BanditFeatureFlags{})) {
		raw, _ := json.Marshal(cfg.Features)
		cfg.FeaturesRaw = raw
	}
	return r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slot"}, {Name: "variant"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"num_segments",
				"num_variants",
				"w_bandit",
				"w_offline",
				"explore_noise",
				"alpha",
				"value_weight",
				"reward_impression",
				"reward_click",
				"reward_atc",
				"reward_order",
				"features",
				"updated_at",
			}),
		}).
		Create(&cfg).Error
}
