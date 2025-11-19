package logger

import (
	"context"

	"github.com/rs/zerolog"
)

// ContextKey is the type for context keys used by this package
type ContextKey string

const (
	// LoggerContextKey is the context key for the logger
	LoggerContextKey ContextKey = "logger"

	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"

	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"

	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"

	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
)

// FromContext extracts a logger from context
// If no logger is found, returns the global logger
func FromContext(ctx context.Context) *zerolog.Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*zerolog.Logger); ok {
		return logger
	}

	// Return global logger as fallback
	globalLogger := zerolog.Ctx(ctx)
	return globalLogger
}

// WithContext adds logger to context
func WithContext(ctx context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, logger)
}

// WithRequestID adds request ID to context and logger
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx).With().Str("request_id", requestID).Logger()
	ctx = WithContext(ctx, &logger)
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds user ID to context and logger
func WithUserID(ctx context.Context, userID string) context.Context {
	logger := FromContext(ctx).With().Str("user_id", userID).Logger()
	ctx = WithContext(ctx, &logger)
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTraceID adds trace ID to context and logger
func WithTraceID(ctx context.Context, traceID string) context.Context {
	logger := FromContext(ctx).With().Str("trace_id", traceID).Logger()
	ctx = WithContext(ctx, &logger)
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context and logger
func WithSpanID(ctx context.Context, spanID string) context.Context {
	logger := FromContext(ctx).With().Str("span_id", spanID).Logger()
	ctx = WithContext(ctx, &logger)
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithFields adds multiple fields to context logger
func WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	logger := FromContext(ctx)
	logContext := logger.With()

	for k, v := range fields {
		logContext = logContext.Interface(k, v)
	}

	newLogger := logContext.Logger()
	return WithContext(ctx, &newLogger)
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID extracts span ID from context
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// Debug logs a debug message with context
func Debug(ctx context.Context) *zerolog.Event {
	return FromContext(ctx).Debug()
}

// Info logs an info message with context
func Info(ctx context.Context) *zerolog.Event {
	return FromContext(ctx).Info()
}

// Warn logs a warning message with context
func Warn(ctx context.Context) *zerolog.Event {
	return FromContext(ctx).Warn()
}

// Error logs an error message with context
func Error(ctx context.Context) *zerolog.Event {
	return FromContext(ctx).Error()
}

// Fatal logs a fatal message with context
func Fatal(ctx context.Context) *zerolog.Event {
	return FromContext(ctx).Fatal()
}
