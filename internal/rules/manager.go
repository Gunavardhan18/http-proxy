package rules

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"http-proxy/pkg/types"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Manager handles dynamic rule management including file watching and reloading
type Manager struct {
	mu           sync.RWMutex
	engine       *Engine
	rulesFile    string
	watchEnabled bool
	stopWatch    chan bool
	reloadTicker *time.Ticker
	lastModTime  time.Time
}

// NewManager creates a new rules manager
func NewManager(config *types.RulesConfig) (*Manager, error) {
	manager := &Manager{
		rulesFile:    config.RulesFile,
		watchEnabled: config.WatchRulesFile,
		stopWatch:    make(chan bool, 1),
	}

	// Initialize engine with rules from config
	manager.engine = NewEngine(config.Rules, config.DefaultAction)

	// If rules file is specified, load rules from file
	if manager.rulesFile != "" {
		if err := manager.loadRulesFromFile(); err != nil {
			return nil, fmt.Errorf("failed to load rules from file: %w", err)
		}

		// Start watching file if enabled
		if manager.watchEnabled {
			manager.startFileWatcher(config.ReloadInterval)
		}
	}

	return manager, nil
}

// GetEngine returns the rules engine
func (rm *Manager) GetEngine() *Engine {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine
}

// UpdateRules updates the rules in the engine
func (rm *Manager) UpdateRules(rules []types.Rule) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.engine.UpdateRules(rules)
}

// LoadRulesFromFile loads rules from the configured file
func (rm *Manager) loadRulesFromFile() error {
	if rm.rulesFile == "" {
		return fmt.Errorf("no rules file configured")
	}

	fileInfo, err := os.Stat(rm.rulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Rules file %s does not exist, using existing rules", rm.rulesFile)
			return nil
		}
		return fmt.Errorf("failed to stat rules file: %w", err)
	}

	// Check if file has been modified
	if !fileInfo.ModTime().After(rm.lastModTime) {
		return nil // File hasn't been modified
	}

	data, err := os.ReadFile(rm.rulesFile)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	rules, err := rm.parseRulesFile(data, rm.rulesFile)
	if err != nil {
		return fmt.Errorf("failed to parse rules file: %w", err)
	}

	rm.mu.Lock()
	rm.engine.UpdateRules(rules)
	rm.lastModTime = fileInfo.ModTime()
	rm.mu.Unlock()

	log.Printf("Loaded %d rules from %s", len(rules), rm.rulesFile)
	return nil
}

// parseRulesFile parses rules from file data based on file extension
func (rm *Manager) parseRulesFile(data []byte, filename string) ([]types.Rule, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	var rulesWrapper struct {
		Rules []types.Rule `yaml:"rules" json:"rules" toml:"rules"`
	}

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &rulesWrapper); err != nil {
			return nil, fmt.Errorf("failed to parse YAML rules: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &rulesWrapper); err != nil {
			return nil, fmt.Errorf("failed to parse JSON rules: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &rulesWrapper); err != nil {
			return nil, fmt.Errorf("failed to parse TOML rules: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported rules file format: %s", ext)
	}

	return rulesWrapper.Rules, nil
}

// SaveRulesToFile saves current rules to the configured file
func (rm *Manager) SaveRulesToFile() error {
	if rm.rulesFile == "" {
		return fmt.Errorf("no rules file configured")
	}

	rm.mu.RLock()
	rules := rm.engine.GetRules()
	rm.mu.RUnlock()

	rulesWrapper := struct {
		Rules []types.Rule `yaml:"rules" json:"rules" toml:"rules"`
	}{
		Rules: rules,
	}

	var data []byte
	var err error

	ext := strings.ToLower(filepath.Ext(rm.rulesFile))
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(&rulesWrapper)
	case ".json":
		data, err = json.MarshalIndent(&rulesWrapper, "", "  ")
	case ".toml":
		data, err = toml.Marshal(&rulesWrapper)
	default:
		return fmt.Errorf("unsupported rules file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := os.WriteFile(rm.rulesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write rules file: %w", err)
	}

	log.Printf("Saved %d rules to %s", len(rules), rm.rulesFile)
	return nil
}

// startFileWatcher starts watching the rules file for changes
func (rm *Manager) startFileWatcher(interval time.Duration) {
	rm.reloadTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-rm.reloadTicker.C:
				if err := rm.loadRulesFromFile(); err != nil {
					log.Printf("Error reloading rules from file: %v", err)
				}
			case <-rm.stopWatch:
				rm.reloadTicker.Stop()
				return
			}
		}
	}()

	log.Printf("Started file watcher for rules file: %s (interval: %v)", rm.rulesFile, interval)
}

// StopFileWatcher stops the file watcher
func (rm *Manager) StopFileWatcher() {
	if rm.reloadTicker != nil {
		select {
		case rm.stopWatch <- true:
		default:
		}
	}
}

// AddRule adds a new rule
func (rm *Manager) AddRule(rule types.Rule) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.engine.AddRule(rule)
	log.Printf("Added rule: %s", rule.ID)
}

// RemoveRule removes a rule by ID
func (rm *Manager) RemoveRule(id string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.engine.RemoveRule(id) {
		log.Printf("Removed rule: %s", id)
		return true
	}
	return false
}

// EnableRule enables a rule by ID
func (rm *Manager) EnableRule(id string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.engine.EnableRule(id) {
		log.Printf("Enabled rule: %s", id)
		return true
	}
	return false
}

// DisableRule disables a rule by ID
func (rm *Manager) DisableRule(id string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.engine.DisableRule(id) {
		log.Printf("Disabled rule: %s", id)
		return true
	}
	return false
}

// GetRules returns all rules
func (rm *Manager) GetRules() []types.Rule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine.GetRules()
}

// GetRuleByID returns a rule by ID
func (rm *Manager) GetRuleByID(id string) (*types.Rule, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine.GetRuleByID(id)
}

// EvaluateRequest evaluates a request against all rules
func (rm *Manager) EvaluateRequest(req *types.RequestInfo) *types.RuleResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine.EvaluateRequest(req)
}

// Close cleans up the manager
func (rm *Manager) Close() {
	rm.StopFileWatcher()
}

// CreateSampleRulesFile creates a sample rules file
func CreateSampleRulesFile(filename string) error {
	sampleRules := struct {
		Rules []types.Rule `yaml:"rules" json:"rules" toml:"rules"`
	}{
		Rules: []types.Rule{
			{
				ID:          "block-admin",
				Name:        "Block Admin Access",
				Description: "Block access to admin endpoints",
				Type:        types.RuleTypeURL,
				Operator:    types.MatchStartsWith,
				Value:       "/admin",
				Action:      types.ActionBlock,
				Priority:    100,
				Enabled:     true,
			},
			{
				ID:          "block-large-uploads",
				Name:        "Block Large Uploads",
				Description: "Block uploads larger than 50MB",
				Type:        types.RuleTypeSize,
				Operator:    types.MatchGTE,
				MinSize:     &[]int64{50 * 1024 * 1024}[0],
				Action:      types.ActionBlock,
				Priority:    200,
				Enabled:     true,
			},
			{
				ID:          "block-suspicious-uas",
				Name:        "Block Suspicious User Agents",
				Description: "Block requests from suspicious user agents",
				Type:        types.RuleTypeUserAgent,
				Operator:    types.MatchRegex,
				Value:       `(?i)(bot|crawler|spider|scraper)`,
				Action:      types.ActionBlock,
				Priority:    300,
				Enabled:     false,
			},
			{
				ID:          "allow-health-checks",
				Name:        "Allow Health Checks",
				Description: "Always allow health check endpoints",
				Type:        types.RuleTypeURL,
				Operator:    types.MatchEquals,
				Value:       "/health",
				Action:      types.ActionAllow,
				Priority:    50,
				Enabled:     true,
			},
			{
				ID:          "block-private-networks",
				Name:        "Block Private Network Access",
				Description: "Block requests from private network ranges",
				Type:        types.RuleTypeIPv4,
				Operator:    types.MatchInRange,
				Value:       "192.168.0.0/16",
				Action:      types.ActionBlock,
				Priority:    150,
				Enabled:     false,
			},
		},
	}

	var data []byte
	var err error

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(&sampleRules)
	case ".json":
		data, err = json.MarshalIndent(&sampleRules, "", "  ")
	case ".toml":
		data, err = toml.Marshal(&sampleRules)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal sample rules: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample rules file: %w", err)
	}

	return nil
}
