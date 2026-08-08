package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"task-manager/internal/domain"
	"task-manager/internal/models"
)

type TeamRepository interface {
	Create(ctx context.Context, name string, userID int64) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]models.Team, error)
	GetUserRole(ctx context.Context, teamID, userID int64) (string, error)
	Invite(ctx context.Context, teamID, userID int64, role string) error
}

type TeamService struct {
	teams        TeamRepository
	inviteSender InviteSender
}

func NewTeamService(teams TeamRepository, inviteSender InviteSender) *TeamService {
	return &TeamService{
		teams:        teams,
		inviteSender: inviteSender,
	}
}

func (s *TeamService) Create(ctx context.Context, userID int64, name string) (int64, error) {
	teamID, err := s.teams.Create(ctx, name, userID)
	if err != nil {
		return 0, fmt.Errorf("create team: %w", err)
	}
	return teamID, nil
}

func (s *TeamService) List(ctx context.Context, userID int64) ([]models.Team, error) {
	teams, err := s.teams.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	return teams, nil
}

func (s *TeamService) Invite(ctx context.Context, currentUserID, teamID, invitedUserID int64, role string) error {
	currentUserRole, err := s.teams.GetUserRole(ctx, teamID, currentUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrForbidden
		}
		return fmt.Errorf("get current user role: %w", err)
	}
	if currentUserRole != "owner" && currentUserRole != "admin" {
		return ErrForbidden
	}

	if err := s.teams.Invite(ctx, teamID, invitedUserID, role); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, domain.ErrTeamMemberExists) {
			return ErrConflict
		}
		return fmt.Errorf("invite team member: %w", err)
	}

	if s.inviteSender != nil {
		if err := s.inviteSender.SendInvite(ctx, teamID, invitedUserID, role); err != nil {
			slog.Warn("invite sender failed", "team_id", teamID, "user_id", invitedUserID, "error", err)
		}
	}

	return nil
}
