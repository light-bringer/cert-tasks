package repository

import (
	"errors"

	"github.com/light-bringer/cert-tasks/internal/models"
)

// ErrTaskNotFound is returned when a task is not found
var ErrTaskNotFound = errors.New("task not found")

// TaskRepository defines the interface for task storage operations
type TaskRepository interface {
	// Create creates a new task and returns it with generated ID
	Create(task *models.Task) (*models.Task, error)

	// GetAll returns all tasks
	GetAll() ([]*models.Task, error)

	// GetByID returns a task by ID or ErrTaskNotFound if not found
	GetByID(id int64) (*models.Task, error)

	// Update updates an existing task and returns the updated task
	Update(id int64, task *models.Task) (*models.Task, error)

	// Delete deletes a task by ID
	Delete(id int64) error
}
