package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Event is the payload delivered to handlers.
type Event struct {
	Channel   string         `json:"channel"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// Handler is a function that processes an event from a subscribed channel.
type Handler func(ctx context.Context, event Event) error

// Listener subscribes to PostgreSQL LISTEN/NOTIFY channels and dispatches
// events to registered handlers. It maintains a dedicated connection outside
// the pool (LISTEN blocks the connection) and reconnects automatically on failure.
type Listener struct {
	connStr     string
	handlers    map[string][]Handler
	mu          sync.RWMutex
	onReconnect func() // called after reconnecting — use to flush cache
	logger      *slog.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// New creates a new Listener. onReconnect is called after every reconnection
// to allow callers to flush stale cache state from the disconnect window.
// Register handlers with On() before calling Start().
func New(connStr string, logger *slog.Logger, onReconnect func()) *Listener {
	ctx, cancel := context.WithCancel(context.Background())
	return &Listener{
		connStr:     connStr,
		handlers:    make(map[string][]Handler),
		onReconnect: onReconnect,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// On registers a handler for the given channel. Handlers must be registered
// before Start() — channels are subscribed on connect, not on registration.
func (l *Listener) On(channel string, handler Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers[channel] = append(l.handlers[channel], handler)
	l.logger.Debug("registered event handler", "channel", channel, "total", len(l.handlers[channel]))
}

// Start begins listening for notifications in a background goroutine.
func (l *Listener) Start() {
	l.wg.Add(1)
	go l.run()
	l.logger.Info("event listener started")
}

// Stop cancels the listener and waits for all in-flight handlers to complete.
func (l *Listener) Stop() {
	l.logger.Info("stopping event listener")
	l.cancel()
	l.wg.Wait()
	l.logger.Info("event listener stopped")
}

// run is the main reconnection loop. On any connection error it backs off
// exponentially (1s → 60s) and reconnects. On reconnect it calls onReconnect
// to flush stale cache state from the disconnect window.
func (l *Listener) run() {
	defer l.wg.Done()

	backoff := time.Second
	isReconnect := false

	for {
		if l.ctx.Err() != nil {
			return
		}

		if err := l.connect(isReconnect); err != nil {
			if l.ctx.Err() != nil {
				return // clean shutdown
			}
			l.logger.Warn("event listener disconnected, will reconnect",
				"error", err,
				"backoff", backoff,
			)
			select {
			case <-l.ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < time.Minute {
					backoff *= 2
				}
			}
		} else {
			backoff = time.Second
		}

		isReconnect = true
	}
}

// connect establishes a dedicated connection, subscribes to all registered
// channels, and blocks processing notifications until the connection fails
// or the listener is stopped. On reconnect it calls onReconnect after
// successfully subscribing so no notifications are missed after the flush.
func (l *Listener) connect(isReconnect bool) error {
	conn, err := pgx.Connect(l.ctx, l.connStr)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(context.Background())

	l.mu.RLock()
	channels := make([]string, 0, len(l.handlers))
	for ch := range l.handlers {
		channels = append(channels, ch)
	}
	l.mu.RUnlock()

	for _, ch := range channels {
		if _, err := conn.Exec(l.ctx, fmt.Sprintf("LISTEN %q", ch)); err != nil {
			return fmt.Errorf("listen %s: %w", ch, err)
		}
	}

	l.logger.Info("event listener connected", "channels", channels)

	if isReconnect {
		l.reconnect()
	}

	return l.processNotifications(conn)
}

// processNotifications blocks, receiving notifications and dispatching them.
// A 90s timeout is used on each wait — on timeout, the connection is pinged
// to detect silent disconnects without a separate goroutine.
func (l *Listener) processNotifications(conn *pgx.Conn) error {
	for {
		waitCtx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
		n, err := conn.WaitForNotification(waitCtx)
		cancel()

		if err != nil {
			if l.ctx.Err() != nil {
				return nil // clean shutdown
			}
			if waitCtx.Err() != nil {
				// Timeout — ping to verify the connection is still alive
				if pingErr := conn.Ping(l.ctx); pingErr != nil {
					return fmt.Errorf("connection lost: %w", pingErr)
				}
				continue
			}
			return fmt.Errorf("wait for notification: %w", err)
		}

		l.dispatch(n)
	}
}

// dispatch parses the notification payload and calls all registered handlers
// for the channel concurrently. Handler errors are logged but do not stop
// other handlers. Invalid JSON is logged and silently dropped.
func (l *Listener) dispatch(n *pgconn.Notification) {
	var data map[string]any
	if err := json.Unmarshal([]byte(n.Payload), &data); err != nil {
		l.logger.Error("failed to parse notification payload",
			"channel", n.Channel,
			"payload", n.Payload,
			"error", err,
		)
		return
	}

	event := Event{
		Channel:   n.Channel,
		Data:      data,
		Timestamp: time.Now(),
	}

	l.logger.Debug("received event", "channel", event.Channel)

	l.mu.RLock()
	handlers := make([]Handler, len(l.handlers[n.Channel]))
	copy(handlers, l.handlers[n.Channel])
	l.mu.RUnlock()

	for _, h := range handlers {
		l.wg.Add(1)
		go func(handler Handler) {
			defer l.wg.Done()
			if err := handler(l.ctx, event); err != nil {
				l.logger.Error("event handler error", "channel", event.Channel, "error", err)
			}
		}(h)
	}
}

// reconnect calls the onReconnect callback if set, logging the flush.
func (l *Listener) reconnect() {
	if l.onReconnect == nil {
		return
	}
	l.logger.Info("event listener reconnected — flushing cache")
	l.onReconnect()
}
