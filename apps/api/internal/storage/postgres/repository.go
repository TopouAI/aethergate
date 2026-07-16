package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

type Repository struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	// PgBouncer transaction pooling cannot rely on connection-local prepared statements.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() { r.pool.Close() }

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) ListOrganizations(ctx context.Context, filter platform.OrganizationFilter) ([]platform.Organization, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.name, o.slug, o.status, o.plan, o.region,
		       (SELECT count(*) FROM workspaces w WHERE w.organization_id = o.id AND w.deleted_at IS NULL),
		       (SELECT count(*) FROM projects p WHERE p.organization_id = o.id AND p.deleted_at IS NULL),
		       (SELECT count(*) FROM members m WHERE m.organization_id = o.id AND m.deleted_at IS NULL),
		       o.monthly_cost_usd, o.budget_usd, o.request_count, o.owner_email, o.created_at
		FROM organizations o
		WHERE o.deleted_at IS NULL
		  AND ($1 = '' OR $1 = 'all' OR o.status = $1)
		  AND ($2 = '' OR lower(o.name) LIKE '%' || lower($2) || '%'
		       OR lower(o.slug) LIKE '%' || lower($2) || '%'
		       OR lower(o.owner_email) LIKE '%' || lower($2) || '%'
		       OR lower(o.region) LIKE '%' || lower($2) || '%')
		ORDER BY o.created_at DESC`, filter.Status, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	organizations := make([]platform.Organization, 0)
	for rows.Next() {
		organization, err := scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		organizations = append(organizations, organization)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}
	return organizations, nil
}

func (r *Repository) GetOrganization(ctx context.Context, id string) (platform.Organization, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT o.id, o.name, o.slug, o.status, o.plan, o.region,
		       (SELECT count(*) FROM workspaces w WHERE w.organization_id = o.id AND w.deleted_at IS NULL),
		       (SELECT count(*) FROM projects p WHERE p.organization_id = o.id AND p.deleted_at IS NULL),
		       (SELECT count(*) FROM members m WHERE m.organization_id = o.id AND m.deleted_at IS NULL),
		       o.monthly_cost_usd, o.budget_usd, o.request_count, o.owner_email, o.created_at
		FROM organizations o
		WHERE o.id = $1 AND o.deleted_at IS NULL`, id)
	organization, err := scanOrganization(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.Organization{}, platform.ErrNotFound
	}
	return organization, err
}

func (r *Repository) CreateOrganization(ctx context.Context, organization platform.Organization) (platform.Organization, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, slug, status, plan, region, owner_email,
			monthly_cost_usd, budget_usd, request_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		organization.ID, organization.Name, organization.Slug, organization.Status, organization.Plan,
		organization.Region, organization.Owner, organization.MonthlyCostUSD, organization.BudgetUSD,
		organization.Requests, organization.CreatedAt)
	if isUniqueViolation(err) {
		return platform.Organization{}, platform.ErrConflict
	}
	if err != nil {
		return platform.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return organization, nil
}

func (r *Repository) ListAPIKeys(ctx context.Context, filter platform.APIKeyFilter) ([]platform.APIKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT k.id, k.organization_id, k.name, k.prefix, k.project_id, k.project_name,
		       k.status, COALESCE(array_agg(km.model_id) FILTER (WHERE km.model_id IS NOT NULL), '{}'),
		       k.rpm, k.tpm, k.spend_usd, k.created_by, k.created_at,
		       k.last_used_at, k.expires_at
		FROM api_keys k
		LEFT JOIN api_key_models km ON km.api_key_id = k.id
		WHERE ($1 = '' OR k.organization_id = $1)
		  AND ($2 = '' OR $2 = 'all' OR k.status = $2)
		  AND ($3 = '' OR lower(k.name) LIKE '%' || lower($3) || '%'
		       OR lower(k.prefix) LIKE '%' || lower($3) || '%'
		       OR lower(k.project_name) LIKE '%' || lower($3) || '%'
		       OR lower(k.created_by) LIKE '%' || lower($3) || '%'
		       OR EXISTS (SELECT 1 FROM api_key_models search_models WHERE search_models.api_key_id = k.id AND lower(search_models.model_id) LIKE '%' || lower($3) || '%'))
		GROUP BY k.id
		ORDER BY k.created_at DESC`, filter.OrganizationID, filter.Status, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	keys := make([]platform.APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return keys, nil
}

func (r *Repository) CreateAPIKey(ctx context.Context, key platform.APIKey) (platform.APIKey, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.APIKey{}, fmt.Errorf("begin api key transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO api_keys (
			id, organization_id, project_id, project_name, name, prefix, secret_digest,
			status, rpm, tpm, spend_usd, created_by, created_at, last_used_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		key.ID, key.Organization, key.ProjectID, key.Project, key.Name, key.Prefix, key.SecretDigest[:],
		key.Status, key.RPM, key.TPM, key.SpendUSD, key.CreatedBy, key.CreatedAt, key.LastUsedAt, key.ExpiresAt)
	if isForeignKeyViolation(err) {
		return platform.APIKey{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.APIKey{}, platform.ErrConflict
	}
	if err != nil {
		return platform.APIKey{}, fmt.Errorf("insert api key: %w", err)
	}
	for _, model := range key.Models {
		if _, err := tx.Exec(ctx, `INSERT INTO api_key_models(api_key_id, model_id) VALUES ($1, $2)`, key.ID, model); err != nil {
			if isForeignKeyViolation(err) {
				return platform.APIKey{}, &platform.ValidationError{Code: "model_not_found", Message: "One or more allowed models do not exist."}
			}
			return platform.APIKey{}, fmt.Errorf("insert api key model: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.APIKey{}, fmt.Errorf("commit api key: %w", err)
	}
	return key, nil
}

func (r *Repository) RevokeAPIKey(ctx context.Context, id, actor string, revokedAt time.Time) (platform.APIKey, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE api_keys
		SET status = 'revoked', revoked_at = $2, revoked_by = $3
		WHERE id = $1 AND status = 'active'`, id, revokedAt, actor)
	if err != nil {
		return platform.APIKey{}, fmt.Errorf("revoke api key: %w", err)
	}
	if command.RowsAffected() == 0 {
		var status string
		err := r.pool.QueryRow(ctx, `SELECT status FROM api_keys WHERE id = $1`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.APIKey{}, platform.ErrNotFound
		}
		if err != nil {
			return platform.APIKey{}, fmt.Errorf("read api key status: %w", err)
		}
		return platform.APIKey{}, platform.ErrInactive
	}
	keys, err := r.ListAPIKeys(ctx, platform.APIKeyFilter{Query: id})
	if err != nil {
		return platform.APIKey{}, err
	}
	for _, key := range keys {
		if key.ID == id {
			return key, nil
		}
	}
	return platform.APIKey{}, platform.ErrNotFound
}

type rowScanner interface {
	Scan(...any) error
}

func scanOrganization(row rowScanner) (platform.Organization, error) {
	var organization platform.Organization
	err := row.Scan(
		&organization.ID, &organization.Name, &organization.Slug, &organization.Status,
		&organization.Plan, &organization.Region, &organization.Workspaces, &organization.Projects,
		&organization.Members, &organization.MonthlyCostUSD, &organization.BudgetUSD,
		&organization.Requests, &organization.Owner, &organization.CreatedAt,
	)
	if err != nil {
		return platform.Organization{}, err
	}
	return organization, nil
}

func scanAPIKey(row rowScanner) (platform.APIKey, error) {
	var key platform.APIKey
	err := row.Scan(
		&key.ID, &key.Organization, &key.Name, &key.Prefix, &key.ProjectID, &key.Project,
		&key.Status, &key.Models, &key.RPM, &key.TPM, &key.SpendUSD, &key.CreatedBy,
		&key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt,
	)
	if err != nil {
		return platform.APIKey{}, err
	}
	return key, nil
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23503"
}
