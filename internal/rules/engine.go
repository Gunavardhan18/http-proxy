package rules

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"http-proxy/pkg/types"
)

// Engine represents the rules engine for request filtering
type Engine struct {
	mu            sync.RWMutex
	rules         []types.Rule
	compiledRegex map[string]*regexp.Regexp
	defaultAction types.Action
}

// NewEngine creates a new rules engine
func NewEngine(rules []types.Rule, defaultAction types.Action) *Engine {
	engine := &Engine{
		rules:         make([]types.Rule, len(rules)),
		compiledRegex: make(map[string]*regexp.Regexp),
		defaultAction: defaultAction,
	}

	// Copy rules and sort by priority (lower number = higher priority)
	copy(engine.rules, rules)
	sort.Slice(engine.rules, func(i, j int) bool {
		return engine.rules[i].Priority < engine.rules[j].Priority
	})

	// Pre-compile regex patterns
	engine.compileRegexPatterns()

	return engine
}

// UpdateRules updates the rules in the engine
func (e *Engine) UpdateRules(rules []types.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make([]types.Rule, len(rules))
	copy(e.rules, rules)

	// Sort by priority
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority < e.rules[j].Priority
	})

	// Clear and recompile regex patterns
	e.compiledRegex = make(map[string]*regexp.Regexp)
	e.compileRegexPatterns()
}

// EvaluateRequest evaluates a request against all rules and returns the action to take
func (e *Engine) EvaluateRequest(req *types.RequestInfo) *types.RuleResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		matched, reason := e.matchRule(&rule, req)
		if matched {
			return &types.RuleResult{
				Rule:    &rule,
				Matched: true,
				Action:  rule.Action,
				Reason:  reason,
			}
		}
	}

	// No rules matched, use default action
	return &types.RuleResult{
		Rule:    nil,
		Matched: false,
		Action:  e.defaultAction,
		Reason:  "no rules matched, using default action",
	}
}

// matchRule checks if a single rule matches the request
func (e *Engine) matchRule(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	switch rule.Type {
	case types.RuleTypeIPv4:
		return e.matchIPv4(rule, req)
	case types.RuleTypeIPv6:
		return e.matchIPv6(rule, req)
	case types.RuleTypeURL:
		return e.matchURL(rule, req)
	case types.RuleTypeDomain:
		return e.matchDomain(rule, req)
	case types.RuleTypeUserAgent:
		return e.matchUserAgent(rule, req)
	case types.RuleTypeURISuffix:
		return e.matchURISuffix(rule, req)
	case types.RuleTypeSize:
		return e.matchSize(rule, req)
	case types.RuleTypeMethod:
		return e.matchMethod(rule, req)
	case types.RuleTypeHeader:
		return e.matchHeader(rule, req)
	default:
		return false, fmt.Sprintf("unknown rule type: %s", rule.Type)
	}
}

// matchIPv4 matches IPv4 addresses
func (e *Engine) matchIPv4(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	if req.ClientIP.To4() == nil {
		return false, "request IP is not IPv4"
	}

	return e.matchIP(rule, req.ClientIP.String())
}

// matchIPv6 matches IPv6 addresses
func (e *Engine) matchIPv6(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	if req.ClientIP.To4() != nil {
		return false, "request IP is not IPv6"
	}

	return e.matchIP(rule, req.ClientIP.String())
}

// matchIP matches IP addresses with CIDR support
func (e *Engine) matchIP(rule *types.Rule, clientIP string) (bool, string) {
	switch rule.Operator {
	case types.MatchEquals:
		if clientIP == rule.Value {
			return true, fmt.Sprintf("IP %s equals %s", clientIP, rule.Value)
		}
	case types.MatchInRange:
		// Check if IP is in CIDR range
		_, network, err := net.ParseCIDR(rule.Value)
		if err != nil {
			return false, fmt.Sprintf("invalid CIDR range %s: %v", rule.Value, err)
		}
		ip := net.ParseIP(clientIP)
		if ip != nil && network.Contains(ip) {
			return true, fmt.Sprintf("IP %s is in range %s", clientIP, rule.Value)
		}
	}

	return false, fmt.Sprintf("IP %s does not match rule value %s with operator %s", clientIP, rule.Value, rule.Operator)
}

// matchURL matches URL paths
func (e *Engine) matchURL(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	return e.matchStringValue(rule, req.URL, "URL")
}

// matchDomain matches domain names
func (e *Engine) matchDomain(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	return e.matchStringValue(rule, req.Domain, "domain")
}

// matchUserAgent matches user agent strings
func (e *Engine) matchUserAgent(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	return e.matchStringValue(rule, req.UserAgent, "user agent")
}

// matchURISuffix matches URI suffixes
func (e *Engine) matchURISuffix(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	switch rule.Operator {
	case types.MatchEquals:
		if strings.HasSuffix(req.Path, rule.Value) {
			return true, fmt.Sprintf("URI path %s ends with %s", req.Path, rule.Value)
		}
	case types.MatchWildcard:
		matched, _ := filepath.Match(rule.Value, req.Path)
		if matched {
			return true, fmt.Sprintf("URI path %s matches wildcard %s", req.Path, rule.Value)
		}
	case types.MatchRegex:
		if regex, ok := e.compiledRegex[rule.ID]; ok && regex.MatchString(req.Path) {
			return true, fmt.Sprintf("URI path %s matches regex %s", req.Path, rule.Value)
		}
	}

	return false, fmt.Sprintf("URI path %s does not match suffix rule", req.Path)
}

// matchSize matches request size
func (e *Engine) matchSize(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	switch rule.Operator {
	case types.MatchGTE:
		if rule.MinSize != nil && req.Size >= *rule.MinSize {
			return true, fmt.Sprintf("request size %d >= %d", req.Size, *rule.MinSize)
		}
	case types.MatchLTE:
		if rule.MaxSize != nil && req.Size <= *rule.MaxSize {
			return true, fmt.Sprintf("request size %d <= %d", req.Size, *rule.MaxSize)
		}
	case types.MatchInRange:
		if rule.MinSize != nil && rule.MaxSize != nil {
			if req.Size >= *rule.MinSize && req.Size <= *rule.MaxSize {
				return true, fmt.Sprintf("request size %d is between %d and %d", req.Size, *rule.MinSize, *rule.MaxSize)
			}
		}
	case types.MatchEquals:
		if size, err := strconv.ParseInt(rule.Value, 10, 64); err == nil && req.Size == size {
			return true, fmt.Sprintf("request size %d equals %d", req.Size, size)
		}
	}

	return false, fmt.Sprintf("request size %d does not match size rule", req.Size)
}

// matchMethod matches HTTP methods
func (e *Engine) matchMethod(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	return e.matchStringValue(rule, req.Method, "HTTP method")
}

// matchHeader matches HTTP headers
func (e *Engine) matchHeader(rule *types.Rule, req *types.RequestInfo) (bool, string) {
	headerName := strings.ToLower(rule.HeaderName)
	if headerName == "" {
		return false, "header name not specified"
	}

	headerValues, exists := req.Headers[headerName]
	if !exists {
		return false, fmt.Sprintf("header %s not present", rule.HeaderName)
	}

	// Check against all header values
	for _, headerValue := range headerValues {
		matched, reason := e.matchStringValueDirect(rule.Operator, rule.HeaderValue, headerValue, fmt.Sprintf("header %s", rule.HeaderName), rule.ID)
		if matched {
			return true, reason
		}
	}

	return false, fmt.Sprintf("header %s values do not match rule", rule.HeaderName)
}

// matchStringValue matches string values using various operators
func (e *Engine) matchStringValue(rule *types.Rule, value, fieldName string) (bool, string) {
	return e.matchStringValueDirect(rule.Operator, rule.Value, value, fieldName, rule.ID)
}

// matchStringValueDirect matches string values directly
func (e *Engine) matchStringValueDirect(operator types.MatchOperator, ruleValue, actualValue, fieldName, ruleID string) (bool, string) {
	switch operator {
	case types.MatchEquals:
		if actualValue == ruleValue {
			return true, fmt.Sprintf("%s '%s' equals '%s'", fieldName, actualValue, ruleValue)
		}
	case types.MatchContains:
		if strings.Contains(strings.ToLower(actualValue), strings.ToLower(ruleValue)) {
			return true, fmt.Sprintf("%s '%s' contains '%s'", fieldName, actualValue, ruleValue)
		}
	case types.MatchStartsWith:
		if strings.HasPrefix(strings.ToLower(actualValue), strings.ToLower(ruleValue)) {
			return true, fmt.Sprintf("%s '%s' starts with '%s'", fieldName, actualValue, ruleValue)
		}
	case types.MatchEndsWith:
		if strings.HasSuffix(strings.ToLower(actualValue), strings.ToLower(ruleValue)) {
			return true, fmt.Sprintf("%s '%s' ends with '%s'", fieldName, actualValue, ruleValue)
		}
	case types.MatchWildcard:
		matched, _ := filepath.Match(ruleValue, actualValue)
		if matched {
			return true, fmt.Sprintf("%s '%s' matches wildcard '%s'", fieldName, actualValue, ruleValue)
		}
	case types.MatchRegex:
		if regex, ok := e.compiledRegex[ruleID]; ok && regex.MatchString(actualValue) {
			return true, fmt.Sprintf("%s '%s' matches regex '%s'", fieldName, actualValue, ruleValue)
		}
	}

	return false, fmt.Sprintf("%s '%s' does not match '%s' with operator %s", fieldName, actualValue, ruleValue, operator)
}

// compileRegexPatterns pre-compiles regex patterns for better performance
func (e *Engine) compileRegexPatterns() {
	for _, rule := range e.rules {
		if rule.Operator == types.MatchRegex && rule.Value != "" {
			if regex, err := regexp.Compile(rule.Value); err == nil {
				e.compiledRegex[rule.ID] = regex
			}
		}
	}
}

// GetRules returns a copy of all rules
func (e *Engine) GetRules() []types.Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]types.Rule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// GetRuleByID returns a rule by its ID
func (e *Engine) GetRuleByID(id string) (*types.Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if rule.ID == id {
			return &rule, true
		}
	}
	return nil, false
}

// AddRule adds a new rule to the engine
func (e *Engine) AddRule(rule types.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = append(e.rules, rule)

	// Re-sort by priority
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority < e.rules[j].Priority
	})

	// Compile regex if needed
	if rule.Operator == types.MatchRegex && rule.Value != "" {
		if regex, err := regexp.Compile(rule.Value); err == nil {
			e.compiledRegex[rule.ID] = regex
		}
	}
}

// RemoveRule removes a rule by its ID
func (e *Engine) RemoveRule(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, rule := range e.rules {
		if rule.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			delete(e.compiledRegex, id)
			return true
		}
	}
	return false
}

// EnableRule enables a rule by its ID
func (e *Engine) EnableRule(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, rule := range e.rules {
		if rule.ID == id {
			e.rules[i].Enabled = true
			return true
		}
	}
	return false
}

// DisableRule disables a rule by its ID
func (e *Engine) DisableRule(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, rule := range e.rules {
		if rule.ID == id {
			e.rules[i].Enabled = false
			return true
		}
	}
	return false
}
