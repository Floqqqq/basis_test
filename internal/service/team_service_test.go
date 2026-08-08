package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"task-manager/internal/models"
	"task-manager/internal/repository"
)

type fakeTeamRepository struct {
	roles     map[int64]string
	members   map[int64]bool
	inviteErr error
	invited   bool
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

func (r *fakeTeamRepository) Invite(ctx context.Context, teamID, userID int64, role string) error {
	if r.inviteErr != nil {
		return r.inviteErr
	}
	r.invited = true
	return nil
}

type fakeInviteSender struct {
	err  error
	sent bool
}

func (s *fakeInviteSender) SendInvite(ctx context.Context, teamID, userID int64, role string) error {
	s.sent = true
	return s.err
}

func TestTeamServiceInvite(t *testing.T) {
	teams := &fakeTeamRepository{roles: map[int64]string{1: "admin"}}
	inviteSender := &fakeInviteSender{}
	service := NewTeamService(teams, inviteSender)

	err := service.Invite(context.Background(), 1, 5, 2, "member")
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if !teams.invited {
		t.Fatal("repository invite was not called")
	}
	if !inviteSender.sent {
		t.Fatal("invite sender was not called")
	}
}

func TestTeamServiceInviteWithoutPermissions(t *testing.T) {
	teams := &fakeTeamRepository{roles: map[int64]string{1: "member"}}
	service := NewTeamService(teams, nil)

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
			}, nil)

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

func TestTeamServiceInviteSenderErrorDoesNotBreakRequest(t *testing.T) {
	teams := &fakeTeamRepository{roles: map[int64]string{1: "owner"}}
	inviteSender := &fakeInviteSender{err: errors.New("sender down")}
	service := NewTeamService(teams, inviteSender)

	err := service.Invite(context.Background(), 1, 5, 2, "member")
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if !inviteSender.sent {
		t.Fatal("invite sender was not called")
	}
}
