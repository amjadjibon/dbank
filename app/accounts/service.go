package accounts

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/amjadjibon/dbank/app/store"
	dbankv1 "github.com/amjadjibon/dbank/gen/go/dbank/v1"
	"github.com/amjadjibon/dbank/pkg/passw"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	logger       *slog.Logger
	accountStore *store.Store
	dbankv1.UnimplementedAccountServiceServer
}

func NewService(
	logger *slog.Logger,
	accountStore *store.Store,
) *Service {
	return &Service{
		accountStore: accountStore,
		logger:       logger,
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

	balance, err := strconv.ParseFloat(request.AccountBalance, 64)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to parse balance", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid balance")
	}

	err = a.accountStore.CreateAccount(ctx, &store.CreateUserRequest{
		ID:            uuid.New().String(),
		Username:      request.Username,
		Email:         request.Email,
		Password:      hashedPassword,
		AccountNumber: request.AccountName,
		Balance:       balance,
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create account")
	}

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
