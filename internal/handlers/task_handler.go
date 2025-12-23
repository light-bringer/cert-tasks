package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/light-bringer/cert-tasks/internal/models"
	"github.com/light-bringer/cert-tasks/internal/repository"
)

// TaskHandler handles HTTP requests for tasks
type TaskHandler struct {
	repo repository.TaskRepository
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(repo repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateTask handles POST /tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	task := &models.Task{
		Title:       req.Title,
		Description: req.Description,
	}

	created, err := h.repo.Create(task)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	respondWithJSON(w, http.StatusCreated, created)
}

// ListTasks handles GET /tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.repo.GetAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to retrieve tasks")
		return
	}

	respondWithJSON(w, http.StatusOK, tasks)
}

// GetTask handles GET /tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			respondWithError(w, http.StatusNotFound, "task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to retrieve task")
		return
	}

	respondWithJSON(w, http.StatusOK, task)
}

// UpdateTask handles PUT /tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var req models.UpdateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	task := &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	updated, err := h.repo.Update(id, task)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			respondWithError(w, http.StatusNotFound, "task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	respondWithJSON(w, http.StatusOK, updated)
}

// DeleteTask handles DELETE /tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	err = h.repo.Delete(id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			respondWithError(w, http.StatusNotFound, "task not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondWithJSON writes a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

// respondWithError writes an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}
