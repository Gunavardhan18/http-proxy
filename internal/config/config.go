package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"http-proxy/pkg/types"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ConfigManager handles configuration loading and management
type ConfigManager struct {
	configPath string
	config     *types.ProxyConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads configuration from file based on file extension
func (cm *ConfigManager) LoadConfig() (*types.ProxyConfig, error) {
	if cm.configPath == "" {
		return cm.getDefaultConfig(), nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", cm.configPath, err)
	}

	config := &types.ProxyConfig{}
	ext := strings.ToLower(filepath.Ext(cm.configPath))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse TOML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Validate and set defaults
	if err := cm.validateAndSetDefaults(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cm.config = config
	return config, nil
}

// SaveConfig saves the current configuration to file
func (cm *ConfigManager) SaveConfig(config *types.ProxyConfig) error {
	if cm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	var data []byte
	var err error

	ext := strings.ToLower(filepath.Ext(cm.configPath))

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	case ".toml":
		data, err = toml.Marshal(config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cm.config = config
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *types.ProxyConfig {
	if cm.config == nil {
		return cm.getDefaultConfig()
	}
	return cm.config
}

// validateAndSetDefaults validates configuration and sets default values
func (cm *ConfigManager) validateAndSetDefaults(config *types.ProxyConfig) error {
	// Server defaults
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 120 * time.Second
	}
	if config.Server.MaxHeaderBytes == 0 {
		config.Server.MaxHeaderBytes = 1 << 20 // 1MB
	}

	// Backend defaults
	if config.Backend.Host == "" {
		config.Backend.Host = "localhost"
	}
	if config.Backend.Port == 0 {
		config.Backend.Port = 8090
	}
	if config.Backend.Timeout == 0 {
		config.Backend.Timeout = 30 * time.Second
	}

	// Health check defaults
	if config.Backend.HealthCheck.Interval == 0 {
		config.Backend.HealthCheck.Interval = 30 * time.Second
	}
	if config.Backend.HealthCheck.Timeout == 0 {
		config.Backend.HealthCheck.Timeout = 5 * time.Second
	}
	if config.Backend.HealthCheck.Path == "" {
		config.Backend.HealthCheck.Path = "/health"
	}

	// Rules defaults
	if config.Rules.DefaultAction == "" {
		config.Rules.DefaultAction = types.ActionAllow
	}
	if config.Rules.ReloadInterval == 0 {
		config.Rules.ReloadInterval = 5 * time.Second
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 100 // 100MB
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 3
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 28 // 28 days
	}

	// Rate limiting defaults
	if config.Security.RateLimiting.Enabled {
		if config.Security.RateLimiting.RequestsPerSec == 0 {
			config.Security.RateLimiting.RequestsPerSec = 100
		}
		if config.Security.RateLimiting.BurstSize == 0 {
			config.Security.RateLimiting.BurstSize = 10
		}
		if config.Security.RateLimiting.CleanupInterval == 0 {
			config.Security.RateLimiting.CleanupInterval = 60 * time.Second
		}
	}

	// Validate rules
	for i, rule := range config.Rules.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule at index %d has no ID", i)
		}
		if rule.Type == "" {
			return fmt.Errorf("rule %s has no type", rule.ID)
		}
		if rule.Action != types.ActionAllow && rule.Action != types.ActionBlock {
			return fmt.Errorf("rule %s has invalid action: %s", rule.ID, rule.Action)
		}
	}

	return nil
}

// getDefaultConfig returns a default configuration
func (cm *ConfigManager) getDefaultConfig() *types.ProxyConfig {
	return &types.ProxyConfig{
		Server: types.ServerConfig{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    120 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Backend: types.BackendConfig{
			Host:    "localhost",
			Port:    8090,
			Timeout: 30 * time.Second,
			HealthCheck: types.HealthCheckConfig{
				Enabled:  true,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Path:     "/health",
			},
		},
		Rules: types.RulesConfig{
			DefaultAction:  types.ActionAllow,
			WatchRulesFile: true,
			ReloadInterval: 5 * time.Second,
			Rules: []types.Rule{
				{
					ID:          "default-allow-all",
					Name:        "Allow all requests by default",
					Description: "Default rule to allow all requests",
					Type:        types.RuleTypeURL,
					Operator:    types.MatchWildcard,
					Value:       "*",
					Action:      types.ActionAllow,
					Priority:    1000,
					Enabled:     true,
				},
			},
		},
		Logging: types.LoggingConfig{
			Level:        "info",
			MaxSize:      100,
			MaxBackups:   3,
			MaxAge:       28,
			Compress:     true,
			AuditEnabled: true,
		},
		Security: types.SecurityConfig{
			RateLimiting: types.RateLimitConfig{
				Enabled:         false,
				RequestsPerSec:  100,
				BurstSize:       10,
				CleanupInterval: 60 * time.Second,
			},
		},
	}
}

// CreateSampleConfigs creates sample configuration files in different formats
func CreateSampleConfigs(dir string) error {
	cm := NewConfigManager("")
	config := cm.getDefaultConfig()

	// Add some sample rules
	config.Rules.Rules = append(config.Rules.Rules, []types.Rule{
		{
			ID:          "block-admin-paths",
			Name:        "Block admin paths",
			Description: "Block access to admin endpoints",
			Type:        types.RuleTypeURL,
			Operator:    types.MatchStartsWith,
			Value:       "/admin",
			Action:      types.ActionBlock,
			Priority:    100,
			Enabled:     true,
		},
		{
			ID:          "block-large-requests",
			Name:        "Block large requests",
			Description: "Block requests larger than 10MB",
			Type:        types.RuleTypeSize,
			Operator:    types.MatchGTE,
			MinSize:     &[]int64{10 * 1024 * 1024}[0], // 10MB
			Action:      types.ActionBlock,
			Priority:    200,
			Enabled:     true,
		},
		{
			ID:          "block-bad-user-agents",
			Name:        "Block suspicious user agents",
			Description: "Block requests from known bad user agents",
			Type:        types.RuleTypeUserAgent,
			Operator:    types.MatchContains,
			Value:       "bot",
			Action:      types.ActionBlock,
			Priority:    300,
			Enabled:     false,
		},
	}...)

	// Create YAML config
	yamlPath := filepath.Join(dir, "proxy.yaml")
	yamlCM := NewConfigManager(yamlPath)
	if err := yamlCM.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to create YAML config: %w", err)
	}

	// Create JSON config
	jsonPath := filepath.Join(dir, "proxy.json")
	jsonCM := NewConfigManager(jsonPath)
	if err := jsonCM.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to create JSON config: %w", err)
	}

	// Create TOML config
	tomlPath := filepath.Join(dir, "proxy.toml")
	tomlCM := NewConfigManager(tomlPath)
	if err := tomlCM.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to create TOML config: %w", err)
	}

	return nil
}
