package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // json or console
	Output     string // stdout, stderr, or file path
	TimeFormat string
}

// InitLogger initializes the global logger
func InitLogger(config *Config) error {
	// Set log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

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
			return err
		}
		output = file
	}

	// Set format
	if config.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	log.Info().
		Str("level", level.String()).
		Str("format", config.Format).
		Str("output", config.Output).
		Msg("Logger initialized")

	return nil
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
	}
}

// DebugConfig returns debug logger configuration (console format)
func DebugConfig() *Config {
	return &Config{
		Level:      "debug",
		Format:     "console",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
	}
}
