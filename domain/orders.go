package domain

import "time"

type Orders struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	ProductID     int       `json:"product_id"`
	Quantity      int       `json:"quantity"`
	PriceEach     float64   `json:"price_each"`
	Subtotal      float64   `json:"subtotal"`
	OrderStatus   string    `json:"order_status"`
	PaymentMethod string    `json:"payment_method"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
