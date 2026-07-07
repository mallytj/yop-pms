package util

import "github.com/google/uuid"

// PtrToUUID returns a pointer to a UUID.
// Returns nil for uuid.Nil so omitempty works in JSON.
func PtrToUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// NullUUIDToPtr converts a nullable UUID to a pointer.
// Returns nil if not valid.
func NullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		return &nu.UUID
	}
	return nil
}

// PtrUUID dereferences a UUID pointer, returning uuid.Nil for nil.
func PtrUUID(u *uuid.UUID) uuid.UUID {
	if u != nil {
		return *u
	}
	return uuid.Nil
}
