package domain

import "time"

type BanditEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null" json:"user_id"`
	Slot      string    `gorm:"column:slot;not null" json:"slot"`
	ProductID uint64    `gorm:"column:product_id;not null" json:"product_id"`
	EventType string    `gorm:"column:event_type;not null" json:"event_type"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Value   float64 `gorm:"-" json:"value"`   // optional GMV/margin
	Variant int     `gorm:"-" json:"variant"` // A/B bucket
}

type BanditRecommendation struct {
	ProductID uint64  `json:"product_id"`
	Score     float64 `json:"score"`
}
