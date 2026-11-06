package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/denis-sofonov/go-task-api/internal/db"
	"github.com/google/uuid"
)

const eventTaskCreated = "task.created"

type EventPublisher interface {
	Publish(ctx context.Context, key string, value []byte) error
}

type taskEvent struct {
	Type       string    `json:"type"`
	TaskID     uuid.UUID `json:"task_id"`
	UserID     uuid.UUID `json:"user_id"`
	Title      string    `json:"title"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (h *Handler) publishCreated(ctx context.Context, t db.Task) {
	payload, err := json.Marshal(taskEvent{
		Type:       eventTaskCreated,
		TaskID:     t.ID,
		UserID:     t.UserID,
		Title:      t.Title,
		OccurredAt: t.CreatedAt,
	})
	if err != nil {
		h.log.Error("marshal task event", "task_id", t.ID, "err", err)
		return
	}

	if err := h.events.Publish(ctx, t.ID.String(), payload); err != nil {
		h.log.Error("publish task event", "task_id", t.ID, "err", err)
	}
}
