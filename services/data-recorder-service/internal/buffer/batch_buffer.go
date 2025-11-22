package buffer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"data-recorder-service/internal/pkg/pb"
)

// MessageType represents the type of market data message
type MessageType int

const (
	MessageTypeTick MessageType = iota
	MessageTypeOrderBook
)

// BufferedMessage wraps a market data message with metadata
type BufferedMessage struct {
	Type      MessageType
	Timestamp time.Time
	Tick      *pb.TickData
	OrderBook *pb.OrderBook
}

// FlushCallback is called when a batch is ready to be flushed
type FlushCallback func(messages []BufferedMessage) error

// BatchBuffer provides thread-safe buffering with automatic flushing
type BatchBuffer struct {
	// Configuration
	maxSize      int
	flushTimeout time.Duration
	callback     FlushCallback

	// Buffer state
	mu       sync.RWMutex
	buffer   []BufferedMessage
	lastFlow time.Time

	// Channels
	flushCh chan struct{}
	stopCh  chan struct{}
	doneCh  chan struct{}

	// Metrics
	totalMessages   atomic.Int64
	totalFlushes    atomic.Int64
	totalErrors     atomic.Int64
	currentSize     atomic.Int32
	droppedMessages atomic.Int64

	// Control
	running atomic.Bool
}

// NewBatchBuffer creates a new batch buffer
func NewBatchBuffer(maxSize int, flushTimeout time.Duration, callback FlushCallback) *BatchBuffer {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if flushTimeout <= 0 {
		flushTimeout = 5 * time.Second
	}

	bb := &BatchBuffer{
		maxSize:      maxSize,
		flushTimeout: flushTimeout,
		callback:     callback,
		buffer:       make([]BufferedMessage, 0, maxSize),
		lastFlow:     time.Now(),
		flushCh:      make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	bb.running.Store(true)
	return bb
}

// Start starts the background flush goroutine
func (bb *BatchBuffer) Start(ctx context.Context) {
	ticker := time.NewTicker(bb.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			bb.finalFlush()
			close(bb.doneCh)
			return

		case <-bb.stopCh:
			bb.finalFlush()
			close(bb.doneCh)
			return

		case <-ticker.C:
			bb.checkAndFlush(false)

		case <-bb.flushCh:
			bb.checkAndFlush(true)
		}
	}
}

// Add adds a message to the buffer
func (bb *BatchBuffer) Add(msg BufferedMessage) error {
	if !bb.running.Load() {
		bb.droppedMessages.Add(1)
		return ErrBufferClosed
	}

	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Check buffer capacity
	if len(bb.buffer) >= bb.maxSize*2 {
		// Buffer overflow - drop message
		bb.droppedMessages.Add(1)
		return ErrBufferFull
	}

	bb.buffer = append(bb.buffer, msg)
	bb.totalMessages.Add(1)
	bb.currentSize.Store(int32(len(bb.buffer)))

	// Trigger flush if size threshold reached
	if len(bb.buffer) >= bb.maxSize {
		select {
		case bb.flushCh <- struct{}{}:
		default:
			// Flush already pending
		}
	}

	return nil
}

// AddTick adds a tick data message to the buffer
func (bb *BatchBuffer) AddTick(tick *pb.TickData) error {
	return bb.Add(BufferedMessage{
		Type:      MessageTypeTick,
		Timestamp: time.Now(),
		Tick:      tick,
	})
}

// AddOrderBook adds an orderbook message to the buffer
func (bb *BatchBuffer) AddOrderBook(orderbook *pb.OrderBook) error {
	return bb.Add(BufferedMessage{
		Type:      MessageTypeOrderBook,
		Timestamp: time.Now(),
		OrderBook: orderbook,
	})
}

// checkAndFlush checks if flush is needed and performs it
func (bb *BatchBuffer) checkAndFlush(forced bool) {
	bb.mu.Lock()

	if len(bb.buffer) == 0 {
		bb.mu.Unlock()
		return
	}

	// Check if should flush
	shouldFlush := forced ||
		len(bb.buffer) >= bb.maxSize ||
		time.Since(bb.lastFlow) >= bb.flushTimeout

	if !shouldFlush {
		bb.mu.Unlock()
		return
	}

	// Get messages to flush
	messages := make([]BufferedMessage, len(bb.buffer))
	copy(messages, bb.buffer)

	// Clear buffer
	bb.buffer = bb.buffer[:0]
	bb.lastFlow = time.Now()
	bb.currentSize.Store(0)

	bb.mu.Unlock()

	// Flush outside of lock
	bb.flush(messages)
}

// flush performs the actual flush operation
func (bb *BatchBuffer) flush(messages []BufferedMessage) {
	if len(messages) == 0 {
		return
	}

	bb.totalFlushes.Add(1)

	if err := bb.callback(messages); err != nil {
		bb.totalErrors.Add(1)
		// TODO: Implement retry logic or dead letter queue
	}
}

// finalFlush flushes remaining messages on shutdown
func (bb *BatchBuffer) finalFlush() {
	bb.mu.Lock()
	if len(bb.buffer) == 0 {
		bb.mu.Unlock()
		return
	}

	messages := make([]BufferedMessage, len(bb.buffer))
	copy(messages, bb.buffer)
	bb.buffer = bb.buffer[:0]
	bb.currentSize.Store(0)
	bb.mu.Unlock()

	// Final flush with retries
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		if err := bb.callback(messages); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
}

// Stop stops the buffer and flushes remaining messages
func (bb *BatchBuffer) Stop() {
	if !bb.running.CompareAndSwap(true, false) {
		return
	}

	close(bb.stopCh)
	<-bb.doneCh
}

// Flush forces an immediate flush
func (bb *BatchBuffer) Flush() {
	select {
	case bb.flushCh <- struct{}{}:
	default:
	}
}

// Metrics returns buffer metrics
func (bb *BatchBuffer) Metrics() BufferMetrics {
	return BufferMetrics{
		TotalMessages:   bb.totalMessages.Load(),
		TotalFlushes:    bb.totalFlushes.Load(),
		TotalErrors:     bb.totalErrors.Load(),
		CurrentSize:     int(bb.currentSize.Load()),
		DroppedMessages: bb.droppedMessages.Load(),
	}
}

// Size returns current buffer size
func (bb *BatchBuffer) Size() int {
	return int(bb.currentSize.Load())
}

// IsFull returns true if buffer is near capacity
func (bb *BatchBuffer) IsFull() bool {
	return bb.Size() >= bb.maxSize
}

// BufferMetrics contains buffer statistics
type BufferMetrics struct {
	TotalMessages   int64
	TotalFlushes    int64
	TotalErrors     int64
	CurrentSize     int
	DroppedMessages int64
}
