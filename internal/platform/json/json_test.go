package json

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/logging"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"message": "hello"}
	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("Message: got %q, want %q", result["message"], "hello")
	}
}

func TestWriteError_APIError(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.Background()
	logger := logging.NewLogger("dev")
	ctx = logging.WithContext(ctx, logger)

	r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	WriteError(w, r, apierror.ErrNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusNotFound)
	}

	var result apierror.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != "NOT_FOUND" {
		t.Errorf("Code: got %q, want %q", result.Code, "NOT_FOUND")
	}
}

func TestWriteError_UnexpectedError(t *testing.T) {
	_ = t
	w := httptest.NewRecorder()
	ctx := context.Background()
	logger := logging.NewLogger("dev")
	ctx = logging.WithContext(ctx, logger)

	r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	WriteError(w, r, apierror.MapPostgresError(nil))

	// WriteError should handle nil from MapPostgresError gracefully
	// This case returns OK status since we passed nil and MapPostgresError returns nil
}

func TestReadJSON_ValidInput(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	body := []byte(`{"name": "John", "age": 30}`)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))

	var result TestPayload
	err := ReadJSON(r, &result)

	if err != nil {
		t.Errorf("ReadJSON returned error: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("Name: got %q, want %q", result.Name, "John")
	}
	if result.Age != 30 {
		t.Errorf("Age: got %d, want %d", result.Age, 30)
	}
}

func TestReadJSON_MalformedJSON(t *testing.T) {
	body := []byte(`{invalid json}`)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))

	var result map[string]string
	err := ReadJSON(r, &result)

	if err == nil {
		t.Fatal("ReadJSON should return error for malformed JSON")
	}

	if err.Code != "BAD_REQUEST" {
		t.Errorf("Code: got %q, want %q", err.Code, "BAD_REQUEST")
	}

	if err.Status != http.StatusBadRequest {
		t.Errorf("Status: got %d, want %d", err.Status, http.StatusBadRequest)
	}
}

func TestReadJSON_UnknownFields(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name"`
	}

	body := []byte(`{"name": "John", "unknown": "field"}`)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))

	var result TestPayload
	err := ReadJSON(r, &result)

	if err == nil {
		t.Fatal("ReadJSON should return error for unknown fields")
	}

	if err.Code != "BAD_REQUEST" {
		t.Errorf("Code: got %q, want %q", err.Code, "BAD_REQUEST")
	}
}

func TestReadJSON_EmptyBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte{}))

	var result map[string]string
	err := ReadJSON(r, &result)

	if err == nil {
		t.Fatal("ReadJSON should return error for empty body")
	}

	if err.Code != "BAD_REQUEST" {
		t.Errorf("Code: got %q, want %q", err.Code, "BAD_REQUEST")
	}
}

func TestWriteError_CustomErrorMessage(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.Background()
	logger := logging.NewLogger("dev")
	ctx = logging.WithContext(ctx, logger)

	r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	customErr := apierror.ErrConflict.WithMessage("custom conflict message")
	WriteError(w, r, customErr)

	if w.Code != http.StatusConflict {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusConflict)
	}

	var result apierror.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Message != "custom conflict message" {
		t.Errorf("Message: got %q, want %q", result.Message, "custom conflict message")
	}
}

func TestWriteJSON_ComplexStructure(t *testing.T) {
	w := httptest.NewRecorder()

	type NestedData struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	}

	data := struct {
		Status string     `json:"status"`
		Data   NestedData `json:"data"`
		Items  []string   `json:"items"`
	}{
		Status: "ok",
		Data:   NestedData{ID: 1, Value: "test"},
		Items:  []string{"a", "b", "c"},
	}

	WriteJSON(w, http.StatusOK, data)

	// Verify the response can be decoded
	var result struct {
		Status string     `json:"status"`
		Data   NestedData `json:"data"`
		Items  []string   `json:"items"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Status != "ok" || result.Data.ID != 1 || len(result.Items) != 3 {
		t.Error("Decoded data does not match original")
	}
}
