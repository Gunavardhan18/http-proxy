package rules

import (
	"net"
	"reflect"
	"testing"

	"http-proxy/pkg/types"
)

func TestNewEngine(t *testing.T) {
	rules := []types.Rule{
		{
			ID:       "rule1",
			Priority: 100,
			Enabled:  true,
		},
		{
			ID:       "rule2",
			Priority: 50,
			Enabled:  true,
		},
	}

	engine := NewEngine(rules, types.ActionAllow)

	if engine.defaultAction != types.ActionAllow {
		t.Errorf("Expected default action to be %v, got %v", types.ActionAllow, engine.defaultAction)
	}

	// Check if rules are sorted by priority (lower number = higher priority)
	engineRules := engine.GetRules()
	if len(engineRules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(engineRules))
	}

	if engineRules[0].ID != "rule2" || engineRules[1].ID != "rule1" {
		t.Errorf("Rules not sorted correctly by priority")
	}
}

func TestEngine_MatchIPv4(t *testing.T) {
	tests := []struct {
		name         string
		rule         types.Rule
		clientIP     string
		expectMatch  bool
		expectReason string
	}{
		{
			name: "IPv4 exact match",
			rule: types.Rule{
				ID:       "ipv4-exact",
				Type:     types.RuleTypeIPv4,
				Operator: types.MatchEquals,
				Value:    "192.168.1.100",
				Action:   types.ActionBlock,
			},
			clientIP:     "192.168.1.100",
			expectMatch:  true,
			expectReason: "IP 192.168.1.100 equals 192.168.1.100",
		},
		{
			name: "IPv4 CIDR range match",
			rule: types.Rule{
				ID:       "ipv4-range",
				Type:     types.RuleTypeIPv4,
				Operator: types.MatchInRange,
				Value:    "192.168.1.0/24",
				Action:   types.ActionBlock,
			},
			clientIP:     "192.168.1.50",
			expectMatch:  true,
			expectReason: "IP 192.168.1.50 is in range 192.168.1.0/24",
		},
		{
			name: "IPv4 no match",
			rule: types.Rule{
				ID:       "ipv4-no-match",
				Type:     types.RuleTypeIPv4,
				Operator: types.MatchEquals,
				Value:    "10.0.0.1",
				Action:   types.ActionBlock,
			},
			clientIP:    "192.168.1.100",
			expectMatch: false,
		},
		{
			name: "IPv6 on IPv4 rule",
			rule: types.Rule{
				ID:       "ipv4-rule",
				Type:     types.RuleTypeIPv4,
				Operator: types.MatchEquals,
				Value:    "192.168.1.100",
				Action:   types.ActionBlock,
			},
			clientIP:    "2001:db8::1",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{}, types.ActionAllow)

			req := &types.RequestInfo{
				ClientIP: net.ParseIP(tt.clientIP),
			}

			matched, reason := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}

			if tt.expectMatch && reason != tt.expectReason {
				t.Errorf("Expected reason: %q, got: %q", tt.expectReason, reason)
			}
		})
	}
}

func TestEngine_MatchURL(t *testing.T) {
	tests := []struct {
		name         string
		rule         types.Rule
		requestURL   string
		expectMatch  bool
		expectReason string
	}{
		{
			name: "URL exact match",
			rule: types.Rule{
				ID:       "url-exact",
				Type:     types.RuleTypeURL,
				Operator: types.MatchEquals,
				Value:    "/admin",
				Action:   types.ActionBlock,
			},
			requestURL:   "/admin",
			expectMatch:  true,
			expectReason: "URL '/admin' equals '/admin'",
		},
		{
			name: "URL starts with",
			rule: types.Rule{
				ID:       "url-starts",
				Type:     types.RuleTypeURL,
				Operator: types.MatchStartsWith,
				Value:    "/api",
				Action:   types.ActionBlock,
			},
			requestURL:   "/api/users",
			expectMatch:  true,
			expectReason: "URL '/api/users' starts with '/api'",
		},
		{
			name: "URL contains",
			rule: types.Rule{
				ID:       "url-contains",
				Type:     types.RuleTypeURL,
				Operator: types.MatchContains,
				Value:    "admin",
				Action:   types.ActionBlock,
			},
			requestURL:   "/secure/admin/panel",
			expectMatch:  true,
			expectReason: "URL '/secure/admin/panel' contains 'admin'",
		},
		{
			name: "URL wildcard match",
			rule: types.Rule{
				ID:       "url-wildcard",
				Type:     types.RuleTypeURL,
				Operator: types.MatchWildcard,
				Value:    "/api/*/delete",
				Action:   types.ActionBlock,
			},
			requestURL:  "/api/users/delete",
			expectMatch: true,
		},
		{
			name: "URL no match",
			rule: types.Rule{
				ID:       "url-no-match",
				Type:     types.RuleTypeURL,
				Operator: types.MatchStartsWith,
				Value:    "/admin",
				Action:   types.ActionBlock,
			},
			requestURL:  "/public/info",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{}, types.ActionAllow)

			req := &types.RequestInfo{
				URL: tt.requestURL,
			}

			matched, reason := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}

			if tt.expectMatch && tt.expectReason != "" && reason != tt.expectReason {
				t.Errorf("Expected reason: %q, got: %q", tt.expectReason, reason)
			}
		})
	}
}

func TestEngine_MatchUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		rule        types.Rule
		userAgent   string
		expectMatch bool
	}{
		{
			name: "User agent contains bot",
			rule: types.Rule{
				ID:       "block-bots",
				Type:     types.RuleTypeUserAgent,
				Operator: types.MatchContains,
				Value:    "bot",
				Action:   types.ActionBlock,
			},
			userAgent:   "BadBot/1.0",
			expectMatch: true,
		},
		{
			name: "User agent regex match",
			rule: types.Rule{
				ID:       "block-crawlers",
				Type:     types.RuleTypeUserAgent,
				Operator: types.MatchRegex,
				Value:    `(?i)(bot|crawler|spider)`,
				Action:   types.ActionBlock,
			},
			userAgent:   "GoogleCrawler/1.0",
			expectMatch: true,
		},
		{
			name: "Normal user agent",
			rule: types.Rule{
				ID:       "block-bots",
				Type:     types.RuleTypeUserAgent,
				Operator: types.MatchContains,
				Value:    "bot",
				Action:   types.ActionBlock,
			},
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{tt.rule}, types.ActionAllow)

			req := &types.RequestInfo{
				UserAgent: tt.userAgent,
			}

			matched, _ := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}
		})
	}
}

func TestEngine_MatchSize(t *testing.T) {
	tests := []struct {
		name        string
		rule        types.Rule
		requestSize int64
		expectMatch bool
	}{
		{
			name: "Size greater than or equal",
			rule: types.Rule{
				ID:       "block-large",
				Type:     types.RuleTypeSize,
				Operator: types.MatchGTE,
				MinSize:  &[]int64{1024 * 1024}[0], // 1MB
				Action:   types.ActionBlock,
			},
			requestSize: 2 * 1024 * 1024, // 2MB
			expectMatch: true,
		},
		{
			name: "Size less than or equal",
			rule: types.Rule{
				ID:       "allow-small",
				Type:     types.RuleTypeSize,
				Operator: types.MatchLTE,
				MaxSize:  &[]int64{1024}[0], // 1KB
				Action:   types.ActionAllow,
			},
			requestSize: 512, // 512 bytes
			expectMatch: true,
		},
		{
			name: "Size in range",
			rule: types.Rule{
				ID:       "medium-size",
				Type:     types.RuleTypeSize,
				Operator: types.MatchInRange,
				MinSize:  &[]int64{1024}[0],      // 1KB
				MaxSize:  &[]int64{1024 * 10}[0], // 10KB
				Action:   types.ActionAllow,
			},
			requestSize: 5 * 1024, // 5KB
			expectMatch: true,
		},
		{
			name: "Size not in range",
			rule: types.Rule{
				ID:       "medium-size",
				Type:     types.RuleTypeSize,
				Operator: types.MatchInRange,
				MinSize:  &[]int64{1024}[0],      // 1KB
				MaxSize:  &[]int64{1024 * 10}[0], // 10KB
				Action:   types.ActionAllow,
			},
			requestSize: 50 * 1024, // 50KB
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{}, types.ActionAllow)

			req := &types.RequestInfo{
				Size: tt.requestSize,
			}

			matched, _ := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}
		})
	}
}

func TestEngine_MatchMethod(t *testing.T) {
	tests := []struct {
		name        string
		rule        types.Rule
		method      string
		expectMatch bool
	}{
		{
			name: "Method exact match",
			rule: types.Rule{
				ID:       "block-delete",
				Type:     types.RuleTypeMethod,
				Operator: types.MatchEquals,
				Value:    "DELETE",
				Action:   types.ActionBlock,
			},
			method:      "DELETE",
			expectMatch: true,
		},
		{
			name: "Method no match",
			rule: types.Rule{
				ID:       "block-delete",
				Type:     types.RuleTypeMethod,
				Operator: types.MatchEquals,
				Value:    "DELETE",
				Action:   types.ActionBlock,
			},
			method:      "GET",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{}, types.ActionAllow)

			req := &types.RequestInfo{
				Method: tt.method,
			}

			matched, _ := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}
		})
	}
}

func TestEngine_MatchHeader(t *testing.T) {
	tests := []struct {
		name        string
		rule        types.Rule
		headers     map[string][]string
		expectMatch bool
	}{
		{
			name: "Header exact match",
			rule: types.Rule{
				ID:          "block-api-key",
				Type:        types.RuleTypeHeader,
				Operator:    types.MatchEquals,
				HeaderName:  "X-API-Key",
				HeaderValue: "blocked-key",
				Action:      types.ActionBlock,
			},
			headers: map[string][]string{
				"x-api-key": {"blocked-key"},
			},
			expectMatch: true,
		},
		{
			name: "Header contains match",
			rule: types.Rule{
				ID:          "block-admin-header",
				Type:        types.RuleTypeHeader,
				Operator:    types.MatchContains,
				HeaderName:  "Authorization",
				HeaderValue: "admin",
				Action:      types.ActionBlock,
			},
			headers: map[string][]string{
				"authorization": {"Bearer admin-token-123"},
			},
			expectMatch: true,
		},
		{
			name: "Header not present",
			rule: types.Rule{
				ID:          "require-auth",
				Type:        types.RuleTypeHeader,
				Operator:    types.MatchEquals,
				HeaderName:  "Authorization",
				HeaderValue: "Bearer token",
				Action:      types.ActionBlock,
			},
			headers:     map[string][]string{},
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine([]types.Rule{}, types.ActionAllow)

			req := &types.RequestInfo{
				Headers: tt.headers,
			}

			matched, _ := engine.matchRule(&tt.rule, req)

			if matched != tt.expectMatch {
				t.Errorf("Expected match: %v, got: %v", tt.expectMatch, matched)
			}
		})
	}
}

func TestEngine_EvaluateRequest(t *testing.T) {
	rules := []types.Rule{
		{
			ID:       "high-priority-block",
			Type:     types.RuleTypeURL,
			Operator: types.MatchStartsWith,
			Value:    "/admin",
			Action:   types.ActionBlock,
			Priority: 10,
			Enabled:  true,
		},
		{
			ID:       "low-priority-allow",
			Type:     types.RuleTypeURL,
			Operator: types.MatchStartsWith,
			Value:    "/",
			Action:   types.ActionAllow,
			Priority: 100,
			Enabled:  true,
		},
		{
			ID:       "disabled-rule",
			Type:     types.RuleTypeURL,
			Operator: types.MatchEquals,
			Value:    "/admin/test",
			Action:   types.ActionAllow,
			Priority: 5,
			Enabled:  false,
		},
	}

	engine := NewEngine(rules, types.ActionAllow)

	tests := []struct {
		name           string
		request        *types.RequestInfo
		expectedAction types.Action
		expectedRuleID string
	}{
		{
			name: "Admin path blocked by high priority rule",
			request: &types.RequestInfo{
				URL: "/admin/users",
			},
			expectedAction: types.ActionBlock,
			expectedRuleID: "high-priority-block",
		},
		{
			name: "Root path allowed by low priority rule",
			request: &types.RequestInfo{
				URL: "/public/info",
			},
			expectedAction: types.ActionAllow,
			expectedRuleID: "low-priority-allow",
		},
		{
			name: "Disabled rule not matched",
			request: &types.RequestInfo{
				URL: "/admin/test",
			},
			expectedAction: types.ActionBlock, // Should match high-priority rule instead
			expectedRuleID: "high-priority-block",
		},
		{
			name: "No rules match, use default",
			request: &types.RequestInfo{
				URL:    "ftp://example.com/file", // Doesn't start with "/"
				Method: "POST",
			},
			expectedAction: types.ActionAllow,
			expectedRuleID: "", // No rule matched
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.EvaluateRequest(tt.request)

			if result.Action != tt.expectedAction {
				t.Errorf("Expected action: %v, got: %v", tt.expectedAction, result.Action)
			}

			if tt.expectedRuleID != "" {
				if result.Rule == nil {
					t.Errorf("Expected rule to be set, got nil")
				} else if result.Rule.ID != tt.expectedRuleID {
					t.Errorf("Expected rule ID: %s, got: %s", tt.expectedRuleID, result.Rule.ID)
				}
			} else if result.Rule != nil {
				t.Errorf("Expected no rule to match, but got: %s", result.Rule.ID)
			}
		})
	}
}

func TestEngine_AddRemoveRule(t *testing.T) {
	engine := NewEngine([]types.Rule{}, types.ActionAllow)

	// Add a rule
	newRule := types.Rule{
		ID:       "new-rule",
		Name:     "New Test Rule",
		Type:     types.RuleTypeURL,
		Operator: types.MatchEquals,
		Value:    "/test",
		Action:   types.ActionBlock,
		Priority: 200,
		Enabled:  true,
	}

	engine.AddRule(newRule)

	rules := engine.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after adding, got %d", len(rules))
	}

	// Check if rule was added correctly
	foundRule, exists := engine.GetRuleByID("new-rule")
	if !exists {
		t.Errorf("Rule should exist after adding")
	}
	if !reflect.DeepEqual(*foundRule, newRule) {
		t.Errorf("Added rule doesn't match expected rule")
	}

	// Remove the rule
	removed := engine.RemoveRule("new-rule")
	if !removed {
		t.Errorf("Rule should have been removed")
	}

	rules = engine.GetRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after removing, got %d", len(rules))
	}

	// Try to remove non-existent rule
	removed = engine.RemoveRule("non-existent")
	if removed {
		t.Errorf("Should not be able to remove non-existent rule")
	}
}

func TestEngine_EnableDisableRule(t *testing.T) {
	rule := types.Rule{
		ID:       "test-rule",
		Name:     "Test Rule",
		Type:     types.RuleTypeURL,
		Operator: types.MatchEquals,
		Value:    "/test",
		Action:   types.ActionBlock,
		Priority: 100,
		Enabled:  true,
	}

	engine := NewEngine([]types.Rule{rule}, types.ActionAllow)

	// Disable rule
	disabled := engine.DisableRule("test-rule")
	if !disabled {
		t.Errorf("Should be able to disable existing rule")
	}

	foundRule, _ := engine.GetRuleByID("test-rule")
	if foundRule.Enabled {
		t.Errorf("Rule should be disabled")
	}

	// Enable rule
	enabled := engine.EnableRule("test-rule")
	if !enabled {
		t.Errorf("Should be able to enable existing rule")
	}

	foundRule, _ = engine.GetRuleByID("test-rule")
	if !foundRule.Enabled {
		t.Errorf("Rule should be enabled")
	}

	// Try to enable non-existent rule
	enabled = engine.EnableRule("non-existent")
	if enabled {
		t.Errorf("Should not be able to enable non-existent rule")
	}
}

func TestEngine_UpdateRules(t *testing.T) {
	initialRules := []types.Rule{
		{
			ID:       "rule1",
			Priority: 100,
		},
		{
			ID:       "rule2",
			Priority: 200,
		},
	}

	engine := NewEngine(initialRules, types.ActionAllow)

	newRules := []types.Rule{
		{
			ID:       "rule3",
			Priority: 50,
		},
		{
			ID:       "rule4",
			Priority: 150,
		},
	}

	engine.UpdateRules(newRules)

	rules := engine.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules after update, got %d", len(rules))
	}

	// Check if rules are sorted by priority
	if rules[0].ID != "rule3" || rules[1].ID != "rule4" {
		t.Errorf("Rules not sorted correctly after update")
	}

	// Check that old rules are gone
	_, exists := engine.GetRuleByID("rule1")
	if exists {
		t.Errorf("Old rule should not exist after update")
	}
}
