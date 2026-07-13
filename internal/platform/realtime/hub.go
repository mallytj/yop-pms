// Package realtime provides a Server-Sent Events hub that fans out PostgreSQL
// LISTEN/NOTIFY events to connected browser clients via HTTP.
//
// Core Requirements: [R-RES-INTEG-003], [ADR-011]
//
// The Hub registers as a handler on internal/platform/events.Listener (no own
// pgx connection). It uses a simple mutex to guard the client map — no select
// loop or background goroutine.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/events"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

const maxSSEClients = 1000

// SSEMessage is a single event delivered to one SSE client.
type SSEMessage struct {
	Event string // e.g. "reservation.changed", "staff.alert", "resync"
	Data  string // compact JSON payload
}

// Hub fans out events from the events.Listener to connected SSE clients.
// It is safe for concurrent use by multiple goroutines.
type Hub struct {
	mu      sync.Mutex
	clients map[chan SSEMessage]uuid.UUID // channel → property_id
	logger  *slog.Logger
}

// NewHub creates a new realtime Hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[chan SSEMessage]uuid.UUID),
		logger:  logger,
	}
}

// Subscribe upgrades the HTTP connection to an SSE stream. It blocks until
// the client disconnects (tab closed, navigate away).
//
// @Summary      Subscribe to real-time SSE events
// @Description  Opens a Server-Sent Events stream for real-time updates
// (reservations, availability, rates). EventSource clients must pass
// property_id as query param since EventSource cannot set custom headers.
// Auth is validated against the authenticated session via StubAuth.
// Heartbeat every 25s keeps reverse proxies from closing idle connections.
// @Tags         Realtime
// @Produce      text/event-stream
// @Param        X-Property-ID  header  string  true  "Property UUID from authenticated session"
// @Param        property_id    query   string  true  "Property UUID (must match authenticated session)"
// @Success      200            "SSE event stream — one ChangeEvent per line"
// @Failure      400            {object}  apierror.APIError  "property_id query parameter is required or invalid UUID"
// @Failure      401            {object}  apierror.APIError  "No authenticated session"
// @Failure      403            {object}  apierror.APIError  "Property access denied"
// @Failure      503            {object}  apierror.APIError  "SSE at capacity"
// @Router       /v1/sse [get]
//
// Auth: property_id is resolved from the query parameter
// (?property_id=...) because EventSource cannot set custom HTTP headers.
// StubAuth validates the resolved property against the authenticated session.
//
// Each client gets a 64-event buffered channel. A 25s heartbeat keeps
// reverse proxies from closing idle connections.
func (h *Hub) Subscribe(w http.ResponseWriter, r *http.Request) {
	// Extract and validate property_id from query param.
	propertyID, ok := h.resolvePropertyID(w, r)
	if !ok {
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		platformjson.WriteError(w, r, apierror.ErrInternal.WithMessage("streaming unsupported"))
		return
	}

	h.mu.Lock()
	if len(h.clients) >= maxSSEClients {
		h.mu.Unlock()
		platformjson.WriteError(w, r, apierror.New("SERVICE_UNAVAILABLE", "sse at capacity", http.StatusServiceUnavailable))
		return
	}

	// Buffered channel per client. Overflow → resync (slow consumer protection).
	ch := make(chan SSEMessage, 64)

	h.clients[ch] = propertyID
	h.mu.Unlock()

	h.logger.Info("sse client connected", "property_id", propertyID)

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		close(ch)
		h.logger.Info("sse client disconnected", "property_id", propertyID)
	}()

	// Heartbeat ticker to keep proxies from closing idle connections.
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case msg := <-ch:
			if msg.Event != "" {
				_, _ = w.Write([]byte("event: " + msg.Event + "\n"))
			}
			_, _ = w.Write([]byte("data: " + msg.Data + "\n\n"))
			flusher.Flush()

		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

// resolvePropertyID extracts and validates property_id from the SSE query
// parameter, then verifies the authenticated user has access to it.
func (h *Hub) resolvePropertyID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	rawPID := r.URL.Query().Get("property_id")
	if rawPID == "" {
		platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("property_id query parameter is required"))
		return uuid.Nil, false
	}
	propertyID, err := uuid.Parse(rawPID)
	if err != nil {
		platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("property_id must be a valid UUID"))
		return uuid.Nil, false
	}

	ctxPropertyID := helpers.GetPropertyIDFromCtx(r.Context())
	if ctxPropertyID == uuid.Nil {
		platformjson.WriteError(w, r, apierror.ErrUnauthorized.WithMessage("no authenticated session"))
		return uuid.Nil, false
	}
	if ctxPropertyID != propertyID {
		platformjson.WriteError(w, r, apierror.ErrForbidden.WithMessage("property access denied"))
		return uuid.Nil, false
	}
	return propertyID, true
}

// OnEvent is an events.Handler — register it via listener.On().
// It fans out the parsed event to all connected clients filtered by property_id.
func (h *Hub) OnEvent(_ context.Context, event events.Event) error {
	propertyIDRaw, ok := event.Data["property_id"].(string)
	if !ok {
		// Cannot filter without property_id — log and drop.
		h.logger.Warn("sse event missing property_id", "channel", event.Channel)
		return nil
	}

	propertyID, err := uuid.Parse(propertyIDRaw)
	if err != nil {
		h.logger.Warn("sse event has invalid property_id", "property_id", propertyIDRaw)
		return nil
	}

	msg := h.ConvertEventToSSEMessage(event)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.fanOut(propertyID, msg)

	return nil
}

// Resync sends a resync event to every connected client.
// Called from events.Listener's onReconnect callback after a pgx disconnect
// to cover the gap where events were missed.
func (h *Hub) Resync(_ context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.clients {
		select {
		case ch <- SSEMessage{Event: "resync", Data: "{}"}:
		default:
		}
	}

	h.logger.Info("sse resync broadcast", "client_count", len(h.clients))
}

// fanOut delivers msg to all connected clients for the given property.
// Slow consumer protection: if a client's 64-event buffer is full, send a
// resync signal instead of blocking. If even the resync channel is full,
// skip the client silently (connection likely dead).
func (h *Hub) fanOut(propertyID uuid.UUID, msg SSEMessage) {
	for ch, clientPropertyID := range h.clients {
		if clientPropertyID != propertyID {
			continue // skip — different property
		}
		select {
		case ch <- msg:
			// delivered
		default:
			// slow consumer — send resync instead of blocking.
			select {
			case ch <- SSEMessage{Event: "resync", Data: "{}"}:
			default:
				// channel is completely backed up — skip silently.
			}
		}
	}
}

// ConvertEventToSSEMessage converts an events.Event into an SSEMessage.
func (h *Hub) ConvertEventToSSEMessage(event events.Event) SSEMessage {
	table, _ := event.Data["table"].(string)
	op, _ := event.Data["op"].(string)
	id, _ := event.Data["id"].(string)
	propertyID, _ := event.Data["property_id"].(string)

	var eventName string
	switch table {
	case "reservations", "reservation_items":
		eventName = "reservation.changed"
	case "room_inventory_ledger":
		eventName = "availability.changed"
	case "booked_daily_rates":
		eventName = "rate.changed"
	default:
		eventName = "change"
	}

	payload := map[string]any{
		"property_id": propertyID,
		"record_id":   id,
		"op":          op,
		"at":          event.Timestamp.Format(time.RFC3339),
	}
	if v, ok := event.Data["version"]; ok {
		payload["version"] = v
	}
	data, _ := json.Marshal(payload)

	return SSEMessage{Event: eventName, Data: string(data)}
}
