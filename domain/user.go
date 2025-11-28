package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID         uint    `gorm:"primaryKey"`
	FullName   string  `gorm:"column:full_name;not null"`
	Email      string  `gorm:"column:email;unique;not null"`
	IsVerified bool    `gorm:"column:is_verified;default:false"`
	Password   string  `gorm:"column:password;not null"`
	Role       string  `gorm:"column:role;default:customer"`
	Wallet     float64 `gorm:"column:wallet;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}
