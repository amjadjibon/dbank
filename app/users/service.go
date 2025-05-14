package users

import (
	"context"
	"fmt"

	usersv1 "github.com/amjadjibon/dbank/gen/go/users/v1"
)

type Service struct {
	usersv1.UnimplementedUsersServiceServer
}

func NewService() *Service {
	return &Service{}
}

var _ usersv1.UsersServiceServer = (*Service)(nil)

func (u Service) CreateUser(context.Context, *usersv1.CreateUserRequest) (*usersv1.CreateUserResponse, error) {
	fmt.Println("CreateUser called")
	return &usersv1.CreateUserResponse{}, nil
}

func (u Service) GetUser(context.Context, *usersv1.GetUserRequest) (*usersv1.GetUserResponse, error) {
	return &usersv1.GetUserResponse{}, nil
}

func (u Service) UpdateUser(context.Context, *usersv1.UpdateUserRequest) (*usersv1.UpdateUserResponse, error) {
	return &usersv1.UpdateUserResponse{}, nil
}

func (u Service) DeleteUser(context.Context, *usersv1.DeleteUserRequest) (*usersv1.DeleteUserResponse, error) {
	return &usersv1.DeleteUserResponse{}, nil
}
