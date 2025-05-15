package accounts

import (
	"context"
	"log/slog"

	dbankv1 "github.com/amjadjibon/dbank/gen/go/dbank/v1"
	"github.com/amjadjibon/dbank/pkg/dbx"
	"github.com/amjadjibon/dbank/pkg/passw"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	logger *slog.Logger
	db     *dbx.Postgres
	dbankv1.UnimplementedAccountServiceServer
}

func NewService(
	logger *slog.Logger,
	db *dbx.Postgres,
) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

var _ dbankv1.AccountServiceServer = (*Service)(nil)

func (a Service) CreateAccount(
	ctx context.Context,
	request *dbankv1.CreateAccountRequest,
) (*dbankv1.CreateAccountResponse, error) {
	// TODO implement me
	_ = ctx
	_ = request

	hashedPassword, err := passw.HashPassword(request.Password)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to hash password", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to hash password")
	}

	err = dbx.RunInTx(ctx, a.db, func(ctx context.Context, tx pgx.Tx) error {
		sql, args, err := a.db.Builder.
			Insert("dbank_users").
			Columns("id", "username", "email", "password").
			Values(uuid.NewString(), request.Username, request.Email, hashedPassword).
			ToSql()
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to execute SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}

		a.logger.InfoContext(ctx, "account created successfully",
			"username", request.Username,
			"email", request.Email,
		)
		sql, args, err = a.db.Builder.
			Insert("dbank_accounts").
			Columns("id", "user_id", "balance").
			Values(uuid.NewString(), request.Username, 0).
			ToSql()
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to build SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to build SQL query")
		}
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to execute SQL query", "error", err)
			return status.Errorf(codes.Internal, "failed to execute SQL query")
		}
		a.logger.InfoContext(ctx, "account created successfully",
			"username", request.Username,
			"email", request.Email,
		)
		return nil
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create account")
	}
	a.logger.InfoContext(ctx, "account created successfully",
		"username", request.Username,
		"email", request.Email,
	)

	return &dbankv1.CreateAccountResponse{
		Username: request.Username,
		Email:    request.Email,
	}, nil
}

func (a Service) GetAccount(
	ctx context.Context,
	request *dbankv1.GetAccountRequest,
) (*dbankv1.GetAccountResponse, error) {
	// TODO implement me
	_ = ctx
	_ = request
	return nil, status.Errorf(codes.Unimplemented, "method GetAccount not implemented")
}

func (a Service) UpdateAccount(
	ctx context.Context,
	request *dbankv1.UpdateAccountRequest,
) (*dbankv1.UpdateAccountResponse, error) {
	// TODO implement me
	_ = ctx
	_ = request
	return nil, status.Errorf(codes.Unimplemented, "method UpdateAccount not implemented")
}

func (a Service) DeleteAccount(
	ctx context.Context,
	request *dbankv1.DeleteAccountRequest,
) (*dbankv1.DeleteAccountResponse, error) {
	// TODO implement me
	_ = ctx
	_ = request
	return nil, status.Errorf(codes.Unimplemented, "method DeleteAccount not implemented")
}
