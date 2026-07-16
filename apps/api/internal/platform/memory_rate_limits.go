package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListRateLimitRules(_ context.Context, filter RateLimitFilter) ([]RateLimitRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]RateLimitRule, 0)
	for _, rule := range r.rateLimitRules {
		if rule.OrganizationID != filter.OrganizationID || filter.Status != "" && filter.Status != "all" && rule.Status != filter.Status || filter.ScopeType != "" && filter.ScopeType != "all" && rule.ScopeType != filter.ScopeType {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(rule.Name+" "+rule.ScopeID+" "+rule.Metric+" "+rule.Action), query) {
			continue
		}
		items = append(items, rule)
	}
	return items, nil
}

func (r *MemoryRepository) GetRateLimitRule(_ context.Context, organizationID, id string) (RateLimitRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rule := range r.rateLimitRules {
		if rule.OrganizationID == organizationID && rule.ID == id {
			return rule, nil
		}
	}
	return RateLimitRule{}, ErrNotFound
}

func (r *MemoryRepository) CreateRateLimitRule(_ context.Context, rule RateLimitRule) (RateLimitRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, rule.OrganizationID); !found {
		return RateLimitRule{}, ErrNotFound
	}
	if slices.ContainsFunc(r.rateLimitRules, func(existing RateLimitRule) bool {
		return existing.OrganizationID == rule.OrganizationID && strings.EqualFold(existing.Name, rule.Name)
	}) {
		return RateLimitRule{}, ErrConflict
	}
	r.rateLimitRules = append([]RateLimitRule{rule}, r.rateLimitRules...)
	return rule, nil
}

func (r *MemoryRepository) UpdateRateLimitRuleStatus(_ context.Context, organizationID, id, status string, updatedAt time.Time) (RateLimitRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.rateLimitRules {
		if r.rateLimitRules[index].OrganizationID == organizationID && r.rateLimitRules[index].ID == id {
			r.rateLimitRules[index].Status = status
			r.rateLimitRules[index].UpdatedAt = updatedAt
			return r.rateLimitRules[index], nil
		}
	}
	return RateLimitRule{}, ErrNotFound
}

func developmentRateLimitRules() []RateLimitRule {
	created := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	return []RateLimitRule{
		{ID: "limit_org_tokens", OrganizationID: "org_topoai", Name: "Organization token ceiling", Status: "enforced", ScopeType: "organization", ScopeID: "org_topoai", Metric: "tokens", Window: "minute", Limit: 2000000, Burst: 200000, Action: "reject", Priority: 100, MatchedRequests: 482340, LimitedRequests: 184, CreatedAt: created, UpdatedAt: created},
		{ID: "limit_engineering_requests", OrganizationID: "org_topoai", Name: "Engineering request budget", Status: "enforced", ScopeType: "workspace", ScopeID: "ws_engineering", Metric: "requests", Window: "minute", Limit: 1200, Burst: 120, Action: "throttle", Priority: 200, MatchedRequests: 251571, LimitedRequests: 39, CreatedAt: created, UpdatedAt: created},
		{ID: "limit_copilot_observe", OrganizationID: "org_topoai", Name: "Copilot concurrency preview", Status: "draft", ScopeType: "project", ScopeID: "project_engineering_copilot", Metric: "concurrency", Window: "second", Limit: 80, Burst: 10, Action: "observe", Priority: 300, CreatedAt: created, UpdatedAt: created},
	}
}
