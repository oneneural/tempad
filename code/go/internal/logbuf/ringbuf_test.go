package logbuf

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_BasicWriteRead(t *testing.T) {
	rb := NewRingBuffer(5)

	rb.Write("line 1", "stdout")
	rb.Write("line 2", "stderr")
	rb.Write("line 3", "tempad")

	assert.Equal(t, 3, rb.Len())

	lines := rb.Lines(0)
	require.Len(t, lines, 3)
	assert.Equal(t, "line 1", lines[0].Text)
	assert.Equal(t, "stdout", lines[0].Stream)
	assert.Equal(t, "line 2", lines[1].Text)
	assert.Equal(t, "stderr", lines[1].Stream)
	assert.Equal(t, "line 3", lines[2].Text)
	assert.Equal(t, "tempad", lines[2].Stream)
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Write("a", "stdout")
	rb.Write("b", "stdout")
	rb.Write("c", "stdout")
	rb.Write("d", "stdout") // evicts "a"
	rb.Write("e", "stdout") // evicts "b"

	assert.Equal(t, 5, rb.Len())

	lines := rb.Lines(0)
	require.Len(t, lines, 3)
	assert.Equal(t, "c", lines[0].Text)
	assert.Equal(t, "d", lines[1].Text)
	assert.Equal(t, "e", lines[2].Text)
}

func TestRingBuffer_IncrementalRead(t *testing.T) {
	rb := NewRingBuffer(10)

	rb.Write("a", "stdout")
	rb.Write("b", "stdout")
	rb.Write("c", "stdout")

	// Read from offset 0 → 3 lines
	lines := rb.Lines(0)
	require.Len(t, lines, 3)

	// Read from offset 2 → 1 line
	lines = rb.Lines(2)
	require.Len(t, lines, 1)
	assert.Equal(t, "c", lines[0].Text)

	// Read from offset 3 → no new lines
	lines = rb.Lines(3)
	assert.Nil(t, lines)

	// Write more, read from offset 3
	rb.Write("d", "stderr")
	lines = rb.Lines(3)
	require.Len(t, lines, 1)
	assert.Equal(t, "d", lines[0].Text)
}

func TestRingBuffer_OffsetBeyondOldest(t *testing.T) {
	rb := NewRingBuffer(3)

	// Write 5 lines: oldest available will be at offset 2.
	for i := 0; i < 5; i++ {
		rb.Write(fmt.Sprintf("line-%d", i), "stdout")
	}

	// Offset 0 is before oldest — should get from oldest (offset 2).
	lines := rb.Lines(0)
	require.Len(t, lines, 3)
	assert.Equal(t, "line-2", lines[0].Text)
	assert.Equal(t, "line-3", lines[1].Text)
	assert.Equal(t, "line-4", lines[2].Text)
}

func TestRingBuffer_DefaultCapacity(t *testing.T) {
	rb := NewRingBuffer(0)
	assert.Equal(t, 0, rb.Len())

	// Should use DefaultCapacity.
	for i := 0; i < DefaultCapacity+10; i++ {
		rb.Write(fmt.Sprintf("line-%d", i), "stdout")
	}
	assert.Equal(t, DefaultCapacity+10, rb.Len())

	lines := rb.Lines(0)
	assert.Len(t, lines, DefaultCapacity)
}

func TestRingBuffer_EmptyRead(t *testing.T) {
	rb := NewRingBuffer(5)
	lines := rb.Lines(0)
	assert.Nil(t, lines)
}

func TestRingBuffer_ConcurrentWriteRead(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writers.
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				rb.Write(fmt.Sprintf("w%d-line-%d", id, i), "stdout")
			}
		}(w)
	}

	// Concurrent readers.
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = rb.Lines(0)
				_ = rb.Len()
			}
		}()
	}

	wg.Wait()

	// Total written should be 10 * 100 = 1000.
	assert.Equal(t, 1000, rb.Len())

	// Buffer should have exactly 100 lines.
	lines := rb.Lines(0)
	assert.Len(t, lines, 100)
}
