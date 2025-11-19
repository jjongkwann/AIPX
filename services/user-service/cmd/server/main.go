package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"user-service/internal/config"
	"user-service/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection pool
	dbPool, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()

	// Initialize repositories
	repos := initRepositories(dbPool)
	_ = repos // Will be used when implementing handlers

	// Initialize HTTP router
	router := setupRouter(cfg)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting User Service on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
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

	log.Println("Database connection established")
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
func setupRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "user-service",
			"version": "1.0.0",
		})
	})

	// API v1 routes (placeholder)
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (to be implemented in T5)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", placeholderHandler("register"))
			auth.POST("/login", placeholderHandler("login"))
			auth.POST("/logout", placeholderHandler("logout"))
			auth.POST("/refresh", placeholderHandler("refresh token"))
			auth.POST("/verify-email", placeholderHandler("verify email"))
			auth.POST("/reset-password", placeholderHandler("reset password"))
		}

		// User routes (to be implemented in T5)
		users := v1.Group("/users")
		{
			users.GET("/me", placeholderHandler("get current user"))
			users.PUT("/me", placeholderHandler("update current user"))
			users.PUT("/me/password", placeholderHandler("change password"))
			users.DELETE("/me", placeholderHandler("delete account"))
		}

		// API key routes (to be implemented in T5)
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.GET("", placeholderHandler("list api keys"))
			apiKeys.POST("", placeholderHandler("create api key"))
			apiKeys.DELETE("/:id", placeholderHandler("delete api key"))
			apiKeys.PUT("/:id/activate", placeholderHandler("activate api key"))
			apiKeys.PUT("/:id/deactivate", placeholderHandler("deactivate api key"))
		}
	}

	return router
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// placeholderHandler returns a placeholder response for unimplemented endpoints
func placeholderHandler(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not implemented",
			"message": fmt.Sprintf("Endpoint for '%s' will be implemented in T5", action),
		})
	}
}
