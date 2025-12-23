package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/light-bringer/cert-tasks/internal/models"
	"github.com/light-bringer/cert-tasks/internal/repository"
)

func TestTaskHandler_CreateTask(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid task",
			body:       `{"title":"Test Task","description":"Test Description"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid task without description",
			body:       `{"title":"Test Task"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing title",
			body:       `{"description":"Test Description"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "title is required",
		},
		{
			name:       "empty title",
			body:       `{"title":"  ","description":"Test Description"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "title is required",
		},
		{
			name:       "invalid JSON",
			body:       `{"title":}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemoryRepository()
			handler := NewTaskHandler(repo)

			req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler.CreateTask(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", rec.Code, tt.wantStatus)
			}

			if tt.wantError != "" {
				var errResp ErrorResponse
				json.NewDecoder(rec.Body).Decode(&errResp)
				if errResp.Error == "" {
					t.Error("expected error response")
				}
			}

			if tt.wantStatus == http.StatusCreated {
				var task models.Task
				json.NewDecoder(rec.Body).Decode(&task)
				if task.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if task.Status != models.StatusTodo {
					t.Errorf("status = %v, want %v", task.Status, models.StatusTodo)
				}
			}
		})
	}
}

func TestTaskHandler_ListTasks(t *testing.T) {
	repo := repository.NewMemoryRepository()
	handler := NewTaskHandler(repo)

	// Create some tasks
	repo.Create(&models.Task{Title: "Task 1"})
	repo.Create(&models.Task{Title: "Task 2"})

	req := httptest.NewRequest("GET", "/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", rec.Code, http.StatusOK)
	}

	var tasks []*models.Task
	json.NewDecoder(rec.Body).Decode(&tasks)

	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestTaskHandler_GetTask(t *testing.T) {
	repo := repository.NewMemoryRepository()
	handler := NewTaskHandler(repo)

	created, _ := repo.Create(&models.Task{Title: "Test Task"})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing task",
			id:         "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent task",
			id:         "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/tasks/"+tt.id, nil)
			rec := httptest.NewRecorder()

			// Create chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetTask(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var task models.Task
				json.NewDecoder(rec.Body).Decode(&task)
				if task.ID != created.ID {
					t.Errorf("ID = %v, want %v", task.ID, created.ID)
				}
			}
		})
	}
}

func TestTaskHandler_UpdateTask(t *testing.T) {
	repo := repository.NewMemoryRepository()
	handler := NewTaskHandler(repo)

	created, _ := repo.Create(&models.Task{Title: "Original Title"})

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid update",
			id:         "1",
			body:       `{"title":"Updated Title","description":"Updated Desc","status":"done"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid status",
			id:         "1",
			body:       `{"title":"Updated Title","status":"invalid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing title",
			id:         "1",
			body:       `{"description":"Updated Desc","status":"done"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existent task",
			id:         "999",
			body:       `{"title":"Updated Title","status":"done"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			body:       `{"title":"Updated Title","status":"done"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/tasks/"+tt.id, bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			// Create chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.UpdateTask(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var task models.Task
				json.NewDecoder(rec.Body).Decode(&task)
				if task.ID != created.ID {
					t.Errorf("ID = %v, want %v", task.ID, created.ID)
				}
			}
		})
	}
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	repo := repository.NewMemoryRepository()
	handler := NewTaskHandler(repo)

	repo.Create(&models.Task{Title: "Test Task"})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing task",
			id:         "1",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "non-existent task",
			id:         "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/tasks/"+tt.id, nil)
			rec := httptest.NewRecorder()

			// Create chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.DeleteTask(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", rec.Code, tt.wantStatus)
			}
		})
	}
}
