package writer

import "errors"

var (
	// ErrWriterClosed is returned when writer is closed
	ErrWriterClosed = errors.New("writer is closed")

	// ErrInvalidData is returned for invalid data
	ErrInvalidData = errors.New("invalid data")

	// ErrUploadFailed is returned when S3 upload fails
	ErrUploadFailed = errors.New("upload failed")

	// ErrDatabaseError is returned for database errors
	ErrDatabaseError = errors.New("database error")
)
