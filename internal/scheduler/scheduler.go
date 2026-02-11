package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

type TaskFunc func(ctx context.Context) error

type Task struct {
	ID          string
	Name        string
	Description string
	CronExpr    string
	Handler     TaskFunc
	Status      TaskStatus
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int
	ErrorCount  int
	LastError   error
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Scheduler struct {
	tasks      map[string]*Task
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	ticker     *time.Ticker
	running    bool
	taskChan   chan *Task
	resultChan chan *TaskResult
}

type TaskResult struct {
	TaskID    string
	Status    TaskStatus
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

type SchedulerConfig struct {
	TickInterval time.Duration
}

func NewScheduler(config *SchedulerConfig) *Scheduler {
	if config == nil {
		config = &SchedulerConfig{
			TickInterval: time.Second,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		tasks:      make(map[string]*Task),
		ctx:        ctx,
		cancel:     cancel,
		ticker:     time.NewTicker(config.TickInterval),
		taskChan:   make(chan *Task, 100),
		resultChan: make(chan *TaskResult, 100),
	}
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true

	go s.run()
	go s.processTasks()

	log.Println("Scheduler started")

	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	s.cancel()
	s.ticker.Stop()

	close(s.taskChan)
	close(s.resultChan)

	log.Println("Scheduler stopped")

	return nil
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if task.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	if task.CronExpr == "" {
		return fmt.Errorf("task cron expression cannot be empty")
	}

	if task.Handler == nil {
		return fmt.Errorf("task handler cannot be nil")
	}

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	now := time.Now()
	task.Status = StatusPending
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Enabled = true

	nextRun, err := s.calculateNextRun(task.CronExpr, now)
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}
	task.NextRun = nextRun

	s.tasks[task.ID] = task

	log.Printf("Task added: %s (ID: %s, Next run: %s)", task.Name, task.ID, task.NextRun)

	return nil
}

func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[taskID]; !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	delete(s.tasks, taskID)

	log.Printf("Task removed: %s", taskID)

	return nil
}

func (s *Scheduler) GetTask(taskID string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	return task, exists
}

func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

func (s *Scheduler) EnableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	log.Printf("Task enabled: %s", taskID)

	return nil
}

func (s *Scheduler) DisableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	log.Printf("Task disabled: %s", taskID)

	return nil
}

func (s *Scheduler) TriggerTask(taskID string) error {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	if !task.Enabled {
		return fmt.Errorf("task %s is disabled", taskID)
	}

	select {
	case s.taskChan <- task:
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

func (s *Scheduler) GetResults() <-chan *TaskResult {
	return s.resultChan
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.checkAndScheduleTasks()
		}
	}
}

func (s *Scheduler) checkAndScheduleTasks() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		if now.After(task.NextRun) || now.Equal(task.NextRun) {
			select {
			case s.taskChan <- task:
				task.LastRun = now
				task.NextRun, _ = s.calculateNextRun(task.CronExpr, now)
			default:
				log.Printf("Task queue is full, skipping task: %s", task.ID)
			}
		}
	}
}

func (s *Scheduler) processTasks() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case task, ok := <-s.taskChan:
			if !ok {
				return
			}

			go s.executeTask(task)
		}
	}
}

func (s *Scheduler) executeTask(task *Task) {
	s.mu.Lock()
	task.Status = StatusRunning
	task.UpdatedAt = time.Now()
	s.mu.Unlock()

	startTime := time.Now()

	log.Printf("Task started: %s (ID: %s)", task.Name, task.ID)

	err := task.Handler(s.ctx)

	duration := time.Since(startTime)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		task.Status = StatusFailed
		task.ErrorCount++
		task.LastError = err
		log.Printf("Task failed: %s (ID: %s, Error: %v)", task.Name, task.ID, err)
	} else {
		task.Status = StatusCompleted
		task.RunCount++
		log.Printf("Task completed: %s (ID: %s, Duration: %v)", task.Name, task.ID, duration)
	}

	task.UpdatedAt = time.Now()

	result := &TaskResult{
		TaskID:    task.ID,
		Status:    task.Status,
		Error:     err,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	select {
	case s.resultChan <- result:
	default:
		log.Printf("Result queue is full, dropping result for task: %s", task.ID)
	}
}

func (s *Scheduler) calculateNextRun(cronExpr string, from time.Time) (time.Time, error) {
	parser := NewCronParser()
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	return schedule.Next(from), nil
}

func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalTasks := len(s.tasks)
	enabledTasks := 0
	runningTasks := 0
	totalRuns := 0
	totalErrors := 0

	for _, task := range s.tasks {
		if task.Enabled {
			enabledTasks++
		}
		if task.Status == StatusRunning {
			runningTasks++
		}
		totalRuns += task.RunCount
		totalErrors += task.ErrorCount
	}

	return map[string]interface{}{
		"total_tasks":   totalTasks,
		"enabled_tasks": enabledTasks,
		"running_tasks": runningTasks,
		"total_runs":    totalRuns,
		"total_errors":  totalErrors,
		"is_running":    s.running,
	}
}
