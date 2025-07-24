package filtering

import (
	"fmt"
	"strings"
	"sync"

	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/risk"
)

// EventFilter defines the interface for event filtering
type EventFilter interface {
	ShouldCapture(event *event.Event) bool
	GetFilterReason(event *event.Event) string
	GetFilterStatistics() FilterStatistics
}

// FilteringRules represents the configuration for event filtering
type FilteringRules struct {
	MethodWhitelist        []string          `json:"method_whitelist"`
	MethodBlacklist        []string          `json:"method_blacklist"`
	PayloadSizeLimit       int               `json:"payload_size_limit"`       // bytes, 0 = no limit
	MinimumRiskLevel       risk.RiskLevel    `json:"minimum_risk_level"`       // minimum risk level to capture
	ExcludePingMessages    bool              `json:"exclude_ping_messages"`    // exclude ping messages
	OnlyHighRiskMethods    bool              `json:"only_high_risk_methods"`   // only capture high-risk methods
	DirectionFilter        []event.Direction `json:"direction_filter"`         // filter by direction (inbound/outbound)
	EnableContentFiltering bool              `json:"enable_content_filtering"` // enable content-based filtering
	ContentBlacklist       []string          `json:"content_blacklist"`        // content patterns to exclude
}

// DefaultFilteringRules returns sensible default filtering rules
func DefaultFilteringRules() FilteringRules {
	return FilteringRules{
		MethodWhitelist:        []string{}, // empty = capture all methods
		MethodBlacklist:        []string{}, // empty = no method exclusions
		PayloadSizeLimit:       0,          // 0 = no limit
		MinimumRiskLevel:       risk.RiskLevelLow,
		ExcludePingMessages:    true, // exclude ping by default for noise reduction
		OnlyHighRiskMethods:    false,
		DirectionFilter:        []event.Direction{}, // empty = capture all directions
		EnableContentFiltering: false,
		ContentBlacklist:       []string{},
	}
}

// CompositeFilter implements filtering using multiple filter strategies
type CompositeFilter struct {
	rules           FilteringRules
	methodFilter    *MethodFilter
	sizeFilter      *SizeFilter
	riskFilter      *RiskFilter
	contentFilter   *ContentFilter
	directionFilter *DirectionFilter
	statistics      FilterStatistics
	statsMu         sync.Mutex // Protects statistics updates
	riskAnalyzer    risk.RiskAnalyzer
}

// FilterStatistics tracks filtering statistics
type FilterStatistics struct {
	TotalEvaluated    int `json:"total_evaluated"`
	TotalCaptured     int `json:"total_captured"`
	TotalFiltered     int `json:"total_filtered"`
	MethodFiltered    int `json:"method_filtered"`
	SizeFiltered      int `json:"size_filtered"`
	RiskFiltered      int `json:"risk_filtered"`
	ContentFiltered   int `json:"content_filtered"`
	DirectionFiltered int `json:"direction_filtered"`
	PingFiltered      int `json:"ping_filtered"`
}

// NewCompositeFilter creates a new composite filter with the given rules
func NewCompositeFilter(rules FilteringRules, riskAnalyzer risk.RiskAnalyzer) *CompositeFilter {
	return &CompositeFilter{
		rules:           rules,
		methodFilter:    NewMethodFilter(rules.MethodWhitelist, rules.MethodBlacklist, rules.ExcludePingMessages),
		sizeFilter:      NewSizeFilter(rules.PayloadSizeLimit),
		riskFilter:      NewRiskFilter(rules.MinimumRiskLevel, rules.OnlyHighRiskMethods),
		contentFilter:   NewContentFilter(rules.ContentBlacklist, rules.EnableContentFiltering),
		directionFilter: NewDirectionFilter(rules.DirectionFilter),
		statistics:      FilterStatistics{},
		riskAnalyzer:    riskAnalyzer,
	}
}

// ShouldCapture determines if an event should be captured based on all filtering rules
func (f *CompositeFilter) ShouldCapture(evt *event.Event) bool {
	// Increment total evaluated counter (protected by mutex)
	f.statsMu.Lock()
	f.statistics.TotalEvaluated++
	f.statsMu.Unlock()

	// Apply risk analysis if risk filtering is enabled
	if f.rules.MinimumRiskLevel != risk.RiskLevelLow || f.rules.OnlyHighRiskMethods {
		if riskScore, err := f.riskAnalyzer.AnalyzeEvent(evt); err == nil {
			evt.UpdateRiskScore(riskScore)
		}
	}

	// Apply filters in order of efficiency (fastest first)

	// 1. Method filtering (fastest)
	if !f.methodFilter.ShouldCapture(evt) {
		f.statsMu.Lock()
		f.statistics.MethodFiltered++
		f.statistics.TotalFiltered++
		f.statsMu.Unlock()
		return false
	}

	// 2. Direction filtering
	if !f.directionFilter.ShouldCapture(evt) {
		f.statsMu.Lock()
		f.statistics.DirectionFiltered++
		f.statistics.TotalFiltered++
		f.statsMu.Unlock()
		return false
	}

	// 3. Size filtering
	if !f.sizeFilter.ShouldCapture(evt) {
		f.statsMu.Lock()
		f.statistics.SizeFiltered++
		f.statistics.TotalFiltered++
		f.statsMu.Unlock()
		return false
	}

	// 4. Risk filtering
	if !f.riskFilter.ShouldCapture(evt) {
		f.statsMu.Lock()
		f.statistics.RiskFiltered++
		f.statistics.TotalFiltered++
		f.statsMu.Unlock()
		return false
	}

	// 5. Content filtering (most expensive)
	if !f.contentFilter.ShouldCapture(evt) {
		f.statsMu.Lock()
		f.statistics.ContentFiltered++
		f.statistics.TotalFiltered++
		f.statsMu.Unlock()
		return false
	}

	// Event passed all filters
	f.statsMu.Lock()
	f.statistics.TotalCaptured++
	f.statsMu.Unlock()
	return true
}

// GetFilterReason returns the reason why an event was filtered (for debugging)
func (f *CompositeFilter) GetFilterReason(evt *event.Event) string {
	if !f.methodFilter.ShouldCapture(evt) {
		return f.methodFilter.GetFilterReason(evt)
	}
	if !f.directionFilter.ShouldCapture(evt) {
		return f.directionFilter.GetFilterReason(evt)
	}
	if !f.sizeFilter.ShouldCapture(evt) {
		return f.sizeFilter.GetFilterReason(evt)
	}
	if !f.riskFilter.ShouldCapture(evt) {
		return f.riskFilter.GetFilterReason(evt)
	}
	if !f.contentFilter.ShouldCapture(evt) {
		return f.contentFilter.GetFilterReason(evt)
	}

	return "Event passed all filters"
}

// GetFilterStatistics returns current filtering statistics
func (f *CompositeFilter) GetFilterStatistics() FilterStatistics {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()
	return f.statistics
}

// UpdateRules updates the filtering rules and reconfigures sub-filters
func (f *CompositeFilter) UpdateRules(rules FilteringRules) {
	f.rules = rules
	f.methodFilter = NewMethodFilter(rules.MethodWhitelist, rules.MethodBlacklist, rules.ExcludePingMessages)
	f.sizeFilter = NewSizeFilter(rules.PayloadSizeLimit)
	f.riskFilter = NewRiskFilter(rules.MinimumRiskLevel, rules.OnlyHighRiskMethods)
	f.contentFilter = NewContentFilter(rules.ContentBlacklist, rules.EnableContentFiltering)
	f.directionFilter = NewDirectionFilter(rules.DirectionFilter)
}

// MethodFilter filters events based on method names
type MethodFilter struct {
	whitelist       []string
	blacklist       []string
	excludePing     bool
	wildcardEnabled bool
}

// NewMethodFilter creates a new method filter
func NewMethodFilter(whitelist, blacklist []string, excludePing bool) *MethodFilter {
	return &MethodFilter{
		whitelist:       whitelist,
		blacklist:       blacklist,
		excludePing:     excludePing,
		wildcardEnabled: true,
	}
}

// ShouldCapture determines if an event should be captured based on method filtering
func (f *MethodFilter) ShouldCapture(evt *event.Event) bool {
	method := evt.Method().Value()

	// Handle ping message exclusion
	if f.excludePing && strings.ToLower(method) == "ping" {
		return false
	}

	// Check blacklist first (exclusion takes precedence)
	if len(f.blacklist) > 0 {
		for _, blacklistedMethod := range f.blacklist {
			if f.matchesPattern(method, blacklistedMethod) {
				return false
			}
		}
	}

	// If whitelist is empty, capture all methods (except blacklisted)
	if len(f.whitelist) == 0 {
		return true
	}

	// Check if method is in whitelist
	for _, whitelistedMethod := range f.whitelist {
		if f.matchesPattern(method, whitelistedMethod) {
			return true
		}
	}

	return false
}

// GetFilterReason returns the reason for filtering
func (f *MethodFilter) GetFilterReason(evt *event.Event) string {
	method := evt.Method().Value()

	if f.excludePing && strings.ToLower(method) == "ping" {
		return "ping message excluded"
	}

	for _, blacklistedMethod := range f.blacklist {
		if f.matchesPattern(method, blacklistedMethod) {
			return fmt.Sprintf("method %s is blacklisted", method)
		}
	}

	if len(f.whitelist) > 0 {
		return fmt.Sprintf("method %s not in whitelist", method)
	}

	return ""
}

// matchesPattern checks if a method matches a pattern (supports wildcards)
func (f *MethodFilter) matchesPattern(method, pattern string) bool {
	if !f.wildcardEnabled {
		return method == pattern
	}

	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	// Prefix matching with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(method, prefix)
	}

	// Suffix matching with *
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(method, suffix)
	}

	// Exact match
	return method == pattern
}

// SizeFilter filters events based on payload size
type SizeFilter struct {
	maxSize int
}

// NewSizeFilter creates a new size filter
func NewSizeFilter(maxSize int) *SizeFilter {
	return &SizeFilter{
		maxSize: maxSize,
	}
}

// ShouldCapture determines if an event should be captured based on size filtering
func (f *SizeFilter) ShouldCapture(evt *event.Event) bool {
	if f.maxSize <= 0 {
		return true // No size limit
	}

	return evt.Size() <= f.maxSize
}

// GetFilterReason returns the reason for filtering
func (f *SizeFilter) GetFilterReason(evt *event.Event) string {
	if f.maxSize > 0 && evt.Size() > f.maxSize {
		return fmt.Sprintf("payload size %d exceeds limit %d", evt.Size(), f.maxSize)
	}
	return ""
}

// RiskFilter filters events based on risk level
type RiskFilter struct {
	minimumLevel risk.RiskLevel
	onlyHighRisk bool
}

// NewRiskFilter creates a new risk filter
func NewRiskFilter(minimumLevel risk.RiskLevel, onlyHighRisk bool) *RiskFilter {
	return &RiskFilter{
		minimumLevel: minimumLevel,
		onlyHighRisk: onlyHighRisk,
	}
}

// ShouldCapture determines if an event should be captured based on risk filtering
func (f *RiskFilter) ShouldCapture(evt *event.Event) bool {
	if f.onlyHighRisk {
		return evt.IsHighRisk()
	}

	// Check minimum risk level
	eventRiskLevel := evt.RiskScore().Level()
	return f.isRiskLevelSufficient(eventRiskLevel)
}

// GetFilterReason returns the reason for filtering
func (f *RiskFilter) GetFilterReason(evt *event.Event) string {
	if f.onlyHighRisk && !evt.IsHighRisk() {
		return "event is not high risk"
	}

	eventRiskLevel := evt.RiskScore().Level()
	if !f.isRiskLevelSufficient(eventRiskLevel) {
		return fmt.Sprintf("event risk level %s below minimum %s", eventRiskLevel, f.minimumLevel)
	}

	return ""
}

// isRiskLevelSufficient checks if the event risk level meets the minimum requirement
func (f *RiskFilter) isRiskLevelSufficient(level string) bool {
	switch f.minimumLevel {
	case risk.RiskLevelLow:
		return true // All levels are sufficient
	case risk.RiskLevelMedium:
		return level == "medium" || level == "high"
	case risk.RiskLevelHigh:
		return level == "high"
	default:
		return true
	}
}

// ContentFilter filters events based on content patterns
type ContentFilter struct {
	blacklist []string
	enabled   bool
}

// NewContentFilter creates a new content filter
func NewContentFilter(blacklist []string, enabled bool) *ContentFilter {
	return &ContentFilter{
		blacklist: blacklist,
		enabled:   enabled,
	}
}

// ShouldCapture determines if an event should be captured based on content filtering
func (f *ContentFilter) ShouldCapture(evt *event.Event) bool {
	if !f.enabled || len(f.blacklist) == 0 {
		return true
	}

	content := string(evt.Payload())
	contentLower := strings.ToLower(content)

	for _, pattern := range f.blacklist {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return false
		}
	}

	return true
}

// GetFilterReason returns the reason for filtering
func (f *ContentFilter) GetFilterReason(evt *event.Event) string {
	if !f.enabled {
		return ""
	}

	content := string(evt.Payload())
	contentLower := strings.ToLower(content)

	for _, pattern := range f.blacklist {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return fmt.Sprintf("content contains blacklisted pattern: %s", pattern)
		}
	}

	return ""
}

// DirectionFilter filters events based on direction (inbound/outbound)
type DirectionFilter struct {
	allowedDirections []event.Direction
}

// NewDirectionFilter creates a new direction filter
func NewDirectionFilter(allowedDirections []event.Direction) *DirectionFilter {
	return &DirectionFilter{
		allowedDirections: allowedDirections,
	}
}

// ShouldCapture determines if an event should be captured based on direction filtering
func (f *DirectionFilter) ShouldCapture(evt *event.Event) bool {
	if len(f.allowedDirections) == 0 {
		return true // No direction filtering
	}

	eventDirection := evt.Direction()
	for _, allowed := range f.allowedDirections {
		if eventDirection == allowed {
			return true
		}
	}

	return false
}

// GetFilterReason returns the reason for filtering
func (f *DirectionFilter) GetFilterReason(evt *event.Event) string {
	if len(f.allowedDirections) == 0 {
		return ""
	}

	eventDirection := evt.Direction()
	for _, allowed := range f.allowedDirections {
		if eventDirection == allowed {
			return ""
		}
	}

	return fmt.Sprintf("direction %s not in allowed directions", eventDirection)
}

// FilterChain represents a chain of filters that can be applied in sequence
type FilterChain struct {
	filters []EventFilter
}

// NewFilterChain creates a new filter chain
func NewFilterChain(filters ...EventFilter) *FilterChain {
	return &FilterChain{
		filters: filters,
	}
}

// ShouldCapture applies all filters in the chain
func (f *FilterChain) ShouldCapture(evt *event.Event) bool {
	for _, filter := range f.filters {
		if !filter.ShouldCapture(evt) {
			return false
		}
	}
	return true
}

// GetFilterReason returns the reason from the first filter that rejects the event
func (f *FilterChain) GetFilterReason(evt *event.Event) string {
	for _, filter := range f.filters {
		if !filter.ShouldCapture(evt) {
			return filter.GetFilterReason(evt)
		}
	}
	return "Event passed all filters"
}

// GetFilterStatistics aggregates statistics from all filters
func (f *FilterChain) GetFilterStatistics() FilterStatistics {
	// For chain, we return the statistics from the first filter that provides them
	for _, filter := range f.filters {
		if stats := filter.GetFilterStatistics(); stats.TotalEvaluated > 0 {
			return stats
		}
	}
	return FilterStatistics{}
}

// AddFilter adds a filter to the chain
func (f *FilterChain) AddFilter(filter EventFilter) {
	f.filters = append(f.filters, filter)
}

// RemoveFilter removes a filter from the chain by index
func (f *FilterChain) RemoveFilter(index int) error {
	if index < 0 || index >= len(f.filters) {
		return fmt.Errorf("filter index %d out of range", index)
	}

	f.filters = append(f.filters[:index], f.filters[index+1:]...)
	return nil
}
