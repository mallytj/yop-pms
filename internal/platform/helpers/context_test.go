package helpers

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestGetPropertyIDFromCtx_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetPropertyIDFromCtx(ctx)
	if got != uuid.Nil {
		t.Errorf("expected uuid.Nil when not set, got %v", got)
	}
}

func TestGetPropertyIDFromCtx_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxPropertyID, "not-a-uuid")
	got := GetPropertyIDFromCtx(ctx)
	if got != uuid.Nil {
		t.Errorf("expected uuid.Nil for wrong type, got %v", got)
	}
}

func TestSetAndGetPropertyID(t *testing.T) {
	id := uuid.New()
	ctx := SetPropertyIDInCtx(context.Background(), id)
	got := GetPropertyIDFromCtx(ctx)
	if got != id {
		t.Errorf("expected %v, got %v", id, got)
	}
}

func TestGetUserIDFromCtx_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetUserIDFromCtx(ctx)
	if got != uuid.Nil {
		t.Errorf("expected uuid.Nil when not set, got %v", got)
	}
}

func TestSetAndGetUserID(t *testing.T) {
	id := uuid.New()
	ctx := SetUserIDInCtx(context.Background(), id)
	got := GetUserIDFromCtx(ctx)
	if got != id {
		t.Errorf("expected %v, got %v", id, got)
	}
}

func TestGetUserIDPtrFromCtx_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetUserIDPtrFromCtx(ctx)
	if got != nil {
		t.Errorf("expected nil when not set, got %v", got)
	}
}

func TestGetUserIDPtrFromCtx_Set(t *testing.T) {
	id := uuid.New()
	ctx := SetUserIDInCtx(context.Background(), id)
	got := GetUserIDPtrFromCtx(ctx)
	if got == nil || *got != id {
		t.Errorf("expected %v, got %v", id, got)
	}
}

func TestGetPermissionsFromCtx_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetPermissionsFromCtx(ctx)
	if got != nil {
		t.Errorf("expected nil when not set, got %v", got)
	}
}

func TestGetPermissionsFromCtx_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxPermissions, "not-a-slice")
	got := GetPermissionsFromCtx(ctx)
	if got != nil {
		t.Errorf("expected nil for wrong type, got %v", got)
	}
}

func TestSetAndGetPermissions(t *testing.T) {
	perms := []string{"read", "write"}
	ctx := SetPermissionsInCtx(context.Background(), perms)
	got := GetPermissionsFromCtx(ctx)
	if len(got) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(got))
	}
}

func TestHasPermission_Found(t *testing.T) {
	ctx := SetPermissionsInCtx(context.Background(), []string{"admin:read", "admin:write"})
	if !HasPermission(ctx, "admin:read") {
		t.Error("expected HasPermission to return true")
	}
}

func TestHasPermission_NotFound(t *testing.T) {
	ctx := SetPermissionsInCtx(context.Background(), []string{"read"})
	if HasPermission(ctx, "write") {
		t.Error("expected HasPermission to return false")
	}
}

func TestHasPermission_Empty(t *testing.T) {
	ctx := context.Background()
	if HasPermission(ctx, "any") {
		t.Error("expected HasPermission to return false with no permissions set")
	}
}

func TestGetIfMatchVersion_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetIfMatchVersion(ctx)
	if got != 0 {
		t.Errorf("expected 0 when not set, got %d", got)
	}
}

func TestGetIfMatchVersion_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxIfMatchVer, "not-int32")
	got := GetIfMatchVersion(ctx)
	if got != 0 {
		t.Errorf("expected 0 for wrong type, got %d", got)
	}
}

func TestSetAndGetIfMatchVersion(t *testing.T) {
	ctx := SetIfMatchVersion(context.Background(), 42)
	got := GetIfMatchVersion(ctx)
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}
