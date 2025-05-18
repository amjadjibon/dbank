package amqpx

// TransactionEvent represents a transaction event to be published to RabbitMQ
type TransactionEvent struct {
	TransactionID   string `json:"transaction_id"`
	FromAccountID   string `json:"from_account_id"`
	ToAccountID     string `json:"to_account_id"`
	TransactionType string `json:"transaction_type"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	Description     string `json:"description,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}

// Constants for AMQP exchanges and routing keys
const (
	TransactionExchange     = "transactions"
	TransactionSuccessRoute = "transaction.success"
	TransactionFailureRoute = "transaction.failure"
)
