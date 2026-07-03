package booking

// Satisfies ADR-022

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/store"
)

// ParseIncludeFlags parses the ?include query parameter from the request.
// Default (no param): items included, guest ID only, no folio.
// ?include=none: lightweight envelope without items.
func ParseIncludeFlags(r *http.Request) IncludeFlags {
	raw := r.URL.Query().Get("include")
	if raw == "" {
		return IncludeFlags{Items: true} // default: items included
	}

	flags := IncludeFlags{Items: true}
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "items":
			flags.Items = true
		case "guest":
			flags.Guest = true
		case "folio_summary":
			flags.FolioSummary = true
		case "none":
			flags.None = true
			flags.Items = false
		}
	}
	return flags
}

// expandInclude fetches related resources (items, guest) for a reservation response.
// If response.Items is nil and IncludeItems is true, items are fetched from the DB.
// If IncludeGuest is true and primaryGuestID is non-nil, the guest is expanded.
func expandInclude(
	ctx context.Context,
	q *store.Queries,
	resp *ReservationResponse,
	flags IncludeFlags,
	propertyID, reservationID uuid.UUID,
	primaryGuestID uuid.NullUUID,
	log *slog.Logger,
) {
	if flags.IncludeItems() && len(resp.Items) == 0 {
		items, err := q.GetReservationItems(ctx, &store.GetReservationItemsParams{
			ReservationID: reservationID,
			PropertyID:    propertyID,
		})
		if err != nil {
			log.Warn("failed to fetch items for expansion", "error", err, "reservation_id", reservationID)
		} else {
			for _, item := range items {
				it := itemToResponse(&item)
				resp.Items = append(resp.Items, *it)
			}
		}
	}

	if flags.Guest && primaryGuestID.Valid {
		guest, err := q.GetGuest(ctx, primaryGuestID.UUID)
		if err != nil {
			log.Warn("failed to expand guest", "error", err, "guest_id", primaryGuestID)
		} else {
			resp.Guest = guestToResponse(&guest)
		}
	}
}
