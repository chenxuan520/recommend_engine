package task

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status represents the status of an asynchronous task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Task represents an asynchronous task.
type Task struct {
	ID        string      `json:"id"`
	Status    Status      `json:"status"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// Manager manages asynchronous tasks using an in-memory store.
type Manager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewManager creates a new task manager.
func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*Task),
	}
}

// NewTask creates a new task, stores it, and returns it.
func (m *Manager) NewTask() *Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	task := &Task{
		ID:        uuid.New().String(),
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	m.tasks[task.ID] = task
	return task
}

// GetTask retrieves a task by its ID.
func (m *Manager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID '%s' not found", id)
	}
	return task, nil
}

// UpdateStatus updates the status of a task.
func (m *Manager) UpdateStatus(id string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID '%s' not found", id)
	}
	task.Status = status
	return nil
}

// SetResult sets the successful result of a task and marks it as completed.
func (m *Manager) SetResult(id string, result interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID '%s' not found", id)
	}
	task.Result = result
	task.Status = StatusCompleted
	task.Error = ""
	return nil
}

// SetError sets the error message for a failed task and marks it as failed.
func (m *Manager) SetError(id string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID '%s' not found", id)
	}
	task.Error = err.Error()
	task.Status = StatusFailed
	return nil
}
