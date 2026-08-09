package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"task-manager/internal/domain"
	"task-manager/internal/events"
	"task-manager/internal/models"
)

type TeamRepository interface {
	Create(ctx context.Context, name string, userID int64) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]models.TeamWithRole, error)
	ListMembers(ctx context.Context, teamID int64) ([]models.TeamMember, error)
	GetUserRole(ctx context.Context, teamID, userID int64) (string, error)
	IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error)
	Invite(ctx context.Context, teamID, userID int64, role string, event events.Event) error
	RemoveMember(ctx context.Context, teamID, userID, actorID int64) error
	CreateLeaveRequest(ctx context.Context, teamID, userID int64) (models.TeamLeaveRequest, error)
	GetPendingLeaveRequest(ctx context.Context, teamID, userID int64) (*models.TeamLeaveRequest, error)
	ListLeaveRequests(ctx context.Context, teamID, excludeUserID int64) ([]models.TeamLeaveRequest, error)
	ResolveLeaveRequest(ctx context.Context, teamID, requestID, resolverID int64, approved bool) error
}

type TeamService struct {
	teams TeamRepository
}

func NewTeamService(teams TeamRepository) *TeamService {
	return &TeamService{teams: teams}
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

	event := events.New(events.TeamMemberAdded, currentUserID, events.TeamMemberPayload{
		TeamID: teamID,
		UserID: invitedUserID,
		Role:   role,
	})
	if err := s.teams.Invite(ctx, teamID, invitedUserID, role, event); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, domain.ErrTeamMemberExists) {
			return ErrConflict
		}
		return fmt.Errorf("invite team member: %w", err)
	}

	return nil
}

func (s *TeamService) RemoveMember(ctx context.Context, currentUserID, teamID, memberID int64) (err error) {
	ctx, span := startSpan(ctx, "TeamService.RemoveMember",
		attribute.Int64("user.id", currentUserID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	currentRole, err := s.userRole(ctx, teamID, currentUserID)
	if err != nil {
		return err
	}
	if currentUserID == memberID {
		return ErrForbidden
	}
	targetRole, err := s.userRole(ctx, teamID, memberID)
	if err != nil {
		return err
	}
	if targetRole == "owner" || currentRole == "member" || (currentRole == "admin" && targetRole != "member") {
		return ErrForbidden
	}

	if err := s.teams.RemoveMember(ctx, teamID, memberID, currentUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("remove team member: %w", err)
	}
	return nil
}

func (s *TeamService) RequestLeave(ctx context.Context, userID, teamID int64) (request models.TeamLeaveRequest, err error) {
	ctx, span := startSpan(ctx, "TeamService.RequestLeave",
		attribute.Int64("user.id", userID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	role, err := s.userRole(ctx, teamID, userID)
	if err != nil {
		return models.TeamLeaveRequest{}, err
	}
	if role == "owner" {
		return models.TeamLeaveRequest{}, ErrForbidden
	}

	request, err = s.teams.CreateLeaveRequest(ctx, teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.TeamLeaveRequest{}, ErrForbidden
	}
	if errors.Is(err, domain.ErrLeaveRequestExists) {
		return models.TeamLeaveRequest{}, ErrConflict
	}
	if err != nil {
		return models.TeamLeaveRequest{}, fmt.Errorf("create team leave request: %w", err)
	}
	return request, nil
}

func (s *TeamService) ListLeaveRequests(ctx context.Context, userID, teamID int64) (requests []models.TeamLeaveRequest, err error) {
	ctx, span := startSpan(ctx, "TeamService.ListLeaveRequests",
		attribute.Int64("user.id", userID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	role, err := s.userRole(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if role != "owner" && role != "admin" {
		return nil, ErrForbidden
	}

	requests, err = s.teams.ListLeaveRequests(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("list team leave requests: %w", err)
	}
	return requests, nil
}

func (s *TeamService) GetOwnLeaveRequest(ctx context.Context, userID, teamID int64) (*models.TeamLeaveRequest, error) {
	if _, err := s.userRole(ctx, teamID, userID); err != nil {
		return nil, err
	}
	request, err := s.teams.GetPendingLeaveRequest(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("get pending team leave request: %w", err)
	}
	return request, nil
}

func (s *TeamService) ResolveLeaveRequest(ctx context.Context, userID, teamID, requestID int64, approved bool) (err error) {
	ctx, span := startSpan(ctx, "TeamService.ResolveLeaveRequest",
		attribute.Int64("user.id", userID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	role, err := s.userRole(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return ErrForbidden
	}

	err = s.teams.ResolveLeaveRequest(ctx, teamID, requestID, userID, approved)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.Is(err, domain.ErrLeaveRequestHandled) {
		return ErrConflict
	}
	if errors.Is(err, domain.ErrCannotResolveOwn) {
		return ErrForbidden
	}
	if err != nil {
		return fmt.Errorf("resolve team leave request: %w", err)
	}
	return nil
}

func (s *TeamService) userRole(ctx context.Context, teamID, userID int64) (string, error) {
	role, err := s.teams.GetUserRole(ctx, teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", fmt.Errorf("get team member role: %w", err)
	}
	return role, nil
}
