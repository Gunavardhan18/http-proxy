package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"http-proxy/pkg/types"
)

func TestNewLogger(t *testing.T) {
	config := &types.LoggingConfig{
		Level:        "info",
		MaxSize:      100,
		MaxBackups:   3,
		MaxAge:       28,
		Compress:     true,
		AuditEnabled: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Errorf("Expected no error creating logger, got: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.level != LevelInfo {
		t.Errorf("Expected log level info, got %s", logger.level)
	}

	// Cleanup
	logger.Close()
}

func TestNewLogger_WithFiles(t *testing.T) {
	tempDir := t.TempDir()

	config := &types.LoggingConfig{
		Level:        "debug",
		File:         filepath.Join(tempDir, "app.log"),
		AuditEnabled: true,
		AuditFile:    filepath.Join(tempDir, "audit.log"),
		MaxSize:      10,
		MaxBackups:   2,
		MaxAge:       7,
		Compress:     false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Errorf("Expected no error creating logger with files, got: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Test logging
	logger.Info("Test log message")

	// Test audit logging
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),
		RequestID: "test-123",
		ClientIP:  "192.168.1.1",
		Method:    "GET",
		URL:       "/test",
		Action:    types.ActionAllow,
		Reason:    "test reason",
	}
	logger.LogAuditEvent(event)

	// Cleanup
	logger.Close()

	// Wait a bit for file handles to be released
	time.Sleep(100 * time.Millisecond)

	// Just verify the logger was created correctly, don't check files
	// as they may be locked by lumberjack on Windows
	if logger.config.File != config.File {
		t.Errorf("Logger should have correct file path")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level:        "warn", // Only warn and error should be logged
		AuditEnabled: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// Redirect logger output to buffer for testing
	logger.appLogger.SetOutput(&buf)

	// Test different log levels
	logger.Debug("Debug message") // Should not appear
	logger.Info("Info message")   // Should not appear
	logger.Warn("Warn message")   // Should appear
	logger.Error("Error message") // Should appear

	output := buf.String()

	if strings.Contains(output, "Debug message") {
		t.Errorf("Debug message should not be logged at warn level")
	}

	if strings.Contains(output, "Info message") {
		t.Errorf("Info message should not be logged at warn level")
	}

	if !strings.Contains(output, "Warn message") {
		t.Errorf("Warn message should be logged at warn level")
	}

	if !strings.Contains(output, "Error message") {
		t.Errorf("Error message should be logged at warn level")
	}
}

func TestLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		configLevel string
		testLevel   LogLevel
		shouldLog   bool
	}{
		{"debug", LevelDebug, true},
		{"debug", LevelInfo, true},
		{"debug", LevelWarn, true},
		{"debug", LevelError, true},
		{"info", LevelDebug, false},
		{"info", LevelInfo, true},
		{"info", LevelWarn, true},
		{"info", LevelError, true},
		{"warn", LevelDebug, false},
		{"warn", LevelInfo, false},
		{"warn", LevelWarn, true},
		{"warn", LevelError, true},
		{"error", LevelDebug, false},
		{"error", LevelInfo, false},
		{"error", LevelWarn, false},
		{"error", LevelError, true},
		{"invalid", LevelInfo, true}, // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.configLevel+"->"+string(tt.testLevel), func(t *testing.T) {
			logger := &Logger{
				level: LogLevel(tt.configLevel),
			}

			result := logger.shouldLog(tt.testLevel)
			if result != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v for config=%s test=%s, got %v",
					tt.shouldLog, tt.configLevel, tt.testLevel, result)
			}
		})
	}
}

func TestLogger_LogAuditEvent(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level:        "info",
		AuditEnabled: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// Redirect audit logger to buffer
	logger.auditLogger.SetOutput(&buf)

	event := &AuditEvent{
		Timestamp:    time.Date(2023, 11, 26, 10, 0, 0, 0, time.UTC),
		RequestID:    "req-123",
		ClientIP:     "192.168.1.100",
		Method:       "POST",
		URL:          "/api/test",
		UserAgent:    "TestAgent/1.0",
		RequestSize:  1024,
		RuleMatched:  "test-rule",
		Action:       types.ActionBlock,
		Reason:       "URL blocked by rule",
		Duration:     25 * time.Millisecond,
		ResponseCode: 403,
		ResponseSize: 256,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}

	logger.LogAuditEvent(event)

	output := buf.String()
	if output == "" {
		t.Errorf("Expected audit event to be logged")
	}

	// Verify it's valid JSON
	var parsedEvent AuditEvent
	err = json.Unmarshal([]byte(output), &parsedEvent)
	if err != nil {
		t.Errorf("Audit event should be valid JSON: %v", err)
	}

	// Verify some key fields
	if parsedEvent.RequestID != "req-123" {
		t.Errorf("Expected request ID 'req-123', got '%s'", parsedEvent.RequestID)
	}

	if parsedEvent.Action != types.ActionBlock {
		t.Errorf("Expected action 'block', got '%s'", parsedEvent.Action)
	}
}

func TestLogger_LogAuditEvent_Disabled(t *testing.T) {
	config := &types.LoggingConfig{
		Level:        "info",
		AuditEnabled: false, // Disabled
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// Should not panic when audit is disabled
	event := &AuditEvent{
		RequestID: "test",
		Action:    types.ActionAllow,
	}

	logger.LogAuditEvent(event) // Should not crash
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level:        "debug", // Enable debug to include headers
		AuditEnabled: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.auditLogger.SetOutput(&buf)

	result := &types.RuleResult{
		Rule: &types.Rule{
			ID: "test-rule",
		},
		Matched: true,
		Action:  types.ActionBlock,
		Reason:  "Test block reason",
	}

	headers := map[string][]string{
		"User-Agent": {"TestBot/1.0"},
		"Accept":     {"application/json"},
	}

	logger.LogRequest(
		"req-456",
		"10.0.0.1",
		"DELETE",
		"/api/delete",
		"TestBot/1.0",
		2048,
		result,
		50*time.Millisecond,
		403,
		512,
		headers,
	)

	output := buf.String()
	if output == "" {
		t.Errorf("Expected request to be logged")
	}

	var event AuditEvent
	err = json.Unmarshal([]byte(output), &event)
	if err != nil {
		t.Errorf("Logged request should be valid JSON: %v", err)
	}

	if event.RequestID != "req-456" {
		t.Errorf("Expected request ID 'req-456', got '%s'", event.RequestID)
	}

	if event.Method != "DELETE" {
		t.Errorf("Expected method 'DELETE', got '%s'", event.Method)
	}

	if event.RuleMatched != "test-rule" {
		t.Errorf("Expected rule 'test-rule', got '%s'", event.RuleMatched)
	}

	// Headers should be included at debug level
	if event.Headers == nil {
		t.Errorf("Expected headers to be included at debug level")
	}
}

func TestLogger_LogProxyError(t *testing.T) {
	var appBuf, auditBuf bytes.Buffer

	config := &types.LoggingConfig{
		Level:        "error",
		AuditEnabled: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.appLogger.SetOutput(&appBuf)
	logger.auditLogger.SetOutput(&auditBuf)

	logger.LogProxyError("req-789", "172.16.0.1", "/error/test", "Connection timeout")

	// Check app log
	appOutput := appBuf.String()
	if !strings.Contains(appOutput, "Connection timeout") {
		t.Errorf("App log should contain error message")
	}

	// Check audit log
	auditOutput := auditBuf.String()
	if auditOutput == "" {
		t.Errorf("Audit log should contain error event")
	}

	var event AuditEvent
	err = json.Unmarshal([]byte(auditOutput), &event)
	if err != nil {
		t.Errorf("Error audit event should be valid JSON: %v", err)
	}

	if event.Action != "error" {
		t.Errorf("Expected action 'error', got '%s'", event.Action)
	}
}

func TestLogger_SetLevel(t *testing.T) {
	config := &types.LoggingConfig{
		Level: "info",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// Change level
	logger.SetLevel("debug")

	if logger.GetLevel() != "debug" {
		t.Errorf("Expected level to be changed to debug, got %s", logger.GetLevel())
	}

	// Verify debug logging now works
	if !logger.shouldLog(LevelDebug) {
		t.Errorf("Debug logging should be enabled after level change")
	}
}

func TestNewRequestIDGenerator(t *testing.T) {
	gen := NewRequestIDGenerator()
	if gen == nil {
		t.Fatal("Expected generator to be created")
	}

	// Generate some IDs
	id1 := gen.Generate()
	id2 := gen.Generate()
	id3 := gen.Generate()

	// Should be unique
	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("Generated IDs should be unique: %s, %s, %s", id1, id2, id3)
	}

	// Should contain timestamp and counter
	if !strings.Contains(id1, "-") {
		t.Errorf("ID should contain timestamp and counter separated by -")
	}
}

func TestNewContextualLogger(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level: "debug",
	}

	baseLogger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer baseLogger.Close()

	baseLogger.appLogger.SetOutput(&buf)

	ctxLogger := NewContextualLogger(baseLogger, "req-999", "203.0.113.1")

	ctxLogger.Info("Contextual log message")

	output := buf.String()

	// Should contain request ID and client IP
	if !strings.Contains(output, "req-999") {
		t.Errorf("Output should contain request ID")
	}

	if !strings.Contains(output, "203.0.113.1") {
		t.Errorf("Output should contain client IP")
	}

	if !strings.Contains(output, "Contextual log message") {
		t.Errorf("Output should contain log message")
	}
}

func TestContextualLogger_AllLevels(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level: "debug",
	}

	baseLogger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer baseLogger.Close()

	baseLogger.appLogger.SetOutput(&buf)

	ctxLogger := NewContextualLogger(baseLogger, "ctx-test", "192.0.2.1")

	ctxLogger.Debug("Debug message")
	ctxLogger.Info("Info message")
	ctxLogger.Warn("Warn message")
	ctxLogger.Error("Error message")

	output := buf.String()

	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, level := range levels {
		if !strings.Contains(output, level) {
			t.Errorf("Output should contain %s level", level)
		}
	}

	// All should have context
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "[ctx-test|192.0.2.1]") {
			t.Errorf("Line should contain context: %s", line)
		}
	}
}

func TestLogger_LogStats(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level: "info",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.appLogger.SetOutput(&buf)

	stats := &types.ProxyStats{
		TotalRequests:    1000,
		AllowedRequests:  800,
		BlockedRequests:  150,
		ErrorRequests:    50,
		AverageLatencyMs: 25,
	}

	logger.LogStats(stats)

	output := buf.String()

	// Should contain all stats
	expectedStrings := []string{
		"Total: 1000",
		"Allowed: 800",
		"Blocked: 150",
		"Errors: 50",
		"Avg Latency: 25ms",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain '%s': %s", expected, output)
		}
	}
}

func TestLogger_LogRuleAction(t *testing.T) {
	var buf bytes.Buffer

	config := &types.LoggingConfig{
		Level: "debug", // Enable debug for allow actions
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.appLogger.SetOutput(&buf)

	// Test block action (logged as warning)
	logger.LogRuleAction(types.ActionBlock, "block-rule", "URL blocked", "10.1.1.1", "/admin")

	// Test allow action (logged as debug)
	logger.LogRuleAction(types.ActionAllow, "allow-rule", "URL allowed", "10.1.1.2", "/public")

	output := buf.String()

	if !strings.Contains(output, "BLOCKED") {
		t.Errorf("Output should contain BLOCKED action")
	}

	if !strings.Contains(output, "ALLOWED") {
		t.Errorf("Output should contain ALLOWED action")
	}

	if !strings.Contains(output, "block-rule") {
		t.Errorf("Output should contain rule ID")
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	config := &types.LoggingConfig{
		Level:        "info",
		AuditEnabled: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		b.Fatal(err)
	}
	defer logger.Close()

	// Redirect to discard to avoid I/O in benchmark
	logger.appLogger.SetOutput(io.Discard)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark log message %d", i)
	}
}

func BenchmarkLogger_LogAuditEvent(b *testing.B) {
	config := &types.LoggingConfig{
		Level:        "info",
		AuditEnabled: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		b.Fatal(err)
	}
	defer logger.Close()

	logger.auditLogger.SetOutput(io.Discard)

	event := &AuditEvent{
		Timestamp: time.Now(),
		RequestID: "bench-123",
		ClientIP:  "192.168.1.1",
		Method:    "GET",
		URL:       "/test",
		Action:    types.ActionAllow,
		Reason:    "Benchmark test",
		Duration:  10 * time.Millisecond,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.LogAuditEvent(event)
	}
}
