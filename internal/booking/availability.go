package booking

// Core Requirements: R-RES-AVAIL-001, R-RES-AVAIL-002, R-RES-AVAIL-003, ADR-013

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

const (
	availabilityCacheTTL = 60 * time.Second
)

// CheckAvailability returns per-night availability for a room type over a date range.
// Results are cached per-day in Redis (TTL 60s). Cache is invalidated reactively
// via NOTIFY reservation_changes. TTL is a safety net only.
func (s *Service) CheckAvailability(
	ctx context.Context,
	propertyID, roomTypeID uuid.UUID,
	startDate, endDate time.Time,
) ([]DateAvailability, error) {
	if !endDate.After(startDate) {
		return nil, ErrInvalidDates.WithMessage("end_date must be after start_date")
	}

	nights := util.NightsBetween(startDate, endDate)
	cachedByDate := make(map[string]int32, len(nights))

	// Batch Redis MGET: one round-trip instead of N individual GETs.
	keys := make([]string, len(nights))
	for i, night := range nights {
		keys[i] = availabilityCacheKey(propertyID, roomTypeID, night)
	}

	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget availability: %w", err)
	}

	var uncached []time.Time
	for i, val := range vals {
		k := nights[i].Format("2006-01-02")
		if val != nil {
			if s, ok := val.(string); ok {
				if n, err := strconv.ParseInt(s, 10, 32); err == nil {
					cachedByDate[k] = int32(n) //nolint:gosec
					continue
				}
			}
		}
		uncached = append(uncached, nights[i])
	}

	if len(uncached) == 0 {
		return buildAvailabilityResult(nights, cachedByDate), nil
	}

	// Compute availability: total rooms of this type minus sold ledger entries per date.
	const availQuery = `
		SELECT d::date AS calendar_date,
			(SELECT COUNT(*) FROM inventory.rooms WHERE room_type_id = $2 AND property_id = $1) -
			COALESCE(
				(SELECT COUNT(*) FROM inventory.room_inventory_ledger
				 WHERE calendar_date = d::date AND status = 'sold'
				 AND room_id IN (SELECT id FROM inventory.rooms WHERE room_type_id = $2 AND property_id = $1)),
			0
		)::INT AS available
		FROM generate_series($3::date, $4::date, '1 day') d
	`
	rows, err := s.pool.Query(ctx, availQuery, propertyID, roomTypeID,
		pgtype.Date{Time: uncached[0], Valid: true},
		pgtype.Date{Time: uncached[len(uncached)-1], Valid: true},
	)
	if err != nil {
		return nil, fmt.Errorf("availability query: %w", err)
	}
	defer rows.Close()

	availByDate := make(map[string]int32)
	for rows.Next() {
		var d pgtype.Date
		var avail int32
		if err := rows.Scan(&d, &avail); err != nil {
			return nil, fmt.Errorf("scan availability: %w", err)
		}
		if d.Valid {
			availByDate[d.Time.Format("2006-01-02")] = avail
		}
	}

	// Pipeline SETs: one round-trip instead of N individual SETs.
	pipe := s.rdb.Pipeline()
	for _, night := range uncached {
		k := night.Format("2006-01-02")
		cachedByDate[k] = availByDate[k]
		key := availabilityCacheKey(propertyID, roomTypeID, night)
		pipe.Set(ctx, key, availByDate[k], availabilityCacheTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("redis pipeline set availability", "error", err)
	}

	return buildAvailabilityResult(nights, cachedByDate), nil
}

// buildAvailabilityResult assembles the response slice from cached values.
func buildAvailabilityResult(nights []time.Time, cachedByDate map[string]int32) []DateAvailability {
	result := make([]DateAvailability, 0, len(nights))
	for _, night := range nights {
		k := night.Format("2006-01-02")
		result = append(result, DateAvailability{
			Date:      types.ISO8601Date{Time: night},
			Available: int(cachedByDate[k]),
		})
	}
	return result
}

// InvalidateAvailabilityCache removes cached availability for all nights of a stay.
func (s *Service) InvalidateAvailabilityCache(ctx context.Context, propertyID, roomTypeID uuid.UUID, arrival, departure time.Time) {
	for _, night := range util.NightsBetween(arrival, departure) {
		key := availabilityCacheKey(propertyID, roomTypeID, night)
		if err := s.rdb.Del(ctx, key).Err(); err != nil {
			s.log.Warn("redis del availability", "key", key, "error", err)
		}
	}
}

// conflictCheck asserts that a specific room has no conflicting bookings on the given dates.
func (s *Service) conflictCheck(
	ctx context.Context,
	qtx *store.Queries,
	roomID uuid.UUID,
	dates []time.Time,
	excludeItemID *uuid.UUID,
) error {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)

	pgDates := make([]pgtype.Date, len(dates))
	for i, d := range dates {
		pgDates[i] = pgtype.Date{Time: d, Valid: true}
	}

	var excludeID uuid.UUID
	if excludeItemID != nil {
		excludeID = *excludeItemID
	}

	conflicts, err := qtx.ConflictCheckOnLedger(ctx, &store.ConflictCheckOnLedgerParams{
		RoomID:        roomID,
		Dates:         pgDates,
		PropertyID:    propertyID,
		ExcludeItemID: excludeID,
	})
	if err != nil {
		return fmt.Errorf("conflict check: %w", err)
	}
	if len(conflicts) > 0 {
		return ErrRoomNotAvailable.WithMessage(fmt.Sprintf("room not available on %d date(s), including %s",
			len(conflicts), conflicts[0].Time.Format("2006-01-02")))
	}
	return nil
}

func availabilityCacheKey(propertyID, roomTypeID uuid.UUID, date time.Time) string {
	return fmt.Sprintf("yop:availability:%s:%s:%s", propertyID, roomTypeID, date.Format("2006-01-02"))
}
