package platform

import (
	"context"
	"slices"
	"time"
)

func (r *MemoryRepository) GetProvider(_ context.Context, organizationID, id string) (ProviderConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, provider := range r.providers {
		if provider.OrganizationID == organizationID && provider.ID == id {
			return provider, nil
		}
	}
	return ProviderConnection{}, ErrNotFound
}

func (r *MemoryRepository) ListProviderHealthEvents(_ context.Context, filter ProviderHealthFilter) ([]ProviderHealthEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ProviderHealthEvent, 0)
	for _, event := range r.providerHealthEvents {
		if event.OrganizationID != filter.OrganizationID || filter.ProviderID != "" && event.ProviderID != filter.ProviderID || filter.Status != "" && filter.Status != "all" && event.Status != filter.Status || filter.Source != "" && filter.Source != "all" && event.Source != filter.Source {
			continue
		}
		items = append(items, event)
	}
	return items, nil
}

func (r *MemoryRepository) ListProviderHealthProbes(_ context.Context, filter ProviderHealthProbeFilter) ([]ProviderHealthProbe, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ProviderHealthProbe, 0)
	for _, probe := range r.providerHealthProbes {
		if probe.OrganizationID != filter.OrganizationID || filter.ProviderID != "" && probe.ProviderID != filter.ProviderID || filter.Status != "" && filter.Status != "all" && probe.Status != filter.Status {
			continue
		}
		items = append(items, probe)
	}
	return items, nil
}

func (r *MemoryRepository) CreateProviderHealthProbe(_ context.Context, probe ProviderHealthProbe) (ProviderHealthProbe, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !slices.ContainsFunc(r.providers, func(provider ProviderConnection) bool {
		return provider.OrganizationID == probe.OrganizationID && provider.ID == probe.ProviderID
	}) {
		return ProviderHealthProbe{}, ErrNotFound
	}
	r.providerHealthProbes = append([]ProviderHealthProbe{probe}, r.providerHealthProbes...)
	return probe, nil
}

func (r *MemoryRepository) RecordProviderHealth(_ context.Context, provider ProviderConnection, event ProviderHealthEvent) (ProviderHealthEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for index := range r.providers {
		if r.providers[index].OrganizationID == provider.OrganizationID && r.providers[index].ID == provider.ID {
			r.providers[index] = provider
			found = true
			break
		}
	}
	if !found {
		return ProviderHealthEvent{}, ErrNotFound
	}
	if event.ProbeID != nil {
		for index := range r.providerHealthProbes {
			if r.providerHealthProbes[index].OrganizationID == provider.OrganizationID && r.providerHealthProbes[index].ID == *event.ProbeID {
				r.providerHealthProbes[index].Status = "succeeded"
				if !event.Success {
					r.providerHealthProbes[index].Status = "failed"
					r.providerHealthProbes[index].ErrorMessage = event.Reason
				}
				r.providerHealthProbes[index].CompletedAt = &event.ObservedAt
				r.providerHealthProbes[index].EventID = &event.ID
				break
			}
		}
	}
	r.providerHealthEvents = append([]ProviderHealthEvent{event}, r.providerHealthEvents...)
	return event, nil
}

func (r *MemoryRepository) UpdateProviderMaintenance(_ context.Context, provider ProviderConnection) (ProviderConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.providers {
		if r.providers[index].OrganizationID == provider.OrganizationID && r.providers[index].ID == provider.ID {
			r.providers[index] = provider
			return provider, nil
		}
	}
	return ProviderConnection{}, ErrNotFound
}

func developmentProviderHealthEvents() []ProviderHealthEvent {
	return []ProviderHealthEvent{
		{ID: "phe_openai_01", OrganizationID: "org_topoai", ProviderID: "provider_openai_primary", ProviderName: "OpenAI Primary", Source: "passive_telemetry", PreviousStatus: "healthy", Status: "healthy", Success: true, RoutingEligible: true, RequestCount: 184220, ErrorCount: 111, ErrorRate: 0.0603, AverageLatencyMS: 842, P95LatencyMS: 1240, Reason: "Passive telemetry is within routing-safe thresholds.", ObservedAt: time.Date(2026, 7, 14, 6, 0, 0, 0, time.UTC)},
		{ID: "phe_deepseek_01", OrganizationID: "org_topoai", ProviderID: "provider_deepseek_apac", ProviderName: "DeepSeek APAC", Source: "passive_telemetry", PreviousStatus: "healthy", Status: "degraded", Transition: true, Success: false, RequestCount: 48120, ErrorCount: 1049, ErrorRate: 2.18, AverageLatencyMS: 1480, P95LatencyMS: 2160, Reason: "Passive telemetry exceeded the degraded error-rate or latency threshold.", ObservedAt: time.Date(2026, 7, 14, 5, 55, 0, 0, time.UTC)},
	}
}

func developmentProviderHealthProbes() []ProviderHealthProbe {
	completed := time.Date(2026, 7, 14, 6, 0, 0, 0, time.UTC)
	eventID := "phe_openai_01"
	return []ProviderHealthProbe{
		{ID: "probe_openai_01", OrganizationID: "org_topoai", ProviderID: "provider_openai_primary", ProviderName: "OpenAI Primary", Status: "succeeded", Region: "apac", Model: "gpt-5-mini", RequestedBy: "system", RequestedAt: time.Date(2026, 7, 14, 5, 59, 55, 0, time.UTC), CompletedAt: &completed, EventID: &eventID},
	}
}

var _ ProviderHealthRepository = (*MemoryRepository)(nil)
