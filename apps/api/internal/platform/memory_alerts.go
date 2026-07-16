package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListAlertRules(_ context.Context, f AlertFilter) ([]AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := strings.ToLower(f.Query)
	out := make([]AlertRule, 0)
	for _, a := range r.alertRules {
		if a.OrganizationID != f.OrganizationID || f.Status != "" && f.Status != "all" && a.Status != f.Status || f.Severity != "" && f.Severity != "all" && a.Severity != f.Severity {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(a.Name+" "+a.Metric+" "+a.Severity), q) {
			continue
		}
		a.Channels = slices.Clone(a.Channels)
		out = append(out, a)
	}
	return out, nil
}
func (r *MemoryRepository) GetAlertRule(_ context.Context, org, id string) (AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.alertRules {
		if a.OrganizationID == org && a.ID == id {
			return a, nil
		}
	}
	return AlertRule{}, ErrNotFound
}
func (r *MemoryRepository) CreateAlertRule(_ context.Context, a AlertRule) (AlertRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := findOrganization(r.organizations, a.OrganizationID); !ok {
		return AlertRule{}, ErrNotFound
	}
	if slices.ContainsFunc(r.alertRules, func(x AlertRule) bool {
		return x.OrganizationID == a.OrganizationID && strings.EqualFold(x.Name, a.Name)
	}) {
		return AlertRule{}, ErrConflict
	}
	r.alertRules = append([]AlertRule{a}, r.alertRules...)
	return a, nil
}
func (r *MemoryRepository) UpdateAlertRuleStatus(_ context.Context, org, id, status string, updated time.Time) (AlertRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.alertRules {
		if r.alertRules[i].OrganizationID == org && r.alertRules[i].ID == id {
			r.alertRules[i].Status = status
			r.alertRules[i].UpdatedAt = updated
			return r.alertRules[i], nil
		}
	}
	return AlertRule{}, ErrNotFound
}
func (r *MemoryRepository) ListAlertIncidents(_ context.Context, f AlertFilter) ([]AlertIncident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AlertIncident, 0)
	for _, i := range r.alertIncidents {
		if i.OrganizationID == f.OrganizationID && (f.Status == "" || f.Status == "all" || i.Status == f.Status) && (f.Severity == "" || f.Severity == "all" || i.Severity == f.Severity) {
			out = append(out, i)
		}
	}
	return out, nil
}
func developmentAlertRules() []AlertRule {
	last := time.Date(2026, 7, 14, 5, 50, 0, 0, time.UTC)
	created := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	return []AlertRule{{ID: "alert_error_rate", OrganizationID: "org_topoai", Name: "Production error rate", Status: "enabled", Metric: "error_rate", Operator: "gte", Threshold: 1, Window: "15m", CooldownMinutes: 30, Severity: "critical", Channels: []string{"in_app", "slack"}, Filters: map[string]string{"environment": "production"}, IncidentCount: 4, LastTriggeredAt: &last, CreatedAt: created, UpdatedAt: created}, {ID: "alert_latency", OrganizationID: "org_topoai", Name: "P95 latency regression", Status: "enabled", Metric: "latency", Operator: "gt", Threshold: 2500, Window: "15m", CooldownMinutes: 20, Severity: "warning", Channels: []string{"in_app", "email"}, Filters: map[string]string{}, IncidentCount: 2, CreatedAt: created, UpdatedAt: created}, {ID: "alert_cost", OrganizationID: "org_topoai", Name: "Hourly cost anomaly", Status: "draft", Metric: "cost", Operator: "gte", Threshold: 500, Window: "1h", CooldownMinutes: 60, Severity: "warning", Channels: []string{"in_app"}, Filters: map[string]string{}, CreatedAt: created, UpdatedAt: created}}
}
func developmentAlertIncidents() []AlertIncident {
	resolved := time.Date(2026, 7, 13, 10, 40, 0, 0, time.UTC)
	return []AlertIncident{{ID: "incident_err_01", OrganizationID: "org_topoai", RuleID: "alert_error_rate", RuleName: "Production error rate", Status: "open", Severity: "critical", Metric: "error_rate", ObservedValue: 1.73, Threshold: 1, Summary: "Production error rate exceeded 1% for 15 minutes.", StartedAt: time.Date(2026, 7, 14, 5, 50, 0, 0, time.UTC)}, {ID: "incident_latency_01", OrganizationID: "org_topoai", RuleID: "alert_latency", RuleName: "P95 latency regression", Status: "resolved", Severity: "warning", Metric: "latency", ObservedValue: 2840, Threshold: 2500, Summary: "P95 latency exceeded 2.5 seconds.", StartedAt: time.Date(2026, 7, 13, 10, 15, 0, 0, time.UTC), ResolvedAt: &resolved}}
}
