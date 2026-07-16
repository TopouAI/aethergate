package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListRateLimitRules(ctx context.Context, filter platform.RateLimitFilter) ([]platform.RateLimitRule, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, organization_id, name, status, scope_type, scope_id, metric, window_unit, limit_value, burst, action, priority, matched_requests, limited_requests, created_at, updated_at FROM rate_limit_rules WHERE organization_id=$1 AND deleted_at IS NULL AND ($2='' OR $2='all' OR status=$2) AND ($3='' OR $3='all' OR scope_type=$3) AND ($4='' OR lower(name) LIKE '%'||lower($4)||'%' OR lower(scope_id) LIKE '%'||lower($4)||'%') ORDER BY priority DESC, created_at DESC`, filter.OrganizationID, filter.Status, filter.ScopeType, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list rate limits: %w", err)
	}
	defer rows.Close()
	items := make([]platform.RateLimitRule, 0)
	for rows.Next() {
		var item platform.RateLimitRule
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Status, &item.ScopeType, &item.ScopeID, &item.Metric, &item.Window, &item.Limit, &item.Burst, &item.Action, &item.Priority, &item.MatchedRequests, &item.LimitedRequests, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rate limit: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetRateLimitRule(ctx context.Context, organizationID, id string) (platform.RateLimitRule, error) {
	var item platform.RateLimitRule
	err := r.pool.QueryRow(ctx, `SELECT id, organization_id, name, status, scope_type, scope_id, metric, window_unit, limit_value, burst, action, priority, matched_requests, limited_requests, created_at, updated_at FROM rate_limit_rules WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id).Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Status, &item.ScopeType, &item.ScopeID, &item.Metric, &item.Window, &item.Limit, &item.Burst, &item.Action, &item.Priority, &item.MatchedRequests, &item.LimitedRequests, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.RateLimitRule{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.RateLimitRule{}, fmt.Errorf("get rate limit: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateRateLimitRule(ctx context.Context, rule platform.RateLimitRule) (platform.RateLimitRule, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO rate_limit_rules(id,organization_id,name,status,scope_type,scope_id,metric,window_unit,limit_value,burst,action,priority,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, rule.ID, rule.OrganizationID, rule.Name, rule.Status, rule.ScopeType, rule.ScopeID, rule.Metric, rule.Window, rule.Limit, rule.Burst, rule.Action, rule.Priority, rule.CreatedAt, rule.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.RateLimitRule{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.RateLimitRule{}, platform.ErrConflict
	}
	if err != nil {
		return platform.RateLimitRule{}, fmt.Errorf("create rate limit: %w", err)
	}
	return rule, nil
}

func (r *Repository) UpdateRateLimitRuleStatus(ctx context.Context, organizationID, id, status string, updatedAt time.Time) (platform.RateLimitRule, error) {
	command, err := r.pool.Exec(ctx, `UPDATE rate_limit_rules SET status=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id, status, updatedAt)
	if err != nil {
		return platform.RateLimitRule{}, fmt.Errorf("update rate limit status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.RateLimitRule{}, platform.ErrNotFound
	}
	return r.GetRateLimitRule(ctx, organizationID, id)
}

var _ platform.RateLimitRepository = (*Repository)(nil)
