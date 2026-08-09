package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"task-manager/internal/events"
	"task-manager/internal/models"
	"task-manager/internal/repository"
)

type fakeTeamRepository struct {
	roles             map[int64]string
	members           map[int64]bool
	inviteErr         error
	invited           bool
	event             events.Event
	removedUserID     int64
	leaveRequest      models.TeamLeaveRequest
	leaveRequests     []models.TeamLeaveRequest
	leaveRequestErr   error
	resolvedRequestID int64
	resolvedApproved  bool
}

func (r *fakeTeamRepository) Create(ctx context.Context, name string, userID int64) (int64, error) {
	return 1, nil
}

func (r *fakeTeamRepository) ListByUser(ctx context.Context, userID int64) ([]models.TeamWithRole, error) {
	return nil, nil
}

func (r *fakeTeamRepository) ListMembers(ctx context.Context, teamID int64) ([]models.TeamMember, error) {
	return nil, nil
}

func (r *fakeTeamRepository) GetUserRole(ctx context.Context, teamID, userID int64) (string, error) {
	role, ok := r.roles[userID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return role, nil
}

func (r *fakeTeamRepository) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return r.members[userID], nil
}

func (r *fakeTeamRepository) Invite(ctx context.Context, teamID, userID int64, role string, event events.Event) error {
	if r.inviteErr != nil {
		return r.inviteErr
	}
	r.invited = true
	r.event = event
	return nil
}

func (r *fakeTeamRepository) RemoveMember(ctx context.Context, teamID, userID, actorID int64) error {
	r.removedUserID = userID
	return nil
}

func (r *fakeTeamRepository) CreateLeaveRequest(ctx context.Context, teamID, userID int64) (models.TeamLeaveRequest, error) {
	if r.leaveRequestErr != nil {
		return models.TeamLeaveRequest{}, r.leaveRequestErr
	}
	r.leaveRequest = models.TeamLeaveRequest{ID: 8, TeamID: teamID, UserID: userID, Status: "pending"}
	return r.leaveRequest, nil
}

func (r *fakeTeamRepository) ListLeaveRequests(ctx context.Context, teamID, excludeUserID int64) ([]models.TeamLeaveRequest, error) {
	return r.leaveRequests, nil
}

func (r *fakeTeamRepository) GetPendingLeaveRequest(ctx context.Context, teamID, userID int64) (*models.TeamLeaveRequest, error) {
	if r.leaveRequest.ID == 0 {
		return nil, nil
	}
	return &r.leaveRequest, nil
}

func (r *fakeTeamRepository) ResolveLeaveRequest(ctx context.Context, teamID, requestID, resolverID int64, approved bool) error {
	if r.leaveRequestErr != nil {
		return r.leaveRequestErr
	}
	r.resolvedRequestID = requestID
	r.resolvedApproved = approved
	return nil
}

func TestTeamServiceInvite(t *testing.T) {
	teams := &fakeTeamRepository{roles: map[int64]string{1: "admin"}}
	service := NewTeamService(teams)

	err := service.Invite(context.Background(), 1, 5, 2, "member")
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if !teams.invited {
		t.Fatal("repository invite was not called")
	}
	if teams.event.EventType != events.TeamMemberAdded {
		t.Fatalf("event type = %q, want team.member_added", teams.event.EventType)
	}
}

func TestTeamServiceInviteWithoutPermissions(t *testing.T) {
	teams := &fakeTeamRepository{roles: map[int64]string{1: "member"}}
	service := NewTeamService(teams)

	err := service.Invite(context.Background(), 1, 5, 2, "member")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Invite() error = %v, want ErrForbidden", err)
	}
	if teams.invited {
		t.Fatal("repository invite was called")
	}
}

func TestTeamServiceInviteRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "user not found", err: repository.ErrUserNotFound, wantErr: ErrNotFound},
		{name: "member exists", err: repository.ErrTeamMemberExists, wantErr: ErrConflict},
		{name: "storage error", err: errors.New("db failed"), wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTeamService(&fakeTeamRepository{
				roles:     map[int64]string{1: "owner"},
				inviteErr: tt.err,
			})

			err := service.Invite(context.Background(), 1, 5, 2, "member")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Invite() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err == nil {
				t.Fatal("Invite() error = nil, want error")
			}
		})
	}
}

func TestTeamServiceRemoveMemberRoleRules(t *testing.T) {
	tests := []struct {
		name       string
		roles      map[int64]string
		actorID    int64
		targetID   int64
		wantErr    error
		wantRemove bool
	}{
		{name: "owner removes admin", roles: map[int64]string{1: "owner", 2: "admin"}, actorID: 1, targetID: 2, wantRemove: true},
		{name: "admin removes member", roles: map[int64]string{1: "admin", 2: "member"}, actorID: 1, targetID: 2, wantRemove: true},
		{name: "admin cannot remove admin", roles: map[int64]string{1: "admin", 2: "admin"}, actorID: 1, targetID: 2, wantErr: ErrForbidden},
		{name: "admin cannot remove owner", roles: map[int64]string{1: "admin", 2: "owner"}, actorID: 1, targetID: 2, wantErr: ErrForbidden},
		{name: "member cannot remove member", roles: map[int64]string{1: "member", 2: "member"}, actorID: 1, targetID: 2, wantErr: ErrForbidden},
		{name: "cannot remove self", roles: map[int64]string{1: "admin"}, actorID: 1, targetID: 1, wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTeamRepository{roles: tt.roles}
			err := NewTeamService(repo).RemoveMember(context.Background(), tt.actorID, 5, tt.targetID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RemoveMember() error = %v, want %v", err, tt.wantErr)
			}
			if got := repo.removedUserID != 0; got != tt.wantRemove {
				t.Fatalf("repository remove called = %v, want %v", got, tt.wantRemove)
			}
		})
	}
}

func TestTeamServiceRequestLeave(t *testing.T) {
	repo := &fakeTeamRepository{roles: map[int64]string{2: "member"}}
	request, err := NewTeamService(repo).RequestLeave(context.Background(), 2, 5)
	if err != nil {
		t.Fatalf("RequestLeave() error = %v", err)
	}
	if request.ID != 8 || request.Status != "pending" {
		t.Fatalf("request = %+v, want pending request 8", request)
	}

	ownerRepo := &fakeTeamRepository{roles: map[int64]string{1: "owner"}}
	if _, err := NewTeamService(ownerRepo).RequestLeave(context.Background(), 1, 5); !errors.Is(err, ErrForbidden) {
		t.Fatalf("owner RequestLeave() error = %v, want ErrForbidden", err)
	}

	duplicateRepo := &fakeTeamRepository{
		roles:           map[int64]string{2: "member"},
		leaveRequestErr: repository.ErrLeaveRequestExists,
	}
	if _, err := NewTeamService(duplicateRepo).RequestLeave(context.Background(), 2, 5); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate RequestLeave() error = %v, want ErrConflict", err)
	}
}

func TestTeamServiceResolveLeaveRequest(t *testing.T) {
	repo := &fakeTeamRepository{roles: map[int64]string{1: "admin"}}
	service := NewTeamService(repo)
	if err := service.ResolveLeaveRequest(context.Background(), 1, 5, 9, true); err != nil {
		t.Fatalf("ResolveLeaveRequest() error = %v", err)
	}
	if repo.resolvedRequestID != 9 || !repo.resolvedApproved {
		t.Fatalf("resolved request = %d approved=%v, want 9 true", repo.resolvedRequestID, repo.resolvedApproved)
	}

	repo.leaveRequestErr = repository.ErrLeaveRequestHandled
	if err := service.ResolveLeaveRequest(context.Background(), 1, 5, 9, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("handled ResolveLeaveRequest() error = %v, want ErrConflict", err)
	}
}
