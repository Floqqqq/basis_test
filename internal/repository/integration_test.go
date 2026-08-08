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
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

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

	if err := teams.Invite(ctx, teamID, memberID, "member"); err != nil {
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
	})
	if err != nil {
		t.Fatalf("tasks.Create() error = %v", err)
	}

	task, err := tasks.GetByID(ctx, taskID)
	if err != nil {
		t.Fatalf("tasks.GetByID() error = %v", err)
	}
	task.Status = "done"

	if err := tasks.Update(ctx, ownerID, *task); err != nil {
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
		"users":         false,
		"teams":         false,
		"team_members":  false,
		"tasks":         false,
		"task_history":  false,
		"task_comments": false,
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
