package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListWorkspaces(ctx context.Context, organizationID string) ([]platform.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.organization_id, w.name, w.slug, w.status, w.environment,
		       count(p.id) FILTER (WHERE p.deleted_at IS NULL), w.created_at
		FROM workspaces w
		LEFT JOIN projects p ON p.workspace_id = w.id
		WHERE w.organization_id = $1 AND w.deleted_at IS NULL
		GROUP BY w.id
		ORDER BY w.created_at DESC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Workspace, 0)
	for rows.Next() {
		var item platform.Workspace
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Slug, &item.Status, &item.Environment, &item.Projects, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateWorkspace(ctx context.Context, workspace platform.Workspace) (platform.Workspace, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspaces(id, organization_id, name, slug, status, environment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, workspace.ID, workspace.OrganizationID, workspace.Name, workspace.Slug, workspace.Status, workspace.Environment, workspace.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.Workspace{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.Workspace{}, platform.ErrConflict
	}
	if err != nil {
		return platform.Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	return workspace, nil
}

func (r *Repository) ListProjects(ctx context.Context, organizationID, workspaceID string) ([]platform.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.organization_id, p.workspace_id, w.name, p.name, p.slug, p.status,
		       p.owner_email, p.budget_usd, p.monthly_cost_usd, p.request_count, p.created_at
		FROM projects p
		JOIN workspaces w ON w.id = p.workspace_id
		WHERE p.organization_id = $1 AND p.deleted_at IS NULL
		  AND ($2 = '' OR p.workspace_id = $2)
		ORDER BY p.created_at DESC`, organizationID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Project, 0)
	for rows.Next() {
		var item platform.Project
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.WorkspaceID, &item.Workspace, &item.Name, &item.Slug, &item.Status, &item.Owner, &item.BudgetUSD, &item.MonthlyCostUSD, &item.Requests, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateProject(ctx context.Context, project platform.Project) (platform.Project, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO projects(id, organization_id, workspace_id, name, slug, status, owner_email, budget_usd, monthly_cost_usd, request_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING (SELECT name FROM workspaces WHERE id = $3)`,
		project.ID, project.OrganizationID, project.WorkspaceID, project.Name, project.Slug, project.Status,
		project.Owner, project.BudgetUSD, project.MonthlyCostUSD, project.Requests, project.CreatedAt)
	if err := row.Scan(&project.Workspace); err != nil {
		if isForeignKeyViolation(err) || errors.Is(err, pgx.ErrNoRows) {
			return platform.Project{}, platform.ErrNotFound
		}
		if isUniqueViolation(err) {
			return platform.Project{}, platform.ErrConflict
		}
		return platform.Project{}, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}

func (r *Repository) ListMembers(ctx context.Context, organizationID string) ([]platform.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.organization_id, m.email, m.display_name, m.status, m.identity_provider,
		       COALESCE(array_agg(DISTINCT rb.role_key) FILTER (WHERE rb.role_key IS NOT NULL), '{}'),
		       m.last_active_at, m.created_at
		FROM members m
		LEFT JOIN role_bindings rb ON rb.member_id = m.id
		WHERE m.organization_id = $1 AND m.deleted_at IS NULL
		GROUP BY m.id
		ORDER BY m.created_at DESC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Member, 0)
	for rows.Next() {
		var item platform.Member
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Email, &item.DisplayName, &item.Status, &item.IdentityProvider, &item.Roles, &item.LastActiveAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateMember(ctx context.Context, member platform.Member, role, invitedBy string) (platform.Member, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.Member{}, fmt.Errorf("begin member transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO members(id, organization_id, email, display_name, status, identity_provider, last_active_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, member.ID, member.OrganizationID, member.Email, member.DisplayName, member.Status, member.IdentityProvider, member.LastActiveAt, member.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.Member{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.Member{}, platform.ErrConflict
	}
	if err != nil {
		return platform.Member{}, fmt.Errorf("create member: %w", err)
	}
	bindingID := "binding_" + member.ID + "_" + role
	_, err = tx.Exec(ctx, `
		INSERT INTO role_bindings(id, organization_id, member_id, role_key, scope_type, scope_id, created_by)
		VALUES ($1, $2, $3, $4, 'organization', $2, $5)`, bindingID, member.OrganizationID, member.ID, role, invitedBy)
	if isForeignKeyViolation(err) {
		return platform.Member{}, &platform.ValidationError{Code: "member_role_invalid", Message: "The selected member role does not exist."}
	}
	if err != nil {
		return platform.Member{}, fmt.Errorf("create role binding: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.Member{}, fmt.Errorf("commit member: %w", err)
	}
	return member, nil
}

func (r *Repository) ListModels(ctx context.Context, filter platform.ModelFilter) ([]platform.Model, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider, display_name, status, context_window, max_output_tokens,
		       input_price_per_million, output_price_per_million,
		       supports_tools, supports_vision, supports_json, regions, created_at
		FROM models
		WHERE ($1 = '' OR $1 = 'all' OR provider = $1)
		  AND ($2 = '' OR $2 = 'all' OR status = $2)
		  AND ($3 = '' OR lower(id) LIKE '%' || lower($3) || '%'
		       OR lower(display_name) LIKE '%' || lower($3) || '%'
		       OR lower(provider) LIKE '%' || lower($3) || '%')
		ORDER BY provider, display_name`, filter.Provider, filter.Status, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Model, 0)
	for rows.Next() {
		var item platform.Model
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.Status, &item.ContextWindow, &item.MaxOutputTokens, &item.InputPricePerMillion, &item.OutputPricePerMillion, &item.SupportsTools, &item.SupportsVision, &item.SupportsJSON, &item.Regions, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpsertModel(ctx context.Context, model platform.Model) (platform.Model, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO models(id, provider, display_name, status, context_window, max_output_tokens,
		  input_price_per_million, output_price_per_million, supports_tools, supports_vision, supports_json, regions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
		  provider = EXCLUDED.provider, display_name = EXCLUDED.display_name, status = EXCLUDED.status,
		  context_window = EXCLUDED.context_window, max_output_tokens = EXCLUDED.max_output_tokens,
		  input_price_per_million = EXCLUDED.input_price_per_million,
		  output_price_per_million = EXCLUDED.output_price_per_million,
		  supports_tools = EXCLUDED.supports_tools, supports_vision = EXCLUDED.supports_vision,
		  supports_json = EXCLUDED.supports_json, regions = EXCLUDED.regions
		RETURNING created_at`, model.ID, model.Provider, model.DisplayName, model.Status, model.ContextWindow,
		model.MaxOutputTokens, model.InputPricePerMillion, model.OutputPricePerMillion, model.SupportsTools,
		model.SupportsVision, model.SupportsJSON, model.Regions, model.CreatedAt)
	if err := row.Scan(&model.CreatedAt); err != nil {
		return platform.Model{}, fmt.Errorf("upsert model: %w", err)
	}
	return model, nil
}

var _ platform.FoundationRepository = (*Repository)(nil)
