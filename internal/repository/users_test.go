package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepositoryCreateWrapsInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	insertErr := errors.New("duplicate")

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO users(email, password_hash) VALUES (?, ?)`)).
		WithArgs("user@example.com", "hash").
		WillReturnError(insertErr)

	_, err = repo.Create(context.Background(), "user@example.com", "hash")
	if !errors.Is(err, insertErr) {
		t.Fatalf("Create() error = %v, want wrapped duplicate error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM users WHERE id = ?
		)
	`)).
		WithArgs(int64(12)).
		WillReturnRows(rows)

	exists, err := repo.Exists(context.Background(), 12)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Fatal("Exists() = true, want false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
