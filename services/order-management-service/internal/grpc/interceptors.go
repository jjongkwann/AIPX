package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request-id"
	userIDKey    contextKey = "user-id"
)

// LoggingInterceptor logs incoming requests and responses
func LoggingInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		requestID := uuid.New().String()

		// Get metadata from context
		md, ok := metadata.FromIncomingContext(ss.Context())
		var userID string
		if ok {
			if ids := md.Get("user-id"); len(ids) > 0 {
				userID = ids[0]
			}
		}

		log.Info().
			Str("request_id", requestID).
			Str("user_id", userID).
			Str("method", info.FullMethod).
			Msg("Starting stream")

		// Add request ID to context
		ctx := context.WithValue(ss.Context(), requestIDKey, requestID)
		ctx = context.WithValue(ctx, userIDKey, userID)
		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}

		// Call handler
		err := handler(srv, wrapped)

		// Log completion
		duration := time.Since(start)
		if err != nil {
			log.Error().
				Err(err).
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("Stream completed with error")
		} else {
			log.Info().
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("Stream completed successfully")
		}

		return err
	}
}

// RecoveryInterceptor recovers from panics
func RecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("method", info.FullMethod).
					Msg("Recovered from panic")

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}

// AuthenticationInterceptor validates JWT tokens
func AuthenticationInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Get metadata
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata not found")
		}

		// Get authorization token
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			// In development, allow requests without auth
			log.Warn().Str("method", info.FullMethod).Msg("No authorization header, allowing in dev mode")
			return handler(srv, ss)
		}

		// Validate token (simplified - in production, use proper JWT validation)
		token := authHeaders[0]
		userID, err := validateToken(token)
		if err != nil {
			log.Warn().Err(err).Msg("Token validation failed")
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add user ID to context
		ctx := context.WithValue(ss.Context(), userIDKey, userID)
		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}

		return handler(srv, wrapped)
	}
}

// MetricsInterceptor collects metrics for Prometheus
func MetricsInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call handler
		err := handler(srv, ss)

		// Record metrics
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		// In production, increment Prometheus counters here
		log.Debug().
			Str("method", info.FullMethod).
			Str("status", statusCode.String()).
			Dur("duration", duration).
			Msg("Metrics recorded")

		return err
	}
}

// validateToken validates a JWT token and returns the user ID
func validateToken(token string) (string, error) {
	// This is a simplified stub - in production, use proper JWT validation
	// with libraries like github.com/golang-jwt/jwt

	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// For development, extract user ID from token
	// In production, verify signature and expiration
	log.Debug().Msg("Token validation not fully implemented, returning mock user ID")
	return "user-123", nil
}

// wrappedStream wraps grpc.ServerStream to inject custom context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
