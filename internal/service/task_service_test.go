package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

type fakeTaskRepository struct {
	createID   int64
	createErr  error
	listTasks  []models.Task
	listErr    error
	task       *models.Task
	getErr     error
	updateErr  error
	comments   []models.TaskComment
	comment    *models.TaskComment
	commentErr error

	createdTask *models.Task
	updatedTask *models.Task
	events      []events.Event
}

func (r *fakeTaskRepository) Create(ctx context.Context, task models.Task, eventFactory func(int64) events.Event) (int64, error) {
	r.createdTask = &task
	if r.createErr != nil {
		return 0, r.createErr
	}
	if eventFactory != nil {
		r.events = append(r.events, eventFactory(r.createID))
	}
	return r.createID, nil
}

func (r *fakeTaskRepository) List(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]models.Task, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listTasks, nil
}

func (r *fakeTaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.task, nil
}

func (r *fakeTaskRepository) Update(ctx context.Context, userID int64, task models.Task, domainEvents []events.Event) error {
	r.updatedTask = &task
	r.events = append(r.events, domainEvents...)
	return r.updateErr
}

func (r *fakeTaskRepository) History(ctx context.Context, taskID int64) ([]models.TaskHistory, error) {
	return nil, nil
}

func (r *fakeTaskRepository) CreateComment(ctx context.Context, taskID, userID int64, comment string) (*models.TaskComment, error) {
	if r.commentErr != nil {
		return nil, r.commentErr
	}
	return r.comment, nil
}

func (r *fakeTaskRepository) ListComments(ctx context.Context, taskID int64) ([]models.TaskComment, error) {
	if r.commentErr != nil {
		return nil, r.commentErr
	}
	return r.comments, nil
}

type fakeTaskCache struct {
	getBody       []byte
	getOK         bool
	getErr        error
	setErr        error
	invalidateErr error

	getCalled        bool
	setCalled        bool
	invalidateCalled bool
}

func (c *fakeTaskCache) GetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]byte, bool, error) {
	c.getCalled = true
	return c.getBody, c.getOK, c.getErr
}

func (c *fakeTaskCache) SetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int, body []byte) error {
	c.setCalled = true
	return c.setErr
}

func (c *fakeTaskCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	c.invalidateCalled = true
	return c.invalidateErr
}

func TestTaskServiceCreate(t *testing.T) {
	taskRepo := &fakeTaskRepository{createID: 10}
	taskCache := &fakeTaskCache{}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{
			1: true,
			2: true,
		},
	}), taskCache)
	assigneeID := int64(2)

	taskID, err := service.Create(context.Background(), 1, models.Task{
		Title:      "Task",
		Status:     "todo",
		TeamID:     5,
		AssigneeID: &assigneeID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if taskID != 10 {
		t.Fatalf("taskID = %d, want 10", taskID)
	}
	if taskRepo.createdTask == nil || taskRepo.createdTask.CreatedBy != 1 {
		t.Fatalf("created task = %+v, want CreatedBy 1", taskRepo.createdTask)
	}
	if len(taskRepo.events) != 1 || taskRepo.events[0].EventType != events.TaskCreated {
		t.Fatalf("events = %+v, want task.created", taskRepo.events)
	}
	payload, ok := taskRepo.events[0].Payload.(events.TaskPayload)
	if !ok || payload.TaskID != 10 || payload.TeamID != 5 {
		t.Fatalf("event payload = %+v, want task 10 team 5", taskRepo.events[0].Payload)
	}
	if !taskCache.invalidateCalled {
		t.Fatal("cache invalidation was not called")
	}
}

func TestTaskServiceUpdateAssignmentEvent(t *testing.T) {
	taskRepo := &fakeTaskRepository{task: &models.Task{ID: 10, Status: "todo", TeamID: 5, CreatedBy: 1}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		roles:   map[int64]string{1: "owner"},
		members: map[int64]bool{2: true},
	}), nil)
	assigneeID := int64(2)

	err := service.Update(context.Background(), 1, 10, TaskUpdate{AssigneeIDSet: true, AssigneeID: &assigneeID})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(taskRepo.events) != 1 || taskRepo.events[0].EventType != events.TaskAssigned {
		t.Fatalf("events = %+v, want task.assigned", taskRepo.events)
	}
}

func TestTaskServiceUpdateNoChangesDoesNotWrite(t *testing.T) {
	taskRepo := &fakeTaskRepository{task: &models.Task{ID: 10, Title: "Same", Status: "todo", TeamID: 5, CreatedBy: 1}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{roles: map[int64]string{1: "owner"}}), nil)
	title := "Same"

	if err := service.Update(context.Background(), 1, 10, TaskUpdate{Title: &title}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if taskRepo.updatedTask != nil || len(taskRepo.events) != 0 {
		t.Fatalf("update = %+v events = %+v, want no write", taskRepo.updatedTask, taskRepo.events)
	}
}

func TestTaskServiceCreateInvalidAssignee(t *testing.T) {
	service := NewTaskService(&fakeTaskRepository{}, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{
			1: true,
		},
	}), nil)
	assigneeID := int64(2)

	_, err := service.Create(context.Background(), 1, models.Task{
		Title:      "Task",
		Status:     "todo",
		TeamID:     5,
		AssigneeID: &assigneeID,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create() error = %v, want ErrInvalidInput", err)
	}
}

func TestTaskServiceListCacheHit(t *testing.T) {
	cachedTasks := []models.Task{{ID: 7, Title: "Cached"}}
	body, err := json.Marshal(cachedTasks)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	taskCache := &fakeTaskCache{getBody: body, getOK: true}
	taskRepo := &fakeTaskRepository{listErr: errors.New("repository should not be called")}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{1: true},
	}), taskCache)

	tasks, err := service.List(context.Background(), 1, 5, "", nil, 20, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 7 {
		t.Fatalf("tasks = %+v, want cached task", tasks)
	}
	if taskCache.setCalled {
		t.Fatal("cache set was called on cache hit")
	}
}

func TestTaskServiceListCacheMiss(t *testing.T) {
	taskCache := &fakeTaskCache{}
	taskRepo := &fakeTaskRepository{listTasks: []models.Task{{ID: 8, Title: "From DB"}}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{1: true},
	}), taskCache)

	tasks, err := service.List(context.Background(), 1, 5, "", nil, 20, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 8 {
		t.Fatalf("tasks = %+v, want repository task", tasks)
	}
	if !taskCache.setCalled {
		t.Fatal("cache set was not called")
	}
}

func TestTaskServiceListCacheFailureDoesNotBreakRequest(t *testing.T) {
	taskCache := &fakeTaskCache{getErr: errors.New("redis down"), setErr: errors.New("redis still down")}
	taskRepo := &fakeTaskRepository{listTasks: []models.Task{{ID: 9, Title: "From DB"}}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{1: true},
	}), taskCache)

	tasks, err := service.List(context.Background(), 1, 5, "", nil, 20, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 9 {
		t.Fatalf("tasks = %+v, want repository task", tasks)
	}
}

func TestTaskServiceUpdate(t *testing.T) {
	taskRepo := &fakeTaskRepository{task: &models.Task{
		ID:        10,
		Title:     "Old",
		Status:    "todo",
		TeamID:    5,
		CreatedBy: 1,
	}}
	taskCache := &fakeTaskCache{}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		roles: map[int64]string{1: "member"},
	}), taskCache)
	title := "New"
	status := "done"

	err := service.Update(context.Background(), 1, 10, TaskUpdate{
		Title:  &title,
		Status: &status,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if taskRepo.updatedTask == nil || taskRepo.updatedTask.Title != "New" || taskRepo.updatedTask.Status != "done" {
		t.Fatalf("updated task = %+v, want title New and status done", taskRepo.updatedTask)
	}
	if !taskCache.invalidateCalled {
		t.Fatal("cache invalidation was not called")
	}
	if len(taskRepo.events) != 2 || taskRepo.events[0].EventType != events.TaskUpdated || taskRepo.events[1].EventType != events.TaskStatusChanged {
		t.Fatalf("events = %+v, want task.updated and task.status_changed", taskRepo.events)
	}
}

func TestTaskServiceForbiddenUpdate(t *testing.T) {
	taskRepo := &fakeTaskRepository{task: &models.Task{
		ID:        10,
		TeamID:    5,
		CreatedBy: 1,
	}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		roles: map[int64]string{2: "member"},
	}), nil)
	title := "New"

	err := service.Update(context.Background(), 2, 10, TaskUpdate{Title: &title})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestTaskServiceGetByIDNotFound(t *testing.T) {
	service := NewTaskService(&fakeTaskRepository{getErr: sql.ErrNoRows}, NewTaskPolicy(fakeTeamAccess{}), nil)

	_, err := service.GetByID(context.Background(), 1, 404)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestTaskServiceUpdateClearsAssignee(t *testing.T) {
	assigneeID := int64(2)
	taskRepo := &fakeTaskRepository{task: &models.Task{
		ID:         10,
		TeamID:     5,
		CreatedBy:  1,
		AssigneeID: &assigneeID,
	}}
	service := NewTaskService(taskRepo, NewTaskPolicy(fakeTeamAccess{
		roles: map[int64]string{1: "owner"},
	}), nil)

	err := service.Update(context.Background(), 1, 10, TaskUpdate{
		AssigneeIDSet: true,
		AssigneeID:    nil,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if taskRepo.updatedTask == nil || taskRepo.updatedTask.AssigneeID != nil {
		t.Fatalf("updated task = %+v, want assignee cleared", taskRepo.updatedTask)
	}
}
