package amqpx

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient holds the RabbitMQ connection and channel
type RabbitMQClient struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQClient creates a new RabbitMQ client
func NewRabbitMQClient(uri string) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &RabbitMQClient{
		Conn:    conn,
		Channel: ch,
	}, nil
}

// Close closes the RabbitMQ connection and channel
func (c *RabbitMQClient) Close() error {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.Conn != nil {
		c.Conn.Close()
	}

	return nil
}

// EnsureExchange ensures that the exchange exists
func (c *RabbitMQClient) EnsureExchange(name, kind string) error {
	return c.Channel.ExchangeDeclare(
		name,  // name
		kind,  // type
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

// PublishEvent publishes an event to the specified exchange
func (c *RabbitMQClient) PublishEvent(ctx context.Context, exchange, routingKey string, event interface{}) error {
	// Ensure the exchange exists
	if err := c.EnsureExchange(exchange, "topic"); err != nil {
		return fmt.Errorf("failed to ensure exchange: %w", err)
	}

	// Marshal event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish the message
	return c.Channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
