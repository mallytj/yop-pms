# ADR 008: Redis Caching Layer

## Status
**Accepted**

## Context

Application data queries can be expensive:
- Repeated database queries for frequently accessed data (e.g., property settings, user preferences)
- Same data fetched multiple times per request from different handlers
- Peak load causes database connection exhaustion

We need a simple, fast cache to reduce database load. Invalidation strategy is covered in ADR-010.

## Decision

We implement a **simple prefix-namespaced Redis cache client**:

1. **Single cache.Client abstraction** — Wraps Redis with a consistent interface:
   ```go
   Set(ctx, key, value, ttl) error                          // Store with TTL
   Get(ctx, key, dst) error                                 // Retrieve and JSON-decode
   GetOrSet(ctx, key, dst, ttl, loader)                     // Read-through cache helper
   Delete(ctx, key) error                                   // Explicit single-key removal
   Invalidate(ctx, pattern) error                           // Pattern batch invalidation (SCAN + DEL)
   InvalidateIf(ctx, pattern, shouldDelete(key)) error      // Filtered batch invalidation (SCAN + predicate + DEL)
   ```

   `InvalidateIf` is used when a glob pattern alone is too broad — for example, evicting only planner cache keys whose date range overlaps a changed reservation, rather than clearing all planner keys for the property.

2. **Key prefixing** — All keys prefixed with configurable namespace (e.g., `"yop:"`) to:
   - Prevent collisions in shared Redis instance
   - Enable pattern-based invalidation (`yop:bookings:*`)

3. **JSON encoding** — Values automatically JSON-marshaled/unmarshaled
   - Type-safe retrieval into Go structs
   - Human-readable in Redis CLI

4. **GetOrSet read-through pattern** — Simplifies cache-aside logic:
   - If key exists in cache, return cached value
   - If miss or error, call loader function to fetch from database
   - Automatically store in cache; don't block on cache write errors

5. **Fail-graceful** — Cache errors don't block responses:
   - Redis unavailable? Log warning, call loader
   - Cache write fails? Log warning, return loaded value
   - Availability over consistency

6. **Pattern-based invalidation** — `SCAN + DEL` (or `SCAN + predicate + DEL`) approach:
   - Non-blocking; never uses `KEYS` which would block Redis
   - `Invalidate` clears all keys matching a glob pattern
   - `InvalidateIf` adds a predicate step for cases where the pattern alone over-selects
   - Logs deleted count for debugging

## Consequences

### ✅ Positive
* **Simple API** — Easy to use; handlers don't need complex cache logic
* **Fast operations** — Redis in-memory storage reduces database load dramatically
* **Long TTLs as safety net** — TTLs (e.g., 24h) are a fallback only; primary invalidation is event-driven (see ADR-010)
* **Type-safe** — JSON encoding ensures deserialization matches types
* **Pattern invalidation** — Efficient bulk cache clearing when related data changes
* **Graceful degradation** — Works without Redis (falls back to database)

### ⚠️ Negative
* **Eventual consistency** — During listener disconnect window, data may be stale until reconnect flush (see ADR-010)
* **Extra Redis calls** — GetOrSet makes two calls (GET + SET) per miss; minor overhead
* **Memory consumption** — Shared Redis instance means cache doesn't have unlimited capacity
* **No cache coherence** — Multiple instances can have inconsistent cache state during updates
* **JSON overhead** — Marshaling/unmarshaling adds CPU cost; alternatives (binary encoding) not supported

## Alternatives Considered

* **In-memory cache (sync.Map)** — Rejected because:
  - No TTL expiry; manual cleanup needed
  - Not shared across multiple instances (horizontal scaling breaks)
  - Unbounded growth in memory

* **Cache-Control headers with HTTP caching** — Rejected because:
  - Only works for read-only endpoints
  - Frontend caching is separate concern from backend optimization
  - Database is still hit on cache miss

* **Message queue + background cache warming** — Rejected because:
  - Over-engineered for current scale
  - Adds significant complexity
  - Can adopt later if cache misses become a bottleneck

## References

* `internal/platform/cache/cache.go` — Implementation
* [Redis documentation](https://redis.io/docs/)
* ADR-007: Idempotency Key Enforcement (uses Redis)
* ADR-009: OpenTelemetry (related; tracing cache hits/misses)
* ADR-010: Reactive Cache Invalidation (invalidation strategy)
