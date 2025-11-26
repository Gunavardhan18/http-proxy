package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"http-proxy/pkg/types"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// AuditEvent represents an audit event for logging proxy decisions
type AuditEvent struct {
	Timestamp    time.Time           `json:"timestamp"`
	RequestID    string              `json:"request_id"`
	ClientIP     string              `json:"client_ip"`
	Method       string              `json:"method"`
	URL          string              `json:"url"`
	UserAgent    string              `json:"user_agent"`
	RequestSize  int64               `json:"request_size"`
	RuleMatched  string              `json:"rule_matched,omitempty"`
	Action       types.Action        `json:"action"`
	Reason       string              `json:"reason"`
	Duration     time.Duration       `json:"duration_ms"`
	ResponseCode int                 `json:"response_code,omitempty"`
	ResponseSize int64               `json:"response_size,omitempty"`
	Headers      map[string][]string `json:"headers,omitempty"`
}

// Logger represents the proxy logger
type Logger struct {
	appLogger   *log.Logger
	auditLogger *log.Logger
	level       LogLevel
	config      *types.LoggingConfig
}

// NewLogger creates a new logger instance
func NewLogger(config *types.LoggingConfig) (*Logger, error) {
	logger := &Logger{
		level:  LogLevel(config.Level),
		config: config,
	}

	// Setup application logger
	appWriter, err := logger.setupAppLogger(config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup app logger: %w", err)
	}
	logger.appLogger = log.New(appWriter, "", log.LstdFlags|log.Lshortfile)

	// Setup audit logger if enabled
	if config.AuditEnabled {
		auditWriter, err := logger.setupAuditLogger(config)
		if err != nil {
			return nil, fmt.Errorf("failed to setup audit logger: %w", err)
		}
		logger.auditLogger = log.New(auditWriter, "", 0) // No standard flags for structured JSON logs
	}

	return logger, nil
}

// setupAppLogger configures the application logger with rotation
func (l *Logger) setupAppLogger(config *types.LoggingConfig) (io.Writer, error) {
	var writers []io.Writer

	// Always write to stdout
	writers = append(writers, os.Stdout)

	// Write to file if specified
	if config.File != "" {
		if err := os.MkdirAll(filepath.Dir(config.File), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		fileWriter := &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
		writers = append(writers, fileWriter)
	}

	return io.MultiWriter(writers...), nil
}

// setupAuditLogger configures the audit logger with rotation
func (l *Logger) setupAuditLogger(config *types.LoggingConfig) (io.Writer, error) {
	auditFile := config.AuditFile
	if auditFile == "" {
		auditFile = "audit.log"
		if config.File != "" {
			dir := filepath.Dir(config.File)
			auditFile = filepath.Join(dir, "audit.log")
		}
	}

	if err := os.MkdirAll(filepath.Dir(auditFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	return &lumberjack.Logger{
		Filename:   auditFile,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.shouldLog(LevelDebug) {
		l.appLogger.Printf("[DEBUG] "+msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.shouldLog(LevelInfo) {
		l.appLogger.Printf("[INFO] "+msg, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.shouldLog(LevelWarn) {
		l.appLogger.Printf("[WARN] "+msg, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.shouldLog(LevelError) {
		l.appLogger.Printf("[ERROR] "+msg, args...)
	}
}

// LogAuditEvent logs an audit event
func (l *Logger) LogAuditEvent(event *AuditEvent) {
	if l.auditLogger == nil {
		return
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		l.Error("Failed to marshal audit event: %v", err)
		return
	}

	l.auditLogger.Println(string(eventJSON))
}

// LogRequest logs a complete request/response cycle for auditing
func (l *Logger) LogRequest(requestID, clientIP, method, url, userAgent string, requestSize int64,
	result *types.RuleResult, duration time.Duration, responseCode int, responseSize int64,
	headers map[string][]string) {

	event := &AuditEvent{
		Timestamp:    time.Now().UTC(),
		RequestID:    requestID,
		ClientIP:     clientIP,
		Method:       method,
		URL:          url,
		UserAgent:    userAgent,
		RequestSize:  requestSize,
		Action:       result.Action,
		Reason:       result.Reason,
		Duration:     duration,
		ResponseCode: responseCode,
		ResponseSize: responseSize,
	}

	if result.Rule != nil {
		event.RuleMatched = result.Rule.ID
	}

	// Only include headers if debug level
	if l.shouldLog(LevelDebug) {
		event.Headers = headers
	}

	l.LogAuditEvent(event)
}

// LogRuleAction logs when a rule action is taken
func (l *Logger) LogRuleAction(action types.Action, ruleID, reason, clientIP, url string) {
	switch action {
	case types.ActionBlock:
		l.Warn("BLOCKED request from %s to %s - Rule: %s, Reason: %s", clientIP, url, ruleID, reason)
	case types.ActionAllow:
		l.Debug("ALLOWED request from %s to %s - Rule: %s, Reason: %s", clientIP, url, ruleID, reason)
	}
}

// LogProxyError logs proxy-related errors
func (l *Logger) LogProxyError(requestID, clientIP, url, error string) {
	l.Error("Proxy error for request %s from %s to %s: %s", requestID, clientIP, url, error)

	if l.auditLogger != nil {
		event := &AuditEvent{
			Timestamp: time.Now().UTC(),
			RequestID: requestID,
			ClientIP:  clientIP,
			URL:       url,
			Action:    "error",
			Reason:    error,
		}
		l.LogAuditEvent(event)
	}
}

// LogStats logs proxy statistics
func (l *Logger) LogStats(stats *types.ProxyStats) {
	l.Info("Proxy Stats - Total: %d, Allowed: %d, Blocked: %d, Errors: %d, Avg Latency: %dms",
		stats.TotalRequests, stats.AllowedRequests, stats.BlockedRequests,
		stats.ErrorRequests, stats.AverageLatencyMs)
}

// shouldLog checks if a message should be logged based on the configured level
func (l *Logger) shouldLog(level LogLevel) bool {
	levelOrder := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}

	configuredLevel, exists := levelOrder[l.level]
	if !exists {
		configuredLevel = levelOrder[LevelInfo] // Default to info
	}

	requestedLevel, exists := levelOrder[level]
	if !exists {
		return false
	}

	return requestedLevel >= configuredLevel
}

// Close closes the logger and flushes any remaining logs
func (l *Logger) Close() error {
	// Lumberjack handles cleanup automatically
	l.Info("Logger shutting down")
	return nil
}

// SetLevel changes the logging level at runtime
func (l *Logger) SetLevel(level string) {
	l.level = LogLevel(level)
	l.Info("Log level changed to: %s", level)
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() string {
	return string(l.level)
}

// RequestIDGenerator generates unique request IDs
type RequestIDGenerator struct {
	counter int64
}

// NewRequestIDGenerator creates a new request ID generator
func NewRequestIDGenerator() *RequestIDGenerator {
	return &RequestIDGenerator{}
}

// Generate generates a new unique request ID
func (r *RequestIDGenerator) Generate() string {
	r.counter++
	return fmt.Sprintf("%d-%d", time.Now().Unix(), r.counter)
}

// ContextualLogger wraps the main logger with request context
type ContextualLogger struct {
	logger    *Logger
	requestID string
	clientIP  string
}

// NewContextualLogger creates a contextual logger for a specific request
func NewContextualLogger(logger *Logger, requestID, clientIP string) *ContextualLogger {
	return &ContextualLogger{
		logger:    logger,
		requestID: requestID,
		clientIP:  clientIP,
	}
}

// Debug logs a debug message with context
func (cl *ContextualLogger) Debug(msg string, args ...interface{}) {
	contextMsg := fmt.Sprintf("[%s|%s] %s", cl.requestID, cl.clientIP, msg)
	cl.logger.Debug(contextMsg, args...)
}

// Info logs an info message with context
func (cl *ContextualLogger) Info(msg string, args ...interface{}) {
	contextMsg := fmt.Sprintf("[%s|%s] %s", cl.requestID, cl.clientIP, msg)
	cl.logger.Info(contextMsg, args...)
}

// Warn logs a warning message with context
func (cl *ContextualLogger) Warn(msg string, args ...interface{}) {
	contextMsg := fmt.Sprintf("[%s|%s] %s", cl.requestID, cl.clientIP, msg)
	cl.logger.Warn(contextMsg, args...)
}

// Error logs an error message with context
func (cl *ContextualLogger) Error(msg string, args ...interface{}) {
	contextMsg := fmt.Sprintf("[%s|%s] %s", cl.requestID, cl.clientIP, msg)
	cl.logger.Error(contextMsg, args...)
}
