package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
	"time"
)

func (r *Repository) ListBudgets(ctx context.Context, f platform.BudgetFilter) ([]platform.Budget, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,organization_id,name,status,scope_type,scope_id,period,limit_usd,warning_percent,critical_percent,action,spent_usd,committed_usd,forecast_usd,starts_at,ends_at,created_at,updated_at FROM budgets WHERE organization_id=$1 AND deleted_at IS NULL AND ($2='' OR $2='all' OR status=$2) AND ($3='' OR $3='all' OR scope_type=$3) AND ($4='' OR lower(name) LIKE '%'||lower($4)||'%' OR lower(scope_id) LIKE '%'||lower($4)||'%') ORDER BY created_at DESC`, f.OrganizationID, f.Status, f.ScopeType, f.Query)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Budget, 0)
	for rows.Next() {
		var b platform.Budget
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.Name, &b.Status, &b.ScopeType, &b.ScopeID, &b.Period, &b.LimitUSD, &b.WarningPercent, &b.CriticalPercent, &b.Action, &b.SpentUSD, &b.CommittedUSD, &b.ForecastUSD, &b.StartsAt, &b.EndsAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}
func (r *Repository) GetBudget(ctx context.Context, org, id string) (platform.Budget, error) {
	var b platform.Budget
	err := r.pool.QueryRow(ctx, `SELECT id,organization_id,name,status,scope_type,scope_id,period,limit_usd,warning_percent,critical_percent,action,spent_usd,committed_usd,forecast_usd,starts_at,ends_at,created_at,updated_at FROM budgets WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, org, id).Scan(&b.ID, &b.OrganizationID, &b.Name, &b.Status, &b.ScopeType, &b.ScopeID, &b.Period, &b.LimitUSD, &b.WarningPercent, &b.CriticalPercent, &b.Action, &b.SpentUSD, &b.CommittedUSD, &b.ForecastUSD, &b.StartsAt, &b.EndsAt, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.Budget{}, platform.ErrNotFound
	}
	return b, err
}
func (r *Repository) CreateBudget(ctx context.Context, b platform.Budget) (platform.Budget, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO budgets(id,organization_id,name,status,scope_type,scope_id,period,limit_usd,warning_percent,critical_percent,action,starts_at,ends_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, b.ID, b.OrganizationID, b.Name, b.Status, b.ScopeType, b.ScopeID, b.Period, b.LimitUSD, b.WarningPercent, b.CriticalPercent, b.Action, b.StartsAt, b.EndsAt, b.CreatedAt, b.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.Budget{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.Budget{}, platform.ErrConflict
	}
	if err != nil {
		return platform.Budget{}, fmt.Errorf("create budget: %w", err)
	}
	return b, nil
}
func (r *Repository) UpdateBudgetStatus(ctx context.Context, org, id, status string, updated time.Time) (platform.Budget, error) {
	c, err := r.pool.Exec(ctx, `UPDATE budgets SET status=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, org, id, status, updated)
	if err != nil {
		return platform.Budget{}, err
	}
	if c.RowsAffected() == 0 {
		return platform.Budget{}, platform.ErrNotFound
	}
	return r.GetBudget(ctx, org, id)
}

var _ platform.BudgetRepository = (*Repository)(nil)
