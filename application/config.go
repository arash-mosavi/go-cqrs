// filepath: application/config.go
package application

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the complete CQRS system configuration
type Config struct {
	CommandBus  CommandBusConfig  `json:"command_bus"`
	QueryBus    QueryBusConfig    `json:"query_bus"`
	Middleware  MiddlewareConfig  `json:"middleware"`
	Monitoring  MonitoringConfig  `json:"monitoring"`
	Performance PerformanceConfig `json:"performance"`
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

// DefaultConfig returns a production-ready default configuration
func DefaultConfig() *Config {
	return &Config{
		CommandBus: CommandBusConfig{
			InitialCapacity:  256,
			EnableMetrics:    true,
			MaxHandlers:      1000,
			TimeoutSeconds:   30,
			EnableValidation: true,
			EnableLogging:    true,
		},
		QueryBus: QueryBusConfig{
			InitialCapacity:  256,
			EnableMetrics:    true,
			MaxHandlers:      1000,
			TimeoutSeconds:   15,
			EnableCaching:    true,
			CacheTTLSeconds:  300, // 5 minutes
			EnableRetry:      true,
			RetryMaxAttempts: 3,
			RetryDelayMs:     100,
		},
		Middleware: MiddlewareConfig{
			Timeout: TimeoutConfig{
				Enabled:        true,
				DefaultSeconds: 30,
				CommandSeconds: 30,
				QuerySeconds:   15,
			},
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				RecoveryTimeMs:   5000,
				HalfOpenMaxCalls: 3,
			},
			RateLimit: RateLimitConfig{
				Enabled:        true,
				RequestsPerSec: 1000,
				BurstSize:      100,
				WindowMs:       1000,
			},
			Caching: CachingConfig{
				Enabled:        true,
				DefaultTTLSec:  300,
				MaxSize:        10000,
				EvictionPolicy: "LRU",
			},
			Retry: RetryConfig{
				Enabled:         true,
				MaxAttempts:     3,
				BaseDelayMs:     100,
				MaxDelayMs:      5000,
				BackoffStrategy: "exponential",
			},
			Validation: ValidationConfig{
				Enabled:    true,
				StrictMode: true,
				FailFast:   true,
			},
			Logging: LoggingConfig{
				Enabled:       true,
				Level:         "info",
				Format:        "json",
				IncludeArgs:   false,
				IncludeTiming: true,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:         true,
			MetricsPort:     9090,
			PrometheusPath:  "/metrics",
			HealthCheckPath: "/health",
			Tracing: TracingConfig{
				Enabled:     true,
				SampleRate:  0.1,
				Endpoint:    "http://localhost:14268/api/traces",
				ServiceName: "cqrs-service",
			},
			Profiling: ProfilingConfig{
				Enabled:   false,
				Port:      6060,
				CPUProf:   true,
				MemProf:   true,
				BlockProf: false,
			},
		},
		Performance: PerformanceConfig{
			EnableObjectPooling:   true,
			PoolSize:              100,
			EnablePreallocation:   true,
			PreallocationSize:     256,
			EnableGCOptimization:  true,
			GCTargetPercent:       100,
			EnableCPUOptimization: false,
			MaxCPUUsage:           80,
			MemoryBallastMB:       0,
			EnableBatchProcessing: false,
			BatchSize:             100,
			BatchTimeout:          1000,
		},
	}
}

// DevelopmentConfig returns a development-optimized configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()

	// Development overrides
	config.CommandBus.InitialCapacity = 64
	config.QueryBus.InitialCapacity = 64
	config.QueryBus.CacheTTLSeconds = 60 // 1 minute

	config.Middleware.Logging.Level = "debug"
	config.Middleware.Logging.IncludeArgs = true
	config.Middleware.Caching.MaxSize = 1000

	config.Monitoring.Tracing.SampleRate = 1.0 // Trace everything in dev
	config.Monitoring.Profiling.Enabled = true

	config.Performance.EnableObjectPooling = false // Easier debugging without pooling

	return config
}

// TestConfig returns a test-optimized configuration
func TestConfig() *Config {
	config := DefaultConfig()

	// Test overrides
	config.CommandBus.InitialCapacity = 16
	config.QueryBus.InitialCapacity = 16
	config.CommandBus.TimeoutSeconds = 1
	config.QueryBus.TimeoutSeconds = 1
	config.QueryBus.CacheTTLSeconds = 10
	config.QueryBus.RetryDelayMs = 10

	config.Middleware.Timeout.DefaultSeconds = 1
	config.Middleware.Timeout.CommandSeconds = 1
	config.Middleware.Timeout.QuerySeconds = 1
	config.Middleware.CircuitBreaker.RecoveryTimeMs = 100
	config.Middleware.RateLimit.RequestsPerSec = 100
	config.Middleware.Caching.DefaultTTLSec = 10
	config.Middleware.Caching.MaxSize = 100
	config.Middleware.Retry.BaseDelayMs = 10
	config.Middleware.Retry.MaxDelayMs = 100

	config.Middleware.Logging.Level = "debug"
	config.Middleware.Logging.IncludeArgs = true

	config.Monitoring.Enabled = false
	config.Performance.EnableObjectPooling = false

	return config
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	return &config, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	// Command Bus
	if val := os.Getenv("CQRS_CMD_INITIAL_CAPACITY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.CommandBus.InitialCapacity = parsed
		}
	}

	if val := os.Getenv("CQRS_CMD_ENABLE_METRICS"); val != "" {
		config.CommandBus.EnableMetrics = val == "true"
	}

	if val := os.Getenv("CQRS_CMD_TIMEOUT_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.CommandBus.TimeoutSeconds = parsed
		}
	}

	// Query Bus
	if val := os.Getenv("CQRS_QUERY_INITIAL_CAPACITY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.QueryBus.InitialCapacity = parsed
		}
	}

	if val := os.Getenv("CQRS_QUERY_ENABLE_METRICS"); val != "" {
		config.QueryBus.EnableMetrics = val == "true"
	}

	if val := os.Getenv("CQRS_QUERY_CACHE_TTL_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.QueryBus.CacheTTLSeconds = parsed
		}
	}

	// Middleware
	if val := os.Getenv("CQRS_TIMEOUT_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.Middleware.Timeout.DefaultSeconds = parsed
		}
	}

	if val := os.Getenv("CQRS_CIRCUIT_BREAKER_THRESHOLD"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.Middleware.CircuitBreaker.FailureThreshold = parsed
		}
	}

	if val := os.Getenv("CQRS_RATE_LIMIT_RPS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.Middleware.RateLimit.RequestsPerSec = parsed
		}
	}

	if val := os.Getenv("CQRS_LOG_LEVEL"); val != "" {
		config.Middleware.Logging.Level = val
	}

	// Monitoring
	if val := os.Getenv("CQRS_METRICS_PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.Monitoring.MetricsPort = parsed
		}
	}

	if val := os.Getenv("CQRS_TRACING_ENDPOINT"); val != "" {
		config.Monitoring.Tracing.Endpoint = val
	}

	if val := os.Getenv("CQRS_SERVICE_NAME"); val != "" {
		config.Monitoring.Tracing.ServiceName = val
	}

	// Performance - removed MaxProcs as it's not in PerformanceConfig
	// Use runtime.GOMAXPROCS() directly if needed

	if val := os.Getenv("GOGC"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.Performance.GCTargetPercent = parsed
		}
	}

	return config
}

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
