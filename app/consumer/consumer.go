package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/amjadjibon/dbank/pkg/amqpx"
)

type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

type Consumer struct {
	logger         *slog.Logger
	rabbitmqClient *amqpx.RabbitMQClient
	handlers       map[string]MessageHandler
	ctx            context.Context
	cancel         context.CancelFunc
}

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

func (c *Consumer) RegisterHandler(routingKey string, handler MessageHandler) {
	c.handlers[routingKey] = handler
}

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

func (c *Consumer) Stop(ctx context.Context) {
	c.logger.InfoContext(ctx, "stopping RabbitMQ consumer")
	c.cancel()
}

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

		if err := delivery.Ack(false); err != nil {
			c.logger.ErrorContext(ctx, "failed to acknowledge message", "error", err)
		}
		return
	}

	if err := handler(ctx, delivery); err != nil {
		c.logger.ErrorContext(ctx, "failed to process message",
			"error", err,
			"routing_key", delivery.RoutingKey,
		)

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

		logger.InfoContext(ctx, "transaction processed successfully",
			"transaction_id", event.TransactionID,
		)

		return nil
	}
}

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

		logger.InfoContext(ctx, "failed transaction processed",
			"transaction_id", event.TransactionID,
		)

		return nil
	}
}
