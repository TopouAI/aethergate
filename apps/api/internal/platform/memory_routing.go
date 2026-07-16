package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListRoutingPolicies(_ context.Context, filter RoutingPolicyFilter) ([]RoutingPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]RoutingPolicy, 0)
	for _, policy := range r.routingPolicies {
		if policy.OrganizationID != filter.OrganizationID || filter.Status != "" && filter.Status != "all" && policy.Status != filter.Status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(policy.Name+" "+policy.Slug+" "+policy.ModelPattern+" "+policy.Strategy), query) {
			continue
		}
		policy.Targets = slices.Clone(policy.Targets)
		items = append(items, policy)
	}
	return items, nil
}

func (r *MemoryRepository) GetRoutingPolicy(_ context.Context, organizationID, id string) (RoutingPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.routingPolicies {
		if policy.OrganizationID == organizationID && policy.ID == id {
			policy.Targets = slices.Clone(policy.Targets)
			return policy, nil
		}
	}
	return RoutingPolicy{}, ErrNotFound
}

func (r *MemoryRepository) CreateRoutingPolicy(_ context.Context, policy RoutingPolicy) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, policy.OrganizationID); !found {
		return RoutingPolicy{}, ErrNotFound
	}
	if slices.ContainsFunc(r.routingPolicies, func(existing RoutingPolicy) bool {
		return existing.OrganizationID == policy.OrganizationID && strings.EqualFold(existing.Slug, policy.Slug)
	}) {
		return RoutingPolicy{}, ErrConflict
	}
	policy.Targets = slices.Clone(policy.Targets)
	r.routingPolicies = append([]RoutingPolicy{policy}, r.routingPolicies...)
	return policy, nil
}

func (r *MemoryRepository) UpdateRoutingPolicyStatus(_ context.Context, organizationID, id, status string, updatedAt time.Time) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.routingPolicies {
		if r.routingPolicies[index].OrganizationID == organizationID && r.routingPolicies[index].ID == id {
			r.routingPolicies[index].Status = status
			r.routingPolicies[index].UpdatedAt = updatedAt
			return r.routingPolicies[index], nil
		}
	}
	return RoutingPolicy{}, ErrNotFound
}

func developmentRoutingPolicies() []RoutingPolicy {
	created := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return []RoutingPolicy{
		{ID: "route_balanced_copilot", OrganizationID: "org_topoai", Name: "Copilot balanced pool", Slug: "copilot-balanced", Status: "active", Strategy: "weighted", ModelPattern: "copilot/*", MaxRetries: 2, RequestTimeoutMS: 45000, Targets: []RoutingTarget{{ID: "target_openai", ProviderID: "provider_openai_primary", ProviderName: "OpenAI Primary", Model: "gpt-5-mini", Priority: 1, Weight: 60, Enabled: true}, {ID: "target_anthropic", ProviderID: "provider_anthropic_primary", ProviderName: "Anthropic Primary", Model: "claude-sonnet-4", Priority: 2, Weight: 40, Enabled: true}}, CreatedAt: created, UpdatedAt: created},
		{ID: "route_research_fallback", OrganizationID: "org_topoai", Name: "Research priority fallback", Slug: "research-priority", Status: "draft", Strategy: "priority", ModelPattern: "research/*", MaxRetries: 3, RequestTimeoutMS: 90000, Targets: []RoutingTarget{{ID: "target_research_anthropic", ProviderID: "provider_anthropic_primary", ProviderName: "Anthropic Primary", Model: "claude-sonnet-4", Priority: 1, Weight: 100, Enabled: true}, {ID: "target_research_deepseek", ProviderID: "provider_deepseek_apac", ProviderName: "DeepSeek APAC", Model: "deepseek-v3", Priority: 2, Weight: 0, Enabled: true}}, CreatedAt: created, UpdatedAt: created},
	}
}
