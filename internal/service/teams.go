package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"

	"task-manager/internal/domain"
	"task-manager/internal/models"
)

type TeamRepository interface {
	Create(ctx context.Context, name string, userID int64) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]models.TeamWithRole, error)
	ListMembers(ctx context.Context, teamID int64) ([]models.TeamMember, error)
	GetUserRole(ctx context.Context, teamID, userID int64) (string, error)
	IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error)
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

func (s *TeamService) Create(ctx context.Context, userID int64, name string) (teamID int64, err error) {
	ctx, span := startSpan(ctx, "TeamService.Create", attribute.Int64("user.id", userID))
	defer func() { finishSpan(span, err) }()

	teamID, err = s.teams.Create(ctx, name, userID)
	if err != nil {
		return 0, fmt.Errorf("create team: %w", err)
	}
	span.SetAttributes(attribute.Int64("team.id", teamID))
	return teamID, nil
}

func (s *TeamService) List(ctx context.Context, userID int64) (teams []models.TeamWithRole, err error) {
	ctx, span := startSpan(ctx, "TeamService.List", attribute.Int64("user.id", userID))
	defer func() { finishSpan(span, err) }()

	teams, err = s.teams.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	return teams, nil
}

func (s *TeamService) ListMembers(ctx context.Context, userID, teamID int64) (members []models.TeamMember, err error) {
	ctx, span := startSpan(ctx, "TeamService.ListMembers",
		attribute.Int64("user.id", userID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	isMember, err := s.teams.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if !isMember {
		return nil, ErrForbidden
	}

	members, err = s.teams.ListMembers(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	return members, nil
}

func (s *TeamService) Invite(ctx context.Context, currentUserID, teamID, invitedUserID int64, role string) (err error) {
	ctx, span := startSpan(ctx, "TeamService.Invite",
		attribute.Int64("user.id", currentUserID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

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
