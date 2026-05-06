package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/events"
)

const dateLayout = "2006-01-02"

// reservationChangePayload matches the JSON shape sent by the database triggers
// in migrations/00005_planner_notifier.sql.
type reservationChangePayload struct {
	Operation    string `json:"operation"`
	PropertyID   string `json:"property_id"`
	RecordID     string `json:"record_id"`
	CheckInDate  string `json:"check_in_date"`
	CheckOutDate string `json:"check_out_date"`
	Table        string `json:"table"`
}

// parsedChange holds the raw payload fields alongside the parsed date values.
// Shared by all invalidation strategies so the event is only unmarshalled once.
type parsedChange struct {
	reservationChangePayload
	CheckIn  time.Time
	CheckOut time.Time
}

// parseReservationChange unmarshals an events.Event into a parsedChange,
// returning an error if the payload is malformed or either date is missing.
func parseReservationChange(event events.Event) (parsedChange, error) {
	raw, err := json.Marshal(event.Data)
	if err != nil {
		return parsedChange{}, fmt.Errorf("marshal event data: %w", err)
	}

	var p reservationChangePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return parsedChange{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	checkIn, err := time.Parse(dateLayout, p.CheckInDate)
	if err != nil {
		return parsedChange{}, fmt.Errorf("parse check_in_date %q: %w", p.CheckInDate, err)
	}

	checkOut, err := time.Parse(dateLayout, p.CheckOutDate)
	if err != nil {
		return parsedChange{}, fmt.Errorf("parse check_out_date %q: %w", p.CheckOutDate, err)
	}

	return parsedChange{reservationChangePayload: p, CheckIn: checkIn, CheckOut: checkOut}, nil
}

// reservationCacheInvalidator combines the two cache operations used by
// NewReservationChangeHandler. *Client satisfies this interface.
type reservationCacheInvalidator interface {
	Invalidate(ctx context.Context, pattern string) error
	InvalidateIf(ctx context.Context, pattern string, shouldDelete func(key string) bool) error
}

// NewReservationChangeHandler returns an events.Handler for the
// reservation_changes channel. On each notification it:
//  1. Invalidates one availability key per night in the stay period
//  2. Invalidates the specific reservation record key
//  3. Evicts all planner cache keys whose date range overlaps the stay period
//
// The event is parsed once and shared across both strategies.
// Individual cache failures are logged but never fail the handler — a cache miss
// is acceptable; serving stale data is not.
func NewReservationChangeHandler(c reservationCacheInvalidator, logger *slog.Logger) events.Handler {
	return func(ctx context.Context, event events.Event) error {
		change, err := parseReservationChange(event)
		if err != nil {
			return err
		}

		invalidateAvailability(ctx, c, logger, change)
		invalidatePlanner(ctx, c, logger, change)

		return nil
	}
}

func invalidateAvailability(ctx context.Context, c reservationCacheInvalidator, logger *slog.Logger, change parsedChange) {
	// One key per night — checkOut is exclusive so iterate d < checkOut.
	for d := change.CheckIn; d.Before(change.CheckOut); d = d.AddDate(0, 0, 1) {
		pattern := fmt.Sprintf("yop:availability:%s:%s", change.PropertyID, d.Format(dateLayout))
		if err := c.Invalidate(ctx, pattern); err != nil {
			logger.Warn("failed to invalidate availability cache", "pattern", pattern, "error", err)
		}
	}

	if err := c.Invalidate(ctx, fmt.Sprintf("yop:reservation:%s", change.RecordID)); err != nil {
		logger.Warn("failed to invalidate reservation cache", "record_id", change.RecordID, "error", err)
	}

	logger.Debug("availability cache invalidated",
		"property_id", change.PropertyID,
		"check_in", change.CheckInDate,
		"check_out", change.CheckOutDate,
		"operation", change.Operation,
	)
}

func invalidatePlanner(ctx context.Context, c reservationCacheInvalidator, logger *slog.Logger, change parsedChange) {
	pattern := fmt.Sprintf("yop:planner:%s:*", change.PropertyID)

	if err := c.InvalidateIf(ctx, pattern, func(key string) bool {
		overlaps, err := plannerKeyOverlaps(key, change.PropertyID, change.CheckIn, change.CheckOut)
		if err != nil {
			logger.Warn("skipping unparseable planner cache key", "key", key, "error", err)
			return false
		}
		return overlaps
	}); err != nil {
		logger.Warn("failed to invalidate planner cache", "property_id", change.PropertyID, "error", err)
	}

	logger.Debug("planner cache invalidated",
		"property_id", change.PropertyID,
		"check_in", change.CheckInDate,
		"check_out", change.CheckOutDate,
		"operation", change.Operation,
	)
}

// plannerKeyOverlaps reports whether the planner cache key's date range overlaps
// with the reservation stay period [checkIn, checkOut).
//
// Expected key format: yop:planner:{propertyID}:{start_date}:{end_date}
// Both intervals are treated as half-open: [start, end).
// Overlap condition: checkIn < keyEnd AND checkOut > keyStart
func plannerKeyOverlaps(key, propertyID string, checkIn, checkOut time.Time) (bool, error) {
	prefix := "yop:planner:" + propertyID + ":"
	if !strings.HasPrefix(key, prefix) {
		return false, nil
	}

	rest := strings.TrimPrefix(key, prefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid planner key format (expected start:end dates): %q", key)
	}

	keyStart, err := time.Parse(dateLayout, parts[0])
	if err != nil {
		return false, fmt.Errorf("parse key start date %q: %w", parts[0], err)
	}

	keyEnd, err := time.Parse(dateLayout, parts[1])
	if err != nil {
		return false, fmt.Errorf("parse key end date %q: %w", parts[1], err)
	}

	// Half-open interval overlap: [checkIn, checkOut) ∩ [keyStart, keyEnd) ≠ ∅
	// ⟺ checkIn < keyEnd AND checkOut > keyStart
	return checkIn.Before(keyEnd) && checkOut.After(keyStart), nil
}
