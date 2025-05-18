package store

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amjadjibon/dbank/pkg/dbx"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Store struct {
	db     *dbx.Postgres
	logger *slog.Logger
}

func NewStore(
	db *dbx.Postgres,
	logger *slog.Logger,
) *Store {
	return &Store{
		db:     db,
		logger: logger,
	}
}

type CreateUserRequest struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	AccountID     string  `json:"account_id"`
	AccountName   string  `json:"account_name"`
	AccountType   string  `json:"account_type"`
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
}

type AccountDetails struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	AccountID     string  `json:"account_id"`
	AccountName   string  `json:"account_name"`
	AccountType   string  `json:"account_type"`
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
}

type UpdateAccountRequest struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	AccountName string  `json:"account_name"`
	AccountType string  `json:"account_type"`
	Balance     float64 `json:"balance"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
}

func (s *Store) CreateAccount(
	ctx context.Context,
	request *CreateUserRequest,
) error {
	err := dbx.RunInTx(ctx, s.logger, s.db, func(ctx context.Context, tx pgx.Tx) error {
		// Insert user data
		userSQL, userArgs, err := s.db.Builder.
			Insert("dbank_users").
			Columns("id", "username", "email", "password").
			Values(request.ID, request.Username, request.Email, request.Password).
			Suffix("RETURNING pk").
			ToSql()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build user SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		var userPK int
		err = tx.QueryRow(ctx, userSQL, userArgs...).Scan(&userPK)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to execute user SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}

		// Insert account data
		accountSQL, accountArgs, err := s.db.Builder.
			Insert("dbank_accounts").
			Columns("id", "user_pk", "account_type", "account_number", "balance", "currency", "status", "account_name").
			Values(
				request.AccountID,
				userPK,
				request.AccountType,
				request.AccountNumber,
				request.Balance,
				request.Currency,
				request.Status,
				request.AccountName,
			).
			ToSql()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build account SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		_, err = tx.Exec(ctx, accountSQL, accountArgs...)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to execute account SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}

		return nil
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return err
	}

	return nil
}

func (s *Store) GetAccount(
	ctx context.Context,
	id string,
) (*AccountDetails, error) {
	// Join dbank_users and dbank_accounts to get all required information
	sql, args, err := s.db.Builder.
		Select(
			"u.id", "u.username", "u.email",
			"a.id as account_id", "a.account_name", "a.account_type",
			"a.account_number", "a.balance", "a.currency", "a.status",
		).
		From("dbank_users u").
		Join("dbank_accounts a ON a.user_pk = u.pk").
		Where("u.id = ?", id).
		Where("u.deleted_at IS NULL").
		Where("a.deleted_at IS NULL").
		ToSql()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to build SQL query")
	}

	var account AccountDetails
	err = s.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.AccountID,
		&account.AccountName,
		&account.AccountType,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.ErrorContext(ctx, "account not found", "id", id)
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		s.logger.ErrorContext(ctx, "failed to query account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to query account")
	}

	return &account, nil
}

func (s *Store) UpdateAccount(
	ctx context.Context,
	request *UpdateAccountRequest,
) (*AccountDetails, error) {
	var updatedAccount *AccountDetails

	err := dbx.RunInTx(ctx, s.logger, s.db, func(ctx context.Context, tx pgx.Tx) error {
		// First get user pk from id
		userPkSQL, userPkArgs, err := s.db.Builder.
			Select("pk").
			From("dbank_users").
			Where("id = ?", request.ID).
			Where("deleted_at IS NULL").
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		var userPk int
		err = tx.QueryRow(ctx, userPkSQL, userPkArgs...).Scan(&userPk)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return status.Errorf(codes.NotFound, "user not found")
			}
			return status.Errorf(codes.Internal, "failed to get user pk")
		}

		// Update user information
		userSQL, userArgs, err := s.db.Builder.
			Update("dbank_users").
			Set("username", request.Username).
			Set("email", request.Email).
			Set("updated_at", "now()").
			Where("pk = ?", userPk).
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build user update SQL")
		}

		_, err = tx.Exec(ctx, userSQL, userArgs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to update user")
		}

		// If password provided, update it
		if request.Password != "" {
			passSQL, passArgs, err := s.db.Builder.
				Update("dbank_users").
				Set("password", request.Password).
				Where("pk = ?", userPk).
				ToSql()
			if err != nil {
				return status.Errorf(codes.Internal, "failed to build password update SQL")
			}

			_, err = tx.Exec(ctx, passSQL, passArgs...)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to update password")
			}
		}

		// Update account information
		accountSQL, accountArgs, err := s.db.Builder.
			Update("dbank_accounts").
			Set("account_name", request.AccountName).
			Set("account_type", request.AccountType).
			Set("balance", request.Balance).
			Set("currency", request.Currency).
			Set("status", request.Status).
			Set("updated_at", "now()").
			Where("user_pk = ?", userPk).
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build account update SQL")
		}

		_, err = tx.Exec(ctx, accountSQL, accountArgs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to update account")
		}

		// Get updated account details
		accountDetailsSQL, accountDetailsArgs, err := s.db.Builder.
			Select(
				"u.id", "u.username", "u.email",
				"a.id as account_id", "a.account_name", "a.account_type",
				"a.account_number", "a.balance", "a.currency", "a.status",
			).
			From("dbank_users u").
			Join("dbank_accounts a ON a.user_pk = u.pk").
			Where("u.pk = ?", userPk).
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build details query")
		}

		updatedAccount = &AccountDetails{}
		err = tx.QueryRow(ctx, accountDetailsSQL, accountDetailsArgs...).Scan(
			&updatedAccount.ID,
			&updatedAccount.Username,
			&updatedAccount.Email,
			&updatedAccount.AccountID,
			&updatedAccount.AccountName,
			&updatedAccount.AccountType,
			&updatedAccount.AccountNumber,
			&updatedAccount.Balance,
			&updatedAccount.Currency,
			&updatedAccount.Status,
		)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get updated account details")
		}

		return nil
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update account", "error", err)
		return nil, err
	}

	return updatedAccount, nil
}

func (s *Store) DeleteAccount(
	ctx context.Context,
	id string,
) error {
	err := dbx.RunInTx(ctx, s.logger, s.db, func(ctx context.Context, tx pgx.Tx) error {
		// Get user pk
		userPkSQL, userPkArgs, err := s.db.Builder.
			Select("pk").
			From("dbank_users").
			Where("id = ?", id).
			Where("deleted_at IS NULL").
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		var userPk int
		err = tx.QueryRow(ctx, userPkSQL, userPkArgs...).Scan(&userPk)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return status.Errorf(codes.NotFound, "user not found")
			}
			return status.Errorf(codes.Internal, "failed to get user pk")
		}

		// Soft delete account
		accountSQL, accountArgs, err := s.db.Builder.
			Update("dbank_accounts").
			Set("deleted_at", "now()").
			Where("user_pk = ?", userPk).
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build account delete SQL")
		}

		_, err = tx.Exec(ctx, accountSQL, accountArgs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to delete account")
		}

		// Soft delete user
		userSQL, userArgs, err := s.db.Builder.
			Update("dbank_users").
			Set("deleted_at", "now()").
			Where("pk = ?", userPk).
			ToSql()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to build user delete SQL")
		}

		_, err = tx.Exec(ctx, userSQL, userArgs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to delete user")
		}

		return nil
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete account", "error", err)
		return err
	}

	return nil
}

type TransactionRequest struct {
	TransactionID      string          `json:"transaction_id"`
	FromAccountID      string          `json:"from_account_id"`
	FromAccountBalance decimal.Decimal `json:"from_account_balance"`
	ToAccountID        string          `json:"to_account_id"`
	ToAccountBalance   decimal.Decimal `json:"to_account_balance"`
	TransactionType    string          `json:"transaction_type"`
	Amount             decimal.Decimal `json:"amount"`
	Currency           string          `json:"currency"`
	Description        string          `json:"description"`
	Status             string          `json:"status"`
}

// CreateTransaction creates a new transaction
func (s *Store) CreateTransaction(
	ctx context.Context,
	request *TransactionRequest,
) error {
	if err := dbx.RunInTx(ctx, s.logger, s.db, func(ctx context.Context, tx pgx.Tx) error {
		// deduct amount from sender's account
		// add amount to receiver's account
		// create transaction record

		sql, args, err := s.db.Builder.
			Update("dbank_accounts").
			Set("balance", request.FromAccountBalance).
			Where("id = ?", request.FromAccountID).
			ToSql()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to execute SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}

		sql, args, err = s.db.Builder.
			Update("dbank_accounts").
			Set("balance", request.ToAccountBalance).
			Where("id = ?", request.ToAccountID).
			ToSql()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to execute SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}

		// Create transaction record
		// Insert transaction data
		sql, args, err = s.db.Builder.
			Insert("dbank_transactions").
			Columns("from_account_id", "to_account_id", "transaction_type", "amount", "currency", "description", "status").
			Values(
				request.FromAccountID,
				request.ToAccountID,
				request.TransactionType,
				request.Amount,
				request.Currency,
				request.Description,
				request.Status,
			).
			Suffix("RETURNING id").
			ToSql()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build transaction SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		var transactionID int
		err = tx.QueryRow(ctx, sql, args...).Scan(&transactionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				s.logger.ErrorContext(ctx, "transaction not found", "id", transactionID)
				return status.Errorf(codes.NotFound, "transaction not found")
			}
			s.logger.ErrorContext(ctx, "failed to execute transaction SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}
		s.logger.InfoContext(ctx, "transaction created", "transaction_id", transactionID)
		return nil
	}); err != nil {
		request.Status = "failed"
		request.Description = "Transaction failed"
		s.logger.ErrorContext(ctx, "failed to create transaction", "error", err)
		return err
	}

	return nil
}

type Transaction struct {
	TransactionID      string          `json:"transaction_id"`
	FromAccountID      string          `json:"from_account_id"`
	FromAccountBalance decimal.Decimal `json:"from_account_balance"`
	ToAccountID        string          `json:"to_account_id"`
	ToAccountBalance   decimal.Decimal `json:"to_account_balance"`
	TransactionType    string          `json:"transaction_type"`
	Amount             decimal.Decimal `json:"amount"`
	Currency           string          `json:"currency"`
	Description        string          `json:"description"`
	Status             string          `json:"status"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

func (s *Store) GetTransaction(
	ctx context.Context,
	id string,
) (*Transaction, error) {
	sql, args, err := s.db.Builder.
		Select(
			"t.id", "t.from_account_id", "t.to_account_id",
			"t.transaction_type", "t.amount", "t.currency",
			"t.description", "t.status", "t.created_at", "t.updated_at",
		).
		From("dbank_transactions t").
		Where("t.id = ?", id).
		Where("t.deleted_at IS NULL").
		ToSql()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to build SQL query")
	}
	var transaction Transaction
	err = s.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&transaction.TransactionID,
		&transaction.FromAccountID,
		&transaction.ToAccountID,
		&transaction.TransactionType,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Description,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.ErrorContext(ctx, "transaction not found", "id", id)
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		s.logger.ErrorContext(ctx, "failed to query transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to query transaction")
	}
	return &transaction, nil
}
