package models

import (
	"errors"
	"strings"
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusTodo TaskStatus = "todo"
	StatusDone TaskStatus = "done"
)

// Task represents a task entity
type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Validate validates the create task request
func (r *CreateTaskRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required and cannot be empty")
	}
	return nil
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
}

// Validate validates the update task request
func (r *UpdateTaskRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required and cannot be empty")
	}
	if r.Status != StatusTodo && r.Status != StatusDone {
		return errors.New("status must be either 'todo' or 'done'")
	}
	return nil
}
