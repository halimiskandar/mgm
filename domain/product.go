package domain

import (
	"time"
)

// CREATE TABLE public.products (
//     id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
//     product_id      BIGINT,
//     product_skuid   BIGINT,
//     is_green_tag    BOOLEAN,
//     product_name    TEXT,
//     product_category TEXT,
//     unit            TEXT,
//     normal_price    NUMERIC,
//     sale_price      NUMERIC,
//     discount        NUMERIC,
//     quantity        NUMERIC,
//     created_at      TIMESTAMPTZ DEFAULT NOW()
// );

type Product struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement"`
	ProductID       uint64    `gorm:"column:product_id"`
	ProductSKUID    uint64    `gorm:"column:product_skuid"`
	CategoryID      uint64    `gorm:"column:category_id;default:0"`
	IsGreenTag      bool      `gorm:"column:is_green_tag;default:false"`
	ProductName     string    `gorm:"column:product_name;type:text"`
	ProductCategory string    `gorm:"column:product_category;type:text"`
	Unit            string    `gorm:"column:unit;type:text"`
	NormalPrice     float64   `gorm:"column:normal_price;type:numeric"`
	SalePrice       float64   `gorm:"column:sale_price;type:numeric"`
	Discount        float64   `gorm:"column:discount;type:numeric"`
	Quantity        float64   `gorm:"column:quantity;type:numeric"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

// TODO: Apakah nambah fitur updated_at dan deleted_at
// UpdatedAt       time.Time      `gorm:"column:updated_at"`
// 	DeletedAt       gorm.DeletedAt `gorm:"index"`

func (Product) TableName() string {
	return "products"
}
