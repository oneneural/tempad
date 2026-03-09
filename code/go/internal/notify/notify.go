// Package notify provides native OS desktop notifications for TEMPAD lifecycle events.
// It uses platform-specific system commands (osascript, notify-send, PowerShell)
// with zero external Go dependencies.
package notify

import (
	"log/slog"
	"sync"
	"time"
)

// Event identifies a notification event type.
type Event string

const (
	EventNewTask          Event = "new_task"
	EventAgentStarted     Event = "agent_started"
	EventAgentCompleted   Event = "agent_completed"
	EventAgentFailed      Event = "agent_failed"
	EventRetriesExhausted Event = "retries_exhausted"
	EventClaimFailed      Event = "claim_failed"
)

// AllEvents returns all supported notification events.
func AllEvents() []Event {
	return []Event{
		EventNewTask,
		EventAgentStarted,
		EventAgentCompleted,
		EventAgentFailed,
		EventRetriesExhausted,
		EventClaimFailed,
	}
}

// Config holds notification configuration.
type Config struct {
	Enabled bool     // whether notifications are enabled (default: true)
	Events  []string // which events trigger notifications (empty = all)
}

// Notifier sends desktop notifications. It is safe for concurrent use.
type Notifier struct {
	enabled    bool
	eventSet   map[Event]bool
	logger     *slog.Logger
	sendFn     func(title, body string) error // platform-specific send function
	mu         sync.Mutex
	lastSendAt time.Time
	minInterval time.Duration
}

// New creates a Notifier that respects the enabled flag and event filter.
// The logger is used for non-blocking error reporting.
func New(cfg Config, logger *slog.Logger) *Notifier {
	if logger == nil {
		logger = slog.Default()
	}

	eventSet := make(map[Event]bool)
	if len(cfg.Events) == 0 {
		// All events enabled by default.
		for _, e := range AllEvents() {
			eventSet[e] = true
		}
	} else {
		for _, e := range cfg.Events {
			eventSet[Event(e)] = true
		}
	}

	return &Notifier{
		enabled:     cfg.Enabled,
		eventSet:    eventSet,
		logger:      logger,
		sendFn:      platformSend,
		minInterval: 1 * time.Second,
	}
}

// Noop returns a notifier that does nothing.
func Noop() *Notifier {
	return &Notifier{
		enabled:  false,
		eventSet: make(map[Event]bool),
		logger:   slog.Default(),
		sendFn:   func(_, _ string) error { return nil },
	}
}

// Send fires a desktop notification for the given event. It is non-blocking:
// the notification is sent in a goroutine and errors are logged, never returned.
// Notifications are rate-limited to at most one per second; excess calls are dropped.
func (n *Notifier) Send(event Event, title, body string) {
	if !n.enabled {
		return
	}
	if !n.eventSet[event] {
		return
	}

	n.mu.Lock()
	now := time.Now()
	if now.Sub(n.lastSendAt) < n.minInterval {
		n.mu.Unlock()
		n.logger.Debug("notification rate-limited", "event", string(event), "title", title)
		return
	}
	n.lastSendAt = now
	n.mu.Unlock()

	go func() {
		if err := n.sendFn(title, body); err != nil {
			n.logger.Warn("notification send failed",
				"event", string(event),
				"title", title,
				"error", err,
			)
		}
	}()
}
