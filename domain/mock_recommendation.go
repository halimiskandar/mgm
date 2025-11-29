package domain

type MockRecommendation struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Slot      string  `gorm:"column:slot;not null" json:"slot"`
	ProductID uint64  `gorm:"column:product_id;not null" json:"product_id"`
	Score     float64 `gorm:"column:score;not null" json:"score"`
}
