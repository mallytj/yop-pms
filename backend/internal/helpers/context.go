package helpers

import (
	"context"

	"ollerod-pms/internal/types"

	"github.com/google/uuid"
)

type contextKey = types.ContextKey

const (
	PropertyIDKey        contextKey = "propertyID"
	UserIDKey            contextKey = "userID"
	LicenceIDKey         contextKey = "licenceID"
	ReservationItemIDKey contextKey = "reservationItemID"
)

// getIdFromCtx is a helper function to retrieve a UUID from the context using the provided key.
// Example usage: getIdFromCtx(ctx, UserIDKey)
// Returns uuid.Nil if not found.
func getIdFromCtx(ctx context.Context, key contextKey) uuid.UUID {
	if ctx == nil {
		return uuid.Nil
	}

	if val := ctx.Value(key); val != nil {
		uuidVal, ok := val.(uuid.UUID)
		if !ok {

			return uuid.Nil
		}
		val = uuidVal

		if val == uuid.Nil {
			return uuid.Nil
		}
		return val.(uuid.UUID)
	}
	return uuid.Nil
}

func SetIDInCtx(ctx context.Context, key contextKey, id uuid.UUID) context.Context {
	return context.WithValue(ctx, key, id)
}

// GetPropertyIDFromCtx retrieves the Property ID from the context.
func GetPropertyIDFromCtx(ctx context.Context) uuid.UUID {
	return getIdFromCtx(ctx, PropertyIDKey)
}

// GetUserIDFromCtx retrieves the User ID from the context.
func GetResItemIDFromCtx(ctx context.Context) uuid.UUID {
	return getIdFromCtx(ctx, ReservationItemIDKey)
}
