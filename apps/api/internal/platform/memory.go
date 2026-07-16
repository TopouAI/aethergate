package platform

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                             sync.RWMutex
	organizations                  []Organization
	apiKeys                        []APIKey
	workspaces                     []Workspace
	projects                       []Project
	members                        []Member
	models                         []Model
	providers                      []ProviderConnection
	routingPolicies                []RoutingPolicy
	rateLimitRules                 []RateLimitRule
	budgets                        []Budget
	alertRules                     []AlertRule
	alertIncidents                 []AlertIncident
	webhooks                       []WebhookEndpoint
	webhookDeliveries              []WebhookDelivery
	providerHealthEvents           []ProviderHealthEvent
	providerHealthProbes           []ProviderHealthProbe
	reports                        []ReportSchedule
	reportRuns                     []ReportRun
	notifications                  []Notification
	notificationPreferences        []NotificationPreference
	notificationEscalationPolicies []NotificationEscalationPolicy
	notificationDeliveries         []NotificationDelivery
	auditEvents                    []AuditEvent
	auditRetentionPolicies         []AuditRetentionPolicy
	auditExports                   []AuditExport
	vaultSecrets                   []VaultSecret
	vaultSecretVersions            []VaultSecretVersion
	vaultAccessEvents              []VaultAccessEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		organizations:                  developmentOrganizations(),
		apiKeys:                        developmentAPIKeys(),
		workspaces:                     developmentWorkspaces(),
		projects:                       developmentProjects(),
		members:                        developmentMembers(),
		models:                         developmentModels(),
		providers:                      developmentProviders(),
		routingPolicies:                developmentRoutingPolicies(),
		rateLimitRules:                 developmentRateLimitRules(),
		budgets:                        developmentBudgets(),
		alertRules:                     developmentAlertRules(),
		alertIncidents:                 developmentAlertIncidents(),
		webhooks:                       developmentWebhooks(),
		webhookDeliveries:              developmentWebhookDeliveries(),
		providerHealthEvents:           developmentProviderHealthEvents(),
		providerHealthProbes:           developmentProviderHealthProbes(),
		reports:                        developmentReports(),
		reportRuns:                     developmentReportRuns(),
		notifications:                  developmentNotifications(),
		notificationPreferences:        developmentNotificationPreferences(),
		notificationEscalationPolicies: developmentNotificationEscalationPolicies(),
		notificationDeliveries:         developmentNotificationDeliveries(),
		auditEvents:                    developmentAuditEvents(),
		auditRetentionPolicies:         developmentAuditRetentionPolicies(),
		auditExports:                   developmentAuditExports(),
		vaultSecrets:                   developmentVaultSecrets(),
		vaultSecretVersions:            developmentVaultSecretVersions(),
		vaultAccessEvents:              developmentVaultAccessEvents(),
	}
}

func (r *MemoryRepository) ListOrganizations(_ context.Context, filter OrganizationFilter) ([]Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]Organization, 0, len(r.organizations))
	for _, organization := range r.organizations {
		if filter.Status != "" && filter.Status != "all" && organization.Status != filter.Status {
			continue
		}
		fields := []string{organization.Name, organization.Slug, organization.Owner, organization.Region}
		if query != "" && !slices.ContainsFunc(fields, func(value string) bool { return strings.Contains(strings.ToLower(value), query) }) {
			continue
		}
		items = append(items, organization)
	}
	return items, nil
}

func (r *MemoryRepository) GetOrganization(_ context.Context, id string) (Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, organization := range r.organizations {
		if organization.ID == id {
			return organization, nil
		}
	}
	return Organization{}, ErrNotFound
}

func (r *MemoryRepository) CreateOrganization(_ context.Context, organization Organization) (Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if slices.ContainsFunc(r.organizations, func(existing Organization) bool { return strings.EqualFold(existing.Slug, organization.Slug) }) {
		return Organization{}, ErrConflict
	}
	r.organizations = append([]Organization{organization}, r.organizations...)
	return organization, nil
}

func (r *MemoryRepository) ListAPIKeys(_ context.Context, filter APIKeyFilter) ([]APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]APIKey, 0, len(r.apiKeys))
	for _, key := range r.apiKeys {
		if filter.OrganizationID != "" && key.Organization != filter.OrganizationID {
			continue
		}
		if filter.Status != "" && filter.Status != "all" && key.Status != filter.Status {
			continue
		}
		fields := append([]string{key.Name, key.Prefix, key.Project, key.CreatedBy}, key.Models...)
		if query != "" && !slices.ContainsFunc(fields, func(value string) bool { return strings.Contains(strings.ToLower(value), query) }) {
			continue
		}
		items = append(items, key)
	}
	return items, nil
}

func (r *MemoryRepository) CreateAPIKey(_ context.Context, key APIKey) (APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, key.Organization); !found {
		return APIKey{}, ErrNotFound
	}
	r.apiKeys = append([]APIKey{key}, r.apiKeys...)
	return key, nil
}

func (r *MemoryRepository) RevokeAPIKey(_ context.Context, id, _ string, _ time.Time) (APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.apiKeys {
		if r.apiKeys[index].ID != id {
			continue
		}
		if r.apiKeys[index].Status != "active" {
			return APIKey{}, ErrInactive
		}
		r.apiKeys[index].Status = "revoked"
		return r.apiKeys[index], nil
	}
	return APIKey{}, ErrNotFound
}

func findOrganization(items []Organization, id string) (Organization, bool) {
	for _, organization := range items {
		if organization.ID == id {
			return organization, true
		}
	}
	return Organization{}, false
}

func developmentOrganizations() []Organization {
	return []Organization{
		{ID: "org_topoai", Name: "TopoAI", Slug: "topoai", Status: "active", Plan: "Enterprise", Region: "China East", Workspaces: 4, Projects: 18, Members: 46, MonthlyCostUSD: 6842, BudgetUSD: 12000, Requests: 482340, Owner: "holden@topoai.dev", CreatedAt: time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC)},
		{ID: "org_acme", Name: "Acme Manufacturing", Slug: "acme-manufacturing", Status: "active", Plan: "Enterprise", Region: "Singapore", Workspaces: 6, Projects: 24, Members: 91, MonthlyCostUSD: 8931, BudgetUSD: 15000, Requests: 591208, Owner: "platform@acme.cn", CreatedAt: time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)},
	}
}

func developmentAPIKeys() []APIKey {
	return []APIKey{
		{ID: "key_01JY8KX2F3", Organization: "org_topoai", Name: "Engineering Copilot · Production", Prefix: "ag_live_7Tx9", Project: "Engineering Copilot", Status: "active", Models: []string{"claude-sonnet-4", "gpt-5-mini"}, RPM: 600, TPM: 1_200_000, SpendUSD: 2148.42, CreatedBy: "li.ming@topoai.dev", CreatedAt: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)},
		{ID: "key_01JY8JZ4DA", Organization: "org_topoai", Name: "Legacy Migration", Prefix: "ag_live_1Lp7", Project: "Code Modernization", Status: "revoked", Models: []string{"deepseek-v3"}, RPM: 300, TPM: 900_000, SpendUSD: 631.88, CreatedBy: "wang.lei@topoai.dev", CreatedAt: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)},
	}
}
