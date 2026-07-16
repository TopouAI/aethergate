package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListWebhookEndpoints(_ context.Context, filter WebhookFilter) ([]WebhookEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]WebhookEndpoint, 0)
	for _, endpoint := range r.webhooks {
		if endpoint.OrganizationID != filter.OrganizationID || filter.Status != "" && filter.Status != "all" && endpoint.Status != filter.Status {
			continue
		}
		if filter.Event != "" && filter.Event != "all" && !slices.Contains(endpoint.Events, filter.Event) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(endpoint.Name+" "+endpoint.Destination+" "+strings.Join(endpoint.Events, " ")), query) {
			continue
		}
		items = append(items, cloneWebhookEndpoint(endpoint))
	}
	return items, nil
}

func (r *MemoryRepository) GetWebhookEndpoint(_ context.Context, organizationID, id string) (WebhookEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, endpoint := range r.webhooks {
		if endpoint.OrganizationID == organizationID && endpoint.ID == id {
			return cloneWebhookEndpoint(endpoint), nil
		}
	}
	return WebhookEndpoint{}, ErrNotFound
}

func (r *MemoryRepository) CreateWebhookEndpoint(_ context.Context, endpoint WebhookEndpoint) (WebhookEndpoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, endpoint.OrganizationID); !found {
		return WebhookEndpoint{}, ErrNotFound
	}
	if slices.ContainsFunc(r.webhooks, func(existing WebhookEndpoint) bool {
		return existing.OrganizationID == endpoint.OrganizationID && strings.EqualFold(existing.Name, endpoint.Name)
	}) {
		return WebhookEndpoint{}, ErrConflict
	}
	r.webhooks = append([]WebhookEndpoint{cloneWebhookEndpoint(endpoint)}, r.webhooks...)
	return cloneWebhookEndpoint(endpoint), nil
}

func (r *MemoryRepository) UpdateWebhookEndpointStatus(_ context.Context, organizationID, id, status string, updatedAt time.Time) (WebhookEndpoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.webhooks {
		if r.webhooks[index].OrganizationID == organizationID && r.webhooks[index].ID == id {
			r.webhooks[index].Status = status
			r.webhooks[index].UpdatedAt = updatedAt
			return cloneWebhookEndpoint(r.webhooks[index]), nil
		}
	}
	return WebhookEndpoint{}, ErrNotFound
}

func (r *MemoryRepository) ListWebhookDeliveries(_ context.Context, filter WebhookDeliveryFilter) ([]WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]WebhookDelivery, 0)
	for _, delivery := range r.webhookDeliveries {
		if delivery.OrganizationID != filter.OrganizationID || filter.WebhookID != "" && delivery.WebhookID != filter.WebhookID || filter.Status != "" && filter.Status != "all" && delivery.Status != filter.Status || filter.EventType != "" && filter.EventType != "all" && delivery.EventType != filter.EventType {
			continue
		}
		items = append(items, delivery)
	}
	return items, nil
}

func (r *MemoryRepository) GetWebhookDelivery(_ context.Context, organizationID, id string) (WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, delivery := range r.webhookDeliveries {
		if delivery.OrganizationID == organizationID && delivery.ID == id {
			return delivery, nil
		}
	}
	return WebhookDelivery{}, ErrNotFound
}

func (r *MemoryRepository) CreateWebhookDelivery(_ context.Context, delivery WebhookDelivery) (WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !slices.ContainsFunc(r.webhooks, func(endpoint WebhookEndpoint) bool {
		return endpoint.OrganizationID == delivery.OrganizationID && endpoint.ID == delivery.WebhookID
	}) {
		return WebhookDelivery{}, ErrNotFound
	}
	if slices.ContainsFunc(r.webhookDeliveries, func(existing WebhookDelivery) bool {
		return existing.WebhookID == delivery.WebhookID && existing.EventID == delivery.EventID && existing.Attempt == delivery.Attempt
	}) {
		return WebhookDelivery{}, ErrConflict
	}
	r.webhookDeliveries = append([]WebhookDelivery{delivery}, r.webhookDeliveries...)
	return delivery, nil
}

func cloneWebhookEndpoint(endpoint WebhookEndpoint) WebhookEndpoint {
	endpoint.Events = slices.Clone(endpoint.Events)
	endpoint.PropertyFilters = slices.Clone(endpoint.PropertyFilters)
	return endpoint
}

func developmentWebhooks() []WebhookEndpoint {
	created := time.Date(2026, 5, 14, 2, 0, 0, 0, time.UTC)
	delivered := time.Date(2026, 7, 14, 6, 42, 0, 0, time.UTC)
	return []WebhookEndpoint{
		{ID: "wh_prod_events", OrganizationID: "org_topoai", Name: "Production event bus", Status: "active", Destination: "https://events.topoai.dev/aethergate", Version: "2026-07-15", Events: []string{"request.completed", "request.failed", "alert.triggered"}, SampleRate: 100, IncludeData: true, PropertyFilters: []WebhookPropertyFilter{{Key: "environment", Value: "production"}}, SigningSecretPrefix: "whsec_xD9m2Q", MaxAttempts: 5, TimeoutSeconds: 10, SuccessCount: 18342, FailureCount: 17, LastDeliveredAt: &delivered, CreatedAt: created, UpdatedAt: created},
		{ID: "wh_finops", OrganizationID: "org_topoai", Name: "FinOps automation", Status: "disabled", Destination: "https://finance.topoai.dev/hooks/usage", Version: "2026-07-15", Events: []string{"budget.threshold_reached"}, SampleRate: 100, IncludeData: false, SigningSecretPrefix: "whsec_7Kp2aN", MaxAttempts: 8, TimeoutSeconds: 15, SuccessCount: 94, FailureCount: 3, CreatedAt: created, UpdatedAt: created},
	}
}

func developmentWebhookDeliveries() []WebhookDelivery {
	delivered := time.Date(2026, 7, 14, 6, 42, 1, 0, time.UTC)
	retry := time.Date(2026, 7, 14, 6, 51, 0, 0, time.UTC)
	ok := 202
	unavailable := 503
	return []WebhookDelivery{
		{ID: "whd_success_01", OrganizationID: "org_topoai", WebhookID: "wh_prod_events", WebhookName: "Production event bus", EventID: "evt_req_01", EventType: "request.completed", Status: "succeeded", Trigger: "event", Attempt: 1, MaxAttempts: 5, ResponseStatus: &ok, DurationMS: 184, DeliveredAt: &delivered, CreatedAt: time.Date(2026, 7, 14, 6, 42, 0, 0, time.UTC)},
		{ID: "whd_failed_02", OrganizationID: "org_topoai", WebhookID: "wh_prod_events", WebhookName: "Production event bus", EventID: "evt_alert_07", EventType: "alert.triggered", Status: "failed", Trigger: "event", Attempt: 2, MaxAttempts: 5, ResponseStatus: &unavailable, DurationMS: 10012, ErrorMessage: "Destination returned HTTP 503.", NextRetryAt: &retry, CreatedAt: time.Date(2026, 7, 14, 6, 41, 0, 0, time.UTC)},
		{ID: "whd_dead_03", OrganizationID: "org_topoai", WebhookID: "wh_prod_events", WebhookName: "Production event bus", EventID: "evt_req_legacy", EventType: "request.failed", Status: "dead_letter", Trigger: "event", Attempt: 5, MaxAttempts: 5, DurationMS: 30000, ErrorMessage: "Delivery timed out after all configured attempts.", CreatedAt: time.Date(2026, 7, 13, 22, 20, 0, 0, time.UTC)},
	}
}

var _ WebhookRepository = (*MemoryRepository)(nil)
