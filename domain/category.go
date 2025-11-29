package domain

import (
	"time"
)

// CREATE TABLE public.category (
//     category_id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
//     product_category    TEXT NOT NULL,
//     created_at          TIMESTAMPTZ DEFAULT NOW()
// );

type Category struct {
	CategoryID      uint64    `gorm:"primaryKey;column:category_id;autoIncrement"`
	ProductCategory string    `gorm:"column:product_category;type:text;not null"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (Category) TableName() string {
	return "categories"
}
