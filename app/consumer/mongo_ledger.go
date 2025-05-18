package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/amjadjibon/dbank/pkg/amqpx"
)

// MongoLedgerEntry represents a single ledger entry in MongoDB
type MongoLedgerEntry struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UUID          string             `bson:"uuid"`
	AccountID     string             `bson:"account_id"`
	TransactionID string             `bson:"transaction_id"`
	EntryType     string             `bson:"entry_type"` // "debit" or "credit"
	Amount        decimal.Decimal    `bson:"amount"`
	Balance       decimal.Decimal    `bson:"balance"`
	Currency      string             `bson:"currency"`
	Description   string             `bson:"description,omitempty"`
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
	DeletedAt     *time.Time         `bson:"deleted_at,omitempty"`
}

// NewMongoLedgerConsumer creates a handler for processing transaction events and storing them in MongoDB
func NewMongoLedgerConsumer(logger *slog.Logger, mongoClient *mongo.Client, dbName string) MessageHandler {
	return func(ctx context.Context, delivery amqp.Delivery) error {
		var event amqpx.TransactionEvent
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal transaction event: %w", err)
		}

		// Log the transaction details
		logger.InfoContext(ctx, "processing transaction for ledger entry",
			"transaction_id", event.TransactionID,
			"type", event.TransactionType,
			"amount", event.Amount,
			"currency", event.Currency,
		)

		// Parse amount from string to decimal.Decimal
		amount, err := decimal.NewFromString(event.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount format: %w", err)
		}

		// Get the ledgers collection
		collection := mongoClient.Database(dbName).Collection("ledgers")
		now := time.Now()

		// Create debit entry for the sender account (decrease sender's balance)
		debitEntry := MongoLedgerEntry{
			UUID:          primitive.NewObjectID().Hex(),
			AccountID:     event.FromAccountID,
			TransactionID: event.TransactionID,
			EntryType:     "debit",
			Amount:        amount,
			Balance:       amount.Neg(), // Negative amount (money going out)
			Currency:      event.Currency,
			Description:   event.Description,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Create credit entry for the receiver account (increase receiver's balance)
		creditEntry := MongoLedgerEntry{
			UUID:          primitive.NewObjectID().Hex(),
			AccountID:     event.ToAccountID,
			TransactionID: event.TransactionID,
			EntryType:     "credit",
			Amount:        amount,
			Balance:       amount, // Positive amount (money coming in)
			Currency:      event.Currency,
			Description:   event.Description,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Start a session for transaction with retry logic
		err = RetryMongoOperation(ctx, logger, "create_ledger_entries", 3, func() error {
			session, err := mongoClient.StartSession()
			if err != nil {
				return fmt.Errorf("failed to start MongoDB session: %w", err)
			}
			defer session.EndSession(ctx)

			// Run the operations in a transaction with options
			if err = session.StartTransaction(MongoTransactionOptions()); err != nil {
				return fmt.Errorf("failed to start MongoDB transaction: %w", err)
			}

			if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
				// Insert debit entry
				if _, err = collection.InsertOne(sc, debitEntry); err != nil {
					return fmt.Errorf("failed to insert debit entry: %w", err)
				}

				// Insert credit entry
				if _, err = collection.InsertOne(sc, creditEntry); err != nil {
					return fmt.Errorf("failed to insert credit entry: %w", err)
				}

				// Commit the transaction
				if err = session.CommitTransaction(sc); err != nil {
					return fmt.Errorf("failed to commit MongoDB transaction: %w", err)
				}

				return nil
			}); err != nil {
				abortErr := session.AbortTransaction(ctx)
				if abortErr != nil {
					logger.ErrorContext(ctx, "failed to abort MongoDB transaction", "error", abortErr)
				}
				return err
			}

			return nil
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to process transaction for ledger entry",
				"transaction_id", event.TransactionID,
				"error", err,
			)
			return fmt.Errorf("failed to process transaction for ledger entry: %w", err)
		}

		logger.InfoContext(ctx, "transaction ledger entries recorded successfully",
			"transaction_id", event.TransactionID,
			"debit_account", event.FromAccountID,
			"credit_account", event.ToAccountID,
		)

		return nil
	}
}

// GetLedgerEntriesByAccount retrieves all ledger entries for a specific account
func GetLedgerEntriesByAccount(
	ctx context.Context,
	mongoClient *mongo.Client,
	dbName string,
	accountID string,
) ([]MongoLedgerEntry, error) {
	collection := mongoClient.Database(dbName).Collection("ledgers")

	filter := bson.M{
		"account_id": accountID,
		"deleted_at": bson.M{"$eq": nil},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []MongoLedgerEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode ledger entries: %w", err)
	}

	return entries, nil
}

// GetLedgerEntriesByTransaction retrieves the ledger entries for a specific transaction
func GetLedgerEntriesByTransaction(
	ctx context.Context,
	mongoClient *mongo.Client,
	dbName, transactionID string,
) ([]MongoLedgerEntry, error) {
	collection := mongoClient.Database(dbName).Collection("ledgers")

	filter := bson.M{
		"transaction_id": transactionID,
		"deleted_at":     bson.M{"$eq": nil},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []MongoLedgerEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode ledger entries: %w", err)
	}

	return entries, nil
}

// CalculateAccountBalance calculates the current balance for an account based on all ledger entries
func CalculateAccountBalance(ctx context.Context,
	mongoClient *mongo.Client,
	dbName, accountID string,
) (decimal.Decimal, error) {
	collection := mongoClient.Database(dbName).Collection("ledgers")

	pipeline := bson.A{
		bson.M{
			"$match": bson.M{
				"account_id": accountID,
				"deleted_at": bson.M{"$eq": nil},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id": nil,
				"balance": bson.M{
					"$sum": "$balance",
				},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to aggregate balance: %w", err)
	}
	defer cursor.Close(ctx)

	type result struct {
		Balance decimal.Decimal `bson:"balance"`
	}

	var results []result
	if err = cursor.All(ctx, &results); err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode balance result: %w", err)
	}

	if len(results) == 0 {
		return decimal.Zero, nil
	}

	return results[0].Balance, nil
}

// EnsureLedgerIndexes creates the necessary indexes on the ledgers collection for better query performance
func EnsureLedgerIndexes(ctx context.Context, mongoClient *mongo.Client, dbName string) error {
	collection := mongoClient.Database(dbName).Collection("ledgers")

	// Create indexes for commonly queried fields
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "account_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_account_created"),
		},
		{
			Keys:    bson.D{{Key: "transaction_id", Value: 1}},
			Options: options.Index().SetName("idx_transaction"),
		},
		{
			Keys: bson.D{
				{Key: "entry_type", Value: 1},
				{Key: "account_id", Value: 1},
			},
			Options: options.Index().SetName("idx_entry_type_account"),
		},
		{
			Keys:    bson.D{{Key: "currency", Value: 1}},
			Options: options.Index().SetName("idx_currency"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// RetryMongoOperation attempts to execute a MongoDB operation with retries
func RetryMongoOperation(ctx context.Context,
	logger *slog.Logger,
	operation string, maxRetries int, fn func() error,
) error {
	var err error
	backoff := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Check if we should retry based on error type
		if mongo.IsNetworkError(err) || mongo.IsTimeout(err) || isMongoDuplicateKeyError(err) {
			// Log retry attempt
			logger.WarnContext(ctx, "MongoDB operation failed, retrying",
				"operation", operation,
				"attempt", attempt,
				"max_retries", maxRetries,
				"backoff_ms", backoff.Milliseconds(),
				"error", err,
			)

			// Wait before retrying with exponential backoff
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff with jitter
				backoff = time.Duration(float64(backoff) * 1.5)
				continue
			}
		}

		// Non-retriable error
		return err
	}

	return fmt.Errorf("operation '%s' failed after %d retries: %w", operation, maxRetries, err)
}

// isMongoDuplicateKeyError checks if the error is a MongoDB duplicate key error
func isMongoDuplicateKeyError(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 11000 // MongoDB duplicate key error code
	}
	return false
}

// MongoTransactionOptions returns the default options for MongoDB transactions
func MongoTransactionOptions() *options.TransactionOptions {
	return options.Transaction().
		SetReadPreference(readpref.Primary()).
		SetReadConcern(readconcern.Majority()).
		SetWriteConcern(writeconcern.Majority())
}
