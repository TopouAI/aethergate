package platform

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"
)

type Budget struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organizationId"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	ScopeType       string    `json:"scopeType"`
	ScopeID         string    `json:"scopeId"`
	Period          string    `json:"period"`
	LimitUSD        float64   `json:"limitUsd"`
	WarningPercent  int       `json:"warningPercent"`
	CriticalPercent int       `json:"criticalPercent"`
	Action          string    `json:"action"`
	SpentUSD        float64   `json:"spentUsd"`
	CommittedUSD    float64   `json:"committedUsd"`
	ForecastUSD     float64   `json:"forecastUsd"`
	StartsAt        time.Time `json:"startsAt"`
	EndsAt          time.Time `json:"endsAt"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type BudgetFilter struct{ OrganizationID, Query, Status, ScopeType string }
type CreateBudgetInput struct {
	OrganizationID  string  `json:"organizationId"`
	Name            string  `json:"name"`
	ScopeType       string  `json:"scopeType"`
	ScopeID         string  `json:"scopeId"`
	Period          string  `json:"period"`
	LimitUSD        float64 `json:"limitUsd"`
	WarningPercent  int     `json:"warningPercent"`
	CriticalPercent int     `json:"criticalPercent"`
	Action          string  `json:"action"`
}
type BudgetEvaluationInput struct {
	OrganizationID   string  `json:"organizationId"`
	WorkspaceID      string  `json:"workspaceId"`
	ProjectID        string  `json:"projectId"`
	CurrentSpendUSD  float64 `json:"currentSpendUsd"`
	ProposedSpendUSD float64 `json:"proposedSpendUsd"`
	ElapsedPercent   float64 `json:"elapsedPercent"`
}
type BudgetMatch struct {
	BudgetID           string  `json:"budgetId"`
	BudgetName         string  `json:"budgetName"`
	ScopeType          string  `json:"scopeType"`
	ScopeID            string  `json:"scopeId"`
	Action             string  `json:"action"`
	LimitUSD           float64 `json:"limitUsd"`
	ProjectedSpendUSD  float64 `json:"projectedSpendUsd"`
	ForecastUSD        float64 `json:"forecastUsd"`
	UtilizationPercent float64 `json:"utilizationPercent"`
	Threshold          string  `json:"threshold"`
	RemainingUSD       float64 `json:"remainingUsd"`
}
type BudgetDecision struct {
	Allowed          bool          `json:"allowed"`
	RequiresApproval bool          `json:"requiresApproval"`
	Mode             string        `json:"mode"`
	Reason           string        `json:"reason"`
	Matches          []BudgetMatch `json:"matches"`
}

type BudgetRepository interface {
	Repository
	ListBudgets(context.Context, BudgetFilter) ([]Budget, error)
	GetBudget(context.Context, string, string) (Budget, error)
	CreateBudget(context.Context, Budget) (Budget, error)
	UpdateBudgetStatus(context.Context, string, string, string, time.Time) (Budget, error)
}
type BudgetService struct {
	repository BudgetRepository
	now        func() time.Time
}

func NewBudgetService(repository BudgetRepository) *BudgetService {
	return &BudgetService{repository: repository, now: time.Now}
}

func (s *BudgetService) List(ctx context.Context, filter BudgetFilter) ([]Budget, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.ScopeType = strings.TrimSpace(filter.ScopeType)
	return s.repository.ListBudgets(ctx, filter)
}
func (s *BudgetService) Create(ctx context.Context, input CreateBudgetInput) (Budget, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.ScopeID = strings.TrimSpace(input.ScopeID)
	if input.Name == "" || input.ScopeID == "" {
		return Budget{}, &ValidationError{Code: "budget_scope_required", Message: "Budget name and scope ID are required."}
	}
	if !slices.Contains([]string{"organization", "workspace", "project"}, input.ScopeType) || !slices.Contains([]string{"monthly", "quarterly", "annual"}, input.Period) {
		return Budget{}, &ValidationError{Code: "budget_dimension_invalid", Message: "Budget scope or period is invalid."}
	}
	if !slices.Contains([]string{"alert", "block", "approval"}, input.Action) {
		return Budget{}, &ValidationError{Code: "budget_action_invalid", Message: "Budget action is invalid."}
	}
	if input.LimitUSD <= 0 || input.WarningPercent < 1 || input.WarningPercent >= input.CriticalPercent || input.CriticalPercent > 100 {
		return Budget{}, &ValidationError{Code: "budget_threshold_invalid", Message: "Budget limit and thresholds are invalid."}
	}
	id, err := randomIdentifier("budget_", 9)
	if err != nil {
		return Budget{}, err
	}
	now := s.now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := 1
	if input.Period == "quarterly" {
		months = 3
	}
	if input.Period == "annual" {
		months = 12
	}
	return s.repository.CreateBudget(ctx, Budget{ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Status: "draft", ScopeType: input.ScopeType, ScopeID: input.ScopeID, Period: input.Period, LimitUSD: input.LimitUSD, WarningPercent: input.WarningPercent, CriticalPercent: input.CriticalPercent, Action: input.Action, StartsAt: start, EndsAt: start.AddDate(0, months, 0), CreatedAt: now, UpdatedAt: now})
}
func (s *BudgetService) Activate(ctx context.Context, organizationID, id string) (Budget, error) {
	return s.repository.UpdateBudgetStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "active", s.now().UTC())
}
func (s *BudgetService) Pause(ctx context.Context, organizationID, id string) (Budget, error) {
	return s.repository.UpdateBudgetStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "paused", s.now().UTC())
}
func (s *BudgetService) Evaluate(ctx context.Context, input BudgetEvaluationInput) (BudgetDecision, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	if input.CurrentSpendUSD < 0 || input.ProposedSpendUSD < 0 || input.ElapsedPercent <= 0 || input.ElapsedPercent > 100 {
		return BudgetDecision{}, &ValidationError{Code: "budget_evaluation_invalid", Message: "Spend and elapsed percentage are invalid."}
	}
	budgets, err := s.repository.ListBudgets(ctx, BudgetFilter{OrganizationID: input.OrganizationID, Status: "active"})
	if err != nil {
		return BudgetDecision{}, err
	}
	sort.SliceStable(budgets, func(i, j int) bool {
		return budgetSpecificity(budgets[i].ScopeType) > budgetSpecificity(budgets[j].ScopeType)
	})
	decision := BudgetDecision{Allowed: true, Mode: "dry-run", Reason: "No active budget would block the proposed spend.", Matches: make([]BudgetMatch, 0)}
	for _, budget := range budgets {
		if !budgetScopeMatches(budget, input) {
			continue
		}
		projected := input.CurrentSpendUSD + input.ProposedSpendUSD
		forecast := input.CurrentSpendUSD/(input.ElapsedPercent/100) + input.ProposedSpendUSD
		utilization := projected / budget.LimitUSD * 100
		threshold := "healthy"
		if utilization >= float64(budget.CriticalPercent) {
			threshold = "critical"
		} else if utilization >= float64(budget.WarningPercent) {
			threshold = "warning"
		}
		remaining := budget.LimitUSD - projected
		if remaining < 0 {
			remaining = 0
		}
		decision.Matches = append(decision.Matches, BudgetMatch{BudgetID: budget.ID, BudgetName: budget.Name, ScopeType: budget.ScopeType, ScopeID: budget.ScopeID, Action: budget.Action, LimitUSD: budget.LimitUSD, ProjectedSpendUSD: projected, ForecastUSD: forecast, UtilizationPercent: utilization, Threshold: threshold, RemainingUSD: remaining})
		if projected > budget.LimitUSD {
			if budget.Action == "block" {
				decision.Allowed = false
				decision.Reason = "An active budget would block the proposed spend."
			}
			if budget.Action == "approval" {
				decision.RequiresApproval = true
				decision.Reason = "The proposed spend requires budget approval."
			}
		}
	}
	return decision, nil
}
func budgetScopeMatches(b Budget, i BudgetEvaluationInput) bool {
	switch b.ScopeType {
	case "organization":
		return b.ScopeID == i.OrganizationID
	case "workspace":
		return b.ScopeID == i.WorkspaceID
	case "project":
		return b.ScopeID == i.ProjectID
	}
	return false
}
func budgetSpecificity(scope string) int {
	return map[string]int{"organization": 1, "workspace": 2, "project": 3}[scope]
}
