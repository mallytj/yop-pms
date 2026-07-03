package booking

// Core Requirements: R-RES-WORKER-001, R-RES-WORKER-002, R-RES-WORKER-003, R-RES-WORKER-004, ADR-016
// Workers run background sweeps (hold expiry, no-show, overstay, archival).
// Single file in booking package is appropriate for this scale.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexxcode1/yop-pms/internal/store"
)

// Workers runs background sweep goroutines for reservation lifecycle management.
// Each sweep runs on a ticker: hold expiry every 30s, no-show every 5min,
// overstay every 5min, archival daily at 02:00.
type Workers struct {
	pool *pgxpool.Pool
	q    *store.Queries
	log  *slog.Logger
}

// NewWorkers creates a new Workers instance.
func NewWorkers(pool *pgxpool.Pool, q *store.Queries, log *slog.Logger) *Workers {
	return &Workers{pool: pool, q: q, log: log}
}

// HoldExpirySweep runs every 30s. Finds expired holds and cancels them
// (status=cancelled, items=cancelled, ledger deleted). No payment auth
// voids in this PR — finance PR (ADR-019) owns that.
func (w *Workers) HoldExpirySweep(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	w.log.Info("hold expiry sweep started (interval: 30s)")
	for {
		select {
		case <-ctx.Done():
			w.log.Info("hold expiry sweep stopped")
			return
		case <-ticker.C:
			w.expireHolds(ctx)
		}
	}
}

func (w *Workers) expireHolds(ctx context.Context) {
	holds, err := w.q.FindExpiredHolds(ctx)
	if err != nil {
		w.log.Error("find expired holds", "error", err)
		return
	}
	if len(holds) == 0 {
		return
	}

	w.log.Info("expiring holds", "count", len(holds))
	for _, h := range holds {
		if err := w.cancelHoldTx(ctx, h); err != nil {
			w.log.Error("cancel hold", "id", h.ID, "error", err)
		}
	}
}

func (w *Workers) cancelHoldTx(ctx context.Context, res store.OperationsReservation) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := w.q.WithTx(tx)

	// Set RLS context so M18 audit trigger and RLS policies work.
	if err := qtx.SetCurrentPropertyID(ctx, res.PropertyID.String()); err != nil {
		return fmt.Errorf("set rls: %w", err)
	}

	// Cancel all non-terminal items
	if err := qtx.CancelReservationItems(ctx, &store.CancelReservationItemsParams{
		ReservationID: res.ID,
		PropertyID:    res.PropertyID,
	}); err != nil {
		return fmt.Errorf("cancel items: %w", err)
	}

	// Delete future ledger rows
	if err := qtx.DeleteLedgerForReservation(ctx, &store.DeleteLedgerForReservationParams{
		ReservationID: uuid.NullUUID{UUID: res.ID, Valid: true},
		PropertyID:    res.PropertyID,
	}); err != nil {
		return fmt.Errorf("delete ledger: %w", err)
	}

	// Update reservation status to cancelled
	rows, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
		ID:      res.ID,
		Version: res.Version,
		Status:  store.OperationsReservationStatusCancelled,
	})
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if rows == 0 {
		w.log.Warn("hold expiry version mismatch (concurrent update)", "id", res.ID)
		return nil // not an error — another worker or user got there first
	}

	// Notify reservation_changes with explicit action for frontend toast routing.
	// DB trigger (trg_reservations_notify) also fires on status change for general SSE
	// broadcast, but this carries a typed action payload the frontend can match on.
	payload, _ := json.Marshal(struct {
		Action string `json:"action"`
		ID     string `json:"reservation_id"`
	}{Action: "hold_expired", ID: res.ID.String()})
	if err := qtx.NotifyChannel(ctx, &store.NotifyChannelParams{
		Channel: "reservation_changes",
		Payload: string(payload),
	}); err != nil {
		w.log.Warn("notify after hold expiry", "id", res.ID, "error", err)
	}

	return tx.Commit(ctx)
}

// NoShowReminder runs at the configured interval. Finds overdue check-ins
// (booked items past arrival + grace period) and sends a staff_alert NOTIFY.
// No status change — staff acts manually (§4.4).
func (w *Workers) NoShowReminder(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	w.log.Info("no-show reminder started (interval: 5min)")
	for {
		select {
		case <-ctx.Done():
			w.log.Info("no-show reminder stopped")
			return
		case <-ticker.C:
			w.checkNoShows(ctx)
		}
	}
}

func (w *Workers) checkNoShows(ctx context.Context) {
	items, err := w.q.FindOverdueCheckins(ctx)
	if err != nil {
		w.log.Error("find overdue checkins", "error", err)
		return
	}
	if len(items) == 0 {
		return
	}

	itemIDs := make([]string, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID.String()
	}

	payload, _ := json.Marshal(map[string]any{
		"type":     "no_show_overdue",
		"item_ids": itemIDs,
	})

	if err := w.q.NotifyChannel(ctx, &store.NotifyChannelParams{
		Channel: "staff_alerts",
		Payload: string(payload),
	}); err != nil {
		w.log.Error("notify staff_alerts for no-show", "error", err)
	}
}

// OverstaySweep runs every 5min. Finds checked-in items past departure +
// grace period and transitions them to overstay status.
func (w *Workers) OverstaySweep(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	w.log.Info("overstay sweep started (interval: 5min)")
	for {
		select {
		case <-ctx.Done():
			w.log.Info("overstay sweep stopped")
			return
		case <-ticker.C:
			w.processOverstays(ctx)
		}
	}
}

func (w *Workers) processOverstays(ctx context.Context) {
	items, err := w.q.FindOverstays(ctx)
	if err != nil {
		w.log.Error("find overstays", "error", err)
		return
	}
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		if err := w.markOverstayTx(ctx, item); err != nil {
			w.log.Error("mark overstay", "item_id", item.ID, "error", err)
		}
	}
}

func (w *Workers) markOverstayTx(ctx context.Context, item store.OperationsReservationItem) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := w.q.WithTx(tx)

	if err := qtx.SetCurrentPropertyID(ctx, item.PropertyID.String()); err != nil {
		return fmt.Errorf("set rls: %w", err)
	}

	// Re-check status inside tx to avoid races
	var currentStatus string
	if err := tx.QueryRow(
		ctx,
		`SELECT status::text FROM operations.reservation_items WHERE id = $1 FOR UPDATE`,
		item.ID,
	).Scan(&currentStatus); err != nil {
		if err == pgx.ErrNoRows {
			return nil // deleted concurrently
		}
		return fmt.Errorf("re-check item: %w", err)
	}
	if currentStatus != "checked_in" {
		return nil // status changed since FindOverstays returned — skip
	}

	_, err = qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
		ID:               item.ID,
		Version:          item.Version,
		Status:           store.OperationsReservationItemStatusOverstay,
		AssignedRoomID:   item.AssignedRoomID,
		BookedRoomTypeID: uuid.NullUUID{UUID: item.BookedRoomTypeID, Valid: true},
		StayPeriod:       item.StayPeriod,
		RatePlanID:       item.RatePlanID,
		AdultsCount:      item.AdultsCount,
		ChildrenCount:    item.ChildrenCount,
	})
	if err != nil {
		return fmt.Errorf("update item to overstay: %w", err)
	}

	// Notify SSE
	payload, _ := json.Marshal(struct {
		Action string `json:"action"`
		ItemID string `json:"item_id"`
	}{Action: "overstay", ItemID: item.ID.String()})
	if err := qtx.NotifyChannel(ctx, &store.NotifyChannelParams{
		Channel: "reservation_changes",
		Payload: string(payload),
	}); err != nil {
		w.log.Warn("notify after overstay", "item_id", item.ID, "error", err)
	}

	return tx.Commit(ctx)
}

// ArchivalSweep runs daily at 02:00 (first tick after start). Archives
// terminal reservations (checked_out/cancelled) past the property's
// archive threshold (reservation_archive_after_days).
func (w *Workers) ArchivalSweep(ctx context.Context) {
	// First tick: calculate delay until next 02:00.
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	delay := time.Until(next)

	w.log.Info("archival sweep scheduled", "first_run", next.Format(time.RFC3339), "interval", "24h")

	// Wait until first scheduled run
	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	w.log.Info("archival sweep started (interval: 24h)")
	for {
		w.archiveTerminal(ctx)

		select {
		case <-ctx.Done():
			w.log.Info("archival sweep stopped")
			return
		case <-ticker.C:
		}
	}
}

func (w *Workers) archiveTerminal(ctx context.Context) {
	archivable, err := w.q.FindArchivableReservations(ctx)
	if err != nil {
		w.log.Error("find archivable reservations", "error", err)
		return
	}
	if len(archivable) == 0 {
		return
	}

	w.log.Info("archiving reservations", "count", len(archivable))
	for _, res := range archivable {
		if err := w.archiveReservationTx(ctx, res); err != nil {
			w.log.Error("archive reservation", "id", res.ID, "error", err)
		}
	}
}

func (w *Workers) archiveReservationTx(ctx context.Context, res store.OperationsReservation) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := w.q.WithTx(tx)

	if err := qtx.SetCurrentPropertyID(ctx, res.PropertyID.String()); err != nil {
		return fmt.Errorf("set rls: %w", err)
	}

	// Soft-delete items (R-RES-INTEG-008)
	if _, err := tx.Exec(
		ctx,
		`UPDATE operations.reservation_items SET deleted_at = NOW(), version = version + 1
		 WHERE reservation_id = $1 AND deleted_at IS NULL`,
		res.ID,
	); err != nil {
		return fmt.Errorf("archive items: %w", err)
	}

	// Soft-archive folios (R-RES-INTEG-008)
	if err := qtx.ArchiveFolios(ctx, &store.ArchiveFoliosParams{
		ReservationID: uuid.NullUUID{UUID: res.ID, Valid: true},
		PropertyID:    res.PropertyID,
	}); err != nil {
		return fmt.Errorf("archive folios: %w", err)
	}

	// Soft-delete reservation
	rows, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
		ID:      res.ID,
		Version: res.Version,
		Status:  store.OperationsReservationStatusArchived,
	})
	if err != nil {
		return fmt.Errorf("archive reservation: %w", err)
	}
	if rows == 0 {
		w.log.Warn("archival version mismatch (concurrent update)", "id", res.ID)
		return nil
	}

	// Notify SSE
	payload, _ := json.Marshal(struct {
		Action string `json:"action"`
		ID     string `json:"reservation_id"`
	}{Action: "archived", ID: res.ID.String()})
	if err := qtx.NotifyChannel(ctx, &store.NotifyChannelParams{
		Channel: "reservation_changes",
		Payload: string(payload),
	}); err != nil {
		w.log.Warn("notify after archival", "id", res.ID, "error", err)
	}

	return tx.Commit(ctx)
}
