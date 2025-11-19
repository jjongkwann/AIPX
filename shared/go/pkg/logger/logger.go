package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds logger configuration
type Config struct {
	// Level specifies the minimum log level (trace, debug, info, warn, error, fatal, panic)
	Level string

	// Format specifies output format (json or console)
	Format string

	// Output specifies where to write logs (stdout, stderr, or file path)
	Output string

	// TimeFormat specifies time format for logs
	TimeFormat string

	// ServiceName is the name of the service (added to all logs)
	ServiceName string

	// Environment specifies the environment (dev, staging, prod)
	Environment string

	// EnableCaller adds caller information to logs
	EnableCaller bool

	// CallerSkipFrameCount adjusts caller frame count
	CallerSkipFrameCount int
}

// Logger wraps zerolog.Logger with additional context
type Logger struct {
	logger zerolog.Logger
	config *Config
}

// NewLogger creates a new logger instance
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Parse log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}

	// Set time format
	if config.TimeFormat != "" {
		zerolog.TimeFieldFormat = config.TimeFormat
	} else {
		zerolog.TimeFieldFormat = time.RFC3339
	}

	// Set output
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	// Set format
	if config.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Create logger
	logger := zerolog.New(output).Level(level).With().Timestamp()

	// Add service name
	if config.ServiceName != "" {
		logger = logger.Str("service", config.ServiceName)
	}

	// Add environment
	if config.Environment != "" {
		logger = logger.Str("env", config.Environment)
	}

	// Add caller
	if config.EnableCaller {
		logger = logger.Caller()
	}

	return &Logger{
		logger: logger.Logger(),
		config: config,
	}, nil
}

// InitLogger initializes the global logger
func InitLogger(config *Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}

	log.Logger = logger.logger

	log.Info().
		Str("level", config.Level).
		Str("format", config.Format).
		Str("output", config.Output).
		Msg("Logger initialized")

	return nil
}

// GetLogger returns the underlying zerolog.Logger
func (l *Logger) GetLogger() zerolog.Logger {
	return l.logger
}

// With creates a child logger with additional context
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

// Level creates a new logger with specified level
func (l *Logger) Level(level zerolog.Level) *Logger {
	return &Logger{
		logger: l.logger.Level(level),
		config: l.config,
	}
}

// Debug logs a debug message
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info logs an info message
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn logs a warning message
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error logs an error message
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// Panic logs a panic message and panics
func (l *Logger) Panic() *zerolog.Event {
	return l.logger.Panic()
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:                "info",
		Format:               "json",
		Output:               "stdout",
		TimeFormat:           time.RFC3339,
		ServiceName:          "aipx",
		Environment:          "development",
		EnableCaller:         true,
		CallerSkipFrameCount: 0,
	}
}

// ProductionConfig returns production logger configuration
func ProductionConfig(serviceName string) *Config {
	return &Config{
		Level:                "info",
		Format:               "json",
		Output:               "stdout",
		TimeFormat:           time.RFC3339,
		ServiceName:          serviceName,
		Environment:          "production",
		EnableCaller:         false,
		CallerSkipFrameCount: 0,
	}
}

// DevelopmentConfig returns development logger configuration
func DevelopmentConfig(serviceName string) *Config {
	return &Config{
		Level:                "debug",
		Format:               "console",
		Output:               "stdout",
		TimeFormat:           time.RFC3339,
		ServiceName:          serviceName,
		Environment:          "development",
		EnableCaller:         true,
		CallerSkipFrameCount: 0,
	}
}
