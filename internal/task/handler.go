package task

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/denis-sofonov/go-task-api/internal/auth"
	"github.com/denis-sofonov/go-task-api/internal/db"
)

type Handler struct {
	q      *db.Queries
	log    *slog.Logger
	events EventPublisher
}

func NewHandler(q *db.Queries, log *slog.Logger, events EventPublisher) *Handler {
	return &Handler{q: q, log: log, events: events}
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// updateRequest uses pointers so a missing field (nil) means "leave unchanged".
type updateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Done        *bool   `json:"done"`
}

type taskResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResponse(t db.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Done:        t.Done,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	tasks, err := h.q.ListTasksByUser(r.Context(), userID)
	if err != nil {
		h.log.Error("list tasks", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, toResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	var req createRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	t, err := h.q.CreateTask(r.Context(), db.CreateTaskParams{
		UserID:      userID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		h.log.Error("create task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.publishCreated(r.Context(), t)
	writeJSON(w, http.StatusCreated, toResponse(t))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := taskID(w, r)
	if !ok {
		return
	}

	t, err := h.q.GetTask(r.Context(), db.GetTaskParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.log.Error("get task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := taskID(w, r)
	if !ok {
		return
	}

	var req updateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeError(w, http.StatusUnprocessableEntity, "title must not be empty")
			return
		}
		req.Title = &title
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		req.Description = &desc
	}

	t, err := h.q.UpdateTask(r.Context(), db.UpdateTaskParams{
		Title:       req.Title,
		Description: req.Description,
		Done:        req.Done,
		ID:          id,
		UserID:      userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.log.Error("update task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, ok := taskID(w, r)
	if !ok {
		return
	}

	n, err := h.q.DeleteTask(r.Context(), db.DeleteTaskParams{ID: id, UserID: userID})
	if err != nil {
		h.log.Error("delete task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// userID reads the authenticated user set by auth.RequireAuth.
func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		h.log.Error("task: no user in context")
		writeError(w, http.StatusInternalServerError, "internal error")
	}
	return id, ok
}

// taskID parses {id} from the path; a malformed id is treated as not found.
func taskID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
