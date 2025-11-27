package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"myGreenMarket/business/bandit"
	"myGreenMarket/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BanditRepository struct {
	DB *gorm.DB
}

func NewBanditRepository(db *gorm.DB) *BanditRepository {
	return &BanditRepository{DB: db}
}

// ---- Events ----

func (r *BanditRepository) SaveEvent(ctx context.Context, event domain.BanditEvent) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	if err := r.DB.WithContext(ctx).Create(&event).Error; err != nil {
		return fmt.Errorf("failed to save bandit event: %w", err)
	}

	return nil
}

// ---- State ----

type banditStateRow struct {
	Slot      string `gorm:"column:slot;primaryKey"`
	StateJSON []byte `gorm:"column:state_json"`
}

func (banditStateRow) TableName() string {
	return "bandit_state"
}

func (r *BanditRepository) GetState(ctx context.Context, slot string) (*bandit.LinUCBState, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	var row banditStateRow
	err := r.DB.WithContext(ctx).First(&row, "slot = ?", slot).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query bandit_state: %w", err)
	}

	var state bandit.LinUCBState
	if err := json.Unmarshal(row.StateJSON, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state_json: %w", err)
	}

	return &state, nil
}

func (r *BanditRepository) SaveState(ctx context.Context, slot string, state *bandit.LinUCBState) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	row := banditStateRow{
		Slot:      slot,
		StateJSON: raw,
	}

	if err := r.DB.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "slot"}},
			UpdateAll: true,
		},
	).Create(&row).Error; err != nil {
		return fmt.Errorf("failed to upsert bandit_state: %w", err)
	}

	return nil
}
