package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListProviders(_ context.Context, filter ProviderFilter) ([]ProviderConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]ProviderConnection, 0)
	for _, provider := range r.providers {
		if provider.OrganizationID != filter.OrganizationID || filter.Status != "" && filter.Status != "all" && provider.Status != filter.Status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(provider.Name+" "+provider.Provider+" "+provider.BaseURL), query) {
			continue
		}
		items = append(items, provider)
	}
	return items, nil
}

func (r *MemoryRepository) CreateProvider(_ context.Context, provider ProviderConnection) (ProviderConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, provider.OrganizationID); !found {
		return ProviderConnection{}, ErrNotFound
	}
	if slices.ContainsFunc(r.providers, func(existing ProviderConnection) bool {
		return existing.OrganizationID == provider.OrganizationID && strings.EqualFold(existing.Name, provider.Name)
	}) {
		return ProviderConnection{}, ErrConflict
	}
	r.providers = append([]ProviderConnection{provider}, r.providers...)
	return provider, nil
}

func developmentProviders() []ProviderConnection {
	now := time.Date(2026, 7, 14, 6, 0, 0, 0, time.UTC)
	return []ProviderConnection{
		{ID: "provider_openai_primary", OrganizationID: "org_topoai", Name: "OpenAI Primary", Provider: "OpenAI", BaseURL: "https://api.openai.com/v1", Status: "healthy", CredentialState: "configured", Models: 8, P95LatencyMS: 1240, SuccessRate: 99.94, LastCheckedAt: &now, RoutingEligible: true, HealthSource: "passive_telemetry", HealthReason: "Passive telemetry is within routing-safe thresholds.", ErrorRate: 0.06, RequestCount24H: 184220, AverageLatencyMS: 842, LastTransitionAt: &now, CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "provider_anthropic_primary", OrganizationID: "org_topoai", Name: "Anthropic Primary", Provider: "Anthropic", BaseURL: "https://api.anthropic.com", Status: "healthy", CredentialState: "configured", Models: 6, P95LatencyMS: 1380, SuccessRate: 99.91, LastCheckedAt: &now, RoutingEligible: true, HealthSource: "passive_telemetry", HealthReason: "Passive telemetry is within routing-safe thresholds.", ErrorRate: 0.09, RequestCount24H: 153880, AverageLatencyMS: 920, LastTransitionAt: &now, CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "provider_deepseek_apac", OrganizationID: "org_topoai", Name: "DeepSeek APAC", Provider: "DeepSeek", BaseURL: "https://api.deepseek.com", Status: "degraded", CredentialState: "rotating", Models: 3, P95LatencyMS: 2160, SuccessRate: 97.82, LastCheckedAt: &now, RoutingEligible: false, HealthSource: "passive_telemetry", HealthReason: "Passive telemetry exceeded the degraded error-rate threshold.", ErrorRate: 2.18, RequestCount24H: 48120, AverageLatencyMS: 1480, LastTransitionAt: &now, CreatedAt: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)},
	}
}
