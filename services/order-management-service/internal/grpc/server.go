package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pb "github.com/jjongkwann/aipx/shared/go/pkg/pb"
)

// Server represents the gRPC server
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	handler    *OrderServiceHandler
	port       string
}

// Config holds the server configuration
type Config struct {
	Port              string
	ShutdownTimeout   time.Duration
	EnableReflection  bool
}

// NewServer creates a new gRPC server
func NewServer(config *Config, handler *OrderServiceHandler) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			RecoveryInterceptor(),
			LoggingInterceptor(),
			AuthenticationInterceptor(),
			MetricsInterceptor(),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxConcurrentStreams(1000),
	)

	// Register service
	pb.RegisterOrderServiceServer(grpcServer, handler)

	// Enable reflection for development
	if config.EnableReflection {
		reflection.Register(grpcServer)
		log.Info().Msg("gRPC reflection enabled")
	}

	server := &Server{
		grpcServer: grpcServer,
		listener:   lis,
		handler:    handler,
		port:       config.Port,
	}

	log.Info().
		Str("port", config.Port).
		Msg("gRPC server created")

	return server, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	log.Info().
		Str("address", s.listener.Addr().String()).
		Msg("Starting gRPC server")

	if err := s.grpcServer.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	log.Info().Msg("Stopping gRPC server")

	// Create a channel to signal when shutdown is complete
	stopped := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-stopped:
		log.Info().Msg("gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		log.Warn().Msg("Forcing gRPC server shutdown")
		s.grpcServer.Stop()
		return ctx.Err()
	}
}
