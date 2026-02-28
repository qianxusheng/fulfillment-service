package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/qianxusheng/fulfillment-service/internal/handler"
	"github.com/qianxusheng/fulfillment-service/internal/repository"
)

func main() {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatal("TABLE_NAME environment variable is required")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	repo := repository.NewDynamoDBRepository(dynamoClient, tableName)
	h := handler.NewHandler(repo)

	lambda.Start(h.HandleSQSEvent)
}
