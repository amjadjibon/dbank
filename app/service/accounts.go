package service

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

type AccountService struct {
	logger       *slog.Logger
	accountStore *store.Store
	dbankv1.UnimplementedAccountServiceServer
}

func NewAccountService(
	logger *slog.Logger,
	accountStore *store.Store,
) *AccountService {
	return &AccountService{
		accountStore: accountStore,
		logger:       logger,
	}
}

var _ dbankv1.AccountServiceServer = (*AccountService)(nil)

func (a *AccountService) CreateAccount(
	ctx context.Context,
	request *dbankv1.CreateAccountRequest,
) (*dbankv1.CreateAccountResponse, error) {
	// Validate required fields
	if request.Username == "" || request.Email == "" || request.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username, email, and password are required")
	}

	if request.AccountName == "" || request.AccountType == "" || request.AccountCurrency == "" {
		return nil, status.Errorf(codes.InvalidArgument, "account name, type, and currency are required")
	}

	// Hash the password
	hashedPassword, err := passw.HashPassword(request.Password)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to hash password", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to hash password")
	}

	// Parse account balance
	balance, err := strconv.ParseFloat(request.AccountBalance, 64)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to parse balance", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid balance format")
	}

	// Set default status if not provided
	accountStatus := request.AccountStatus
	if accountStatus == "" {
		accountStatus = "active"
	}

	// Generate UUIDs
	userID := uuid.New().String()
	accountID := uuid.New().String()

	// Generate an account number (this is a simple implementation, you might want a more sophisticated approach)
	accountNumber := uuid.New().String()[:8]

	// Create the account
	err = a.accountStore.CreateAccount(ctx, &store.CreateUserRequest{
		ID:            userID,
		Username:      request.Username,
		Email:         request.Email,
		Password:      hashedPassword,
		AccountID:     accountID,
		AccountName:   request.AccountName,
		AccountType:   request.AccountType,
		AccountNumber: accountNumber,
		Balance:       balance,
		Currency:      request.AccountCurrency,
		Status:        accountStatus,
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &dbankv1.CreateAccountResponse{
		Id:              userID,
		Username:        request.Username,
		Email:           request.Email,
		AccountName:     request.AccountName,
		AccountType:     request.AccountType,
		AccountBalance:  request.AccountBalance,
		AccountCurrency: request.AccountCurrency,
		AccountStatus:   accountStatus,
	}, nil
}

// GetAccounts
func (a *AccountService) ListAccounts(
	ctx context.Context,
	request *dbankv1.ListAccountsRequest,
) (*dbankv1.ListAccountsResponse, error) {
	// Get all accounts
	accounts, err := a.accountStore.GetAllAccounts(ctx, request.Page, request.PageSize)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get accounts", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get accounts: %v", err)
	}

	var accountList []*dbankv1.GetAccountResponse
	for _, account := range accounts {
		accountList = append(accountList, &dbankv1.GetAccountResponse{
			Id:              account.ID,
			Username:        account.Username,
			Email:           account.Email,
			AccountName:     account.AccountName,
			AccountType:     account.AccountType,
			AccountBalance:  strconv.FormatFloat(account.Balance, 'f', 2, 64),
			AccountCurrency: account.Currency,
			AccountStatus:   account.Status,
		})
	}

	return &dbankv1.ListAccountsResponse{
		Accounts:   accountList,
		TotalCount: uint64(len(accountList)),
	}, nil
}

func (a *AccountService) GetAccount(
	ctx context.Context,
	request *dbankv1.GetAccountRequest,
) (*dbankv1.GetAccountResponse, error) {
	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "account ID is required")
	}

	// Get account details
	account, err := a.accountStore.GetAccount(ctx, request.Id)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get account", "error", err, "id", request.Id)
		return nil, status.Errorf(codes.NotFound, "account not found: %v", err)
	}

	// Format balance as string for the response
	balanceStr := strconv.FormatFloat(account.Balance, 'f', 2, 64)

	return &dbankv1.GetAccountResponse{
		Id:              account.ID,
		Username:        account.Username,
		Email:           account.Email,
		AccountName:     account.AccountName,
		AccountType:     account.AccountType,
		AccountBalance:  balanceStr,
		AccountCurrency: account.Currency,
		AccountStatus:   account.Status,
	}, nil
}

func (a *AccountService) UpdateAccount(
	ctx context.Context,
	request *dbankv1.UpdateAccountRequest,
) (*dbankv1.UpdateAccountResponse, error) {
	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "account ID is required")
	}

	// Get current account to verify it exists
	existingAccount, err := a.accountStore.GetAccount(ctx, request.Id)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get account for update", "error", err, "id", request.Id)
		return nil, status.Errorf(codes.NotFound, "account not found: %v", err)
	}

	// Prepare update data
	updateData := &store.UpdateAccountRequest{
		ID: request.Id,
	}

	// Update fields that are provided
	if request.Username != "" {
		updateData.Username = request.Username
	} else {
		updateData.Username = existingAccount.Username
	}

	if request.Email != "" {
		updateData.Email = request.Email
	} else {
		updateData.Email = existingAccount.Email
	}

	if request.Password != "" {
		hashedPassword, err := passw.HashPassword(request.Password)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to hash password", "error", err)
			return nil, status.Errorf(codes.Internal, "failed to hash password")
		}
		updateData.Password = hashedPassword
	}

	if request.AccountName != "" {
		updateData.AccountName = request.AccountName
	} else {
		updateData.AccountName = existingAccount.AccountName
	}

	if request.AccountType != "" {
		updateData.AccountType = request.AccountType
	} else {
		updateData.AccountType = existingAccount.AccountType
	}

	if request.AccountBalance != "" {
		balance, err := strconv.ParseFloat(request.AccountBalance, 64)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to parse balance", "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "invalid balance format")
		}
		updateData.Balance = balance
	} else {
		updateData.Balance = existingAccount.Balance
	}

	if request.AccountCurrency != "" {
		updateData.Currency = request.AccountCurrency
	} else {
		updateData.Currency = existingAccount.Currency
	}

	if request.AccountStatus != "" {
		updateData.Status = request.AccountStatus
	} else {
		updateData.Status = existingAccount.Status
	}

	// Update the account
	updatedAccount, err := a.accountStore.UpdateAccount(ctx, updateData)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update account", "error", err, "id", request.Id)
		return nil, status.Errorf(codes.Internal, "failed to update account: %v", err)
	}

	// Format balance for response
	balanceStr := strconv.FormatFloat(updatedAccount.Balance, 'f', 2, 64)

	return &dbankv1.UpdateAccountResponse{
		Id:              updatedAccount.ID,
		Username:        updatedAccount.Username,
		Email:           updatedAccount.Email,
		AccountName:     updatedAccount.AccountName,
		AccountType:     updatedAccount.AccountType,
		AccountBalance:  balanceStr,
		AccountCurrency: updatedAccount.Currency,
		AccountStatus:   updatedAccount.Status,
	}, nil
}

func (a *AccountService) DeleteAccount(
	ctx context.Context,
	request *dbankv1.DeleteAccountRequest,
) (*dbankv1.DeleteAccountResponse, error) {
	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "account ID is required")
	}

	// First check if the account exists
	_, err := a.accountStore.GetAccount(ctx, request.Id)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get account for deletion", "error", err, "id", request.Id)
		return nil, status.Errorf(codes.NotFound, "account not found: %v", err)
	}

	// Delete the account
	err = a.accountStore.DeleteAccount(ctx, request.Id)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to delete account", "error", err, "id", request.Id)
		return nil, status.Errorf(codes.Internal, "failed to delete account: %v", err)
	}

	return &dbankv1.DeleteAccountResponse{
		Id:      request.Id,
		Message: "Account successfully deleted",
	}, nil
}
