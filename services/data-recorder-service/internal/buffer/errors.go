package buffer

import "errors"

var (
	// ErrBufferFull is returned when buffer is full
	ErrBufferFull = errors.New("buffer is full")

	// ErrBufferClosed is returned when buffer is closed
	ErrBufferClosed = errors.New("buffer is closed")

	// ErrInvalidMessage is returned for invalid messages
	ErrInvalidMessage = errors.New("invalid message")
)
