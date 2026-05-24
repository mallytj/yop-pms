// Package helpers provides shared context utilities for extracting and storing
// request-scoped values (property ID, user ID, permissions, If-Match version).
//
// These are set by middleware (StubAuth, RequireIfMatch) and consumed by service
// methods and store helpers (ExecuteTx).
package helpers

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxPropertyID  contextKey = "property_id"
	ctxPermissions contextKey = "permissions"
	ctxIfMatchVer  contextKey = "if_match_version"
	ctxUserID      contextKey = "user_id"
)

// GetPropertyIDFromCtx extracts the property UUID from context.
// Returns uuid.Nil if not set (handlers must return 400 before reaching service).
func GetPropertyIDFromCtx(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(ctxPropertyID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// SetPropertyIDInCtx stores the property UUID in context.
func SetPropertyIDInCtx(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxPropertyID, id)
}

// GetUserIDFromCtx extracts the user UUID from context.
// Returns uuid.Nil if not set (workers and anonymous paths).
func GetUserIDFromCtx(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(ctxUserID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// SetUserIDInCtx stores the user UUID in context.
func SetUserIDInCtx(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxUserID, id)
}

// GetPermissionsFromCtx extracts the permission slice from context.
// Returns nil if not set.
func GetPermissionsFromCtx(ctx context.Context) []string {
	perms, ok := ctx.Value(ctxPermissions).([]string)
	if !ok {
		return nil
	}
	return perms
}

// SetPermissionsInCtx stores the permission slice in context.
func SetPermissionsInCtx(ctx context.Context, perms []string) context.Context {
	return context.WithValue(ctx, ctxPermissions, perms)
}

// HasPermission checks whether a specific permission is held in the context.
func HasPermission(ctx context.Context, perm string) bool {
	for _, p := range GetPermissionsFromCtx(ctx) {
		if p == perm {
			return true
		}
	}
	return false
}

// GetIfMatchVersion extracts the optimistic-lock version from context.
// Returns 0 if not set (non-mutation endpoints).
func GetIfMatchVersion(ctx context.Context) int32 {
	v, ok := ctx.Value(ctxIfMatchVer).(int32)
	if !ok {
		return 0
	}
	return v
}

// SetIfMatchVersion stores the optimistic-lock version in context.
func SetIfMatchVersion(ctx context.Context, version int32) context.Context {
	return context.WithValue(ctx, ctxIfMatchVer, version)
}
