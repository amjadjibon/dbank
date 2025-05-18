package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/amjadjibon/dbank/pkg/amqpx"
)

// MessageHandler defines a function to process amqp messages
type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

// Consumer handles consuming and processing RabbitMQ messages
type Consumer struct {
	logger         *slog.Logger
	rabbitmqClient *amqpx.RabbitMQClient
	handlers       map[string]MessageHandler
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(rabbitmqClient *amqpx.RabbitMQClient, logger *slog.Logger) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		logger:         logger,
		rabbitmqClient: rabbitmqClient,
		handlers:       make(map[string]MessageHandler),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// RegisterHandler registers a handler for a specific routing key
func (c *Consumer) RegisterHandler(routingKey string, handler MessageHandler) {
	c.handlers[routingKey] = handler
}

// Start begins consuming messages from the specified exchange
func (c *Consumer) Start(ctx context.Context) error {
	exchange := amqpx.TransactionExchange
	queueName := "dbank.transactions.consumer"

	// We'll bind to all routing keys initially (using "#" wildcard)
	// and filter by specific routing keys in the message handler
	deliveries, err := c.rabbitmqClient.ConsumeEvents(exchange, queueName, "#")
	if err != nil {
		return fmt.Errorf("failed to consume events: %w", err)
	}

	c.logger.InfoContext(ctx, "starting RabbitMQ consumer",
		"exchange", exchange,
		"queue", queueName,
	)

	go c.consumeMessages(deliveries)
	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop(ctx context.Context) {
	c.logger.InfoContext(ctx, "stopping RabbitMQ consumer")
	c.cancel()
}

// consumeMessages processes messages from the delivery channel
func (c *Consumer) consumeMessages(deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.InfoContext(context.Background(), "stopping RabbitMQ consumer")
			return
		case delivery, ok := <-deliveries:
			if !ok {
				c.logger.WarnContext(context.Background(), "RabbitMQ channel closed")
				return
			}

			c.handleMessage(delivery)
		}
	}
}

// handleMessage processes an individual message
func (c *Consumer) handleMessage(delivery amqp.Delivery) {
	// Create a new context for handling this specific message
	ctx := context.Background()

	c.logger.DebugContext(ctx, "received message",
		"routing_key", delivery.RoutingKey,
		"exchange", delivery.Exchange,
	)

	// Find the appropriate handler based on the routing key
	handler, exists := c.handlers[delivery.RoutingKey]
	if !exists {
		c.logger.WarnContext(ctx, "no handler registered for routing key",
			"routing_key", delivery.RoutingKey,
		)
		// Acknowledge the message to remove it from the queue
		// since we don't have a handler for it
		if err := delivery.Ack(false); err != nil {
			c.logger.ErrorContext(ctx, "failed to acknowledge message", "error", err)
		}
		return
	}

	// Process the message with the handler
	if err := handler(ctx, delivery); err != nil {
		c.logger.ErrorContext(ctx, "failed to process message",
			"error", err,
			"routing_key", delivery.RoutingKey,
		)

		// Reject the message, requeue it for later processing
		if err := delivery.Reject(true); err != nil {
			c.logger.ErrorContext(ctx, "failed to reject message", "error", err)
		}
		return
	}

	// Acknowledge successful processing
	if err := delivery.Ack(false); err != nil {
		c.logger.ErrorContext(ctx, "failed to acknowledge message", "error", err)
	}
}

// DefaultTransactionHandler provides a default implementation for handling transaction events
func DefaultTransactionHandler(logger *slog.Logger) MessageHandler {
	return func(ctx context.Context, delivery amqp.Delivery) error {
		var event amqpx.TransactionEvent
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal transaction event: %w", err)
		}

		logger.InfoContext(ctx, "processed transaction event",
			"transaction_id", event.TransactionID,
			"type", event.TransactionType,
			"amount", event.Amount,
			"currency", event.Currency,
			"status", event.Status,
		)

		return nil
	}
}

// ProcessSuccessfulTransaction creates a handler that processes successful transactions
// This is an example of a more advanced transaction handler that could be used in production
func ProcessSuccessfulTransaction(logger *slog.Logger) MessageHandler {
	return func(ctx context.Context, delivery amqp.Delivery) error {
		var event amqpx.TransactionEvent
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal transaction event: %w", err)
		}

		// Log the transaction details
		logger.InfoContext(ctx, "processing successful transaction",
			"transaction_id", event.TransactionID,
			"type", event.TransactionType,
			"amount", event.Amount,
			"currency", event.Currency,
		)

		// Here you could implement:
		// 1. Updating transaction status in database
		// 2. Sending notifications
		// 3. Generating reports
		// 4. Triggering downstream processes

		// Example: Simulate some processing time
		// time.Sleep(100 * time.Millisecond)

		logger.InfoContext(ctx, "transaction processed successfully",
			"transaction_id", event.TransactionID,
		)

		return nil
	}
}

// ProcessFailedTransaction handles failed transactions
func ProcessFailedTransaction(logger *slog.Logger) MessageHandler {
	return func(ctx context.Context, delivery amqp.Delivery) error {
		var event amqpx.TransactionEvent
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal transaction event: %w", err)
		}

		logger.WarnContext(ctx, "processing failed transaction",
			"transaction_id", event.TransactionID,
			"type", event.TransactionType,
			"amount", event.Amount,
			"currency", event.Currency,
			"status", event.Status,
		)

		// Here you could implement:
		// 1. Recording failure details
		// 2. Notification to support team
		// 3. Automatic retry logic
		// 4. Customer notification

		logger.InfoContext(ctx, "failed transaction processed",
			"transaction_id", event.TransactionID,
		)

		return nil
	}
}
