package types

import (
	"net"
	"time"
)

// Action defines what to do with a request
type Action string

const (
	ActionAllow Action = "allow"
	ActionBlock Action = "block"
)

// RuleType defines the type of rule for matching
type RuleType string

const (
	RuleTypeIPv4      RuleType = "ipv4"
	RuleTypeIPv6      RuleType = "ipv6"
	RuleTypeURL       RuleType = "url"
	RuleTypeDomain    RuleType = "domain"
	RuleTypeUserAgent RuleType = "user_agent"
	RuleTypeURISuffix RuleType = "uri_suffix"
	RuleTypeSize      RuleType = "size"
	RuleTypeMethod    RuleType = "method"
	RuleTypeHeader    RuleType = "header"
)

// MatchOperator defines how to match the rule
type MatchOperator string

const (
	MatchEquals     MatchOperator = "equals"
	MatchContains   MatchOperator = "contains"
	MatchStartsWith MatchOperator = "starts_with"
	MatchEndsWith   MatchOperator = "ends_with"
	MatchRegex      MatchOperator = "regex"
	MatchWildcard   MatchOperator = "wildcard"
	MatchGTE        MatchOperator = "gte" // Greater than or equal (for size)
	MatchLTE        MatchOperator = "lte" // Less than or equal (for size)
	MatchInRange    MatchOperator = "in_range"
)

// Rule represents a filtering rule
type Rule struct {
	ID          string        `yaml:"id" json:"id" toml:"id"`
	Name        string        `yaml:"name" json:"name" toml:"name"`
	Description string        `yaml:"description,omitempty" json:"description,omitempty" toml:"description,omitempty"`
	Type        RuleType      `yaml:"type" json:"type" toml:"type"`
	Operator    MatchOperator `yaml:"operator" json:"operator" toml:"operator"`
	Value       string        `yaml:"value" json:"value" toml:"value"`
	Action      Action        `yaml:"action" json:"action" toml:"action"`
	Priority    int           `yaml:"priority" json:"priority" toml:"priority"`
	Enabled     bool          `yaml:"enabled" json:"enabled" toml:"enabled"`

	// For size-based rules
	MinSize *int64 `yaml:"min_size,omitempty" json:"min_size,omitempty" toml:"min_size,omitempty"`
	MaxSize *int64 `yaml:"max_size,omitempty" json:"max_size,omitempty" toml:"max_size,omitempty"`

	// For header-based rules
	HeaderName  string `yaml:"header_name,omitempty" json:"header_name,omitempty" toml:"header_name,omitempty"`
	HeaderValue string `yaml:"header_value,omitempty" json:"header_value,omitempty" toml:"header_value,omitempty"`
}

// ProxyConfig represents the main proxy configuration
type ProxyConfig struct {
	Server   ServerConfig   `yaml:"server" json:"server" toml:"server"`
	Backend  BackendConfig  `yaml:"backend" json:"backend" toml:"backend"`
	Rules    RulesConfig    `yaml:"rules" json:"rules" toml:"rules"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging" toml:"logging"`
	Security SecurityConfig `yaml:"security,omitempty" json:"security,omitempty" toml:"security,omitempty"`
}

// ServerConfig represents proxy server configuration
type ServerConfig struct {
	Host           string        `yaml:"host" json:"host" toml:"host"`
	Port           int           `yaml:"port" json:"port" toml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" json:"idle_timeout" toml:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" json:"max_header_bytes" toml:"max_header_bytes"`
}

// BackendConfig represents backend server configuration
type BackendConfig struct {
	Host    string        `yaml:"host" json:"host" toml:"host"`
	Port    int           `yaml:"port" json:"port" toml:"port"`
	Timeout time.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`

	// Health check settings
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check" toml:"health_check"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled" toml:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval" toml:"interval"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`
	Path     string        `yaml:"path" json:"path" toml:"path"`
}

// RulesConfig represents rules configuration
type RulesConfig struct {
	Rules          []Rule        `yaml:"rules" json:"rules" toml:"rules"`
	DefaultAction  Action        `yaml:"default_action" json:"default_action" toml:"default_action"`
	RulesFile      string        `yaml:"rules_file,omitempty" json:"rules_file,omitempty" toml:"rules_file,omitempty"`
	WatchRulesFile bool          `yaml:"watch_rules_file" json:"watch_rules_file" toml:"watch_rules_file"`
	ReloadInterval time.Duration `yaml:"reload_interval" json:"reload_interval" toml:"reload_interval"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level" json:"level" toml:"level"`
	File       string `yaml:"file,omitempty" json:"file,omitempty" toml:"file,omitempty"`
	MaxSize    int    `yaml:"max_size" json:"max_size" toml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups" json:"max_backups" toml:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age" toml:"max_age"` // days
	Compress   bool   `yaml:"compress" json:"compress" toml:"compress"`

	// Audit logging
	AuditEnabled bool   `yaml:"audit_enabled" json:"audit_enabled" toml:"audit_enabled"`
	AuditFile    string `yaml:"audit_file,omitempty" json:"audit_file,omitempty" toml:"audit_file,omitempty"`
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	RateLimiting RateLimitConfig `yaml:"rate_limiting,omitempty" json:"rate_limiting,omitempty" toml:"rate_limiting,omitempty"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled" toml:"enabled"`
	RequestsPerSec  int           `yaml:"requests_per_sec" json:"requests_per_sec" toml:"requests_per_sec"`
	BurstSize       int           `yaml:"burst_size" json:"burst_size" toml:"burst_size"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" toml:"cleanup_interval"`
}

// RequestInfo represents information about an HTTP request for rule evaluation
type RequestInfo struct {
	Method     string
	URL        string
	Domain     string
	Path       string
	Headers    map[string][]string
	UserAgent  string
	ClientIP   net.IP
	Size       int64
	RemoteAddr string
}

// RuleResult represents the result of rule evaluation
type RuleResult struct {
	Rule    *Rule
	Matched bool
	Action  Action
	Reason  string
}

// ProxyStats represents proxy statistics
type ProxyStats struct {
	TotalRequests    int64 `json:"total_requests"`
	AllowedRequests  int64 `json:"allowed_requests"`
	BlockedRequests  int64 `json:"blocked_requests"`
	ErrorRequests    int64 `json:"error_requests"`
	AverageLatencyMs int64 `json:"average_latency_ms"`
	RulesEvaluated   int64 `json:"rules_evaluated"`
}
