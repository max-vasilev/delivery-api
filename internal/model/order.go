package model

import (
	"time"
	"uuid"
)

type Order struct {
	ID        uuid.UUID   `json:"id"`
	Address   string      `json:"address"`
	Price     int64       `json:"price"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID       uuid.UUID `json:"id"`
	OrderID  int       `json:"order_id"`
	Name     string    `json:"name"`
	Quantity int       `json:"quantity"`
	Price    int64     `json:"price"`
}
