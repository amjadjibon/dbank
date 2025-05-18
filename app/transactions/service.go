package transactions

import (
	"context"
	"log/slog"
	"time"

	"github.com/amjadjibon/dbank/app/store"
	dbankv1 "github.com/amjadjibon/dbank/gen/go/dbank/v1"
	"github.com/amjadjibon/dbank/pkg/amqpx"
	"github.com/shopspring/decimal" // Added import
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service represents the transaction service
type Service struct {
	logger           *slog.Logger
	transactionStore *store.Store
	rabbitmqClient   *amqpx.RabbitMQClient
	dbankv1.UnimplementedTransactionServiceServer
}

// NewService creates a new transaction service
func NewService(
	logger *slog.Logger,
	transactionStore *store.Store,
	rabbitmqClient *amqpx.RabbitMQClient,
) *Service {
	return &Service{
		logger:           logger,
		transactionStore: transactionStore,
		rabbitmqClient:   rabbitmqClient,
	}
}

// Ensure Service implements the TransactionServiceServer interface
var _ dbankv1.TransactionServiceServer = (*Service)(nil)

// CreateTransaction creates a new transaction
func (t *Service) CreateTransaction(
	ctx context.Context,
	request *dbankv1.CreateTransactionRequest,
) (*dbankv1.CreateTransactionResponse, error) {
	t.logger.InfoContext(ctx, "Creating transaction",
		"from_account_id", request.FromAccountId,
		"to_account_id", request.ToAccountId,
		"amount", request.Amount,
	)

	// Validate request
	if request.FromAccountId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "from_account_id is required")
	}

	if request.ToAccountId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "to_account_id is required")
	}

	amountDecimal, err := decimal.NewFromString(request.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}

	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		return nil, status.Errorf(codes.InvalidArgument, "amount must be positive")
	}

	if request.Currency == "" {
		return nil, status.Errorf(codes.InvalidArgument, "currency is required")
	}

	// get account by ids
	fromAccount, err := t.transactionStore.GetAccount(ctx, request.FromAccountId)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get from account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get from account: %v", err)
	}

	toAccount, err := t.transactionStore.GetAccount(ctx, request.ToAccountId)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get to account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get to account: %v", err)
	}

	fromAccountBalance := decimal.NewFromFloat(fromAccount.Balance)
	toAccountBalance := decimal.NewFromFloat(toAccount.Balance)
	if fromAccountBalance.LessThan(amountDecimal) {
		return nil, status.Errorf(codes.InvalidArgument, "insufficient balance in from account")
	}
	if fromAccount.ID == toAccount.ID {
		return nil, status.Errorf(codes.InvalidArgument, "from and to account cannot be the same")
	}

	newFromAccountBalance := fromAccountBalance.Sub(amountDecimal)
	newToAccountBalance := toAccountBalance.Add(amountDecimal)

	// Create transaction
	if err := t.transactionStore.CreateTransaction(ctx, &store.TransactionRequest{
		FromAccountID:      request.FromAccountId,
		ToAccountID:        request.ToAccountId,
		TransactionType:    request.TransactionType,
		FromAccountBalance: newFromAccountBalance,
		ToAccountBalance:   newToAccountBalance,
		Amount:             amountDecimal,
		Currency:           request.Currency,
		Description:        request.Description,
		Status:             "success",
	}); err != nil {
		t.logger.ErrorContext(ctx, "failed to create transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	// Generate transaction ID
	transactionID := request.FromAccountId + request.ToAccountId

	// Create a transaction response
	response := &dbankv1.CreateTransactionResponse{
		Id:              transactionID,
		FromAccountId:   request.FromAccountId,
		ToAccountId:     request.ToAccountId,
		TransactionType: request.TransactionType,
		Amount:          request.Amount,
		Currency:        request.Currency,
		Description:     request.Description,
		Status:          "success",
	}

	// Publish the transaction event to RabbitMQ
	if t.rabbitmqClient != nil {
		event := &amqpx.TransactionEvent{
			TransactionID:   transactionID,
			FromAccountID:   request.FromAccountId,
			ToAccountID:     request.ToAccountId,
			TransactionType: request.TransactionType,
			Amount:          request.Amount,
			Currency:        request.Currency,
			Status:          "success",
			Description:     request.Description,
			Timestamp:       time.Now().Unix(),
		}

		if err := t.rabbitmqClient.PublishEvent(
			ctx,
			amqpx.TransactionExchange,
			amqpx.TransactionSuccessRoute,
			event,
		); err != nil {
			t.logger.WarnContext(ctx, "Failed to publish transaction event", "error", err)
			// Don't fail the transaction if event publishing fails
		} else {
			t.logger.InfoContext(ctx, "Published transaction success event",
				"transaction_id", transactionID,
			)
		}
	}

	return response, nil
}

// GetTransaction retrieves a transaction by ID
func (t *Service) GetTransaction(
	ctx context.Context,
	request *dbankv1.GetTransactionRequest,
) (*dbankv1.GetTransactionResponse, error) {
	t.logger.InfoContext(ctx, "Getting transaction", "id", request.Id)

	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "transaction id is required")
	}

	transaction, err := t.transactionStore.GetTransaction(ctx, request.Id)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get transaction: %v", err)
	}

	if transaction == nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}

	return &dbankv1.GetTransactionResponse{
		Id:              transaction.TransactionID,
		FromAccountId:   transaction.FromAccountID,
		ToAccountId:     transaction.ToAccountID,
		TransactionType: transaction.TransactionType,
		Amount:          transaction.Amount.String(), // Convert decimal.Decimal to string
		Currency:        transaction.Currency,
		Description:     transaction.Description,
		Status:          transaction.Status,
		CreatedAt:       transaction.CreatedAt,
	}, nil
}
