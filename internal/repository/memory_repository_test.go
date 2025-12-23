package repository

import (
	"sync"
	"testing"

	"github.com/light-bringer/cert-tasks/internal/models"
)

func TestMemoryRepository_Create(t *testing.T) {
	repo := NewMemoryRepository()

	task := &models.Task{
		Title:       "Test Task",
		Description: "Test Description",
	}

	created, err := repo.Create(task)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == 0 {
		t.Error("Expected non-zero ID")
	}

	if created.Title != task.Title {
		t.Errorf("Title = %v, want %v", created.Title, task.Title)
	}

	if created.Description != task.Description {
		t.Errorf("Description = %v, want %v", created.Description, task.Description)
	}

	if created.Status != models.StatusTodo {
		t.Errorf("Status = %v, want %v", created.Status, models.StatusTodo)
	}

	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if created.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestMemoryRepository_GetAll(t *testing.T) {
	repo := NewMemoryRepository()

	// Create multiple tasks
	task1 := &models.Task{Title: "Task 1"}
	task2 := &models.Task{Title: "Task 2"}

	repo.Create(task1)
	repo.Create(task2)

	tasks, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("GetAll() returned %d tasks, want 2", len(tasks))
	}
}

func TestMemoryRepository_GetByID(t *testing.T) {
	repo := NewMemoryRepository()

	task := &models.Task{Title: "Test Task"}
	created, _ := repo.Create(task)

	t.Run("existing task", func(t *testing.T) {
		found, err := repo.GetByID(created.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if found.ID != created.ID {
			t.Errorf("ID = %v, want %v", found.ID, created.ID)
		}

		if found.Title != created.Title {
			t.Errorf("Title = %v, want %v", found.Title, created.Title)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		_, err := repo.GetByID(999)
		if err != ErrTaskNotFound {
			t.Errorf("Expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestMemoryRepository_Update(t *testing.T) {
	repo := NewMemoryRepository()

	task := &models.Task{Title: "Original Title"}
	created, _ := repo.Create(task)

	t.Run("existing task", func(t *testing.T) {
		updateData := &models.Task{
			Title:       "Updated Title",
			Description: "Updated Description",
			Status:      models.StatusDone,
		}

		updated, err := repo.Update(created.ID, updateData)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Title != updateData.Title {
			t.Errorf("Title = %v, want %v", updated.Title, updateData.Title)
		}

		if updated.Description != updateData.Description {
			t.Errorf("Description = %v, want %v", updated.Description, updateData.Description)
		}

		if updated.Status != updateData.Status {
			t.Errorf("Status = %v, want %v", updated.Status, updateData.Status)
		}

		if updated.UpdatedAt.Before(created.CreatedAt) {
			t.Error("UpdatedAt should not be before CreatedAt")
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		updateData := &models.Task{Title: "Test"}
		_, err := repo.Update(999, updateData)
		if err != ErrTaskNotFound {
			t.Errorf("Expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestMemoryRepository_Delete(t *testing.T) {
	repo := NewMemoryRepository()

	task := &models.Task{Title: "Test Task"}
	created, _ := repo.Create(task)

	t.Run("existing task", func(t *testing.T) {
		err := repo.Delete(created.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify task is deleted
		_, err = repo.GetByID(created.ID)
		if err != ErrTaskNotFound {
			t.Error("Task should be deleted")
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		err := repo.Delete(999)
		if err != ErrTaskNotFound {
			t.Errorf("Expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryRepository()
	var wg sync.WaitGroup

	// Create tasks concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			task := &models.Task{Title: "Concurrent Task"}
			repo.Create(task)
		}(i)
	}

	wg.Wait()

	tasks, _ := repo.GetAll()
	if len(tasks) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(tasks))
	}

	// Check all IDs are unique
	ids := make(map[int64]bool)
	for _, task := range tasks {
		if ids[task.ID] {
			t.Errorf("Duplicate ID found: %d", task.ID)
		}
		ids[task.ID] = true
	}
}
