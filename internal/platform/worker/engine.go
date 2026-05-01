// Package worker implements a transactional outbox worker backed by
// internal.outbox_events. It polls for pending events using SELECT FOR UPDATE
// SKIP LOCKED, dispatches each to a registered Handler, and retries failed
// events with exponential backoff (min(2^n, 1800) seconds) up to MaxRetries.
// Exhausted events are dead-lettered (status = 'failed') and a pg_notify is
// emitted on the 'outbox_dead_lettered' channel. A 5-minute visibility timeout
// on claimed rows ensures a crashed worker does not permanently strand events.
//
// Enqueue events from route handlers via Enqueue or EnqueueAt; register
// handlers with Register before calling Start.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config tunes the worker engine. Zero values apply defaults.
type Config struct {
	PollInterval time.Duration // default: 5s
	BatchSize    int           // default: 10
	MaxRetries   int           // default: 3
}

// Engine polls internal.outbox_events, dispatches registered handlers, and manages
// retries with exponential backoff. Use New to create, Register to add handlers,
// Start to begin polling, and Stop for graceful shutdown.
type Engine struct {
	db           *pgxpool.Pool
	handlers     map[string]Handler
	mu           sync.RWMutex
	pollInterval time.Duration
	batchSize    int
	maxRetries   int
	logger       *slog.Logger
	wg           sync.WaitGroup
	cancel       context.CancelFunc
	ctx          context.Context
}

// New creates a stopped Engine. Call Register, then Start.
func New(db *pgxpool.Pool, logger *slog.Logger, cfg Config) *Engine {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		db:           db,
		handlers:     make(map[string]Handler),
		pollInterval: cfg.PollInterval,
		batchSize:    cfg.BatchSize,
		maxRetries:   cfg.MaxRetries,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Register maps an event_type to a handler. Must be called before Start.
// Registering the same event_type twice overwrites the previous handler.
func (e *Engine) Register(eventType string, h Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[eventType] = h
}

// Start begins polling in a background goroutine.
func (e *Engine) Start() {
	e.wg.Add(1)
	go e.run()
	e.logger.Info("outbox worker started",
		"pollInterval", e.pollInterval,
		"batchSize", e.batchSize,
		"maxRetries", e.maxRetries,
	)
}

// Stop cancels the poll loop and waits for all in-flight handlers to finish.
func (e *Engine) Stop() {
	e.cancel()
	e.wg.Wait()
}

// run is the main poll loop.
func (e *Engine) run() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			if err := e.poll(); err != nil {
				e.logger.Error("outbox worker poll error", "error", err)
			}
		}
	}
}

// claimedRow holds the data returned from the claim CTE.
type claimedRow struct {
	id         uuid.UUID
	eventType  string
	payload    json.RawMessage
	retryCount int
}

// poll claims a batch of pending/timed-out rows and dispatches each in its own goroutine.
func (e *Engine) poll() error {
	rows, err := e.claimBatch()
	if err != nil {
		return fmt.Errorf("claim batch: %w", err)
	}
	for _, row := range rows {
		e.wg.Add(1)
		go func(r claimedRow) {
			defer e.wg.Done()
			e.process(r)
		}(row)
	}
	return nil
}

// claimBatch atomically claims up to batchSize pending (or timed-out processing)
// rows, setting their status to 'processing' and forwarding process_at by 5
// minutes as a visibility timeout. Uses SELECT … FOR UPDATE SKIP LOCKED so
// concurrent workers (future) or restarted instances never double-process.
func (e *Engine) claimBatch() ([]claimedRow, error) {
	const sql = `
WITH claimed AS (
    SELECT id FROM internal.outbox_events
    WHERE (status = 'pending' OR status = 'processing')
      AND process_at <= NOW()
    ORDER BY process_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE internal.outbox_events
SET
    status     = 'processing',
    process_at = NOW() + INTERVAL '5 minutes',
    updated_at = NOW()
FROM claimed
WHERE internal.outbox_events.id = claimed.id
RETURNING
    internal.outbox_events.id,
    internal.outbox_events.event_type,
    internal.outbox_events.payload,
    internal.outbox_events.retry_count
`
	pgRows, err := e.db.Query(e.ctx, sql, e.batchSize)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var rows []claimedRow
	for pgRows.Next() {
		var r claimedRow
		if err := pgRows.Scan(&r.id, &r.eventType, &r.payload, &r.retryCount); err != nil {
			return nil, fmt.Errorf("scan claimed row: %w", err)
		}
		rows = append(rows, r)
	}
	return rows, pgRows.Err()
}

// process runs the registered handler for a claimed row and updates its status.
func (e *Engine) process(r claimedRow) {
	e.mu.RLock()
	h, ok := e.handlers[r.eventType]
	e.mu.RUnlock()

	// DB status updates must complete even after Stop() cancels e.ctx.
	// Using a detached context prevents rows from being stranded in 'processing'
	// and avoids double-processing after the 5-minute visibility timeout.
	dbCtx := context.Background()

	if !ok {
		e.logger.Warn("outbox worker: no handler registered", "event_type", r.eventType, "id", r.id)
		e.markFailed(dbCtx, r, "no handler registered for event_type: "+r.eventType)
		return
	}

	err := h(e.ctx, r.payload)
	if err == nil {
		e.markCompleted(dbCtx, r.id)
		return
	}

	e.logger.Warn("outbox worker handler error",
		"event_type", r.eventType,
		"id", r.id,
		"retry_count", r.retryCount,
		"error", err,
	)

	if r.retryCount+1 >= e.maxRetries {
		e.markFailed(dbCtx, r, err.Error())
		return
	}
	e.scheduleRetry(dbCtx, r, err.Error())
}

func (e *Engine) markCompleted(ctx context.Context, id uuid.UUID) {
	const sql = `
UPDATE internal.outbox_events
SET status = 'completed', updated_at = NOW()
WHERE id = $1
`
	if _, err := e.db.Exec(ctx, sql, id); err != nil {
		e.logger.Error("outbox worker: mark completed failed", "id", id, "error", err)
	}
}

func (e *Engine) scheduleRetry(ctx context.Context, r claimedRow, errMsg string) {
	const sql = `
UPDATE internal.outbox_events
SET
    status      = 'pending',
    retry_count = retry_count + 1,
    last_error  = $2,
    process_at  = NOW() + ($3 * INTERVAL '1 second'),
    updated_at  = NOW()
WHERE id = $1
`
	backoff := backoffSeconds(r.retryCount)
	if _, err := e.db.Exec(ctx, sql, r.id, errMsg, backoff); err != nil {
		e.logger.Error("outbox worker: schedule retry failed", "id", r.id, "error", err)
	}
}

// markFailed dead-letters the event and emits a pg_notify on 'outbox_dead_lettered'
// so the event listener can surface recurring failures in the master dashboard.
func (e *Engine) markFailed(ctx context.Context, r claimedRow, errMsg string) {
	const sql = `
UPDATE internal.outbox_events
SET
    status      = 'failed',
    retry_count = retry_count + 1,
    last_error  = $2,
    updated_at  = NOW()
WHERE id = $1
`
	if _, err := e.db.Exec(ctx, sql, r.id, errMsg); err != nil {
		e.logger.Error("outbox worker: mark failed failed", "id", r.id, "error", err)
		return
	}

	type deadLetterPayload struct {
		ID        string `json:"id"`
		EventType string `json:"event_type"`
		LastError string `json:"last_error"`
	}
	notifyPayload, _ := json.Marshal(deadLetterPayload{
		ID:        r.id.String(),
		EventType: r.eventType,
		LastError: errMsg,
	})

	if _, err := e.db.Exec(ctx, `SELECT pg_notify('outbox_dead_lettered', $1)`, string(notifyPayload)); err != nil {
		e.logger.Error("outbox worker: pg_notify dead lettered failed", "id", r.id, "error", err)
	}

	e.logger.Error("outbox worker: event dead-lettered",
		"id", r.id,
		"event_type", r.eventType,
		"retry_count", r.retryCount+1,
		"error", errMsg,
	)
}

// backoffSeconds returns min(2^retryCount, 1800) seconds.
func backoffSeconds(retryCount int) int {
	const max = 1800
	if retryCount >= 11 {
		return max
	}
	return 1 << retryCount
}
