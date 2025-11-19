package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"user-service/internal/auth"
	"user-service/internal/config"
	"user-service/internal/handlers"
	"user-service/internal/middleware"
	"user-service/internal/models"
	"user-service/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger
	if err := initLogger(cfg); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	log.Info().
		Str("environment", cfg.Server.Environment).
		Int("port", cfg.Server.Port).
		Msg("Starting User Service")

	// Initialize database connection pool
	dbPool, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbPool.Close()

	// Initialize repositories
	repos := initRepositories(dbPool)

	// Initialize password hasher
	passwordHasher := auth.NewArgon2Hasher(auth.DefaultArgon2Params())

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
		cfg.JWT.Issuer,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize JWT manager")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(
		repos.User,
		repos.RefreshToken,
		passwordHasher,
		jwtManager,
	)
	userHandler := handlers.NewUserHandler(repos.User)

	// Initialize HTTP router
	router := setupRouter(cfg, jwtManager, authHandler, userHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("host", cfg.Server.Host).
			Int("port", cfg.Server.Port).
			Msg("HTTP server listening")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

// initLogger initializes the logger based on configuration
func initLogger(cfg *config.Config) error {
	// The shared logger package is already initialized globally
	// We just log the configuration
	log.Info().
		Str("level", cfg.Logger.Level).
		Str("format", cfg.Logger.Format).
		Msg("Logger initialized")
	return nil
}

// initDatabase initializes the PostgreSQL connection pool
func initDatabase(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("Database connection established")
	return pool, nil
}

// Repositories holds all repository instances
type Repositories struct {
	User         repository.UserRepository
	APIKey       repository.APIKeyRepository
	RefreshToken repository.RefreshTokenRepository
	AuditLog     repository.AuditLogRepository
}

// initRepositories initializes all repository instances
func initRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:         repository.NewPostgresUserRepository(pool),
		APIKey:       repository.NewPostgresAPIKeyRepository(pool),
		RefreshToken: repository.NewPostgresRefreshTokenRepository(pool),
		AuditLog:     repository.NewPostgresAuditLogRepository(pool),
	}
}

// setupRouter configures the HTTP router with middleware and routes
func setupRouter(
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.CORSMiddleware(getCORSConfig(cfg)))
	r.Use(chimiddleware.Compress(5))

	// Health check endpoint
	r.Get("/health", healthCheckHandler)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public authentication routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authHandler.Signup)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// Protected user routes
		r.Route("/users", func(r chi.Router) {
			// Apply JWT authentication middleware
			r.Use(middleware.AuthMiddleware(jwtManager))

			r.Get("/me", userHandler.GetMe)
			r.Patch("/me", userHandler.UpdateMe)
			r.Delete("/me", userHandler.DeleteMe)
		})
	})

	return r
}

// getCORSConfig returns CORS configuration based on environment
func getCORSConfig(cfg *config.Config) *middleware.CORSConfig {
	if cfg.Server.IsProduction() {
		// In production, specify allowed origins
		return middleware.ProductionCORSConfig([]string{
			"https://yourdomain.com",
			// Add more allowed origins
		})
	}
	return middleware.DefaultCORSConfig()
}

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := models.NewSuccessResponse(map[string]interface{}{
		"status":  "ok",
		"service": "user-service",
		"version": "1.0.0",
	}, "Service is healthy")

	json.NewEncoder(w).Encode(response)
}
