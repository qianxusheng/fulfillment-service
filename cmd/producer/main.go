package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/qianxusheng/fulfillment-service/internal/models"
)

func main() {
	orderID := flag.String("order-id", "", "Order ID (if empty, auto-generate)")
	userID := flag.String("user-id", "USER-12345", "User ID")
	amount := flag.Int("amount", 99900, "Order amount in cents")
	currency := flag.String("currency", "USD", "Currency code")
	productType := flag.String("product", "FLIGHT_HOTEL_PACKAGE", "Product type")

	count := flag.Int("count", 1, "Number of messages to send")
	interval := flag.Duration("interval", 0, "Interval between messages (e.g., 1s, 100ms)")
	send := flag.Bool("send", false, "Send to SQS (default: print only)")
	queueURL := flag.String("queue-url", "", "SQS queue URL (required if --send is used)")

	flag.Parse()

	if *send && *queueURL == "" {
		log.Fatal("--queue-url is required when --send is used")
	}

	var sqsClient *sqs.Client
	ctx := context.Background()

	if *send {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("Failed to load AWS config: %v", err)
		}
		sqsClient = sqs.NewFromConfig(cfg)
		fmt.Printf("Sending %d message(s) to SQS...\n", *count)
	}

	for i := 0; i < *count; i++ {
		order := generateOrder(*orderID, *userID, *productType, *amount, *currency, i)

		jsonData, err := json.Marshal(order)
		if err != nil {
			log.Fatalf("Failed to marshal order: %v", err)
		}

		if *send {
			_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    aws.String(*queueURL),
				MessageBody: aws.String(string(jsonData)),
			})
			if err != nil {
				log.Printf("Failed to send message %d: %v", i+1, err)
				continue
			}
			fmt.Printf("[%d/%d] Sent order: %s\n", i+1, *count, order.OrderID)
		} else {
			jsonDataPretty, _ := json.MarshalIndent(order, "", "  ")
			fmt.Println("Generated SQS Message:")
			fmt.Println(string(jsonDataPretty))
		}

		if *interval > 0 && i < *count-1 {
			time.Sleep(*interval)
		}
	}

	if *send {
		fmt.Printf("\n✓ Successfully sent %d message(s) to SQS\n", *count)
	}
}

func generateOrder(orderID, userID, productType string, amount int, currency string, seq int) models.Order {
	if orderID == "" {
		orderID = fmt.Sprintf("order-%d-%d", time.Now().Unix(), seq)
	} else if seq > 0 {
		orderID = fmt.Sprintf("%s-%d", orderID, seq)
	}

	hotels := []string{"Hilton Tokyo", "Marriott Shanghai", "Hyatt Beijing", "Sheraton Hong Kong"}
	flights := []string{"CA1234", "MU5678", "CZ9012", "HU3456"}

	return models.Order{
		OrderID:     orderID,
		UserID:      userID,
		ProductType: productType,
		Amount:      amount + rand.Intn(10000),
		Currency:    currency,
		Metadata: map[string]interface{}{
			"flight_number":  flights[rand.Intn(len(flights))],
			"hotel_name":     hotels[rand.Intn(len(hotels))],
			"check_in_date":  "2026-03-01",
			"check_out_date": "2026-03-05",
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
