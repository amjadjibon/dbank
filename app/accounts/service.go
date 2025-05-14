package accounts

import (
	"context"

	dbankv1 "github.com/amjadjibon/dbank/gen/go/dbank/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	dbankv1.UnimplementedAccountServiceServer
}

func NewService() *Service {
	return &Service{}
}

var _ dbankv1.AccountServiceServer = (*Service)(nil)

func (a Service) CreateAccount(
	ctx context.Context,
	request *dbankv1.CreateAccountRequest,
) (*dbankv1.CreateAccountResponse, error) {
	// TODO implement me
	return nil, status.Errorf(codes.Unimplemented, "method CreateAccount not implemented")
}

func (a Service) GetAccount(
	ctx context.Context,
	request *dbankv1.GetAccountRequest,
) (*dbankv1.GetAccountResponse, error) {
	// TODO implement me
	return nil, status.Errorf(codes.Unimplemented, "method GetAccount not implemented")
}
func (a Service) UpdateAccount(
	ctx context.Context,
	request *dbankv1.UpdateAccountRequest,
) (*dbankv1.UpdateAccountResponse, error) {
	// TODO implement me
	return nil, status.Errorf(codes.Unimplemented, "method UpdateAccount not implemented")
}
func (a Service) DeleteAccount(
	ctx context.Context,
	request *dbankv1.DeleteAccountRequest,
) (*dbankv1.DeleteAccountResponse, error) {
	// TODO implement me
	return nil, status.Errorf(codes.Unimplemented, "method DeleteAccount not implemented")
}
