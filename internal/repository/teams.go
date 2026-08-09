package repository

import (
	"context"
	"database/sql"
	"fmt"

	"task-manager/internal/domain"
	"task-manager/internal/events"
	"task-manager/internal/models"
)

type TeamRepository struct {
	db     *sql.DB
	outbox *OutboxRepository
}

var (
	ErrUserNotFound     = domain.ErrUserNotFound
	ErrTeamMemberExists = domain.ErrTeamMemberExists
)

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db, outbox: NewOutboxRepository(db)}
}

func (r *TeamRepository) Create(ctx context.Context, name string, userID int64) (teamID int64, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin create team tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
				err = fmt.Errorf("rollback create team tx: %w", rollbackErr)
			}
		}
	}()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO teams(name, created_by) VALUES ($1, $2) RETURNING id`,
		name,
		userID,
	).Scan(&teamID)
	if err != nil {
		return 0, fmt.Errorf("insert team: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO team_members(user_id, team_id, role) VALUES ($1, $2, 'owner')`,
		userID,
		teamID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert owner team member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit create team tx: %w", err)
	}
	committed = true

	return teamID, nil
}

func (r *TeamRepository) ListByUser(ctx context.Context, userID int64) ([]models.TeamWithRole, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		ORDER BY t.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams by user: %w", err)
	}
	defer rows.Close()

	teams := make([]models.TeamWithRole, 0)

	for rows.Next() {
		var t models.TeamWithRole
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt, &t.Role); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}

		teams = append(teams, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}

	return teams, nil
}

func (r *TeamRepository) ListMembers(ctx context.Context, teamID int64) ([]models.TeamMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email, tm.role, tm.created_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY tm.created_at, u.id
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()

	members := make([]models.TeamMember, 0)
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.ID, &member.Email, &member.Role, &member.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team members: %w", err)
	}

	return members, nil
}

func (r *TeamRepository) GetUserRole(ctx context.Context, teamID, userID int64) (string, error) {
	var role string

	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2`,
		teamID,
		userID,
	).Scan(&role)

	if err != nil {
		return "", fmt.Errorf("get user team role: %w", err)
	}

	return role, nil
}

func (r *TeamRepository) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`, teamID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check team membership: %w", err)
	}

	return exists, nil
}

func (r *TeamRepository) Invite(ctx context.Context, teamID, userID int64, role string, event events.Event) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin invite team member tx: %w", err)
	}
	defer rollbackUnlessCommitted(tx, &err)

	var userExists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE id = $1
		)
	`, userID).Scan(&userExists); err != nil {
		return fmt.Errorf("check invited user exists: %w", err)
	}

	if !userExists {
		return ErrUserNotFound
	}

	var memberExists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`, teamID, userID).Scan(&memberExists); err != nil {
		return fmt.Errorf("check team member exists: %w", err)
	}

	if memberExists {
		return ErrTeamMemberExists
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO team_members(user_id, team_id, role) VALUES ($1, $2, $3)`,
		userID,
		teamID,
		role,
	)
	if err != nil {
		return fmt.Errorf("insert team member invite: %w", err)
	}
	if err := r.outbox.Create(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit invite team member tx: %w", err)
	}
	tx = nil

	return nil
}
