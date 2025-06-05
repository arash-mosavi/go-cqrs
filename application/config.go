package application

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the complete CQRS system configuration
type Config struct {
	CommandBus CommandBusConfig `json:"command_bus"`
	QueryBus   QueryBusConfig   `json:"query_bus"`
	Middleware MiddlewareConfig `json:"middleware"`
	Monitoring MonitoringConfig `json:"monitoring"`
}

// CommandBusConfig holds command bus specific configuration
type CommandBusConfig struct {
	InitialCapacity  int  `json:"initial_capacity"`
	EnableMetrics    bool `json:"enable_metrics"`
	MaxHandlers      int  `json:"max_handlers"`
	TimeoutSeconds   int  `json:"timeout_seconds"`
	EnableValidation bool `json:"enable_validation"`
	EnableLogging    bool `json:"enable_logging"`
}

// QueryBusConfig holds query bus specific configuration
type QueryBusConfig struct {
	InitialCapacity  int  `json:"initial_capacity"`
	EnableMetrics    bool `json:"enable_metrics"`
	MaxHandlers      int  `json:"max_handlers"`
	TimeoutSeconds   int  `json:"timeout_seconds"`
	EnableCaching    bool `json:"enable_caching"`
	CacheTTLSeconds  int  `json:"cache_ttl_seconds"`
	EnableRetry      bool `json:"enable_retry"`
	RetryMaxAttempts int  `json:"retry_max_attempts"`
	RetryDelayMs     int  `json:"retry_delay_ms"`
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	Timeout        TimeoutConfig        `json:"timeout"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
	RateLimit      RateLimitConfig      `json:"rate_limit"`
	Caching        CachingConfig        `json:"caching"`
	Retry          RetryConfig          `json:"retry"`
	Validation     ValidationConfig     `json:"validation"`
	Logging        LoggingConfig        `json:"logging"`
}

// TimeoutConfig configures timeout middleware
type TimeoutConfig struct {
	Enabled        bool `json:"enabled"`
	DefaultSeconds int  `json:"default_seconds"`
	CommandSeconds int  `json:"command_seconds"`
	QuerySeconds   int  `json:"query_seconds"`
}

// CircuitBreakerConfig configures circuit breaker middleware
type CircuitBreakerConfig struct {
	Enabled          bool `json:"enabled"`
	FailureThreshold int  `json:"failure_threshold"`
	RecoveryTimeMs   int  `json:"recovery_time_ms"`
	HalfOpenMaxCalls int  `json:"half_open_max_calls"`
}

// RateLimitConfig configures rate limiting middleware
type RateLimitConfig struct {
	Enabled        bool `json:"enabled"`
	RequestsPerSec int  `json:"requests_per_sec"`
	BurstSize      int  `json:"burst_size"`
	WindowMs       int  `json:"window_ms"`
}

// CachingConfig configures caching middleware
type CachingConfig struct {
	Enabled        bool   `json:"enabled"`
	DefaultTTLSec  int    `json:"default_ttl_sec"`
	MaxSize        int    `json:"max_size"`
	EvictionPolicy string `json:"eviction_policy"` // LRU, LFU, TTL
}

// RetryConfig configures retry middleware
type RetryConfig struct {
	Enabled         bool   `json:"enabled"`
	MaxAttempts     int    `json:"max_attempts"`
	BaseDelayMs     int    `json:"base_delay_ms"`
	MaxDelayMs      int    `json:"max_delay_ms"`
	BackoffStrategy string `json:"backoff_strategy"` // linear, exponential
}

// ValidationConfig configures validation middleware
type ValidationConfig struct {
	Enabled    bool `json:"enabled"`
	StrictMode bool `json:"strict_mode"`
	FailFast   bool `json:"fail_fast"`
}

// LoggingConfig configures logging middleware
type LoggingConfig struct {
	Enabled       bool   `json:"enabled"`
	Level         string `json:"level"`  // debug, info, warn, error
	Format        string `json:"format"` // json, text
	IncludeArgs   bool   `json:"include_args"`
	IncludeTiming bool   `json:"include_timing"`
}

// MonitoringConfig configures monitoring and observability
type MonitoringConfig struct {
	Enabled         bool            `json:"enabled"`
	MetricsPort     int             `json:"metrics_port"`
	PrometheusPath  string          `json:"prometheus_path"`
	HealthCheckPath string          `json:"health_check_path"`
	Tracing         TracingConfig   `json:"tracing"`
	Profiling       ProfilingConfig `json:"profiling"`
}

// TracingConfig configures distributed tracing
type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	SampleRate  float64 `json:"sample_rate"`
	Endpoint    string  `json:"endpoint"`
	ServiceName string  `json:"service_name"`
}

// ProfilingConfig configures performance profiling
type ProfilingConfig struct {
	Enabled   bool `json:"enabled"`
	Port      int  `json:"port"`
	CPUProf   bool `json:"cpu_prof"`
	MemProf   bool `json:"mem_prof"`
	BlockProf bool `json:"block_prof"`
}

// Note: PerformanceConfig is defined in performance_optimizations.go

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.CommandBus.InitialCapacity <= 0 {
		return fmt.Errorf("command bus initial capacity must be positive")
	}

	if c.QueryBus.InitialCapacity <= 0 {
		return fmt.Errorf("query bus initial capacity must be positive")
	}

	if c.CommandBus.TimeoutSeconds <= 0 {
		return fmt.Errorf("command bus timeout must be positive")
	}

	if c.QueryBus.TimeoutSeconds <= 0 {
		return fmt.Errorf("query bus timeout must be positive")
	}

	if c.Middleware.CircuitBreaker.FailureThreshold <= 0 {
		return fmt.Errorf("circuit breaker failure threshold must be positive")
	}

	if c.Middleware.RateLimit.RequestsPerSec <= 0 {
		return fmt.Errorf("rate limit requests per second must be positive")
	}

	if c.Monitoring.MetricsPort <= 0 || c.Monitoring.MetricsPort > 65535 {
		return fmt.Errorf("metrics port must be between 1 and 65535")
	}

	return nil
}

// ApplyToCommandBus applies configuration to a command bus
func (c *Config) ApplyToCommandBus() CommandBusOptions {
	var middleware []CommandMiddleware

	// Add middleware based on configuration
	if c.Middleware.Validation.Enabled {
		middleware = append(middleware, ValidationMiddleware())
	}

	if c.Middleware.Timeout.Enabled {
		timeout := time.Duration(c.Middleware.Timeout.CommandSeconds) * time.Second
		middleware = append(middleware, TimeoutMiddleware(timeout))
	}

	if c.Middleware.CircuitBreaker.Enabled {
		threshold := c.Middleware.CircuitBreaker.FailureThreshold
		recovery := time.Duration(c.Middleware.CircuitBreaker.RecoveryTimeMs) * time.Millisecond
		circuitBreaker := NewCircuitBreakerMiddleware(threshold, recovery)
		middleware = append(middleware, circuitBreaker.ForCommands())
	}

	if c.Middleware.RateLimit.Enabled {
		rate := c.Middleware.RateLimit.RequestsPerSec
		window := time.Duration(c.Middleware.RateLimit.WindowMs) * time.Millisecond
		rateLimiter := NewRateLimitingMiddleware(rate, window)
		middleware = append(middleware, rateLimiter.ForCommands())
	}

	return CommandBusOptions{
		InitialCapacity: c.CommandBus.InitialCapacity,
		EnableMetrics:   c.CommandBus.EnableMetrics,
		Middleware:      middleware,
	}
}

// ApplyToQueryBus applies configuration to a query bus
func (c *Config) ApplyToQueryBus() QueryBusOptions {
	var middleware []QueryMiddleware

	// Add middleware based on configuration
	if c.Middleware.Caching.Enabled {
		ttl := time.Duration(c.Middleware.Caching.DefaultTTLSec) * time.Second
		middleware = append(middleware, CachingMiddleware(ttl))
	}

	if c.Middleware.Retry.Enabled {
		attempts := c.Middleware.Retry.MaxAttempts
		delay := time.Duration(c.Middleware.Retry.BaseDelayMs) * time.Millisecond
		middleware = append(middleware, RetryMiddleware(attempts, delay))
	}

	return QueryBusOptions{
		InitialCapacity: c.QueryBus.InitialCapacity,
		EnableMetrics:   c.QueryBus.EnableMetrics,
		Middleware:      middleware,
	}
}
