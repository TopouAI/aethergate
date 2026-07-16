package platform

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
)

type AlertRule struct {
	ID              string            `json:"id"`
	OrganizationID  string            `json:"organizationId"`
	Name            string            `json:"name"`
	Status          string            `json:"status"`
	Metric          string            `json:"metric"`
	Operator        string            `json:"operator"`
	Threshold       float64           `json:"threshold"`
	Window          string            `json:"window"`
	CooldownMinutes int               `json:"cooldownMinutes"`
	Severity        string            `json:"severity"`
	Channels        []string          `json:"channels"`
	Filters         map[string]string `json:"filters"`
	IncidentCount   int               `json:"incidentCount"`
	LastTriggeredAt *time.Time        `json:"lastTriggeredAt"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}
type AlertIncident struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	RuleID         string     `json:"ruleId"`
	RuleName       string     `json:"ruleName"`
	Status         string     `json:"status"`
	Severity       string     `json:"severity"`
	Metric         string     `json:"metric"`
	ObservedValue  float64    `json:"observedValue"`
	Threshold      float64    `json:"threshold"`
	Summary        string     `json:"summary"`
	StartedAt      time.Time  `json:"startedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
}
type AlertFilter struct{ OrganizationID, Query, Status, Severity string }
type CreateAlertInput struct {
	OrganizationID  string            `json:"organizationId"`
	Name            string            `json:"name"`
	Metric          string            `json:"metric"`
	Operator        string            `json:"operator"`
	Threshold       float64           `json:"threshold"`
	Window          string            `json:"window"`
	CooldownMinutes int               `json:"cooldownMinutes"`
	Severity        string            `json:"severity"`
	Channels        []string          `json:"channels"`
	Filters         map[string]string `json:"filters"`
}
type AlertEvaluationInput struct {
	OrganizationID string            `json:"organizationId"`
	Metric         string            `json:"metric"`
	Value          float64           `json:"value"`
	Dimensions     map[string]string `json:"dimensions"`
}
type AlertEvaluationMatch struct {
	RuleID        string   `json:"ruleId"`
	RuleName      string   `json:"ruleName"`
	Severity      string   `json:"severity"`
	Operator      string   `json:"operator"`
	Threshold     float64  `json:"threshold"`
	ObservedValue float64  `json:"observedValue"`
	ConditionMet  bool     `json:"conditionMet"`
	InCooldown    bool     `json:"inCooldown"`
	WouldTrigger  bool     `json:"wouldTrigger"`
	Channels      []string `json:"channels"`
	Reason        string   `json:"reason"`
}
type AlertEvaluation struct {
	Triggered bool                   `json:"triggered"`
	Mode      string                 `json:"mode"`
	Matches   []AlertEvaluationMatch `json:"matches"`
}
type AlertRepository interface {
	Repository
	ListAlertRules(context.Context, AlertFilter) ([]AlertRule, error)
	GetAlertRule(context.Context, string, string) (AlertRule, error)
	CreateAlertRule(context.Context, AlertRule) (AlertRule, error)
	UpdateAlertRuleStatus(context.Context, string, string, string, time.Time) (AlertRule, error)
	ListAlertIncidents(context.Context, AlertFilter) ([]AlertIncident, error)
}
type AlertService struct {
	repository AlertRepository
	now        func() time.Time
}

func NewAlertService(r AlertRepository) *AlertService {
	return &AlertService{repository: r, now: time.Now}
}
func (s *AlertService) List(ctx context.Context, f AlertFilter) ([]AlertRule, error) {
	f.OrganizationID = defaultOrganization(f.OrganizationID)
	f.Query = strings.TrimSpace(f.Query)
	return s.repository.ListAlertRules(ctx, f)
}
func (s *AlertService) ListIncidents(ctx context.Context, f AlertFilter) ([]AlertIncident, error) {
	f.OrganizationID = defaultOrganization(f.OrganizationID)
	return s.repository.ListAlertIncidents(ctx, f)
}
func (s *AlertService) Create(ctx context.Context, i CreateAlertInput) (AlertRule, error) {
	i.OrganizationID = defaultOrganization(i.OrganizationID)
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return AlertRule{}, &ValidationError{Code: "alert_name_required", Message: "Alert name is required."}
	}
	if !slices.Contains([]string{"cost", "error_rate", "latency", "requests", "tokens", "budget_utilization"}, i.Metric) || !slices.Contains([]string{"gt", "gte", "lt", "lte"}, i.Operator) || !slices.Contains([]string{"5m", "15m", "1h", "24h"}, i.Window) {
		return AlertRule{}, &ValidationError{Code: "alert_condition_invalid", Message: "Alert metric, operator, or window is invalid."}
	}
	if !slices.Contains([]string{"info", "warning", "critical"}, i.Severity) || i.CooldownMinutes < 0 || i.CooldownMinutes > 10080 {
		return AlertRule{}, &ValidationError{Code: "alert_delivery_invalid", Message: "Alert severity or cooldown is invalid."}
	}
	if len(i.Channels) == 0 {
		return AlertRule{}, &ValidationError{Code: "alert_channel_required", Message: "At least one alert channel is required."}
	}
	for _, c := range i.Channels {
		if !slices.Contains([]string{"in_app", "email", "webhook", "slack", "teams"}, c) {
			return AlertRule{}, &ValidationError{Code: "alert_channel_invalid", Message: "Alert channel is invalid."}
		}
	}
	id, err := randomIdentifier("alert_", 9)
	if err != nil {
		return AlertRule{}, err
	}
	now := s.now().UTC()
	return s.repository.CreateAlertRule(ctx, AlertRule{ID: id, OrganizationID: i.OrganizationID, Name: i.Name, Status: "draft", Metric: i.Metric, Operator: i.Operator, Threshold: i.Threshold, Window: i.Window, CooldownMinutes: i.CooldownMinutes, Severity: i.Severity, Channels: slices.Clone(i.Channels), Filters: i.Filters, CreatedAt: now, UpdatedAt: now})
}
func (s *AlertService) Enable(ctx context.Context, org, id string) (AlertRule, error) {
	return s.repository.UpdateAlertRuleStatus(ctx, defaultOrganization(org), strings.TrimSpace(id), "enabled", s.now().UTC())
}
func (s *AlertService) Disable(ctx context.Context, org, id string) (AlertRule, error) {
	return s.repository.UpdateAlertRuleStatus(ctx, defaultOrganization(org), strings.TrimSpace(id), "disabled", s.now().UTC())
}
func (s *AlertService) Evaluate(ctx context.Context, i AlertEvaluationInput) (AlertEvaluation, error) {
	i.OrganizationID = defaultOrganization(i.OrganizationID)
	if !slices.Contains([]string{"cost", "error_rate", "latency", "requests", "tokens", "budget_utilization"}, i.Metric) {
		return AlertEvaluation{}, &ValidationError{Code: "alert_evaluation_invalid", Message: "Evaluation metric is invalid."}
	}
	rules, err := s.repository.ListAlertRules(ctx, AlertFilter{OrganizationID: i.OrganizationID, Status: "enabled"})
	if err != nil {
		return AlertEvaluation{}, err
	}
	out := AlertEvaluation{Mode: "dry-run", Matches: make([]AlertEvaluationMatch, 0)}
	now := s.now()
	for _, r := range rules {
		if r.Metric != i.Metric || !alertFiltersMatch(r.Filters, i.Dimensions) {
			continue
		}
		met := alertCompare(i.Value, r.Operator, r.Threshold)
		cool := r.LastTriggeredAt != nil && now.Before(r.LastTriggeredAt.Add(time.Duration(r.CooldownMinutes)*time.Minute))
		would := met && !cool
		reason := "Condition is not met."
		if met {
			reason = "Condition is met and would trigger."
		}
		if cool {
			reason = fmt.Sprintf("Condition is met but rule remains in its %d minute cooldown.", r.CooldownMinutes)
		}
		out.Matches = append(out.Matches, AlertEvaluationMatch{RuleID: r.ID, RuleName: r.Name, Severity: r.Severity, Operator: r.Operator, Threshold: r.Threshold, ObservedValue: i.Value, ConditionMet: met, InCooldown: cool, WouldTrigger: would, Channels: slices.Clone(r.Channels), Reason: reason})
		if would {
			out.Triggered = true
		}
	}
	return out, nil
}
func alertCompare(v float64, op string, t float64) bool {
	switch op {
	case "gt":
		return v > t
	case "gte":
		return v >= t
	case "lt":
		return v < t
	case "lte":
		return v <= t
	}
	return false
}
func alertFiltersMatch(filters, dimensions map[string]string) bool {
	for k, v := range filters {
		if dimensions[k] != v {
			return false
		}
	}
	return true
}
