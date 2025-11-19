package config

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Kafka validation
	if len(c.KafkaBrokers) == 0 {
		errors = append(errors, ValidationError{
			Field:   "KafkaBrokers",
			Message: "at least one Kafka broker is required",
		})
	} else {
		for i, broker := range c.KafkaBrokers {
			if err := validateHostPort(broker); err != nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("KafkaBrokers[%d]", i),
					Message: err.Error(),
				})
			}
		}
	}

	// Redis validation
	if err := validateHostPort(c.RedisHost + ":" + c.RedisPort); err != nil {
		errors = append(errors, ValidationError{
			Field:   "Redis",
			Message: err.Error(),
		})
	}

	if c.RedisDB < 0 || c.RedisDB > 15 {
		errors = append(errors, ValidationError{
			Field:   "RedisDB",
			Message: "must be between 0 and 15",
		})
	}

	// PostgreSQL validation
	if c.PostgresHost == "" {
		errors = append(errors, ValidationError{
			Field:   "PostgresHost",
			Message: "cannot be empty",
		})
	}

	if c.PostgresPort == "" {
		errors = append(errors, ValidationError{
			Field:   "PostgresPort",
			Message: "cannot be empty",
		})
	}

	if c.PostgresDB == "" {
		errors = append(errors, ValidationError{
			Field:   "PostgresDB",
			Message: "cannot be empty",
		})
	}

	// Port validation
	if err := validatePort(c.DataIngestionPort); err != nil {
		errors = append(errors, ValidationError{
			Field:   "DataIngestionPort",
			Message: err.Error(),
		})
	}

	if err := validatePort(c.OMSGRPCPort); err != nil {
		errors = append(errors, ValidationError{
			Field:   "OMSGRPCPort",
			Message: err.Error(),
		})
	}

	// JWT validation
	if c.JWTSecret == "" || c.JWTSecret == "your-super-secret-jwt-key-change-this-in-production" {
		if c.Environment == "production" || c.Environment == "prod" {
			errors = append(errors, ValidationError{
				Field:   "JWTSecret",
				Message: "must be set to a secure value in production",
			})
		}
	}

	if c.JWTExpiration <= 0 {
		errors = append(errors, ValidationError{
			Field:   "JWTExpiration",
			Message: "must be positive",
		})
	}

	// Trading parameters validation
	if c.MaxPositionSize <= 0 {
		errors = append(errors, ValidationError{
			Field:   "MaxPositionSize",
			Message: "must be positive",
		})
	}

	if c.MaxDailyLossPercent <= 0 || c.MaxDailyLossPercent > 100 {
		errors = append(errors, ValidationError{
			Field:   "MaxDailyLossPercent",
			Message: "must be between 0 and 100",
		})
	}

	if c.MaxLossPerTradePercent <= 0 || c.MaxLossPerTradePercent > 100 {
		errors = append(errors, ValidationError{
			Field:   "MaxLossPerTradePercent",
			Message: "must be between 0 and 100",
		})
	}

	// Feature flags validation
	if c.EnableLiveTrading && c.Environment == "dev" {
		errors = append(errors, ValidationError{
			Field:   "EnableLiveTrading",
			Message: "live trading should not be enabled in development environment",
		})
	}

	// Log level validation
	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true,
		"warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errors = append(errors, ValidationError{
			Field:   "LogLevel",
			Message: fmt.Sprintf("invalid log level: %s", c.LogLevel),
		})
	}

	// Environment validation
	validEnvironments := map[string]bool{
		"dev": true, "development": true,
		"staging": true, "stage": true,
		"prod": true, "production": true,
	}
	if !validEnvironments[strings.ToLower(c.Environment)] {
		errors = append(errors, ValidationError{
			Field:   "Environment",
			Message: fmt.Sprintf("invalid environment: %s", c.Environment),
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateHostPort validates a host:port string
func validateHostPort(hostPort string) error {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}

	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// validatePort validates a port string
func validatePort(port string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// validateURL validates a URL string
func validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	_, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return nil
}

// validateEmail validates an email address
func validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(c.Environment)
	return env == "dev" || env == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Environment)
	return env == "prod" || env == "production"
}

// IsStaging returns true if running in staging environment
func (c *Config) IsStaging() bool {
	env := strings.ToLower(c.Environment)
	return env == "staging" || env == "stage"
}

// GetRedisAddr returns the Redis address in host:port format
func (c *Config) GetRedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// GetPostgresConnectionString returns the PostgreSQL connection string
func (c *Config) GetPostgresConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.PostgresDB)
}

// GetTimescaleConnectionString returns the TimescaleDB connection string
func (c *Config) GetTimescaleConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.TimescaleHost, c.TimescalePort, c.TimescaleUser, c.TimescalePassword, c.TimescaleDB)
}

// MustValidate validates the configuration and panics on error
func (c *Config) MustValidate() {
	if err := c.Validate(); err != nil {
		panic(fmt.Sprintf("configuration validation failed: %v", err))
	}
}
