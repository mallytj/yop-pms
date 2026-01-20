package helpers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"ollerod-pms/internal/types"
)

type contextKey = types.ContextKey
const (
	PropertyIDKey contextKey = "propertyID"
	UserIDKey     contextKey = "userID"
	LicenceIDKey  contextKey = "licenceID"
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
		fmt.Println("Retrieved", key, "from context:", val)

		if val == uuid.Nil {
			return uuid.Nil
		}
		return val.(uuid.UUID)
	}
	fmt.Println("No value found in context for key:", key)
	return uuid.Nil
}

// GetUserID retrieves the userID from the context.
// Returns nil if not found.
func GetUserID(ctx context.Context) uuid.UUID {
	return getIdFromCtx(ctx, UserIDKey)
}

// GetLicenceID retrieves the licenceID from the context.
// Returns nil if not found.
func GetLicenceID(ctx context.Context) uuid.UUID {
	return getIdFromCtx(ctx, LicenceIDKey)
}

// GetPropertyID retrieves the propertyID from the context.
// Returns nil if not found.
func GetPropertyID(ctx context.Context) uuid.UUID {
	return getIdFromCtx(ctx, PropertyIDKey)
}
