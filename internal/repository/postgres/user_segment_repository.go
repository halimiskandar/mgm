package postgres

import (
	"context"
	"myGreenMarket/business/bandit"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserBanditSegment struct {
	UserID    uint  `gorm:"column:user_id;primaryKey"`
	Segment   int   `gorm:"column:segment;not null"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type UserSegmentRepository struct {
	DB *gorm.DB
}

var _ bandit.SegmentRepository = (*UserSegmentRepository)(nil)

func NewUserSegmentRepository(db *gorm.DB) *UserSegmentRepository {
	return &UserSegmentRepository{DB: db}
}

func (r *UserSegmentRepository) GetSegment(ctx context.Context, userID uint) (int, bool, error) {
	var row UserBanditSegment
	err := r.DB.WithContext(ctx).First(&row, "user_id = ?", userID).Error
	if err == gorm.ErrRecordNotFound {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return row.Segment, true, nil
}

func (r *UserSegmentRepository) UpsertSegment(ctx context.Context, userID uint, segment int) error {
	row := UserBanditSegment{
		UserID:  userID,
		Segment: segment,
	}
	return r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"segment", "updated_at"}),
		}).
		Create(&row).Error
}
