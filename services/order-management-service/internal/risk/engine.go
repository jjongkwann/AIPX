package risk

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"

	"order-management-service/internal/repository"
)

// RiskRule defines the interface that all risk rules must implement
type RiskRule interface {
	// Validate checks if an order passes this risk rule
	Validate(ctx context.Context, order *repository.Order) error
	// Name returns the name of the risk rule
	Name() string
	// Enabled returns whether this rule is currently enabled
	Enabled() bool
}

// Engine orchestrates risk validation by applying multiple rules
type Engine struct {
	rules    []RiskRule
	mu       sync.RWMutex
	config   *Config
	enabled  bool
}

// Config holds the configuration for the risk engine
type Config struct {
	MaxOrderValue     float64
	PriceDeviation    float64
	DailyLossLimit    float64
	AllowedSymbols    []string
	DuplicateWindow   int64 // seconds
	EnableRiskChecks  bool
}

// NewEngine creates a new risk engine with the provided configuration
func NewEngine(config *Config, deps *Dependencies) (*Engine, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	engine := &Engine{
		config:  config,
		rules:   make([]RiskRule, 0),
		enabled: config.EnableRiskChecks,
	}

	// Register all risk rules
	if err := engine.registerRules(deps); err != nil {
		return nil, fmt.Errorf("failed to register rules: %w", err)
	}

	log.Info().
		Int("rule_count", len(engine.rules)).
		Bool("enabled", engine.enabled).
		Msg("Risk engine initialized")

	return engine, nil
}

// registerRules registers all available risk rules
func (e *Engine) registerRules(deps *Dependencies) error {
	// 1. Max Order Value Rule
	maxOrderValueRule := NewMaxOrderValueRule(e.config.MaxOrderValue, true)
	e.rules = append(e.rules, maxOrderValueRule)

	// 2. Price Deviation Rule
	if deps.RedisClient != nil {
		priceDeviationRule := NewPriceDeviationRule(deps.RedisClient, e.config.PriceDeviation, true)
		e.rules = append(e.rules, priceDeviationRule)
	}

	// 3. Daily Loss Limit Rule
	if deps.Repository != nil && deps.RedisClient != nil {
		dailyLossLimitRule := NewDailyLossLimitRule(deps.Repository, deps.RedisClient, e.config.DailyLossLimit, true)
		e.rules = append(e.rules, dailyLossLimitRule)
	}

	// 4. Duplicate Order Rule
	if deps.RedisClient != nil {
		duplicateOrderRule := NewDuplicateOrderRule(deps.RedisClient, e.config.DuplicateWindow, true)
		e.rules = append(e.rules, duplicateOrderRule)
	}

	// 5. Allowed Symbol Rule
	allowedSymbolRule := NewAllowedSymbolRule(e.config.AllowedSymbols, true)
	e.rules = append(e.rules, allowedSymbolRule)

	return nil
}

// Validate validates an order against all enabled risk rules
func (e *Engine) Validate(ctx context.Context, order *repository.Order) error {
	if !e.enabled {
		log.Debug().Msg("Risk checks disabled, skipping validation")
		return nil
	}

	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	log.Debug().
		Str("order_id", order.ID).
		Str("symbol", order.Symbol).
		Int("rule_count", len(rules)).
		Msg("Starting risk validation")

	// Apply all enabled rules
	for _, rule := range rules {
		if !rule.Enabled() {
			log.Debug().
				Str("rule", rule.Name()).
				Msg("Rule disabled, skipping")
			continue
		}

		if err := rule.Validate(ctx, order); err != nil {
			log.Warn().
				Err(err).
				Str("order_id", order.ID).
				Str("rule", rule.Name()).
				Msg("Risk rule validation failed")
			return &RiskError{
				Rule:    rule.Name(),
				Message: err.Error(),
				Order:   order,
			}
		}

		log.Debug().
			Str("rule", rule.Name()).
			Msg("Rule passed")
	}

	log.Info().
		Str("order_id", order.ID).
		Msg("Risk validation passed")

	return nil
}

// AddRule adds a new rule to the engine
func (e *Engine) AddRule(rule RiskRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
	log.Info().Str("rule", rule.Name()).Msg("Risk rule added")
}

// RemoveRule removes a rule from the engine by name
func (e *Engine) RemoveRule(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, rule := range e.rules {
		if rule.Name() == name {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			log.Info().Str("rule", name).Msg("Risk rule removed")
			return
		}
	}
}

// GetRules returns a copy of all registered rules
func (e *Engine) GetRules() []RiskRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]RiskRule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// Enable enables the risk engine
func (e *Engine) Enable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = true
	log.Info().Msg("Risk engine enabled")
}

// Disable disables the risk engine
func (e *Engine) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
	log.Warn().Msg("Risk engine disabled")
}

// IsEnabled returns whether the risk engine is enabled
func (e *Engine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// RiskError represents a risk validation error
type RiskError struct {
	Rule    string
	Message string
	Order   *repository.Order
}

func (e *RiskError) Error() string {
	return fmt.Sprintf("risk check failed [%s]: %s", e.Rule, e.Message)
}
