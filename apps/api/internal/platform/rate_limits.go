package platform

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"
)

type RateLimitRule struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organizationId"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	ScopeType       string    `json:"scopeType"`
	ScopeID         string    `json:"scopeId"`
	Metric          string    `json:"metric"`
	Window          string    `json:"window"`
	Limit           int64     `json:"limit"`
	Burst           int64     `json:"burst"`
	Action          string    `json:"action"`
	Priority        int       `json:"priority"`
	MatchedRequests int64     `json:"matchedRequests"`
	LimitedRequests int64     `json:"limitedRequests"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type RateLimitFilter struct {
	OrganizationID string
	Query          string
	Status         string
	ScopeType      string
}

type CreateRateLimitInput struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	ScopeType      string `json:"scopeType"`
	ScopeID        string `json:"scopeId"`
	Metric         string `json:"metric"`
	Window         string `json:"window"`
	Limit          int64  `json:"limit"`
	Burst          int64  `json:"burst"`
	Action         string `json:"action"`
	Priority       int    `json:"priority"`
}

type RateLimitEvaluationInput struct {
	OrganizationID string `json:"organizationId"`
	WorkspaceID    string `json:"workspaceId"`
	ProjectID      string `json:"projectId"`
	APIKeyID       string `json:"apiKeyId"`
	UserID         string `json:"userId"`
	Metric         string `json:"metric"`
	CurrentUsage   int64  `json:"currentUsage"`
	RequestedUnits int64  `json:"requestedUnits"`
}

type RateLimitMatch struct {
	RuleID            string `json:"ruleId"`
	RuleName          string `json:"ruleName"`
	ScopeType         string `json:"scopeType"`
	ScopeID           string `json:"scopeId"`
	Window            string `json:"window"`
	Action            string `json:"action"`
	Limit             int64  `json:"limit"`
	Burst             int64  `json:"burst"`
	CurrentUsage      int64  `json:"currentUsage"`
	ProjectedUsage    int64  `json:"projectedUsage"`
	Remaining         int64  `json:"remaining"`
	Exceeded          bool   `json:"exceeded"`
	RetryAfterSeconds int    `json:"retryAfterSeconds"`
}

type RateLimitDecision struct {
	Allowed bool             `json:"allowed"`
	Mode    string           `json:"mode"`
	Reason  string           `json:"reason"`
	Matches []RateLimitMatch `json:"matches"`
}

type RateLimitRepository interface {
	Repository
	ListRateLimitRules(context.Context, RateLimitFilter) ([]RateLimitRule, error)
	GetRateLimitRule(context.Context, string, string) (RateLimitRule, error)
	CreateRateLimitRule(context.Context, RateLimitRule) (RateLimitRule, error)
	UpdateRateLimitRuleStatus(context.Context, string, string, string, time.Time) (RateLimitRule, error)
}

type RateLimitService struct {
	repository RateLimitRepository
	now        func() time.Time
}

func NewRateLimitService(repository RateLimitRepository) *RateLimitService {
	return &RateLimitService{repository: repository, now: time.Now}
}

func (s *RateLimitService) List(ctx context.Context, filter RateLimitFilter) ([]RateLimitRule, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.ScopeType = strings.TrimSpace(filter.ScopeType)
	return s.repository.ListRateLimitRules(ctx, filter)
}

func (s *RateLimitService) Create(ctx context.Context, input CreateRateLimitInput) (RateLimitRule, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.ScopeID = strings.TrimSpace(input.ScopeID)
	if input.Name == "" || input.ScopeID == "" {
		return RateLimitRule{}, &ValidationError{Code: "rate_limit_scope_required", Message: "Rate-limit name and scope ID are required."}
	}
	if !slices.Contains([]string{"organization", "workspace", "project", "api_key", "user"}, input.ScopeType) {
		return RateLimitRule{}, &ValidationError{Code: "rate_limit_scope_invalid", Message: "Rate-limit scope is invalid."}
	}
	if !slices.Contains([]string{"requests", "tokens", "concurrency"}, input.Metric) || !slices.Contains([]string{"second", "minute", "hour", "day"}, input.Window) {
		return RateLimitRule{}, &ValidationError{Code: "rate_limit_dimension_invalid", Message: "Rate-limit metric or window is invalid."}
	}
	if !slices.Contains([]string{"reject", "throttle", "observe"}, input.Action) {
		return RateLimitRule{}, &ValidationError{Code: "rate_limit_action_invalid", Message: "Rate-limit action is invalid."}
	}
	if input.Limit <= 0 || input.Burst < 0 || input.Priority < 0 || input.Priority > 1000 {
		return RateLimitRule{}, &ValidationError{Code: "rate_limit_bounds_invalid", Message: "Rate-limit value, burst, or priority is outside the supported range."}
	}
	id, err := randomIdentifier("limit_", 9)
	if err != nil {
		return RateLimitRule{}, err
	}
	now := s.now().UTC()
	return s.repository.CreateRateLimitRule(ctx, RateLimitRule{ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Status: "draft", ScopeType: input.ScopeType, ScopeID: input.ScopeID, Metric: input.Metric, Window: input.Window, Limit: input.Limit, Burst: input.Burst, Action: input.Action, Priority: input.Priority, CreatedAt: now, UpdatedAt: now})
}

func (s *RateLimitService) Enforce(ctx context.Context, organizationID, id string) (RateLimitRule, error) {
	return s.repository.UpdateRateLimitRuleStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "enforced", s.now().UTC())
}

func (s *RateLimitService) Disable(ctx context.Context, organizationID, id string) (RateLimitRule, error) {
	return s.repository.UpdateRateLimitRuleStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "disabled", s.now().UTC())
}

func (s *RateLimitService) Evaluate(ctx context.Context, input RateLimitEvaluationInput) (RateLimitDecision, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	if !slices.Contains([]string{"requests", "tokens", "concurrency"}, input.Metric) || input.CurrentUsage < 0 || input.RequestedUnits <= 0 {
		return RateLimitDecision{}, &ValidationError{Code: "rate_limit_evaluation_invalid", Message: "Evaluation metric and usage values are invalid."}
	}
	rules, err := s.repository.ListRateLimitRules(ctx, RateLimitFilter{OrganizationID: input.OrganizationID, Status: "enforced"})
	if err != nil {
		return RateLimitDecision{}, err
	}
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rateLimitSpecificity(rules[i].ScopeType) > rateLimitSpecificity(rules[j].ScopeType)
	})
	decision := RateLimitDecision{Allowed: true, Mode: "dry-run", Reason: "No enforced rule would block the request.", Matches: make([]RateLimitMatch, 0)}
	for _, rule := range rules {
		if rule.Metric != input.Metric || !rateLimitScopeMatches(rule, input) {
			continue
		}
		projected := input.CurrentUsage + input.RequestedUnits
		ceiling := rule.Limit + rule.Burst
		remaining := ceiling - projected
		if remaining < 0 {
			remaining = 0
		}
		match := RateLimitMatch{RuleID: rule.ID, RuleName: rule.Name, ScopeType: rule.ScopeType, ScopeID: rule.ScopeID, Window: rule.Window, Action: rule.Action, Limit: rule.Limit, Burst: rule.Burst, CurrentUsage: input.CurrentUsage, ProjectedUsage: projected, Remaining: remaining, Exceeded: projected > ceiling, RetryAfterSeconds: rateLimitWindowSeconds(rule.Window)}
		decision.Matches = append(decision.Matches, match)
		if match.Exceeded && rule.Action != "observe" && decision.Allowed {
			decision.Allowed = false
			decision.Reason = "An enforced rate-limit rule would block or throttle this request."
		}
	}
	return decision, nil
}

func rateLimitScopeMatches(rule RateLimitRule, input RateLimitEvaluationInput) bool {
	switch rule.ScopeType {
	case "organization":
		return rule.ScopeID == input.OrganizationID
	case "workspace":
		return rule.ScopeID == input.WorkspaceID
	case "project":
		return rule.ScopeID == input.ProjectID
	case "api_key":
		return rule.ScopeID == input.APIKeyID
	case "user":
		return rule.ScopeID == input.UserID
	default:
		return false
	}
}

func rateLimitSpecificity(scope string) int {
	return map[string]int{"organization": 1, "workspace": 2, "project": 3, "user": 4, "api_key": 5}[scope]
}

func rateLimitWindowSeconds(window string) int {
	return map[string]int{"second": 1, "minute": 60, "hour": 3600, "day": 86400}[window]
}
