package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/qianxusheng/fulfillment-service/internal/models"
)

var ErrOrderAlreadyExists = errors.New("order already exists")

type DynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBRepository(client *dynamodb.Client, tableName string) *DynamoDBRepository {
	return &DynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *DynamoDBRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	order.SetTimestamps()
	order.SetInitialStatus()

	item, err := attributevalue.MarshalMap(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(OrderID)"),
	})

	if err != nil {
		var condCheckErr *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckErr) {
			return ErrOrderAlreadyExists
		}
		return fmt.Errorf("failed to save order: %w", err)
	}

	return nil
}

func (r *DynamoDBRepository) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"OrderID": &types.AttributeValueMemberS{Value: orderID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	var order models.Order
	err = attributevalue.UnmarshalMap(result.Item, &order)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &order, nil
}

func (r *DynamoDBRepository) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"OrderID": &types.AttributeValueMemberS{Value: orderID},
		},
		UpdateExpression: aws.String("SET #status = :status, UpdatedAt = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "Status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}
