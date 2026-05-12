package cache

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := New(rdb, "test:", logger)

	cleanup := func() {
		rdb.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestSet_And_Get(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	original := TestData{Name: "John", Age: 30}

	// Set a value
	err := c.Set(ctx, "user:1", original, 1*time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get it back
	var retrieved TestData
	err = c.Get(ctx, "user:1", &retrieved)

	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != original.Name || retrieved.Age != original.Age {
		t.Errorf("Data mismatch: got %+v, want %+v", retrieved, original)
	}
}

func TestGet_CacheMiss(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	var data interface{}
	err := c.Get(ctx, "nonexistent", &data)

	if err != ErrCacheMiss {
		t.Errorf("Error: got %v, want ErrCacheMiss", err)
	}
}

func TestDelete(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Set a value
	err := c.Set(ctx, "key:1", "value", 1*time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	var value string
	err = c.Get(ctx, "key:1", &value)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Delete it
	err = c.Delete(ctx, "key:1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	err = c.Get(ctx, "key:1", &value)
	if err != ErrCacheMiss {
		t.Errorf("Error: got %v, want ErrCacheMiss", err)
	}
}

func TestInvalidate_Pattern(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Set multiple values with different patterns
	_ = c.Set(ctx, "availability:uuid1:date1", "data1", 1*time.Hour)
	_ = c.Set(ctx, "availability:uuid1:date2", "data2", 1*time.Hour)
	_ = c.Set(ctx, "availability:uuid2:date1", "data3", 1*time.Hour)
	_ = c.Set(ctx, "other:key", "data4", 1*time.Hour)

	// Invalidate all availability keys for uuid1
	err := c.Invalidate(ctx, "test:availability:uuid1:*")
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// Verify uuid1 keys are gone
	var value string
	if err := c.Get(ctx, "availability:uuid1:date1", &value); err != ErrCacheMiss {
		t.Error("Key should be invalidated")
	}

	// Verify other keys still exist
	if err := c.Get(ctx, "availability:uuid2:date1", &value); err != nil {
		t.Error("Other key should still exist")
	}

	if err := c.Get(ctx, "other:key", &value); err != nil {
		t.Error("Unrelated key should still exist")
	}
}

func TestGetOrSet_CacheHit(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Pre-populate cache
	_ = c.Set(ctx, "data:1", "cached_value", 1*time.Hour)

	loaderCalled := false
	var result string

	err := c.GetOrSet(ctx, "data:1", &result, 1*time.Hour, func(context.Context) (any, error) {
		loaderCalled = true
		return "loader_value", nil
	})

	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if result != "cached_value" {
		t.Errorf("Result: got %q, want %q", result, "cached_value")
	}

	if loaderCalled {
		t.Error("Loader should not be called on cache hit")
	}
}

func TestGetOrSet_CacheMiss(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	loaderCalled := false
	var result string

	err := c.GetOrSet(ctx, "data:2", &result, 1*time.Hour, func(context.Context) (any, error) {
		loaderCalled = true
		return "loaded_value", nil
	})

	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if result != "loaded_value" {
		t.Errorf("Result: got %q, want %q", result, "loaded_value")
	}

	if !loaderCalled {
		t.Error("Loader should be called on cache miss")
	}

	// Verify it was cached
	var cached string
	err = c.Get(ctx, "data:2", &cached)
	if err != nil {
		t.Fatalf("Cached value not found: %v", err)
	}

	if cached != "loaded_value" {
		t.Errorf("Cached value: got %q, want %q", cached, "loaded_value")
	}
}

func TestPrefix_Namespace(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Set with "test:" prefix (from newTestCache)
	_ = c.Set(ctx, "mykey", "myvalue", 1*time.Hour)

	// Try to get with same prefix
	var value string
	err := c.Get(ctx, "mykey", &value)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "myvalue" {
		t.Errorf("Value: got %q, want %q", value, "myvalue")
	}
}

func TestTTL_Expiration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	c := New(rdb, "test:", logger)

	ctx := context.Background()

	// Set with very short TTL
	_ = c.Set(ctx, "shortlived", "value", 100*time.Millisecond)

	// Should exist immediately
	var value string
	if err := c.Get(ctx, "shortlived", &value); err != nil {
		t.Fatal("Key should exist immediately after set")
	}

	// Fast-forward time in miniredis and check expiration
	mr.FastForward(200 * time.Millisecond)

	if err := c.Get(ctx, "shortlived", &value); err != ErrCacheMiss {
		t.Error("Key should have expired")
	}
}

func TestSet_InvalidJSON(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	// Channels can't be JSON-marshaled
	ch := make(chan int)

	err := c.Set(ctx, "badkey", ch, 1*time.Hour)

	if err == nil {
		t.Fatal("Set should fail for non-JSON-serializable value")
	}
}

func TestInvalidateIf_SelectiveDelete(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	_ = c.Set(ctx, "planner:prop-1:2026-03-01:2026-03-10", "data1", time.Hour)
	_ = c.Set(ctx, "planner:prop-1:2026-03-10:2026-03-20", "data2", time.Hour)
	_ = c.Set(ctx, "planner:prop-1:2026-03-20:2026-03-30", "data3", time.Hour)

	// Only delete the middle key
	target := "test:planner:prop-1:2026-03-10:2026-03-20"
	err := c.InvalidateIf(ctx, "test:planner:prop-1:*", func(key string) bool {
		return key == target
	})
	if err != nil {
		t.Fatalf("InvalidateIf failed: %v", err)
	}

	var v string
	if err := c.Get(ctx, "planner:prop-1:2026-03-01:2026-03-10", &v); err != nil {
		t.Error("first key should still exist")
	}
	if err := c.Get(ctx, "planner:prop-1:2026-03-10:2026-03-20", &v); err != ErrCacheMiss {
		t.Error("middle key should have been deleted")
	}
	if err := c.Get(ctx, "planner:prop-1:2026-03-20:2026-03-30", &v); err != nil {
		t.Error("last key should still exist")
	}
}

func TestInvalidateIf_PredicateFalse_NothingDeleted(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()
	_ = c.Set(ctx, "keep:1", "value", time.Hour)

	err := c.InvalidateIf(ctx, "test:keep:*", func(string) bool { return false })
	if err != nil {
		t.Fatalf("InvalidateIf failed: %v", err)
	}

	var v string
	if err := c.Get(ctx, "keep:1", &v); err != nil {
		t.Error("key should still exist when predicate always returns false")
	}
}

func TestInvalidateIf_NoMatchingKeys_NoOp(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	err := c.InvalidateIf(context.Background(), "test:nonexistent:*", func(string) bool { return true })
	if err != nil {
		t.Fatalf("InvalidateIf with no matching keys should not error: %v", err)
	}
}

func TestComplexTypes(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()

	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string   `json:"name"`
		Address Address  `json:"address"`
		Tags    []string `json:"tags"`
	}

	original := Person{
		Name: "Alice",
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
		},
		Tags: []string{"engineer", "golang"},
	}

	// Set and get complex type
	_ = c.Set(ctx, "person:1", original, 1*time.Hour)

	var retrieved Person
	err := c.Get(ctx, "person:1", &retrieved)

	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != original.Name || retrieved.Address.City != original.Address.City {
		t.Errorf("Data mismatch: got %+v, want %+v", retrieved, original)
	}

	if len(retrieved.Tags) != len(original.Tags) || retrieved.Tags[0] != original.Tags[0] {
		t.Error("Tags not properly stored/retrieved")
	}
}
