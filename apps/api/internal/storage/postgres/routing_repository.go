package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListRoutingPolicies(ctx context.Context, filter platform.RoutingPolicyFilter) ([]platform.RoutingPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, name, slug, status, strategy, model_pattern, max_retries, request_timeout_ms, created_at, updated_at
		FROM routing_policies
		WHERE organization_id = $1 AND deleted_at IS NULL
		  AND ($2 = '' OR $2 = 'all' OR status = $2)
		  AND ($3 = '' OR lower(name) LIKE '%' || lower($3) || '%' OR lower(slug) LIKE '%' || lower($3) || '%' OR lower(model_pattern) LIKE '%' || lower($3) || '%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Status, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list routing policies: %w", err)
	}
	defer rows.Close()
	items := make([]platform.RoutingPolicy, 0)
	for rows.Next() {
		var item platform.RoutingPolicy
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Slug, &item.Status, &item.Strategy, &item.ModelPattern, &item.MaxRetries, &item.RequestTimeoutMS, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan routing policy: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Targets, err = r.listRoutingTargets(ctx, items[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *Repository) GetRoutingPolicy(ctx context.Context, organizationID, id string) (platform.RoutingPolicy, error) {
	var item platform.RoutingPolicy
	err := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, name, slug, status, strategy, model_pattern, max_retries, request_timeout_ms, created_at, updated_at
		FROM routing_policies WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`, organizationID, id).Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Slug, &item.Status, &item.Strategy, &item.ModelPattern, &item.MaxRetries, &item.RequestTimeoutMS, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.RoutingPolicy{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.RoutingPolicy{}, fmt.Errorf("get routing policy: %w", err)
	}
	item.Targets, err = r.listRoutingTargets(ctx, item.ID)
	return item, err
}

func (r *Repository) CreateRoutingPolicy(ctx context.Context, policy platform.RoutingPolicy) (platform.RoutingPolicy, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.RoutingPolicy{}, fmt.Errorf("begin routing policy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO routing_policies(id, organization_id, name, slug, status, strategy, model_pattern, max_retries, request_timeout_ms, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, policy.ID, policy.OrganizationID, policy.Name, policy.Slug, policy.Status, policy.Strategy, policy.ModelPattern, policy.MaxRetries, policy.RequestTimeoutMS, policy.CreatedAt, policy.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.RoutingPolicy{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.RoutingPolicy{}, platform.ErrConflict
	}
	if err != nil {
		return platform.RoutingPolicy{}, fmt.Errorf("create routing policy: %w", err)
	}
	for _, target := range policy.Targets {
		_, err = tx.Exec(ctx, `INSERT INTO routing_targets(id, policy_id, provider_id, model, priority, weight, enabled) VALUES ($1,$2,$3,$4,$5,$6,$7)`, target.ID, policy.ID, target.ProviderID, target.Model, target.Priority, target.Weight, target.Enabled)
		if isForeignKeyViolation(err) {
			return platform.RoutingPolicy{}, &platform.ValidationError{Code: "routing_target_invalid", Message: "A routing target references an unavailable provider."}
		}
		if err != nil {
			return platform.RoutingPolicy{}, fmt.Errorf("create routing target: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.RoutingPolicy{}, fmt.Errorf("commit routing policy: %w", err)
	}
	return policy, nil
}

func (r *Repository) UpdateRoutingPolicyStatus(ctx context.Context, organizationID, id, status string, updatedAt time.Time) (platform.RoutingPolicy, error) {
	command, err := r.pool.Exec(ctx, `UPDATE routing_policies SET status=$3, updated_at=$4 WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id, status, updatedAt)
	if err != nil {
		return platform.RoutingPolicy{}, fmt.Errorf("update routing policy status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.RoutingPolicy{}, platform.ErrNotFound
	}
	return r.GetRoutingPolicy(ctx, organizationID, id)
}

func (r *Repository) listRoutingTargets(ctx context.Context, policyID string) ([]platform.RoutingTarget, error) {
	rows, err := r.pool.Query(ctx, `SELECT t.id, t.provider_id, p.name, t.model, t.priority, t.weight, t.enabled FROM routing_targets t JOIN provider_connections p ON p.id=t.provider_id WHERE t.policy_id=$1 ORDER BY t.priority, t.id`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list routing targets: %w", err)
	}
	defer rows.Close()
	items := make([]platform.RoutingTarget, 0)
	for rows.Next() {
		var item platform.RoutingTarget
		if err := rows.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.Model, &item.Priority, &item.Weight, &item.Enabled); err != nil {
			return nil, fmt.Errorf("scan routing target: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ platform.RoutingRepository = (*Repository)(nil)
