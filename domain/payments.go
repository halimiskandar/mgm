package domain

import "time"

type (
	Payments struct {
		ID            int       `json:"id"`
		OrderID       int       `json:"order_id"`
		PaymentStatus string    `json:"payment_status"`
		PaymentMethod string    `json:"payment_method"`
		CreatedAt     time.Time `json:"created_at"`
	}

	PaymentWithLink struct {
		ID            int       `json:"id"`
		OrderID       int       `json:"order_id"`
		PaymentStatus string    `json:"payment_status"`
		PaymentMethod string    `json:"payment_method"`
		PaymentLink   string    `json:"payment_link"`
		CreatedAt     time.Time `json:"created_at"`
	}
)
