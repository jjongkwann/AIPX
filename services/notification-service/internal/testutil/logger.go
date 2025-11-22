package testutil

import (
	"shared/pkg/logger"
)

// NewTestLogger creates a logger configured for testing
func NewTestLogger() *logger.Logger {
	config := &logger.Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		ServiceName: "test",
		Environment: "test",
	}

	log, err := logger.NewLogger(config)
	if err != nil {
		panic(err)
	}

	return log
}

// NewSilentLogger creates a logger that doesn't output anything
func NewSilentLogger() *logger.Logger {
	config := &logger.Config{
		Level:       "error",
		Format:      "json",
		Output:      "/dev/null",
		ServiceName: "test",
		Environment: "test",
	}

	log, err := logger.NewLogger(config)
	if err != nil {
		panic(err)
	}

	return log
}
