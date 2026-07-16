package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListBudgets(_ context.Context, f BudgetFilter) ([]Budget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := strings.ToLower(f.Query)
	items := make([]Budget, 0)
	for _, b := range r.budgets {
		if b.OrganizationID != f.OrganizationID || f.Status != "" && f.Status != "all" && b.Status != f.Status || f.ScopeType != "" && f.ScopeType != "all" && b.ScopeType != f.ScopeType {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(b.Name+" "+b.ScopeID+" "+b.Action), q) {
			continue
		}
		items = append(items, b)
	}
	return items, nil
}
func (r *MemoryRepository) GetBudget(_ context.Context, org, id string) (Budget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.budgets {
		if b.OrganizationID == org && b.ID == id {
			return b, nil
		}
	}
	return Budget{}, ErrNotFound
}
func (r *MemoryRepository) CreateBudget(_ context.Context, b Budget) (Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := findOrganization(r.organizations, b.OrganizationID); !ok {
		return Budget{}, ErrNotFound
	}
	if slices.ContainsFunc(r.budgets, func(x Budget) bool { return x.OrganizationID == b.OrganizationID && strings.EqualFold(x.Name, b.Name) }) {
		return Budget{}, ErrConflict
	}
	r.budgets = append([]Budget{b}, r.budgets...)
	return b, nil
}
func (r *MemoryRepository) UpdateBudgetStatus(_ context.Context, org, id, status string, updated time.Time) (Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.budgets {
		if r.budgets[i].OrganizationID == org && r.budgets[i].ID == id {
			r.budgets[i].Status = status
			r.budgets[i].UpdatedAt = updated
			return r.budgets[i], nil
		}
	}
	return Budget{}, ErrNotFound
}
func developmentBudgets() []Budget {
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := created.AddDate(0, 1, 0)
	return []Budget{{ID: "budget_org_monthly", OrganizationID: "org_topoai", Name: "TopoAI monthly envelope", Status: "active", ScopeType: "organization", ScopeID: "org_topoai", Period: "monthly", LimitUSD: 12000, WarningPercent: 70, CriticalPercent: 90, Action: "approval", SpentUSD: 6842, CommittedUSD: 920, ForecastUSD: 10430, StartsAt: created, EndsAt: end, CreatedAt: created, UpdatedAt: created}, {ID: "budget_engineering", OrganizationID: "org_topoai", Name: "Engineering AI budget", Status: "active", ScopeType: "workspace", ScopeID: "ws_engineering", Period: "monthly", LimitUSD: 7000, WarningPercent: 75, CriticalPercent: 95, Action: "block", SpentUSD: 2780.30, CommittedUSD: 630, ForecastUSD: 4890, StartsAt: created, EndsAt: end, CreatedAt: created, UpdatedAt: created}, {ID: "budget_finance", OrganizationID: "org_topoai", Name: "Finance analyst budget", Status: "draft", ScopeType: "project", ScopeID: "project_finance_analyst", Period: "monthly", LimitUSD: 2500, WarningPercent: 70, CriticalPercent: 90, Action: "alert", SpentUSD: 994.16, ForecastUSD: 1680, StartsAt: created, EndsAt: end, CreatedAt: created, UpdatedAt: created}}
}
