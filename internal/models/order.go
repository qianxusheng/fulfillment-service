package models

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	StatusReceived   OrderStatus = "RECEIVED"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusCompleted  OrderStatus = "COMPLETED"
	StatusFailed     OrderStatus = "FAILED"
)

type Order struct {
	OrderID     string                 `json:"order_id" dynamodbav:"OrderID"`
	UserID      string                 `json:"user_id" dynamodbav:"UserID"`
	ProductType string                 `json:"product_type" dynamodbav:"ProductType"`
	Amount      int                    `json:"amount" dynamodbav:"Amount"`
	Currency    string                 `json:"currency" dynamodbav:"Currency"`
	Status      OrderStatus            `json:"status" dynamodbav:"Status"`
	CreatedAt   string                 `json:"created_at" dynamodbav:"CreatedAt"`
	UpdatedAt   string                 `json:"updated_at" dynamodbav:"UpdatedAt"`
	Version     int                    `json:"version,omitempty" dynamodbav:"Version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" dynamodbav:"Metadata,omitempty"`
}

func (o *Order) Validate() error {
	if o.OrderID == "" {
		return errors.New("order_id is required")
	}
	if o.UserID == "" {
		return errors.New("user_id is required")
	}
	if o.ProductType == "" {
		return errors.New("product_type is required")
	}
	if o.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if o.Currency == "" {
		return errors.New("currency is required")
	}
	return nil
}

func (o *Order) SetTimestamps() {
	now := time.Now().UTC().Format(time.RFC3339)
	if o.CreatedAt == "" {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
}

func (o *Order) SetInitialStatus() {
	if o.Status == "" {
		o.Status = StatusReceived
	}
}
