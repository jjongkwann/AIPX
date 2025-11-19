package logger

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor for logging
func UnaryServerInterceptor(logger *Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)

		// Extract request ID from metadata (or generate one)
		requestID := extractMetadata(md, "x-request-id")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Create logger with context
		logContext := logger.With().
			Str("method", info.FullMethod).
			Str("request_id", requestID)

		// Extract trace ID if available
		if traceID := extractMetadata(md, "x-trace-id"); traceID != "" {
			logContext = logContext.Str("trace_id", traceID)
		}

		// Extract user ID if available
		if userID := extractMetadata(md, "x-user-id"); userID != "" {
			logContext = logContext.Str("user_id", userID)
		}

		contextLogger := logContext.Logger()

		// Add logger to context
		ctx = WithContext(ctx, &contextLogger)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Log request
		contextLogger.Info().
			Msg("gRPC request started")

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on error
		var logEvent *zerolog.Event
		if err != nil {
			st, _ := status.FromError(err)
			code := st.Code()

			if code == codes.Internal || code == codes.Unknown || code == codes.DataLoss {
				logEvent = contextLogger.Error()
			} else {
				logEvent = contextLogger.Warn()
			}

			logEvent = logEvent.
				Err(err).
				Str("grpc_code", code.String())
		} else {
			logEvent = contextLogger.Info()
		}

		// Log response
		logEvent.
			Dur("duration_ms", duration).
			Msg("gRPC request completed")

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for logging
func StreamServerInterceptor(logger *Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		ctx := ss.Context()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)

		// Extract request ID
		requestID := extractMetadata(md, "x-request-id")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Create logger with context
		logContext := logger.With().
			Str("method", info.FullMethod).
			Str("request_id", requestID).
			Bool("is_client_stream", info.IsClientStream).
			Bool("is_server_stream", info.IsServerStream)

		contextLogger := logContext.Logger()

		// Log stream start
		contextLogger.Info().
			Msg("gRPC stream started")

		// Call handler
		err := handler(srv, &loggedServerStream{
			ServerStream: ss,
			logger:       &contextLogger,
		})

		// Calculate duration
		duration := time.Since(start)

		// Log stream completion
		if err != nil {
			st, _ := status.FromError(err)
			contextLogger.Error().
				Err(err).
				Str("grpc_code", st.Code().String()).
				Dur("duration_ms", duration).
				Msg("gRPC stream completed with error")
		} else {
			contextLogger.Info().
				Dur("duration_ms", duration).
				Msg("gRPC stream completed")
		}

		return err
	}
}

// loggedServerStream wraps grpc.ServerStream with logging
type loggedServerStream struct {
	grpc.ServerStream
	logger *zerolog.Logger
}

func (s *loggedServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	return WithContext(ctx, s.logger)
}

// extractMetadata extracts a value from gRPC metadata
func extractMetadata(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// UnaryClientInterceptor returns a gRPC unary client interceptor for logging
func UnaryClientInterceptor(logger *Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		// Extract request ID from context
		requestID := GetRequestID(ctx)
		if requestID == "" {
			requestID = generateRequestID()
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
		}

		// Add request ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		// Create logger
		contextLogger := logger.With().
			Str("method", method).
			Str("request_id", requestID).
			Logger()

		contextLogger.Debug().
			Msg("gRPC client request started")

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)

		if err != nil {
			st, _ := status.FromError(err)
			contextLogger.Error().
				Err(err).
				Str("grpc_code", st.Code().String()).
				Dur("duration_ms", duration).
				Msg("gRPC client request failed")
		} else {
			contextLogger.Debug().
				Dur("duration_ms", duration).
				Msg("gRPC client request completed")
		}

		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor for logging
func StreamClientInterceptor(logger *Logger) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Extract request ID
		requestID := GetRequestID(ctx)
		if requestID == "" {
			requestID = generateRequestID()
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
		}

		// Add to metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		contextLogger := logger.With().
			Str("method", method).
			Str("request_id", requestID).
			Logger()

		contextLogger.Debug().
			Msg("gRPC client stream started")

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Msg("gRPC client stream creation failed")
			return nil, err
		}

		return stream, nil
	}
}
