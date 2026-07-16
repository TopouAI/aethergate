package postgres

import (
	"context"
	"fmt"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListProviders(ctx context.Context, filter platform.ProviderFilter) ([]platform.ProviderConnection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, name, provider, base_url, status, credential_state,
		       model_count,p95_latency_ms,success_rate,last_checked_at,created_at,
		       routing_eligible,health_source,health_reason,error_rate,request_count_24h,
		       average_latency_ms,consecutive_failures,last_transition_at,maintenance_until,maintenance_reason
		FROM provider_connections
		WHERE organization_id = $1 AND deleted_at IS NULL
		  AND ($2 = '' OR $2 = 'all' OR status = $2)
		  AND ($3 = '' OR lower(name) LIKE '%' || lower($3) || '%'
		       OR lower(provider) LIKE '%' || lower($3) || '%'
		       OR lower(base_url) LIKE '%' || lower($3) || '%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Status, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()
	items := make([]platform.ProviderConnection, 0)
	for rows.Next() {
		item, err := scanProviderConnection(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateProvider(ctx context.Context, provider platform.ProviderConnection) (platform.ProviderConnection, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO provider_connections(id,organization_id,name,provider,base_url,status,credential_state,routing_eligible,health_source,health_reason,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, provider.ID, provider.OrganizationID,
		provider.Name, provider.Provider, provider.BaseURL, provider.Status, provider.CredentialState,
		provider.RoutingEligible, provider.HealthSource, provider.HealthReason, provider.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.ProviderConnection{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.ProviderConnection{}, platform.ErrConflict
	}
	if err != nil {
		return platform.ProviderConnection{}, fmt.Errorf("create provider: %w", err)
	}
	return provider, nil
}

var _ platform.ProviderRepository = (*Repository)(nil)
