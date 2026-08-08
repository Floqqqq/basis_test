package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"task-manager/internal/models"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*models.User, error)
}

type UserService struct {
	users UserRepository
}

func NewUserService(users UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) GetByID(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
