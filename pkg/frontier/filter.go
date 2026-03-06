package frontier

import (
	"regexp"
	"sync"
)

// FilterRule defines a URL filtering rule.
type FilterRule struct {
	Pattern *regexp.Regexp
	Allow   bool // true = allow, false = deny
}

// Filter evaluates URLs against a set of allow/deny rules.
// Rules are evaluated in order; the first match wins.
// If no rules match, the URL is allowed by default.
type Filter struct {
	mu    sync.RWMutex
	rules []FilterRule
}

// NewFilter creates a new URL filter with no rules.
func NewFilter() *Filter {
	return &Filter{}
}

// AddAllowRule adds a pattern that explicitly allows matching URLs.
func (f *Filter) AddAllowRule(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.rules = append(f.rules, FilterRule{Pattern: re, Allow: true})
	f.mu.Unlock()
	return nil
}

// AddDenyRule adds a pattern that blocks matching URLs.
func (f *Filter) AddDenyRule(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.rules = append(f.rules, FilterRule{Pattern: re, Allow: false})
	f.mu.Unlock()
	return nil
}

// IsAllowed returns true if the URL passes the filter rules.
// Rules are evaluated in order; the first match determines the result.
// If no rules match, the URL is allowed.
func (f *Filter) IsAllowed(url string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, rule := range f.rules {
		if rule.Pattern.MatchString(url) {
			return rule.Allow
		}
	}
	return true // default allow
}
