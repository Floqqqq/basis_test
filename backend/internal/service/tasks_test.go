package service

import (
	"context"
	"database/sql"
	"testing"

	"task-manager/internal/models"
)

type fakeTeamAccess struct {
	roles   map[int64]string
	members map[int64]bool
}

func (f fakeTeamAccess) GetUserRole(ctx context.Context, teamID, userID int64) (string, error) {
	role, ok := f.roles[userID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return role, nil
}

func (f fakeTeamAccess) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return f.members[userID], nil
}

func TestTaskPolicyIsAssigneeValidForTeam(t *testing.T) {
	policy := NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{
			2: true,
		},
	})

	if ok, err := policy.IsAssigneeValidForTeam(context.Background(), 1, nil); err != nil || !ok {
		t.Fatalf("nil assignee valid = %v, %v; want true, nil", ok, err)
	}

	assigneeID := int64(2)
	if ok, err := policy.IsAssigneeValidForTeam(context.Background(), 1, &assigneeID); err != nil || !ok {
		t.Fatalf("team member assignee valid = %v, %v; want true, nil", ok, err)
	}

	outsiderID := int64(3)
	if ok, err := policy.IsAssigneeValidForTeam(context.Background(), 1, &outsiderID); err != nil || ok {
		t.Fatalf("outsider assignee valid = %v, %v; want false, nil", ok, err)
	}
}

func TestTaskPolicyCanUpdateTask(t *testing.T) {
	assigneeID := int64(4)
	task := &models.Task{
		ID:         10,
		TeamID:     1,
		CreatedBy:  3,
		AssigneeID: &assigneeID,
	}
	policy := NewTaskPolicy(fakeTeamAccess{
		roles: map[int64]string{
			1: "owner",
			2: "admin",
			3: "member",
			4: "member",
			5: "member",
		},
	})

	tests := []struct {
		name   string
		userID int64
		want   bool
	}{
		{name: "owner", userID: 1, want: true},
		{name: "admin", userID: 2, want: true},
		{name: "creator", userID: 3, want: true},
		{name: "assignee", userID: 4, want: true},
		{name: "other member", userID: 5, want: false},
		{name: "outsider", userID: 6, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := policy.CanUpdateTask(context.Background(), task, tt.userID)
			if err != nil {
				t.Fatalf("CanUpdateTask() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("CanUpdateTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskPolicyCanViewTaskHistory(t *testing.T) {
	assigneeID := int64(4)
	task := &models.Task{
		ID:         10,
		TeamID:     1,
		CreatedBy:  3,
		AssigneeID: &assigneeID,
	}
	policy := NewTaskPolicy(fakeTeamAccess{
		roles: map[int64]string{
			1: "owner",
			2: "admin",
			3: "member",
			4: "member",
			5: "member",
		},
	})

	for _, userID := range []int64{1, 2, 3, 4} {
		got, err := policy.CanViewTaskHistory(context.Background(), task, userID)
		if err != nil {
			t.Fatalf("CanViewTaskHistory(%d) error = %v", userID, err)
		}
		if !got {
			t.Fatalf("CanViewTaskHistory(%d) = false, want true", userID)
		}
	}

	got, err := policy.CanViewTaskHistory(context.Background(), task, 5)
	if err != nil {
		t.Fatalf("CanViewTaskHistory(other member) error = %v", err)
	}
	if got {
		t.Fatal("CanViewTaskHistory(other member) = true, want false")
	}
}
