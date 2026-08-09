package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTeamRepositoryCreateCreatesTeamAndOwnerInTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTeamRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO teams(name, created_by) VALUES ($1, $2) RETURNING id`)).
		WithArgs("Team", int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO team_members(user_id, team_id, role) VALUES ($1, $2, 'owner')`)).
		WithArgs(int64(10), int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	teamID, err := repo.Create(context.Background(), "Team", 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if teamID != 5 {
		t.Fatalf("teamID = %d, want 5", teamID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTeamRepositoryListByUserClosesRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTeamRepository(db)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "created_by", "created_at", "role"}).
		AddRow(int64(5), "Team", int64(10), now, "owner")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		ORDER BY t.created_at DESC
	`)).
		WithArgs(int64(10)).
		WillReturnRows(rows).
		RowsWillBeClosed()

	teams, err := repo.ListByUser(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(teams) != 1 || teams[0].ID != 5 || teams[0].Role != "owner" {
		t.Fatalf("teams = %+v, want one team with id 5 and owner role", teams)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTeamRepositoryIsTeamMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTeamRepository(db)

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(int64(5), int64(10)).
		WillReturnRows(rows)

	exists, err := repo.IsTeamMember(context.Background(), 5, 10)
	if err != nil {
		t.Fatalf("IsTeamMember() error = %v", err)
	}
	if !exists {
		t.Fatal("IsTeamMember() = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
