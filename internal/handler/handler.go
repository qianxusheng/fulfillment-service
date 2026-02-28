package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/qianxusheng/fulfillment-service/internal/models"
	"github.com/qianxusheng/fulfillment-service/internal/repository"
)

type Handler struct {
	repo *repository.DynamoDBRepository
}

func NewHandler(repo *repository.DynamoDBRepository) *Handler {
	return &Handler{
		repo: repo,
	}
}

func (h *Handler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		if err := h.processMessage(ctx, record); err != nil {
			log.Printf("Failed to process message %s: %v", record.MessageId, err)
			return err
		}
	}
	return nil
}

func (h *Handler) processMessage(ctx context.Context, record events.SQSMessage) error {
	var order models.Order
	if err := json.Unmarshal([]byte(record.Body), &order); err != nil {
		log.Printf("Invalid message format: %v", err)
		return nil
	}

	if err := order.Validate(); err != nil {
		log.Printf("Invalid order %s: %v", order.OrderID, err)
		return nil
	}

	order.SetTimestamps()
	order.SetInitialStatus()

	err := h.repo.SaveOrder(ctx, &order)
	if err != nil {
		if errors.Is(err, repository.ErrOrderAlreadyExists) {
			log.Printf("Order %s already exists (idempotent), skipping", order.OrderID)
			return nil
		}
		return err
	}

	log.Printf("Successfully processed order %s for user %s, amount: %d %s",
		order.OrderID, order.UserID, order.Amount, order.Currency)

	return nil
}
