package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TaskManager struct {
	scheduler *Scheduler
	tasksFile string
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

type TaskConfig struct {
	ID          string
	Name        string
	Description string
	CronExpr    string
	Enabled     bool
}

type TaskManagerConfig struct {
	TasksFile string
}

func NewTaskManager(scheduler *Scheduler, config *TaskManagerConfig) *TaskManager {
	if config == nil {
		config = &TaskManagerConfig{
			TasksFile: "./data/tasks.json",
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TaskManager{
		scheduler: scheduler,
		tasksFile: config.TasksFile,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (m *TaskManager) Start() error {
	if err := m.loadTasks(); err != nil {
		log.Printf("Warning: failed to load tasks: %v", err)
	}

	go m.watchResults()

	return nil
}

func (m *TaskManager) Stop() error {
	m.cancel()

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err)
	}

	return nil
}

func (m *TaskManager) AddTask(config *TaskConfig, handler TaskFunc) error {
	task := &Task{
		ID:          config.ID,
		Name:        config.Name,
		Description: config.Description,
		CronExpr:    config.CronExpr,
		Handler:     handler,
		Enabled:     config.Enabled,
	}

	if err := m.scheduler.AddTask(task); err != nil {
		return err
	}

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err)
	}

	return nil
}

func (m *TaskManager) RemoveTask(taskID string) error {
	if err := m.scheduler.RemoveTask(taskID); err != nil {
		return err
	}

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err)
	}

	return nil
}

func (m *TaskManager) GetTask(taskID string) (*Task, bool) {
	return m.scheduler.GetTask(taskID)
}

func (m *TaskManager) ListTasks() []*Task {
	return m.scheduler.ListTasks()
}

func (m *TaskManager) EnableTask(taskID string) error {
	if err := m.scheduler.EnableTask(taskID); err != nil {
		return err
	}

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err)
	}

	return nil
}

func (m *TaskManager) DisableTask(taskID string) error {
	if err := m.scheduler.DisableTask(taskID); err != nil {
		return err
	}

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks: %v", err)
	}

	return nil
}

func (m *TaskManager) TriggerTask(taskID string) error {
	return m.scheduler.TriggerTask(taskID)
}

func (m *TaskManager) GetStats() map[string]interface{} {
	return m.scheduler.GetStats()
}

func (m *TaskManager) GetScheduler() *Scheduler {
	return m.scheduler
}

func (m *TaskManager) loadTasks() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.tasksFile); os.IsNotExist(err) {
		log.Printf("Tasks file does not exist: %s", m.tasksFile)
		return nil
	}

	data, err := os.ReadFile(m.tasksFile)
	if err != nil {
		return fmt.Errorf("failed to read tasks file: %w", err)
	}

	var configs []TaskConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	for _, config := range configs {
		task := &Task{
			ID:          config.ID,
			Name:        config.Name,
			Description: config.Description,
			CronExpr:    config.CronExpr,
			Enabled:     config.Enabled,
			Status:      StatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := m.scheduler.AddTask(task); err != nil {
			log.Printf("Warning: failed to add task %s: %v", config.ID, err)
			continue
		}

		log.Printf("Task loaded: %s (ID: %s)", task.Name, task.ID)
	}

	log.Printf("Loaded %d tasks from file", len(configs))

	return nil
}

func (m *TaskManager) saveTasks() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks := m.scheduler.ListTasks()
	configs := make([]TaskConfig, 0, len(tasks))

	for _, task := range tasks {
		configs = append(configs, TaskConfig{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			CronExpr:    task.CronExpr,
			Enabled:     task.Enabled,
		})
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	dir := filepath.Dir(m.tasksFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(m.tasksFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

func (m *TaskManager) watchResults() {
	resultChan := m.scheduler.GetResults()

	for {
		select {
		case <-m.ctx.Done():
			return
		case result, ok := <-resultChan:
			if !ok {
				return
			}

			m.handleResult(result)
		}
	}
}

func (m *TaskManager) handleResult(result *TaskResult) {
	task, exists := m.scheduler.GetTask(result.TaskID)
	if !exists {
		log.Printf("Warning: task %s not found for result", result.TaskID)
		return
	}

	log.Printf("Task result: %s - Status: %s, Duration: %v", task.Name, result.Status, result.Duration)

	if result.Error != nil {
		log.Printf("Task error: %s - %v", task.Name, result.Error)
	}

	if err := m.saveTasks(); err != nil {
		log.Printf("Warning: failed to save tasks after result: %v", err)
	}
}

func (m *TaskManager) ExportTasks() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := m.scheduler.ListTasks()
	configs := make([]TaskConfig, 0, len(tasks))

	for _, task := range tasks {
		configs = append(configs, TaskConfig{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			CronExpr:    task.CronExpr,
			Enabled:     task.Enabled,
		})
	}

	return json.MarshalIndent(configs, "", "  ")
}

func (m *TaskManager) ImportTasks(data []byte) error {
	var configs []TaskConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	for _, config := range configs {
		task, exists := m.scheduler.GetTask(config.ID)
		if exists {
			task.Name = config.Name
			task.Description = config.Description
			task.CronExpr = config.CronExpr
			task.Enabled = config.Enabled
			task.UpdatedAt = time.Now()

			nextRun, err := m.scheduler.calculateNextRun(task.CronExpr, time.Now())
			if err != nil {
				log.Printf("Warning: failed to calculate next run for task %s: %v", config.ID, err)
				continue
			}
			task.NextRun = nextRun

			log.Printf("Task updated: %s (ID: %s)", task.Name, task.ID)
		}
	}

	if err := m.saveTasks(); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	return nil
}

func (m *TaskManager) GetTaskHistory(taskID string, limit int) ([]*TaskResult, error) {
	return nil, fmt.Errorf("task history not implemented")
}

func (m *TaskManager) ClearTaskHistory(taskID string) error {
	return fmt.Errorf("task history not implemented")
}

func (m *TaskManager) ValidateCronExpression(expr string) error {
	parser := NewCronParser()
	_, err := parser.Parse(expr)
	return err
}

func (m *TaskManager) GetNextRunTime(taskID string) (time.Time, error) {
	task, exists := m.scheduler.GetTask(taskID)
	if !exists {
		return time.Time{}, fmt.Errorf("task %s not found", taskID)
	}

	return task.NextRun, nil
}

func (m *TaskManager) GetAllNextRunTimes() (map[string]time.Time, error) {
	tasks := m.scheduler.ListTasks()
	result := make(map[string]time.Time)

	for _, task := range tasks {
		result[task.ID] = task.NextRun
	}

	return result, nil
}

func (m *TaskManager) PauseScheduler() error {
	if err := m.scheduler.Stop(); err != nil {
		return err
	}
	log.Println("Scheduler paused")
	return nil
}

func (m *TaskManager) ResumeScheduler() error {
	if err := m.scheduler.Start(); err != nil {
		return err
	}
	log.Println("Scheduler resumed")
	return nil
}

func (m *TaskManager) IsSchedulerRunning() bool {
	return m.scheduler.IsRunning()
}
