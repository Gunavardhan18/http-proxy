package rules

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

func TestNewManager(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction:  types.ActionAllow,
		WatchRulesFile: false,
		ReloadInterval: 5 * time.Second,
		Rules: []types.Rule{
			{
				ID:       "test-rule",
				Name:     "Test Rule",
				Type:     types.RuleTypeURL,
				Operator: types.MatchEquals,
				Value:    "/test",
				Action:   types.ActionBlock,
				Priority: 100,
				Enabled:  true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	// Check if rules were loaded
	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].ID != "test-rule" {
		t.Errorf("Expected rule ID 'test-rule', got '%s'", rules[0].ID)
	}
}

func TestNewManager_WithRulesFile(t *testing.T) {
	// Create temporary rules file
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "test-rules.yaml")

	rulesData := struct {
		Rules []types.Rule `yaml:"rules"`
	}{
		Rules: []types.Rule{
			{
				ID:       "file-rule",
				Name:     "File Rule",
				Type:     types.RuleTypeURL,
				Operator: types.MatchStartsWith,
				Value:    "/api",
				Action:   types.ActionAllow,
				Priority: 200,
				Enabled:  true,
			},
		},
	}

	yamlData, err := yaml.Marshal(&rulesData)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(rulesFile, yamlData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &types.RulesConfig{
		DefaultAction:  types.ActionBlock,
		RulesFile:      rulesFile,
		WatchRulesFile: false,
		ReloadInterval: 5 * time.Second,
		Rules: []types.Rule{
			{
				ID: "config-rule",
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager with rules file, got: %v", err)
	}

	// Should load rules from file (overriding config rules)
	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule from file, got %d", len(rules))
	}

	if rules[0].ID != "file-rule" {
		t.Errorf("Expected rule ID 'file-rule', got '%s'", rules[0].ID)
	}
}

func TestNewManager_NonExistentFile(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction:  types.ActionAllow,
		RulesFile:      "/non/existent/file.yaml",
		WatchRulesFile: false,
		Rules:          []types.Rule{},
	}

	// Should not fail even if file doesn't exist
	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error with non-existent file, got: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
}

func TestManager_LoadRulesFromFile_YAML(t *testing.T) {
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "test.yaml")

	rulesData := struct {
		Rules []types.Rule `yaml:"rules"`
	}{
		Rules: []types.Rule{
			{
				ID:       "yaml-rule-1",
				Name:     "YAML Rule 1",
				Type:     types.RuleTypeURL,
				Operator: types.MatchEquals,
				Value:    "/yaml1",
				Action:   types.ActionBlock,
				Priority: 100,
				Enabled:  true,
			},
			{
				ID:       "yaml-rule-2",
				Name:     "YAML Rule 2",
				Type:     types.RuleTypeUserAgent,
				Operator: types.MatchContains,
				Value:    "bot",
				Action:   types.ActionBlock,
				Priority: 200,
				Enabled:  false,
			},
		},
	}

	yamlData, err := yaml.Marshal(&rulesData)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(rulesFile, yamlData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &types.RulesConfig{
		RulesFile:      rulesFile,
		DefaultAction:  types.ActionAllow,
		WatchRulesFile: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	rules := manager.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	// Verify rules are sorted by priority
	if rules[0].ID != "yaml-rule-1" || rules[1].ID != "yaml-rule-2" {
		t.Errorf("Rules not sorted correctly by priority")
	}
}

func TestManager_LoadRulesFromFile_JSON(t *testing.T) {
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "test.json")

	rulesData := struct {
		Rules []types.Rule `json:"rules"`
	}{
		Rules: []types.Rule{
			{
				ID:       "json-rule",
				Name:     "JSON Rule",
				Type:     types.RuleTypeSize,
				Operator: types.MatchGTE,
				MinSize:  &[]int64{1024 * 1024}[0],
				Action:   types.ActionBlock,
				Priority: 300,
				Enabled:  true,
			},
		},
	}

	jsonData, err := json.MarshalIndent(&rulesData, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(rulesFile, jsonData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &types.RulesConfig{
		RulesFile:      rulesFile,
		DefaultAction:  types.ActionAllow,
		WatchRulesFile: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].ID != "json-rule" {
		t.Errorf("Expected rule ID 'json-rule', got '%s'", rules[0].ID)
	}
}

func TestManager_SaveRulesToFile(t *testing.T) {
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "save-test.yaml")

	config := &types.RulesConfig{
		RulesFile:     rulesFile,
		DefaultAction: types.ActionAllow,
		Rules: []types.Rule{
			{
				ID:       "save-rule",
				Name:     "Save Test Rule",
				Type:     types.RuleTypeURL,
				Operator: types.MatchEquals,
				Value:    "/save",
				Action:   types.ActionBlock,
				Priority: 100,
				Enabled:  true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	// Save rules to file
	err = manager.SaveRulesToFile()
	if err != nil {
		t.Errorf("Expected no error saving rules, got: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(rulesFile); os.IsNotExist(err) {
		t.Errorf("Rules file should exist after saving")
	}

	// Load and verify content
	manager2, err := NewManager(&types.RulesConfig{
		RulesFile:     rulesFile,
		DefaultAction: types.ActionBlock,
	})
	if err != nil {
		t.Errorf("Expected no error loading saved rules, got: %v", err)
	}

	rules := manager2.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 saved rule, got %d", len(rules))
	}

	if rules[0].ID != "save-rule" {
		t.Errorf("Expected saved rule ID 'save-rule', got '%s'", rules[0].ID)
	}
}

func TestManager_AddRule(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules:         []types.Rule{},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	newRule := types.Rule{
		ID:       "new-rule",
		Name:     "New Rule",
		Type:     types.RuleTypeURL,
		Operator: types.MatchEquals,
		Value:    "/new",
		Action:   types.ActionBlock,
		Priority: 100,
		Enabled:  true,
	}

	// Add rule
	manager.AddRule(newRule)

	// Verify rule was added
	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after adding, got %d", len(rules))
	}

	foundRule, exists := manager.GetRuleByID("new-rule")
	if !exists {
		t.Errorf("Added rule should exist")
	}

	if !reflect.DeepEqual(*foundRule, newRule) {
		t.Errorf("Added rule doesn't match expected rule")
	}
}

func TestManager_RemoveRule(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules: []types.Rule{
			{
				ID:     "remove-me",
				Action: types.ActionBlock,
			},
			{
				ID:     "keep-me",
				Action: types.ActionAllow,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	// Remove rule
	removed := manager.RemoveRule("remove-me")
	if !removed {
		t.Errorf("Should be able to remove existing rule")
	}

	// Verify rule was removed
	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after removal, got %d", len(rules))
	}

	if rules[0].ID != "keep-me" {
		t.Errorf("Wrong rule remained after removal")
	}

	// Try to remove non-existent rule
	removed = manager.RemoveRule("non-existent")
	if removed {
		t.Errorf("Should not be able to remove non-existent rule")
	}
}

func TestManager_EnableDisableRule(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules: []types.Rule{
			{
				ID:      "toggle-rule",
				Action:  types.ActionBlock,
				Enabled: true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	// Disable rule
	disabled := manager.DisableRule("toggle-rule")
	if !disabled {
		t.Errorf("Should be able to disable existing rule")
	}

	rule, _ := manager.GetRuleByID("toggle-rule")
	if rule.Enabled {
		t.Errorf("Rule should be disabled")
	}

	// Enable rule
	enabled := manager.EnableRule("toggle-rule")
	if !enabled {
		t.Errorf("Should be able to enable existing rule")
	}

	rule, _ = manager.GetRuleByID("toggle-rule")
	if !rule.Enabled {
		t.Errorf("Rule should be enabled")
	}

	// Try to toggle non-existent rule
	enabled = manager.EnableRule("non-existent")
	if enabled {
		t.Errorf("Should not be able to enable non-existent rule")
	}
}

func TestManager_UpdateRules(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules: []types.Rule{
			{ID: "old-rule-1"},
			{ID: "old-rule-2"},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	newRules := []types.Rule{
		{
			ID:       "new-rule-1",
			Priority: 50,
		},
		{
			ID:       "new-rule-2",
			Priority: 100,
		},
	}

	// Update rules
	manager.UpdateRules(newRules)

	// Verify rules were updated
	rules := manager.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules after update, got %d", len(rules))
	}

	// Verify old rules are gone
	_, exists := manager.GetRuleByID("old-rule-1")
	if exists {
		t.Errorf("Old rule should not exist after update")
	}

	// Verify new rules exist and are sorted
	if rules[0].ID != "new-rule-1" || rules[1].ID != "new-rule-2" {
		t.Errorf("New rules not properly updated or sorted")
	}
}

func TestManager_EvaluateRequest(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules: []types.Rule{
			{
				ID:       "block-admin",
				Type:     types.RuleTypeURL,
				Operator: types.MatchStartsWith,
				Value:    "/admin",
				Action:   types.ActionBlock,
				Priority: 100,
				Enabled:  true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	tests := []struct {
		name           string
		request        *types.RequestInfo
		expectedAction types.Action
		expectedRuleID string
	}{
		{
			name: "Admin path blocked",
			request: &types.RequestInfo{
				URL: "/admin/users",
			},
			expectedAction: types.ActionBlock,
			expectedRuleID: "block-admin",
		},
		{
			name: "Normal path allowed",
			request: &types.RequestInfo{
				URL: "/public/info",
			},
			expectedAction: types.ActionAllow,
			expectedRuleID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.EvaluateRequest(tt.request)

			if result.Action != tt.expectedAction {
				t.Errorf("Expected action %v, got %v", tt.expectedAction, result.Action)
			}

			if tt.expectedRuleID != "" {
				if result.Rule == nil {
					t.Errorf("Expected rule to be set")
				} else if result.Rule.ID != tt.expectedRuleID {
					t.Errorf("Expected rule ID %s, got %s", tt.expectedRuleID, result.Rule.ID)
				}
			}
		})
	}
}

func TestManager_FileWatching_DisabledByDefault(t *testing.T) {
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "watch-test.yaml")

	// Create initial rules file
	initialRules := struct {
		Rules []types.Rule `yaml:"rules"`
	}{
		Rules: []types.Rule{
			{ID: "initial-rule", Action: types.ActionAllow},
		},
	}

	yamlData, _ := yaml.Marshal(&initialRules)
	os.WriteFile(rulesFile, yamlData, 0644)

	config := &types.RulesConfig{
		RulesFile:      rulesFile,
		WatchRulesFile: false, // Disabled
		ReloadInterval: 100 * time.Millisecond,
		DefaultAction:  types.ActionAllow,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}
	defer manager.Close()

	// Verify initial rule count
	if len(manager.GetRules()) != 1 {
		t.Errorf("Expected 1 initial rule")
	}

	// Update file
	updatedRules := struct {
		Rules []types.Rule `yaml:"rules"`
	}{
		Rules: []types.Rule{
			{ID: "initial-rule", Action: types.ActionAllow},
			{ID: "new-rule", Action: types.ActionBlock},
		},
	}

	yamlData, _ = yaml.Marshal(&updatedRules)
	os.WriteFile(rulesFile, yamlData, 0644)

	// Wait longer than reload interval
	time.Sleep(300 * time.Millisecond)

	// Should still have only 1 rule (watching disabled)
	if len(manager.GetRules()) != 1 {
		t.Errorf("Expected rules not to be reloaded when watching is disabled")
	}
}

func TestCreateSampleRulesFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
	}{
		{"YAML sample", "sample.yaml"},
		{"JSON sample", "sample.json"},
		{"TOML sample", "sample.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := filepath.Join(tempDir, tt.filename)

			err := CreateSampleRulesFile(filepath)
			if err != nil {
				t.Errorf("Expected no error creating sample file, got: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				t.Errorf("Sample file should exist after creation")
			}

			// Try to load the file
			config := &types.RulesConfig{
				RulesFile:     filepath,
				DefaultAction: types.ActionAllow,
			}

			manager, err := NewManager(config)
			if err != nil {
				t.Errorf("Expected to load sample rules file, got: %v", err)
			}

			// Should have some sample rules
			rules := manager.GetRules()
			if len(rules) == 0 {
				t.Errorf("Sample rules file should contain rules")
			}
		})
	}
}

func TestCreateSampleRulesFile_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	filepath := filepath.Join(tempDir, "sample.unsupported")

	err := CreateSampleRulesFile(filepath)
	if err == nil {
		t.Errorf("Expected error for unsupported file format")
	}
}

func TestManager_Close(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules:         []types.Rule{},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	// Close should not panic or error
	manager.Close()
}

func TestManager_ParseRulesFile_InvalidFormat(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
	}{
		{
			name:     "Invalid YAML",
			filename: "invalid.yaml",
			content:  "invalid: yaml: content: {",
		},
		{
			name:     "Invalid JSON",
			filename: "invalid.json",
			content:  `{"invalid": json}`,
		},
		{
			name:     "Invalid TOML",
			filename: "invalid.toml",
			content:  "[invalid toml content",
		},
		{
			name:     "Unsupported extension",
			filename: "rules.txt",
			content:  "some content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(filepath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			config := &types.RulesConfig{
				RulesFile:     filepath,
				DefaultAction: types.ActionAllow,
			}

			_, err = NewManager(config)
			if err == nil {
				t.Errorf("Expected error loading invalid rules file")
			}
		})
	}
}

func TestManager_GetEngine(t *testing.T) {
	config := &types.RulesConfig{
		DefaultAction: types.ActionAllow,
		Rules:         []types.Rule{},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Errorf("Expected no error creating manager, got: %v", err)
	}

	engine := manager.GetEngine()
	if engine == nil {
		t.Errorf("Expected engine to be returned")
	}

	// Engine should have same rules as manager
	engineRules := engine.GetRules()
	managerRules := manager.GetRules()

	if len(engineRules) != len(managerRules) {
		t.Errorf("Engine and manager should have same number of rules")
	}
}
