package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lib/pq"
)

type Event struct {
	Channel   string                 `json:"channel"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type EventHandler func(ctx context.Context, event Event) error

type EventListener struct {
	listener *pq.Listener
	handlers map[string][]EventHandler // channel -> handler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *slog.Logger
}

func NewEventListener(dbConnStr string, logger *slog.Logger) *EventListener {
	ctx, cancel := context.WithCancel(context.Background())

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logger.Error("Postgres listener error", "error", err, "event", ev)
		}
	}

	listener := pq.NewListener(
		dbConnStr,
		10*time.Second, // minReconnectInterval
		time.Minute,    // maxReconnectInterval
		reportProblem,  // Callback
	)

	return &EventListener{
		listener: listener,
		handlers: make(map[string][]EventHandler),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
	}
}

func (el *EventListener) On(channel string, handler EventHandler) error {
	// locks the read write for use in goroutines
	el.mu.Lock()
	defer el.mu.Unlock()
	el.logger.Debug("Registering handler", "channel", channel)

	firstHandler := len(el.handlers[channel]) == 0

	if firstHandler {
		if err := el.listener.Listen(channel); err != nil {
			return fmt.Errorf("Failed to listen to channel %s: %w", channel, err)
		}
		el.logger.Info("Subscribed to channel", "channel", channel)
	}

	// Add new handler to channel
	el.handlers[channel] = append(el.handlers[channel], handler)

	el.logger.Debug("Registered handler",
		"channel", channel,
		"handler_count", len(el.handlers[channel]))

	return nil
}

func (el *EventListener) Start() error {
	// Add a process to the wait group
	el.wg.Add(1)
	go el.processEvents()

	el.logger.Info("Event listener started")
	return nil
}

func (el *EventListener) Stop() error {
	el.logger.Info("Stopping event listener")

	// Cancel context
	el.cancel()

	// Wait until all tasks are completed
	el.wg.Wait()

	el.listener.Close()

	el.logger.Info("Event listener stopped")

	return nil
}

func (el *EventListener) processEvents() {
	defer el.wg.Done()

	pingTicker := time.NewTicker(90 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-el.ctx.Done():
			return
		case noti := <-el.listener.Notify:
			if noti == nil {
				continue // Reconnection, skip nil
			}

			el.handleNotification(noti)
		case <-pingTicker.C:
			// Lightweight ping
			go func() {
				if err := el.listener.Ping(); err != nil {
					el.logger.Warn("Ping failed", "error", err)
				}
			}()
		}
	}
}

func (el *EventListener) handleNotification(n *pq.Notification) {
	// Parse payload
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(n.Extra), &data); err != nil {
		el.logger.Error("Failed to unmarshal event",
			"error", err,
			"channel", n.Channel,
			"payload", n.Extra)
		return
	}

	event := Event{
		Channel:   n.Channel,
		Data:      data,
		Timestamp: time.Now(),
	}

	el.logger.Debug("Received event",
		"channel", event.Channel,
		"data", event.Data)

	// Lock reading
	el.mu.RLock()
	handlers := el.handlers[n.Channel]
	el.mu.RUnlock()

	for _, handler := range handlers {
		el.wg.Add(1)
		// Run each handler function
		go func(h EventHandler) {
			defer el.wg.Done()

			if err := h(el.ctx, event); err != nil {
				el.logger.Error("Handler error",
					"channel", event.Channel,
					"error", err)
			}
		}(handler)
	}
}
