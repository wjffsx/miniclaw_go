package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	if scheduler == nil {
		t.Error("Expected scheduler to be created")
	}

	if scheduler.tasks == nil {
		t.Error("Expected tasks map to be initialized")
	}

	if scheduler.running {
		t.Error("Expected scheduler to not be running initially")
	}
}

func TestAddTask(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scheduler.tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(scheduler.tasks))
	}

	if _, exists := scheduler.tasks["test-task"]; !exists {
		t.Error("Expected task to be added")
	}
}

func TestAddTaskDuplicate(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = scheduler.AddTask(task)
	if err == nil {
		t.Error("Expected error when adding duplicate task")
	}
}

func TestRemoveTask(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	scheduler.AddTask(task)

	err := scheduler.RemoveTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scheduler.tasks) != 0 {
		t.Errorf("Expected 0 tasks after removal, got %d", len(scheduler.tasks))
	}
}

func TestRemoveTaskNonExistent(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	err := scheduler.RemoveTask("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent task")
	}
}

func TestGetTask(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	scheduler.AddTask(task)

	retrieved, exists := scheduler.GetTask("test-task")
	if !exists {
		t.Fatal("Expected task to exist")
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestListTasks(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task1 := &Task{
		ID:          "task-1",
		Name:        "Task 1",
		Description: "First task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	task2 := &Task{
		ID:          "task-2",
		Name:        "Task 2",
		Description: "Second task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	tasks := scheduler.ListTasks()

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestStart(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !scheduler.running {
		t.Error("Expected scheduler to be running")
	}

	scheduler.Stop()
}

func TestStartAlreadyRunning(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	scheduler.Start()

	err := scheduler.Start()
	if err == nil {
		t.Error("Expected error when starting already running scheduler")
	}

	scheduler.Stop()
}

func TestStop(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	scheduler.Start()

	err := scheduler.Stop()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if scheduler.running {
		t.Error("Expected scheduler to not be running")
	}
}

func TestStopNotRunning(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	err := scheduler.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping not running scheduler, got %v", err)
	}
}

func TestEnableTask(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     false,
	}

	scheduler.AddTask(task)

	err := scheduler.EnableTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	task, _ = scheduler.GetTask("test-task")
	if !task.Enabled {
		t.Error("Expected task to be enabled")
	}
}

func TestDisableTask(t *testing.T) {
	config := &SchedulerConfig{
		TickInterval: time.Second,
	}
	scheduler := NewScheduler(config)

	task := &Task{
		ID:          "test-task",
		Name:        "Test Task",
		Description: "A test task",
		CronExpr:    "0 * * * *",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	scheduler.AddTask(task)

	err := scheduler.DisableTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	task, _ = scheduler.GetTask("test-task")
	if task.Enabled {
		t.Error("Expected task to be disabled")
	}
}

func TestCronParser(t *testing.T) {
	parser := NewCronParser()

	schedule, err := parser.Parse("0 * * * *")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if schedule == nil {
		t.Error("Expected schedule to be parsed")
	}
}

func TestCronParserInvalid(t *testing.T) {
	parser := NewCronParser()

	_, err := parser.Parse("invalid")
	if err == nil {
		t.Error("Expected error for invalid cron expression")
	}
}
