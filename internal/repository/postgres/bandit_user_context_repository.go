package postgres

import (
	"context"
	"database/sql"

	"myGreenMarket/business/bandit"

	"gorm.io/gorm"
)

// PostgresUserContextRepository implements bandit.UserContextRepository
// using the "users" table.

type PostgresUserContextRepository struct {
	DB *gorm.DB
}

// Compile-time check that the struct implements the interface.
var _ bandit.UserContextRepository = (*PostgresUserContextRepository)(nil)

func NewUserContextRepository(db *gorm.DB) *PostgresUserContextRepository {
	return &PostgresUserContextRepository{DB: db}
}

func (r *PostgresUserContextRepository) GetUserContext(
	ctx context.Context,
	userID uint,
) (bandit.UserContext, error) {

	var row struct {
		Tier       sql.NullString `gorm:"column:tier"`
		CampaignID sql.NullString `gorm:"column:current_campaign_id"`
	}

	err := r.DB.WithContext(ctx).
		Table("users").
		Select("tier, current_campaign_id").
		Where("id = ?", userID).
		Scan(&row).Error
	if err != nil {
		// If user not found, just return empty context (no error)
		if err == gorm.ErrRecordNotFound {
			return bandit.UserContext{}, nil
		}
		return bandit.UserContext{}, err
	}

	uc := bandit.UserContext{}
	if row.Tier.Valid {
		uc.Tier = row.Tier.String
	}
	if row.CampaignID.Valid {
		uc.CampaignID = row.CampaignID.String
	}

	return uc, nil
}
