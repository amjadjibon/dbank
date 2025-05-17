package store

import (
	"context"
	"log/slog"

	"github.com/amjadjibon/dbank/pkg/dbx"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Store struct {
	db     *dbx.Postgres
	logger slog.Logger
}

func NewStore(
	db *dbx.Postgres,
	logger *slog.Logger,
) *Store {
	return &Store{
		db:     db,
		logger: *logger,
	}
}

type CreateUserRequest struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
}

func (s *Store) CreateAccount(
	ctx context.Context,
	request *CreateUserRequest,
) error {
	err := dbx.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		sql, args, err := s.db.Builder.
			Insert("dbank_users").
			Columns("id", "username", "email", "password").
			Values(request.ID, request.Username, request.Email, request.Password).
			Suffix("RETURNING id").
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

		// Build the SQL query to insert a new account
		sql, args, err = s.db.Builder.
			Insert("dbank_accounts").
			Columns("account_number", "balance").
			Values(request.AccountNumber, request.Balance).
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

		return nil
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return status.Errorf(codes.Internal, "failed to create account")
	}

	return nil
}
