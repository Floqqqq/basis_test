package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"task-manager/internal/events"
	"task-manager/internal/models"
	"task-manager/internal/service"
)

func TestRepositoryIntegrationWithPostgresContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	postgresContainer, err := runPostgresContainer(ctx)
	if err != nil {
		skipIfDockerUnavailable(t, err)
		t.Fatalf("start postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, postgresContainer)

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if err := pingDB(ctx, db); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	applyMigrations(t, ctx, db)
	assertTablesExist(t, ctx, db)

	users := NewUserRepository(db)
	teams := NewTeamRepository(db)
	tasks := NewTaskRepository(db)
	policy := service.NewTaskPolicy(teams)

	ownerID, err := users.Create(ctx, "owner@example.com", "hash")
	if err != nil {
		t.Fatalf("users.Create(owner) error = %v", err)
	}
	memberID, err := users.Create(ctx, "member@example.com", "hash")
	if err != nil {
		t.Fatalf("users.Create(member) error = %v", err)
	}
	outsiderID, err := users.Create(ctx, "outsider@example.com", "hash")
	if err != nil {
		t.Fatalf("users.Create(outsider) error = %v", err)
	}

	teamID, err := teams.Create(ctx, "Integration Team", ownerID)
	if err != nil {
		t.Fatalf("teams.Create() error = %v", err)
	}
	role, err := teams.GetUserRole(ctx, teamID, ownerID)
	if err != nil {
		t.Fatalf("teams.GetUserRole(owner) error = %v", err)
	}
	if role != "owner" {
		t.Fatalf("owner role = %q, want owner", role)
	}

	if err := teams.Invite(ctx, teamID, memberID, "member", events.New(events.TeamMemberAdded, ownerID, events.TeamMemberPayload{TeamID: teamID, UserID: memberID, Role: "member"})); err != nil {
		t.Fatalf("teams.Invite() error = %v", err)
	}
	isMember, err := teams.IsTeamMember(ctx, teamID, memberID)
	if err != nil {
		t.Fatalf("teams.IsTeamMember(member) error = %v", err)
	}
	if !isMember {
		t.Fatal("invited user is not team member")
	}

	if ok, err := policy.IsAssigneeValidForTeam(ctx, teamID, &outsiderID); err != nil || ok {
		t.Fatalf("outsider assignee valid = %v, %v; want false, nil", ok, err)
	}
	if ok, err := policy.IsAssigneeValidForTeam(ctx, teamID, &memberID); err != nil || !ok {
		t.Fatalf("member assignee valid = %v, %v; want true, nil", ok, err)
	}

	taskID, err := tasks.Create(ctx, models.Task{
		Title:      "Initial",
		Status:     "todo",
		AssigneeID: &memberID,
		TeamID:     teamID,
		CreatedBy:  ownerID,
	}, func(taskID int64) events.Event {
		return events.New(events.TaskCreated, ownerID, events.TaskPayload{TaskID: taskID, TeamID: teamID})
	})
	if err != nil {
		t.Fatalf("tasks.Create() error = %v", err)
	}

	task, err := tasks.GetByID(ctx, taskID)
	if err != nil {
		t.Fatalf("tasks.GetByID() error = %v", err)
	}
	task.Status = "done"

	if err := tasks.Update(ctx, ownerID, *task, []events.Event{events.New(events.TaskStatusChanged, ownerID, events.TaskPayload{TaskID: taskID, TeamID: teamID, Status: "done"})}); err != nil {
		t.Fatalf("tasks.Update() error = %v", err)
	}

	updatedTask, err := tasks.GetByID(ctx, taskID)
	if err != nil {
		t.Fatalf("tasks.GetByID(updated) error = %v", err)
	}
	if updatedTask.CompletedAt == nil {
		t.Fatal("completed_at is nil after transition to done")
	}

	history, err := tasks.History(ctx, taskID)
	if err != nil {
		t.Fatalf("tasks.History() error = %v", err)
	}
	if len(history) != 1 || history[0].FieldName != "status" {
		t.Fatalf("history = %+v, want one status change", history)
	}

	assertTeamStatsReport(t, ctx, db, teamID)
	assertTopUsersReport(t, ctx, db, teamID, ownerID)
	assertInvalidAssigneesReport(t, ctx, db)
	assertOutboxTransactionsAndClaims(t, ctx, db, tasks, teams, ownerID, outsiderID, teamID)
	assertActivityPersistence(t, ctx, db, tasks, ownerID, teamID)
	assertTeamMembershipManagement(t, ctx, db, users, teams, ownerID, teamID)
}

func applyMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatalf("find migrations error = %v", err)
	}

	sort.Strings(files)

	if len(files) == 0 {
		t.Fatal("no migration files found")
	}

	for _, file := range files {
		migration, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s error = %v", file, err)
		}

		for _, statement := range strings.Split(string(migration), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}

			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatalf("migration %s statement %q error = %v", file, statement, err)
			}
		}
	}
}

func assertTablesExist(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	expected := map[string]bool{
		"users":               false,
		"teams":               false,
		"team_members":        false,
		"tasks":               false,
		"task_history":        false,
		"task_comments":       false,
		"notification_outbox": false,
		"team_leave_requests": false,
		"activity_events":     false,
	}

	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
	`)
	if err != nil {
		t.Fatalf("list tables error = %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan table error = %v", err)
		}
		if _, ok := expected[table]; ok {
			expected[table] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error = %v", err)
	}

	for table, found := range expected {
		if !found {
			t.Fatalf("table %s was not created", table)
		}
	}
}

func assertActivityPersistence(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	tasks *TaskRepository,
	ownerID, teamID int64,
) {
	t.Helper()

	if count := tableCount(t, ctx, db, "activity_events"); count != 3 {
		t.Fatalf("activity count = %d, want 3 committed domain events", count)
	}

	var olderID, newerID int64
	for _, target := range []*int64{&olderID, &newerID} {
		if err := db.QueryRowContext(ctx, `
			INSERT INTO activity_events(
				event_id, team_id, actor_id, event_type, entity_type, entity_id, payload, created_at
			)
			VALUES (md5(random()::text), $1, $2, 'task.updated', 'task', 1,
			        '{"task_id":1,"team_id":1,"title":"Ordering"}'::jsonb,
			        '2030-01-01T00:00:00Z')
			RETURNING id
		`, teamID, ownerID).Scan(target); err != nil {
			t.Fatalf("insert ordered activity: %v", err)
		}
	}

	activity := NewActivityRepository(db)
	firstPage, err := activity.List(ctx, teamID, 2, 0)
	if err != nil {
		t.Fatalf("activity.List(first page) error = %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].ID != newerID || firstPage[1].ID != olderID {
		t.Fatalf("first activity page ids = %+v, want %d then %d", firstPage, newerID, olderID)
	}
	secondPage, err := activity.List(ctx, teamID, 2, 2)
	if err != nil {
		t.Fatalf("activity.List(second page) error = %v", err)
	}
	if len(secondPage) != 2 || secondPage[0].ID == firstPage[0].ID || secondPage[0].ID == firstPage[1].ID {
		t.Fatalf("second activity page = %+v, want two non-overlapping events", secondPage)
	}

	duplicateActivityEventID := "activity-rollback-event"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO activity_events(
			event_id, team_id, actor_id, event_type, entity_type, entity_id, payload
		)
		VALUES ($1, $2, $3, 'task.created', 'task', 999, '{}')
	`, duplicateActivityEventID, teamID, ownerID); err != nil {
		t.Fatalf("seed duplicate activity event id: %v", err)
	}

	beforeTasks := tableCount(t, ctx, db, "tasks")
	_, err = tasks.Create(ctx, models.Task{
		Title:     "Activity must roll back",
		Status:    "todo",
		TeamID:    teamID,
		CreatedBy: ownerID,
	}, func(taskID int64) events.Event {
		event := events.New(events.TaskCreated, ownerID, events.TaskPayload{TaskID: taskID, TeamID: teamID, Title: "Activity must roll back"})
		event.EventID = duplicateActivityEventID
		return event
	})
	if err == nil {
		t.Fatal("tasks.Create(activity duplicate) error = nil, want rollback")
	}
	if count := tableCount(t, ctx, db, "tasks"); count != beforeTasks {
		t.Fatalf("task count after activity rollback = %d, want %d", count, beforeTasks)
	}
	var outboxCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_id = $1`, duplicateActivityEventID).Scan(&outboxCount); err != nil {
		t.Fatalf("count rolled back outbox event: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("outbox event after activity rollback = %d, want 0", outboxCount)
	}
}

func assertTeamMembershipManagement(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	users *UserRepository,
	teams *TeamRepository,
	ownerID, teamID int64,
) {
	t.Helper()

	adminID, err := users.Create(ctx, "membership-admin@example.com", "hash")
	if err != nil {
		t.Fatalf("create membership admin: %v", err)
	}
	leavingID, err := users.Create(ctx, "leaving-member@example.com", "hash")
	if err != nil {
		t.Fatalf("create leaving member: %v", err)
	}
	for _, invited := range []struct {
		id   int64
		role string
	}{
		{id: adminID, role: "admin"},
		{id: leavingID, role: "member"},
	} {
		event := events.New(events.TeamMemberAdded, ownerID, events.TeamMemberPayload{
			TeamID: teamID,
			UserID: invited.id,
			Role:   invited.role,
		})
		if err := teams.Invite(ctx, teamID, invited.id, invited.role, event); err != nil {
			t.Fatalf("invite %s for membership test: %v", invited.role, err)
		}
	}

	request, err := teams.CreateLeaveRequest(ctx, teamID, leavingID)
	if err != nil {
		t.Fatalf("CreateLeaveRequest() error = %v", err)
	}
	if _, err := teams.CreateLeaveRequest(ctx, teamID, leavingID); !errors.Is(err, ErrLeaveRequestExists) {
		t.Fatalf("duplicate CreateLeaveRequest() error = %v, want ErrLeaveRequestExists", err)
	}
	for _, reviewerID := range []int64{ownerID, adminID} {
		requests, err := teams.ListLeaveRequests(ctx, teamID, reviewerID)
		if err != nil {
			t.Fatalf("ListLeaveRequests(reviewer %d) error = %v", reviewerID, err)
		}
		if len(requests) != 1 || requests[0].ID != request.ID {
			t.Fatalf("reviewer %d requests = %+v, want request %d", reviewerID, requests, request.ID)
		}
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var reviewers sync.WaitGroup
	for _, reviewerID := range []int64{ownerID, adminID} {
		reviewerID := reviewerID
		reviewers.Add(1)
		go func() {
			defer reviewers.Done()
			<-start
			results <- teams.ResolveLeaveRequest(ctx, teamID, request.ID, reviewerID, true)
		}()
	}
	close(start)
	reviewers.Wait()
	close(results)

	var successes, conflicts int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrLeaveRequestHandled):
			conflicts++
		default:
			t.Fatalf("concurrent ResolveLeaveRequest() unexpected error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent resolutions: successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
	if member, err := teams.IsTeamMember(ctx, teamID, leavingID); err != nil || member {
		t.Fatalf("approved leaving member membership = %v, %v; want false, nil", member, err)
	}
	requests, err := teams.ListLeaveRequests(ctx, teamID, ownerID)
	if err != nil {
		t.Fatalf("ListLeaveRequests(after resolution) error = %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("pending requests after resolution = %+v, want empty", requests)
	}

	regularID, err := users.Create(ctx, "removed-member@example.com", "hash")
	if err != nil {
		t.Fatalf("create removable member: %v", err)
	}
	event := events.New(events.TeamMemberAdded, ownerID, events.TeamMemberPayload{TeamID: teamID, UserID: regularID, Role: "member"})
	if err := teams.Invite(ctx, teamID, regularID, "member", event); err != nil {
		t.Fatalf("invite removable member: %v", err)
	}
	if err := teams.RemoveMember(ctx, teamID, regularID, adminID); err != nil {
		t.Fatalf("admin RemoveMember(member) error = %v", err)
	}
	if err := teams.RemoveMember(ctx, teamID, adminID, ownerID); err != nil {
		t.Fatalf("owner RemoveMember(admin) error = %v", err)
	}
}

func assertOutboxTransactionsAndClaims(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	tasks *TaskRepository,
	teams *TeamRepository,
	ownerID, outsiderID, teamID int64,
) {
	t.Helper()

	if count := tableCount(t, ctx, db, "notification_outbox"); count != 3 {
		t.Fatalf("outbox count = %d, want 3 committed business events", count)
	}

	beforeTasks := tableCount(t, ctx, db, "tasks")
	_, err := tasks.Create(ctx, models.Task{
		Title:     "Invalid status",
		Status:    "invalid",
		TeamID:    teamID,
		CreatedBy: ownerID,
	}, func(taskID int64) events.Event {
		return events.New(events.TaskCreated, ownerID, events.TaskPayload{TaskID: taskID, TeamID: teamID})
	})
	if err == nil {
		t.Fatal("tasks.Create(invalid status) error = nil, want error")
	}
	if count := tableCount(t, ctx, db, "tasks"); count != beforeTasks {
		t.Fatalf("task count after business rollback = %d, want %d", count, beforeTasks)
	}
	if count := tableCount(t, ctx, db, "notification_outbox"); count != 3 {
		t.Fatalf("outbox count after business rollback = %d, want 3", count)
	}

	var duplicateEventID string
	if err := db.QueryRowContext(ctx, `SELECT event_id FROM notification_outbox ORDER BY id LIMIT 1`).Scan(&duplicateEventID); err != nil {
		t.Fatalf("select existing event id: %v", err)
	}
	_, err = tasks.Create(ctx, models.Task{
		Title:     "Must roll back",
		Status:    "todo",
		TeamID:    teamID,
		CreatedBy: ownerID,
	}, func(taskID int64) events.Event {
		event := events.New(events.TaskCreated, ownerID, events.TaskPayload{TaskID: taskID, TeamID: teamID})
		event.EventID = duplicateEventID
		return event
	})
	if err == nil {
		t.Fatal("tasks.Create(duplicate event) error = nil, want error")
	}
	var rolledBackTasks int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE title = 'Must roll back'`).Scan(&rolledBackTasks); err != nil {
		t.Fatalf("count rolled back tasks: %v", err)
	}
	if rolledBackTasks != 0 {
		t.Fatalf("rolled back tasks = %d, want 0", rolledBackTasks)
	}

	duplicateInvite := events.New(events.TeamMemberAdded, ownerID, events.TeamMemberPayload{TeamID: teamID, UserID: outsiderID, Role: "member"})
	duplicateInvite.EventID = duplicateEventID
	if err := teams.Invite(ctx, teamID, outsiderID, "member", duplicateInvite); err == nil {
		t.Fatal("teams.Invite(duplicate event) error = nil, want error")
	}
	if member, err := teams.IsTeamMember(ctx, teamID, outsiderID); err != nil || member {
		t.Fatalf("outsider membership after rollback = %v, %v; want false, nil", member, err)
	}

	outbox := NewOutboxRepository(db)
	start := make(chan struct{})
	results := make(chan []OutboxEntry, 2)
	errCh := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			entries, err := outbox.FetchPending(ctx, 2)
			if err != nil {
				errCh <- err
				return
			}
			results <- entries
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errCh)
	for err := range errCh {
		t.Fatalf("FetchPending() error = %v", err)
	}

	claimed := make(map[int64]OutboxEntry)
	for batch := range results {
		for _, entry := range batch {
			if _, exists := claimed[entry.ID]; exists {
				t.Fatalf("outbox event %d claimed by two workers", entry.ID)
			}
			claimed[entry.ID] = entry
		}
	}
	if len(claimed) != 3 {
		t.Fatalf("claimed events = %d, want 3", len(claimed))
	}

	entries := make([]OutboxEntry, 0, len(claimed))
	for _, entry := range claimed {
		entries = append(entries, entry)
	}
	if err := outbox.MarkProcessed(ctx, entries[0].ID); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	retryAt := time.Now().Add(time.Minute).UTC().Truncate(time.Microsecond)
	if err := outbox.MarkFailed(ctx, entries[1].ID, errors.New("delivery failed"), retryAt); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	var processedStatus string
	var processedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT status, processed_at FROM notification_outbox WHERE id = $1`, entries[0].ID).Scan(&processedStatus, &processedAt); err != nil {
		t.Fatalf("select processed event: %v", err)
	}
	if processedStatus != "sent" || !processedAt.Valid {
		t.Fatalf("processed event status = %q processed_at = %v", processedStatus, processedAt.Valid)
	}

	var failedStatus, lastError string
	var attempts int
	var nextAttemptAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT status, attempts, next_attempt_at, last_error FROM notification_outbox WHERE id = $1`, entries[1].ID).Scan(&failedStatus, &attempts, &nextAttemptAt, &lastError); err != nil {
		t.Fatalf("select failed event: %v", err)
	}
	if failedStatus != "failed" || attempts != 1 || lastError != "delivery failed" || !nextAttemptAt.Equal(retryAt) {
		t.Fatalf("failed event = status %q attempts %d retry %v error %q", failedStatus, attempts, nextAttemptAt, lastError)
	}
}

func tableCount(t *testing.T, ctx context.Context, db *sql.DB, table string) int {
	t.Helper()
	allowed := map[string]bool{"tasks": true, "notification_outbox": true, "activity_events": true}
	if !allowed[table] {
		t.Fatalf("unsupported table %q", table)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func assertTeamStatsReport(t *testing.T, ctx context.Context, db *sql.DB, teamID int64) {
	t.Helper()

	var gotTeamID, membersCount, doneTasksLast7Days int64
	var name string
	err := db.QueryRowContext(ctx, `
		SELECT
			t.id,
			t.name,
			COUNT(DISTINCT tm.user_id) AS members_count,
			COUNT(DISTINCT CASE
				WHEN tasks.status = 'done'
				AND tasks.completed_at >= CURRENT_TIMESTAMP - INTERVAL '7 days'
				THEN tasks.id
			END) AS done_tasks_last_7_days
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks ON tasks.team_id = t.id
		WHERE t.id = $1
		GROUP BY t.id, t.name
	`, teamID).Scan(&gotTeamID, &name, &membersCount, &doneTasksLast7Days)
	if err != nil {
		t.Fatalf("team stats query error = %v", err)
	}
	if gotTeamID != teamID || membersCount != 2 || doneTasksLast7Days != 1 {
		t.Fatalf("team stats = team %d members %d done %d; want team %d members 2 done 1", gotTeamID, membersCount, doneTasksLast7Days, teamID)
	}
}

func assertTopUsersReport(t *testing.T, ctx context.Context, db *sql.DB, teamID, ownerID int64) {
	t.Helper()

	var gotTeamID, userID, tasksCount, rank int64
	err := db.QueryRowContext(ctx, `
		SELECT
			team_id,
			user_id,
			tasks_count,
			rn
		FROM (
			SELECT
				t.team_id,
				t.created_by AS user_id,
				COUNT(*) AS tasks_count,
				ROW_NUMBER() OVER (
					PARTITION BY t.team_id
					ORDER BY COUNT(*) DESC
				) AS rn
			FROM tasks t
			WHERE t.created_at >= date_trunc('month', CURRENT_DATE)
			GROUP BY t.team_id, t.created_by
		) ranked
		WHERE rn <= 3 AND team_id = $1
	`, teamID).Scan(&gotTeamID, &userID, &tasksCount, &rank)
	if err != nil {
		t.Fatalf("top users query error = %v", err)
	}
	if gotTeamID != teamID || userID != ownerID || tasksCount != 1 || rank != 1 {
		t.Fatalf("top users = team %d user %d count %d rank %d; want team %d user %d count 1 rank 1", gotTeamID, userID, tasksCount, rank, teamID, ownerID)
	}
}

func assertInvalidAssigneesReport(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tasks
		LEFT JOIN team_members tm
			ON tm.team_id = tasks.team_id
			AND tm.user_id = tasks.assignee_id
		WHERE tasks.assignee_id IS NOT NULL
		  AND tm.user_id IS NULL
	`).Scan(&count)
	if err != nil {
		t.Fatalf("invalid assignees query error = %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid assignees count = %d, want 0", count)
	}
}

func runPostgresContainer(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start postgres container: %v", recovered)
		}
	}()

	return tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("task_manager_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
}

func pingDB(ctx context.Context, db *sql.DB) error {
	var err error
	for range 30 {
		if err = db.PingContext(ctx); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return err
}

func skipIfDockerUnavailable(t *testing.T, err error) {
	t.Helper()

	message := strings.ToLower(err.Error())
	if errors.Is(err, context.Canceled) ||
		strings.Contains(message, "cannot connect to the docker daemon") ||
		strings.Contains(message, "docker daemon") ||
		strings.Contains(message, "rootless docker not found") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "permission denied") {
		t.Skipf("Docker is not available for testcontainers: %v", err)
	}
}
