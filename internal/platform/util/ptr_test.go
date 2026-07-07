package util

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestPtrToUUID(t *testing.T) {
	id := uuid.New()
	got := PtrToUUID(id)
	if got == nil || *got != id {
		t.Errorf("PtrToUUID(%v) = %v, want *%v", id, got, id)
	}
}

func TestNullUUIDToPtr_Valid(t *testing.T) {
	id := uuid.New()
	nu := uuid.NullUUID{UUID: id, Valid: true}
	got := NullUUIDToPtr(nu)
	if got == nil || *got != id {
		t.Errorf("NullUUIDToPtr(valid) = %v, want *%v", got, id)
	}
}

func TestNullUUIDToPtr_Invalid(t *testing.T) {
	nu := uuid.NullUUID{Valid: false}
	got := NullUUIDToPtr(nu)
	if got != nil {
		t.Errorf("NullUUIDToPtr(invalid) = %v, want nil", got)
	}
}

func TestPtrUUID_NonNil(t *testing.T) {
	id := uuid.New()
	got := PtrUUID(&id)
	if got != id {
		t.Errorf("PtrUUID(*%v) = %v, want %v", id, got, id)
	}
}

func TestPtrUUID_Nil(t *testing.T) {
	got := PtrUUID(nil)
	if got != uuid.Nil {
		t.Errorf("PtrUUID(nil) = %v, want uuid.Nil", got)
	}
}

func TestNullText_Nil(t *testing.T) {
	got := NullText(pgtype.Text{})
	if got != "" {
		t.Errorf("NullText(empty) = %q, want empty string", got)
	}
}

func TestNullText_Valid(t *testing.T) {
	got := NullText(pgtype.Text{String: "hello", Valid: true})
	if got != "hello" {
		t.Errorf("NullText(valid) = %q, want %q", got, "hello")
	}
}
