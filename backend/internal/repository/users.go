package repository

import (
	"context"
	"database/sql"
	"fmt"

	"task-manager/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (int64, error) {
	var userID int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users(email, password_hash) VALUES ($1, $2) RETURNING id`,
		email,
		passwordHash,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}

	return userID, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (*models.User, error) {
	var u models.User

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) Exists(ctx context.Context, userID int64) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE id = $1
		)
	`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}
