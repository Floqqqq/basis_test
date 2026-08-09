package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"task-manager/internal/events"
	"task-manager/internal/repository"
)

func TestWorkerIntegrationWithPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runNotificationPostgres(ctx)
	if err != nil {
		if dockerUnavailable(err) {
			t.Skipf("Docker is not available for testcontainers: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, container)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	applyNotificationMigrations(t, ctx, db)

	ownerID, memberID, taskID := seedNotificationData(t, ctx, db)
	outbox := repository.NewOutboxRepository(db)
	data := repository.NewNotificationRepository(db)
	sender := &fakeEmailSender{}
	service := NewNotificationService(NewNotificationBuilder(data), sender)
	worker := NewWorker(outbox, service, 1, 5, time.Second)

	successEvent := events.New(events.TaskAssigned, ownerID, events.TaskPayload{TaskID: taskID, CreatorID: ownerID, AssigneeID: &memberID})
	if err := outbox.Create(ctx, db, successEvent); err != nil {
		t.Fatalf("create successful event: %v", err)
	}
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatalf("ProcessBatch(success) error = %v", err)
	}
	assertOutboxState(t, ctx, db, successEvent.EventID, "sent", 0)
	if len(sender.messages) != 1 || len(sender.messages[0].To) != 1 || sender.messages[0].To[0] != "member@example.com" {
		t.Fatalf("sent messages = %+v", sender.messages)
	}

	sender.err = errors.New("SMTP unavailable")
	retryEvent := events.New(events.TaskAssigned, ownerID, events.TaskPayload{TaskID: taskID, CreatorID: ownerID, AssigneeID: &memberID})
	if err := outbox.Create(ctx, db, retryEvent); err != nil {
		t.Fatalf("create retry event: %v", err)
	}
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatalf("ProcessBatch(retry) error = %v", err)
	}
	assertOutboxState(t, ctx, db, retryEvent.EventID, "pending", 1)

	failedEvent := events.New(events.TaskAssigned, ownerID, events.TaskPayload{TaskID: taskID, CreatorID: ownerID, AssigneeID: &memberID})
	if err := outbox.Create(ctx, db, failedEvent); err != nil {
		t.Fatalf("create max-attempt event: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE notification_outbox SET attempts = 4 WHERE event_id = $1`, failedEvent.EventID); err != nil {
		t.Fatalf("prepare max attempts: %v", err)
	}
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatalf("ProcessBatch(max attempts) error = %v", err)
	}
	assertOutboxState(t, ctx, db, failedEvent.EventID, "failed", 5)
}

func runNotificationPostgres(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start postgres container: %v", recovered)
		}
	}()
	return tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("notifications_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
}

func seedNotificationData(t *testing.T, ctx context.Context, db *sql.DB) (ownerID, memberID, taskID int64) {
	t.Helper()
	if err := db.QueryRowContext(ctx, `INSERT INTO users(email, password_hash) VALUES ('owner@example.com', 'hash') RETURNING id`).Scan(&ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO users(email, password_hash) VALUES ('member@example.com', 'hash') RETURNING id`).Scan(&memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	var teamID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO teams(name, created_by) VALUES ('Backend', $1) RETURNING id`, ownerID).Scan(&teamID); err != nil {
		t.Fatalf("insert team: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		INSERT INTO tasks(title, status, assignee_id, team_id, created_by)
		VALUES ('Check API', 'todo', $1, $2, $3)
		RETURNING id
	`, memberID, teamID, ownerID).Scan(&taskID); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	return ownerID, memberID, taskID
}

func assertOutboxState(t *testing.T, ctx context.Context, db *sql.DB, eventID, wantStatus string, wantAttempts int) {
	t.Helper()
	var status string
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT status, attempts FROM notification_outbox WHERE event_id = $1`, eventID).Scan(&status, &attempts); err != nil {
		t.Fatalf("select outbox event: %v", err)
	}
	if status != wantStatus || attempts != wantAttempts {
		t.Fatalf("outbox state = %s/%d, want %s/%d", status, attempts, wantStatus, wantAttempts)
	}
}

func applyNotificationMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatalf("find migrations: %v", err)
	}
	sort.Strings(files)
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration: %v", err)
		}
		for _, statement := range strings.Split(string(body), ";") {
			if statement = strings.TrimSpace(statement); statement != "" {
				if _, err := db.ExecContext(ctx, statement); err != nil {
					t.Fatalf("apply migration %s: %v", file, err)
				}
			}
		}
	}
}

func dockerUnavailable(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "docker daemon") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "permission denied")
}
