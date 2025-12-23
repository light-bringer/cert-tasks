package repository

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/light-bringer/cert-tasks/internal/models"
)

// MemoryRepository is an in-memory implementation of TaskRepository
type MemoryRepository struct {
	mu     sync.RWMutex
	tasks  map[int64]*models.Task
	nextID int64
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tasks:  make(map[int64]*models.Task),
		nextID: 0,
	}
}

// Create creates a new task with generated ID and timestamps
func (r *MemoryRepository) Create(task *models.Task) (*models.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate new ID using atomic operation
	id := atomic.AddInt64(&r.nextID, 1)

	now := time.Now()
	newTask := &models.Task{
		ID:          id,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set default status if not provided
	if newTask.Status == "" {
		newTask.Status = models.StatusTodo
	}

	r.tasks[id] = newTask
	return newTask, nil
}

// GetAll returns all tasks
func (r *MemoryRepository) GetAll() ([]*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*models.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetByID returns a task by ID
func (r *MemoryRepository) GetByID(id int64) (*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return task, nil
}

// Update updates an existing task
func (r *MemoryRepository) Update(id int64, task *models.Task) (*models.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	// Update fields
	existing.Title = task.Title
	existing.Description = task.Description
	existing.Status = task.Status
	existing.UpdatedAt = time.Now()

	return existing, nil
}

// Delete deletes a task by ID
func (r *MemoryRepository) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}
