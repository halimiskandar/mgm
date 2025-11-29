package domain

import "time"

type (
	Payments struct {
		ID            int       `json:"id"`
		UserID        int       `json:"user_id"`
		OrderID       *int      `json:"order_id"`
		PaymentType   string    `json:"payment_type"`
		PaymentStatus string    `json:"payment_status"`
		PaymentMethod string    `json:"payment_method"`
		CreatedAt     time.Time `json:"created_at"`
	}

	PaymentWithLink struct {
		ID            int       `json:"id"`
		UserID        int       `json:"user_id"`
		OrderID       int       `json:"order_id"`
		PaymentStatus string    `json:"payment_status"`
		PaymentMethod string    `json:"payment_method"`
		PaymentLink   string    `json:"payment_link"`
		CreatedAt     time.Time `json:"created_at"`
	}

	TopUp struct {
		ID        int     `json:"id"`
		UserID    uint    `json:"user_id"`
		Amount    float64 `json:"amount"`
		TopUpLink string  `json:"top_up_link"`
	}
)
