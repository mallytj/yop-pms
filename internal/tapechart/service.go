package tapechart

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// Service loads tape chart grid data (reservations, inventory, and
// maintenance blocks) for a property and date-range window.
type Service struct {
	pool *pgxpool.Pool
	q    *store.Queries
	log  *slog.Logger
}

// NewService builds a Service backed by the given pool, queries, and logger.
func NewService(pool *pgxpool.Pool, q *store.Queries, log *slog.Logger) *Service {
	return &Service{
		pool: pool,
		q:    q,
		log:  log,
	}
}

// GetTapeChart loads the grid for propertyID over [from, to], including
// room/room-type metadata when include is IncludeFull.
func (s *Service) GetTapeChart(
	ctx context.Context,
	propertyID uuid.UUID,
	from time.Time,
	to time.Time,
	include IncludeMode,
) (TapeChartData, error) {
	inventory, reservations, reservationItems, maintenanceBlocks, err := s.loadCoreData(ctx, propertyID, from, to)
	if err != nil {
		return TapeChartData{}, err
	}

	accentByReservation := assignAccentsToReservations(reservations)

	reservationBlocks := buildReservationBlocks(reservationItems, reservations, accentByReservation)
	maintenanceBlockDTOs := buildMaintenanceBlocks(maintenanceBlocks)
	inventoryDays := buildInventoryDays(inventory)

	data := TapeChartData{
		From:      types.ISO8601Date{Time: from},
		To:        types.ISO8601Date{Time: to},
		Inventory: inventoryDays,
	}

	if include == IncludeFull {
		roomTypes, rooms, err := s.loadRoomTypesAndRooms(ctx, propertyID)
		if err != nil {
			return TapeChartData{}, err
		}
		data.RoomTypes = nestRoomTypes(roomTypes, rooms, reservationBlocks, maintenanceBlockDTOs)
		return data, nil
	}

	data.Reservations = reservationBlocks
	data.MaintenanceBlocks = maintenanceBlockDTOs
	return data, nil
}

// loadCoreData runs the three independent block sources concurrently:
// inventory, the reservations→reservation-items chain, and maintenance
// blocks. Reservation items depend on the reservation IDs loaded first, so
// that pair runs as one sequential chain within its own goroutine.
func (s *Service) loadCoreData(
	ctx context.Context,
	propertyID uuid.UUID,
	from time.Time,
	to time.Time,
) (
	inventory []store.GetTapeChartInventoryRow,
	reservations []store.GetTapeChartReservationsRow,
	reservationItems []store.GetTapeChartReservationItemsRow,
	maintenanceBlocks []store.GetTapeChartMaintenanceBlocksRow,
	err error,
) {
	var (
		wg                                            sync.WaitGroup
		inventoryErr, reservationsErr, maintenanceErr error
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		inventory, inventoryErr = s.loadInventory(ctx, propertyID, from, to)
	}()
	go func() {
		defer wg.Done()
		reservations, reservationsErr = s.loadReservations(ctx, propertyID, from, to)
		if reservationsErr != nil {
			return
		}
		reservationItems, reservationsErr = s.loadReservationItems(ctx, reservations)
	}()
	go func() {
		defer wg.Done()
		maintenanceBlocks, maintenanceErr = s.loadMaintenanceBlocks(ctx, propertyID, from, to)
	}()
	wg.Wait()

	if inventoryErr != nil {
		return nil, nil, nil, nil, inventoryErr
	}
	if reservationsErr != nil {
		return nil, nil, nil, nil, reservationsErr
	}
	if maintenanceErr != nil {
		return nil, nil, nil, nil, maintenanceErr
	}
	return inventory, reservations, reservationItems, maintenanceBlocks, nil
}

// logTapeChartError logs a load failure with a consistent "tape-chart: " prefix.
func (s *Service) logTapeChartError(msg string, err error, args ...any) {
	s.log.Error("tape-chart: "+msg, append([]any{"err", err}, args...)...)
}

// loadRoomTypesAndRooms runs both lookups concurrently; they're independent
// reads with no shared transaction requirement.
func (s *Service) loadRoomTypesAndRooms(
	ctx context.Context,
	propertyID uuid.UUID,
) ([]store.GetTapeChartRoomTypesRow, []store.GetTapeChartRoomsRow, error) {
	var (
		wg                     sync.WaitGroup
		roomTypes              []store.GetTapeChartRoomTypesRow
		rooms                  []store.GetTapeChartRoomsRow
		roomTypesErr, roomsErr error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		roomTypes, roomTypesErr = s.loadRoomTypes(ctx, propertyID)
	}()
	go func() {
		defer wg.Done()
		rooms, roomsErr = s.loadRooms(ctx, propertyID)
	}()
	wg.Wait()

	if roomTypesErr != nil {
		return nil, nil, roomTypesErr
	}
	if roomsErr != nil {
		return nil, nil, roomsErr
	}
	return roomTypes, rooms, nil
}

func (s *Service) loadRoomTypes(ctx context.Context, propertyID uuid.UUID) ([]store.GetTapeChartRoomTypesRow, error) {
	rows, err := s.q.GetTapeChartRoomTypes(ctx, propertyID)
	if err != nil {
		s.logTapeChartError("load room types failed", err, "property_id", propertyID)
		return nil, fmt.Errorf("load room types: %w", err)
	}
	return rows, nil
}

func (s *Service) loadRooms(ctx context.Context, propertyID uuid.UUID) ([]store.GetTapeChartRoomsRow, error) {
	rows, err := s.q.GetTapeChartRooms(ctx, propertyID)
	if err != nil {
		s.logTapeChartError("load rooms failed", err, "property_id", propertyID)
		return nil, fmt.Errorf("load rooms: %w", err)
	}
	return rows, nil
}

func (s *Service) loadInventory(
	ctx context.Context,
	propertyID uuid.UUID,
	from time.Time,
	to time.Time,
) ([]store.GetTapeChartInventoryRow, error) {
	rows, err := s.q.GetTapeChartInventory(ctx, &store.GetTapeChartInventoryParams{
		PropertyID: propertyID,
		FromDate:   pgtype.Date{Time: from, Valid: true},
		ToDate:     pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		s.logTapeChartError("load inventory failed", err, "property_id", propertyID, "from", from, "to", to)
		return nil, fmt.Errorf("load inventory: %w", err)
	}
	return rows, nil
}

func (s *Service) loadReservations(
	ctx context.Context,
	propertyID uuid.UUID,
	from time.Time,
	to time.Time,
) ([]store.GetTapeChartReservationsRow, error) {
	rows, err := s.q.GetTapeChartReservations(ctx, &store.GetTapeChartReservationsParams{
		PropertyID: propertyID,
		FromDate:   pgtype.Timestamptz{Time: from, Valid: true},
		ToDate:     pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		s.logTapeChartError("load reservations failed", err, "property_id", propertyID, "from", from, "to", to)
		return nil, fmt.Errorf("load reservations: %w", err)
	}
	return rows, nil
}

func (s *Service) loadReservationItems(
	ctx context.Context,
	reservations []store.GetTapeChartReservationsRow,
) ([]store.GetTapeChartReservationItemsRow, error) {
	if len(reservations) == 0 {
		return nil, nil
	}

	reservationIDs := make([]uuid.UUID, len(reservations))
	for i, reservation := range reservations {
		reservationIDs[i] = reservation.ID
	}

	rows, err := s.q.GetTapeChartReservationItems(ctx, reservationIDs)
	if err != nil {
		s.logTapeChartError("load reservation items failed", err, "count", len(reservationIDs))
		return nil, fmt.Errorf("load reservation items: %w", err)
	}
	return rows, nil
}

func (s *Service) loadMaintenanceBlocks(
	ctx context.Context,
	propertyID uuid.UUID,
	from time.Time,
	to time.Time,
) ([]store.GetTapeChartMaintenanceBlocksRow, error) {
	rows, err := s.q.GetTapeChartMaintenanceBlocks(ctx, &store.GetTapeChartMaintenanceBlocksParams{
		PropertyID: propertyID,
		FromDate:   pgtype.Date{Time: from, Valid: true},
		ToDate:     pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		s.logTapeChartError("load maintenance blocks failed", err, "property_id", propertyID, "from", from, "to", to)
		return nil, fmt.Errorf("load maintenance blocks: %w", err)
	}
	return rows, nil
}
