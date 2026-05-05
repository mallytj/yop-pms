package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// outboxInserter is the narrow interface Enqueue/EnqueueAt need from the store.
type outboxInserter interface {
	CreateOutboxEvent(ctx context.Context, arg *store.CreateOutboxEventParams) (uuid.UUID, error)
}

// Enqueue inserts an outbox event for immediate processing.
func Enqueue(ctx context.Context, q outboxInserter, eventType string, payload any) error {
	return EnqueueAt(ctx, q, eventType, payload, time.Now())
}

// EnqueueAt inserts an outbox event scheduled for processing at processAt.
func EnqueueAt(ctx context.Context, q outboxInserter, eventType string, payload any, processAt time.Time) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("worker: marshal payload: %w", err)
	}
	if _, err = q.CreateOutboxEvent(ctx, &store.CreateOutboxEventParams{
		EventType: eventType,
		Payload:   b,
		ProcessAt: pgtype.Timestamptz{Time: processAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("worker: create outbox event: %w", err)
	}
	return nil
}
