package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListAuditEvents(_ context.Context, filter AuditFilter) ([]AuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.ToLower(filter.Query)
	items := make([]AuditEvent, 0)
	for _, event := range r.auditEvents {
		if event.OrganizationID != filter.OrganizationID ||
			filter.Actor != "" && !strings.Contains(strings.ToLower(event.ActorEmail+" "+event.ActorID), strings.ToLower(filter.Actor)) ||
			filter.Action != "" && filter.Action != "all" && event.Action != filter.Action ||
			filter.ResourceType != "" && filter.ResourceType != "all" && event.ResourceType != filter.ResourceType ||
			filter.ResourceID != "" && event.ResourceID != filter.ResourceID ||
			filter.Outcome != "" && filter.Outcome != "all" && event.Outcome != filter.Outcome ||
			filter.RiskLevel != "" && filter.RiskLevel != "all" && event.RiskLevel != filter.RiskLevel ||
			filter.StartAt != nil && event.CreatedAt.Before(*filter.StartAt) || filter.EndAt != nil && event.CreatedAt.After(*filter.EndAt) {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(event.ActorEmail+" "+event.Action+" "+event.ResourceType+" "+event.ResourceID+" "+event.Reason+" "+event.RequestID), needle) {
			continue
		}
		items = append(items, cloneAuditEvent(event))
	}
	return items, nil
}

func (r *MemoryRepository) LatestAuditHash(_ context.Context, organizationID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, event := range r.auditEvents {
		if event.OrganizationID == organizationID {
			return event.IntegrityHash, nil
		}
	}
	return "", ErrNotFound
}

func (r *MemoryRepository) AppendAuditEvent(_ context.Context, event AuditEvent) (AuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, event.OrganizationID); !found {
		return AuditEvent{}, ErrNotFound
	}
	latest := ""
	for _, existing := range r.auditEvents {
		if existing.OrganizationID == event.OrganizationID {
			latest = existing.IntegrityHash
			break
		}
	}
	if event.PreviousHash != latest {
		return AuditEvent{}, ErrConflict
	}
	r.auditEvents = append([]AuditEvent{cloneAuditEvent(event)}, r.auditEvents...)
	return cloneAuditEvent(event), nil
}

func (r *MemoryRepository) GetAuditRetentionPolicy(_ context.Context, organizationID string) (AuditRetentionPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.auditRetentionPolicies {
		if policy.OrganizationID == organizationID {
			return policy, nil
		}
	}
	return AuditRetentionPolicy{}, ErrNotFound
}

func (r *MemoryRepository) UpsertAuditRetentionPolicy(_ context.Context, policy AuditRetentionPolicy) (AuditRetentionPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, policy.OrganizationID); !found {
		return AuditRetentionPolicy{}, ErrNotFound
	}
	for index := range r.auditRetentionPolicies {
		if r.auditRetentionPolicies[index].OrganizationID == policy.OrganizationID {
			r.auditRetentionPolicies[index] = policy
			return policy, nil
		}
	}
	r.auditRetentionPolicies = append([]AuditRetentionPolicy{policy}, r.auditRetentionPolicies...)
	return policy, nil
}

func (r *MemoryRepository) ListAuditExports(_ context.Context, filter AuditExportFilter) ([]AuditExport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]AuditExport, 0)
	for _, export := range r.auditExports {
		if export.OrganizationID != filter.OrganizationID ||
			filter.Status != "" && filter.Status != "all" && export.Status != filter.Status ||
			filter.Format != "" && filter.Format != "all" && export.Format != filter.Format {
			continue
		}
		items = append(items, cloneAuditExport(export))
	}
	return items, nil
}

func (r *MemoryRepository) GetAuditExport(_ context.Context, organizationID, id string) (AuditExport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, export := range r.auditExports {
		if export.OrganizationID == organizationID && export.ID == id {
			return cloneAuditExport(export), nil
		}
	}
	return AuditExport{}, ErrNotFound
}

func (r *MemoryRepository) CreateAuditExport(_ context.Context, export AuditExport) (AuditExport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, export.OrganizationID); !found {
		return AuditExport{}, ErrNotFound
	}
	if slices.ContainsFunc(r.auditExports, func(existing AuditExport) bool { return existing.ID == export.ID }) {
		return AuditExport{}, ErrConflict
	}
	r.auditExports = append([]AuditExport{cloneAuditExport(export)}, r.auditExports...)
	return cloneAuditExport(export), nil
}

func developmentAuditEvents() []AuditEvent {
	items := []AuditEvent{
		{ID: "audit_org_created", OrganizationID: "org_topoai", ActorID: "user_holden", ActorEmail: "holden@topoai.dev", Action: "organization.created", ResourceType: "organization", ResourceID: "org_topoai", Outcome: "success", RiskLevel: "medium", Source: "control-plane", Reason: "Enterprise tenant bootstrap", RequestID: "req_audit_001", IPAddress: "10.12.0.8", UserAgent: "AetherGate/1.0", BeforeState: map[string]any{}, AfterState: map[string]any{"slug": "topoai", "plan": "Enterprise"}, CreatedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC)},
		{ID: "audit_key_created", OrganizationID: "org_topoai", ActorID: "user_holden", ActorEmail: "holden@topoai.dev", Action: "api_key.created", ResourceType: "api_key", ResourceID: "key_01JY8KX2F3", Outcome: "success", RiskLevel: "high", Source: "control-plane", Reason: "Production application credential", RequestID: "req_audit_002", IPAddress: "10.12.0.8", UserAgent: "Mozilla/5.0", BeforeState: map[string]any{}, AfterState: map[string]any{"project": "Engineering Copilot", "models": []any{"claude-sonnet-4", "gpt-5-mini"}}, CreatedAt: time.Date(2026, 4, 12, 2, 15, 0, 0, time.UTC)},
		{ID: "audit_role_granted", OrganizationID: "org_topoai", ActorID: "user_holden", ActorEmail: "holden@topoai.dev", Action: "member.role_granted", ResourceType: "member", ResourceID: "member_li_ming", Outcome: "success", RiskLevel: "critical", Source: "control-plane", Reason: "Platform operations ownership", RequestID: "req_audit_003", IPAddress: "10.12.0.8", UserAgent: "Mozilla/5.0", BeforeState: map[string]any{"role": "developer"}, AfterState: map[string]any{"role": "admin"}, CreatedAt: time.Date(2026, 7, 14, 4, 30, 0, 0, time.UTC)},
	}
	previous := ""
	for index := range items {
		items[index].PreviousHash = previous
		items[index].IntegrityHash, _ = calculateAuditHash(items[index])
		previous = items[index].IntegrityHash
	}
	slices.Reverse(items)
	return items
}

func developmentAuditRetentionPolicies() []AuditRetentionPolicy {
	return []AuditRetentionPolicy{{OrganizationID: "org_topoai", RetentionDays: 365, ExportFormat: "csv", UpdatedBy: "holden@topoai.dev", UpdatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)}}
}

func developmentAuditExports() []AuditExport {
	completed := time.Date(2026, 7, 1, 0, 0, 15, 0, time.UTC)
	return []AuditExport{
		{ID: "aexp_q2_success", OrganizationID: "org_topoai", RequestedBy: "holden@topoai.dev", Format: "csv", Status: "succeeded", Filters: map[string]string{"riskLevel": "high,critical"}, PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), RowCount: 1824, SizeBytes: 642810, ObjectKey: "audit/org_topoai/2026-q2.csv", Checksum: strings.Repeat("a", 64), CreatedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), CompletedAt: &completed},
		{ID: "aexp_failed", OrganizationID: "org_topoai", RequestedBy: "security@topoai.dev", Format: "jsonl", Status: "failed", Filters: map[string]string{}, PeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), ErrorMessage: "Object storage write timed out.", CreatedAt: time.Date(2026, 7, 1, 0, 5, 0, 0, time.UTC)},
	}
}

func cloneAuditEvent(event AuditEvent) AuditEvent {
	event.BeforeState = cloneAnyMap(event.BeforeState)
	event.AfterState = cloneAnyMap(event.AfterState)
	return event
}

func cloneAuditExport(export AuditExport) AuditExport {
	export.Filters = mapsCloneString(export.Filters)
	return export
}

var _ AuditRepository = (*MemoryRepository)(nil)
