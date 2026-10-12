package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/denis-sofonov/go-task-api/internal/db"
)

type Handler struct {
	q   *db.Queries
	log *slog.Logger
}

func NewHandler(q *db.Queries, log *slog.Logger) *Handler {
	return &Handler{q: q, log: log}
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// userResponse is the public view of a user; it never includes the password hash.
type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req registerRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)

	// ParseAddress also accepts "Name <addr>", so require a bare address.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		writeError(w, http.StatusUnprocessableEntity, "invalid email")
		return
	}
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	// bcrypt rejects passwords longer than 72 bytes.
	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeError(w, http.StatusUnprocessableEntity, "password must be 8-72 bytes")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("register: hash password", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		h.log.Error("register: create user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
