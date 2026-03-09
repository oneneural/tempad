// Package logbuf provides a thread-safe ring buffer for per-issue log lines.
package logbuf

import (
	"sync"
	"time"
)

// DefaultCapacity is the default number of log lines a ring buffer holds.
const DefaultCapacity = 1000

// LogLine represents a single log line with metadata.
type LogLine struct {
	// Time is when the line was recorded.
	Time time.Time `json:"time"`

	// Text is the log line content.
	Text string `json:"text"`

	// Stream identifies the source: "stdout", "stderr", or "tempad".
	Stream string `json:"stream"`
}

// RingBuffer is a thread-safe, fixed-capacity ring buffer for log lines.
// It uses a monotonically increasing counter so consumers can do incremental
// reads without missing lines.
type RingBuffer struct {
	mu       sync.RWMutex
	lines    []LogLine
	capacity int
	total    int // monotonic counter of all lines ever written
}

// NewRingBuffer creates a ring buffer with the given capacity.
// If capacity <= 0, DefaultCapacity is used.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &RingBuffer{
		lines:    make([]LogLine, 0, capacity),
		capacity: capacity,
	}
}

// Write appends a log line to the buffer. If the buffer is full, the oldest
// line is evicted.
func (rb *RingBuffer) Write(text, stream string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	line := LogLine{
		Time:   time.Now(),
		Text:   text,
		Stream: stream,
	}

	if len(rb.lines) < rb.capacity {
		rb.lines = append(rb.lines, line)
	} else {
		// Overwrite at position total % capacity.
		rb.lines[rb.total%rb.capacity] = line
	}
	rb.total++
}

// Len returns the monotonic count of all lines ever written.
// This can be used as an offset for incremental reads.
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.total
}

// Lines returns log lines starting from the given offset (based on Len()).
// If offset is before the oldest available line, returns from the oldest.
// Returns nil if offset >= Len() (no new lines).
func (rb *RingBuffer) Lines(offset int) []LogLine {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if offset >= rb.total {
		return nil
	}

	// Determine the oldest available offset.
	oldest := 0
	if rb.total > rb.capacity {
		oldest = rb.total - rb.capacity
	}
	if offset < oldest {
		offset = oldest
	}

	count := rb.total - offset
	result := make([]LogLine, count)

	for i := 0; i < count; i++ {
		idx := (offset + i) % rb.capacity
		result[i] = rb.lines[idx]
	}

	return result
}
