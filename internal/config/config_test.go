package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"http-proxy/pkg/types"

	"gopkg.in/yaml.v3"
)

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager("test.yaml")
	if cm.configPath != "test.yaml" {
		t.Errorf("Expected config path 'test.yaml', got '%s'", cm.configPath)
	}
}

func TestConfigManager_LoadConfig_DefaultConfig(t *testing.T) {
	// Test loading default config when no path is provided
	cm := NewConfigManager("")
	config, err := cm.LoadConfig()

	if err != nil {
		t.Errorf("Expected no error loading default config, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be returned, got nil")
	}

	// Verify default values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}

	if config.Backend.Host != "localhost" {
		t.Errorf("Expected default backend host 'localhost', got '%s'", config.Backend.Host)
	}

	if config.Backend.Port != 8090 {
		t.Errorf("Expected default backend port 8090, got %d", config.Backend.Port)
	}
}

func TestConfigManager_LoadConfig_YAML(t *testing.T) {
	// Create temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	configData := map[string]interface{}{
		"server": map[string]interface{}{
			"host":             "0.0.0.0",
			"port":             9090,
			"read_timeout":     "45s",
			"write_timeout":    "45s",
			"idle_timeout":     "180s",
			"max_header_bytes": 2097152,
		},
		"backend": map[string]interface{}{
			"host":    "backend.example.com",
			"port":    8000,
			"timeout": "60s",
			"health_check": map[string]interface{}{
				"enabled":  true,
				"interval": "60s",
				"timeout":  "10s",
				"path":     "/api/health",
			},
		},
		"rules": map[string]interface{}{
			"default_action":   "block",
			"watch_rules_file": false,
			"reload_interval":  "10s",
			"rules": []map[string]interface{}{
				{
					"id":          "test-rule",
					"name":        "Test Rule",
					"description": "A test rule",
					"type":        "url",
					"operator":    "starts_with",
					"value":       "/test",
					"action":      "allow",
					"priority":    100,
					"enabled":     true,
				},
			},
		},
		"logging": map[string]interface{}{
			"level":         "debug",
			"file":          "/var/log/proxy.log",
			"max_size":      200,
			"max_backups":   5,
			"max_age":       14,
			"compress":      false,
			"audit_enabled": false,
			"audit_file":    "/var/log/audit.log",
		},
	}

	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load config
	cm := NewConfigManager(configFile)
	config, err := cm.LoadConfig()

	if err != nil {
		t.Errorf("Expected no error loading YAML config, got: %v", err)
	}

	// Verify loaded values
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}

	if config.Backend.Host != "backend.example.com" {
		t.Errorf("Expected backend host 'backend.example.com', got '%s'", config.Backend.Host)
	}

	if config.Rules.DefaultAction != types.ActionBlock {
		t.Errorf("Expected default action 'block', got '%s'", config.Rules.DefaultAction)
	}

	if len(config.Rules.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(config.Rules.Rules))
	}

	rule := config.Rules.Rules[0]
	if rule.ID != "test-rule" {
		t.Errorf("Expected rule ID 'test-rule', got '%s'", rule.ID)
	}
}

func TestConfigManager_LoadConfig_JSON(t *testing.T) {
	// Create temporary JSON config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.json")

	config := &types.ProxyConfig{
		Server: types.ServerConfig{
			Host:           "jsonhost",
			Port:           7777,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    120 * time.Second,
			MaxHeaderBytes: 1048576,
		},
		Backend: types.BackendConfig{
			Host:    "jsonbackend",
			Port:    7000,
			Timeout: 30 * time.Second,
			HealthCheck: types.HealthCheckConfig{
				Enabled:  false,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Path:     "/health",
			},
		},
		Rules: types.RulesConfig{
			DefaultAction:  types.ActionAllow,
			WatchRulesFile: true,
			ReloadInterval: 5 * time.Second,
			Rules:          []types.Rule{},
		},
		Logging: types.LoggingConfig{
			Level:        "warn",
			MaxSize:      50,
			MaxBackups:   2,
			MaxAge:       7,
			Compress:     true,
			AuditEnabled: true,
		},
	}

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load config
	cm := NewConfigManager(configFile)
	loadedConfig, err := cm.LoadConfig()

	if err != nil {
		t.Errorf("Expected no error loading JSON config, got: %v", err)
	}

	// Verify loaded values
	if loadedConfig.Server.Host != "jsonhost" {
		t.Errorf("Expected host 'jsonhost', got '%s'", loadedConfig.Server.Host)
	}

	if loadedConfig.Server.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", loadedConfig.Server.Port)
	}

	if loadedConfig.Logging.Level != "warn" {
		t.Errorf("Expected log level 'warn', got '%s'", loadedConfig.Logging.Level)
	}
}

func TestConfigManager_LoadConfig_TOML(t *testing.T) {
	// Create temporary TOML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.toml")

	tomlContent := `
[server]
host = "tomlhost"
port = 6666
read_timeout = "25s"
write_timeout = "25s"
idle_timeout = "100s"
max_header_bytes = 524288

[backend]
host = "tomlbackend"
port = 6000
timeout = "25s"

[backend.health_check]
enabled = true
interval = "45s"
timeout = "8s"
path = "/toml/health"

[rules]
default_action = "allow"
watch_rules_file = false
reload_interval = "15s"

[logging]
level = "error"
max_size = 25
max_backups = 1
max_age = 3
compress = false
audit_enabled = false
`

	err := os.WriteFile(configFile, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load config
	cm := NewConfigManager(configFile)
	config, err := cm.LoadConfig()

	if err != nil {
		t.Errorf("Expected no error loading TOML config, got: %v", err)
	}

	// Verify loaded values
	if config.Server.Host != "tomlhost" {
		t.Errorf("Expected host 'tomlhost', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 6666 {
		t.Errorf("Expected port 6666, got %d", config.Server.Port)
	}

	if config.Backend.HealthCheck.Path != "/toml/health" {
		t.Errorf("Expected health path '/toml/health', got '%s'", config.Backend.HealthCheck.Path)
	}
}

func TestConfigManager_LoadConfig_InvalidFile(t *testing.T) {
	// Test loading non-existent file
	cm := NewConfigManager("non-existent-file.yaml")
	_, err := cm.LoadConfig()

	if err == nil {
		t.Errorf("Expected error loading non-existent file, got nil")
	}
}

func TestConfigManager_LoadConfig_InvalidFormat(t *testing.T) {
	// Create file with invalid JSON
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(configFile, []byte("invalid json content {"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cm := NewConfigManager(configFile)
	_, err = cm.LoadConfig()

	if err == nil {
		t.Errorf("Expected error loading invalid JSON, got nil")
	}
}

func TestConfigManager_SaveConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		format   string
	}{
		{
			name:     "Save YAML config",
			filename: "save_test.yaml",
			format:   "yaml",
		},
		{
			name:     "Save JSON config",
			filename: "save_test.json",
			format:   "json",
		},
		{
			name:     "Save TOML config",
			filename: "save_test.toml",
			format:   "toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(tempDir, tt.filename)
			cm := NewConfigManager(configFile)

			// Create a test config
			config := &types.ProxyConfig{
				Server: types.ServerConfig{
					Host: "testhost",
					Port: 8888,
				},
				Backend: types.BackendConfig{
					Host: "testbackend",
					Port: 9999,
				},
				Rules: types.RulesConfig{
					DefaultAction: types.ActionBlock,
					Rules: []types.Rule{
						{
							ID:       "save-test-rule",
							Name:     "Save Test Rule",
							Type:     types.RuleTypeURL,
							Operator: types.MatchEquals,
							Value:    "/save-test",
							Action:   types.ActionAllow,
							Priority: 100,
							Enabled:  true,
						},
					},
				},
				Logging: types.LoggingConfig{
					Level: "debug",
				},
			}

			// Save config
			err := cm.SaveConfig(config)
			if err != nil {
				t.Errorf("Expected no error saving config, got: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				t.Errorf("Config file should exist after saving")
			}

			// Load and verify content
			loadedConfig, err := cm.LoadConfig()
			if err != nil {
				t.Errorf("Expected no error loading saved config, got: %v", err)
			}

			if loadedConfig.Server.Host != "testhost" {
				t.Errorf("Expected host 'testhost', got '%s'", loadedConfig.Server.Host)
			}

			if len(loadedConfig.Rules.Rules) != 1 {
				t.Errorf("Expected 1 rule, got %d", len(loadedConfig.Rules.Rules))
			}
		})
	}
}

func TestConfigManager_ValidateAndSetDefaults(t *testing.T) {
	cm := NewConfigManager("")

	// Test config with missing values
	config := &types.ProxyConfig{
		Rules: types.RulesConfig{
			Rules: []types.Rule{
				{
					ID:     "valid-rule",
					Type:   types.RuleTypeURL,
					Action: types.ActionAllow,
				},
				{
					// Missing ID - should cause validation error
					Type:   types.RuleTypeURL,
					Action: types.ActionAllow,
				},
			},
		},
	}

	err := cm.validateAndSetDefaults(config)
	if err == nil {
		t.Errorf("Expected validation error for rule with missing ID")
	}

	// Test config with invalid action
	config = &types.ProxyConfig{
		Rules: types.RulesConfig{
			Rules: []types.Rule{
				{
					ID:     "invalid-action-rule",
					Type:   types.RuleTypeURL,
					Action: "invalid-action",
				},
			},
		},
	}

	err = cm.validateAndSetDefaults(config)
	if err == nil {
		t.Errorf("Expected validation error for rule with invalid action")
	}

	// Test valid config with defaults
	config = &types.ProxyConfig{
		Rules: types.RulesConfig{
			Rules: []types.Rule{
				{
					ID:     "valid-rule",
					Type:   types.RuleTypeURL,
					Action: types.ActionAllow,
				},
			},
		},
	}

	err = cm.validateAndSetDefaults(config)
	if err != nil {
		t.Errorf("Expected no validation error for valid config, got: %v", err)
	}

	// Verify defaults were set
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}

	if config.Rules.DefaultAction != types.ActionAllow {
		t.Errorf("Expected default action 'allow', got '%s'", config.Rules.DefaultAction)
	}
}

func TestCreateSampleConfigs(t *testing.T) {
	tempDir := t.TempDir()

	err := CreateSampleConfigs(tempDir)
	if err != nil {
		t.Errorf("Expected no error creating sample configs, got: %v", err)
	}

	// Verify files were created
	expectedFiles := []string{
		"proxy.yaml",
		"proxy.json",
		"proxy.toml",
	}

	for _, filename := range expectedFiles {
		filepath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Sample config file %s should exist", filename)
		}
	}

	// Verify one of the files can be loaded
	yamlPath := filepath.Join(tempDir, "proxy.yaml")
	cm := NewConfigManager(yamlPath)
	config, err := cm.LoadConfig()

	if err != nil {
		t.Errorf("Expected to load sample YAML config, got error: %v", err)
	}

	if config == nil {
		t.Errorf("Expected config to be loaded")
	}

	// Verify sample rules exist
	if len(config.Rules.Rules) == 0 {
		t.Errorf("Expected sample rules to exist")
	}
}

func TestConfigManager_GetConfig(t *testing.T) {
	cm := NewConfigManager("")

	// Should return default config when none is loaded
	config := cm.GetConfig()
	if config == nil {
		t.Errorf("Expected config to be returned")
	}

	// Load a specific config
	_, err := cm.LoadConfig()
	if err != nil {
		t.Errorf("Expected no error loading config: %v", err)
	}

	// Should return loaded config
	config2 := cm.GetConfig()
	if config2 == nil {
		t.Errorf("Expected loaded config to be returned")
	}

	// Should be the same instance
	if !reflect.DeepEqual(config, config2) {
		t.Errorf("GetConfig should return consistent results")
	}
}
