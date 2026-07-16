package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

const auditSelect = `id,organization_id,actor_id,actor_email,action,resource_type,resource_id,outcome,risk_level,source,
	COALESCE(reason,''),COALESCE(request_id,''),COALESCE(ip_address::text,''),COALESCE(user_agent,''),
	COALESCE(before_state,'{}'::jsonb),COALESCE(after_state,'{}'::jsonb),previous_hash,integrity_hash,created_at`

func (r *Repository) ListAuditEvents(ctx context.Context, filter platform.AuditFilter) ([]platform.AuditEvent, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+auditSelect+`
		FROM audit_events
		WHERE organization_id=$1
		  AND ($2='' OR lower(actor_email) LIKE '%'||lower($2)||'%' OR lower(actor_id) LIKE '%'||lower($2)||'%')
		  AND ($3='' OR $3='all' OR action=$3)
		  AND ($4='' OR $4='all' OR resource_type=$4)
		  AND ($5='' OR resource_id=$5)
		  AND ($6='' OR $6='all' OR outcome=$6)
		  AND ($7='' OR $7='all' OR risk_level=$7)
		  AND ($8::timestamptz IS NULL OR created_at >= $8)
		  AND ($9::timestamptz IS NULL OR created_at <= $9)
		  AND ($10='' OR lower(actor_email||' '||action||' '||resource_type||' '||resource_id||' '||COALESCE(reason,'')||' '||COALESCE(request_id,'')) LIKE '%'||lower($10)||'%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Actor, filter.Action, filter.ResourceType,
		filter.ResourceID, filter.Outcome, filter.RiskLevel, filter.StartAt, filter.EndAt, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	items := make([]platform.AuditEvent, 0)
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

func (r *Repository) LatestAuditHash(ctx context.Context, organizationID string) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT integrity_hash FROM audit_events WHERE organization_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, organizationID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", platform.ErrNotFound
	}
	return hash, err
}

func (r *Repository) AppendAuditEvent(ctx context.Context, event platform.AuditEvent) (platform.AuditEvent, error) {
	beforeState, err := json.Marshal(event.BeforeState)
	if err != nil {
		return platform.AuditEvent{}, fmt.Errorf("marshal audit before state: %w", err)
	}
	afterState, err := json.Marshal(event.AfterState)
	if err != nil {
		return platform.AuditEvent{}, fmt.Errorf("marshal audit after state: %w", err)
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO audit_events(
		id,organization_id,actor_id,actor_email,action,resource_type,resource_id,outcome,risk_level,source,
		reason,request_id,ip_address,user_agent,before_state,after_state,previous_hash,integrity_hash,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,'')::inet,$14,$15,$16,$17,$18,$19)`,
		event.ID, event.OrganizationID, event.ActorID, event.ActorEmail, event.Action, event.ResourceType,
		event.ResourceID, event.Outcome, event.RiskLevel, event.Source, event.Reason, event.RequestID,
		event.IPAddress, event.UserAgent, beforeState, afterState, event.PreviousHash, event.IntegrityHash, event.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.AuditEvent{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.AuditEvent{}, platform.ErrConflict
	}
	if err != nil {
		return platform.AuditEvent{}, fmt.Errorf("append audit event: %w", err)
	}
	return event, nil
}

func (r *Repository) GetAuditRetentionPolicy(ctx context.Context, organizationID string) (platform.AuditRetentionPolicy, error) {
	var policy platform.AuditRetentionPolicy
	err := r.pool.QueryRow(ctx, `SELECT organization_id,retention_days,legal_hold,export_format,updated_by,updated_at FROM audit_retention_policies WHERE organization_id=$1`, organizationID).Scan(
		&policy.OrganizationID, &policy.RetentionDays, &policy.LegalHold, &policy.ExportFormat, &policy.UpdatedBy, &policy.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.AuditRetentionPolicy{}, platform.ErrNotFound
	}
	return policy, err
}

func (r *Repository) UpsertAuditRetentionPolicy(ctx context.Context, policy platform.AuditRetentionPolicy) (platform.AuditRetentionPolicy, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO audit_retention_policies(organization_id,retention_days,legal_hold,export_format,updated_by,updated_at)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(organization_id) DO UPDATE SET retention_days=EXCLUDED.retention_days,
		legal_hold=EXCLUDED.legal_hold,export_format=EXCLUDED.export_format,updated_by=EXCLUDED.updated_by,updated_at=EXCLUDED.updated_at`,
		policy.OrganizationID, policy.RetentionDays, policy.LegalHold, policy.ExportFormat, policy.UpdatedBy, policy.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.AuditRetentionPolicy{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.AuditRetentionPolicy{}, fmt.Errorf("upsert audit retention policy: %w", err)
	}
	return policy, nil
}

func (r *Repository) ListAuditExports(ctx context.Context, filter platform.AuditExportFilter) ([]platform.AuditExport, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,organization_id,requested_by,format,status,filters,period_start,period_end,row_count,size_bytes,
		object_key,checksum,error_message,parent_export_id,created_at,completed_at FROM audit_exports
		WHERE organization_id=$1 AND ($2='' OR $2='all' OR status=$2) AND ($3='' OR $3='all' OR format=$3)
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Status, filter.Format)
	if err != nil {
		return nil, fmt.Errorf("list audit exports: %w", err)
	}
	defer rows.Close()
	items := make([]platform.AuditExport, 0)
	for rows.Next() {
		export, err := scanAuditExport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, export)
	}
	return items, rows.Err()
}

func (r *Repository) GetAuditExport(ctx context.Context, organizationID, id string) (platform.AuditExport, error) {
	row := r.pool.QueryRow(ctx, `SELECT id,organization_id,requested_by,format,status,filters,period_start,period_end,row_count,size_bytes,
		object_key,checksum,error_message,parent_export_id,created_at,completed_at FROM audit_exports WHERE organization_id=$1 AND id=$2`, organizationID, id)
	export, err := scanAuditExport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.AuditExport{}, platform.ErrNotFound
	}
	return export, err
}

func (r *Repository) CreateAuditExport(ctx context.Context, export platform.AuditExport) (platform.AuditExport, error) {
	filters, err := json.Marshal(export.Filters)
	if err != nil {
		return platform.AuditExport{}, fmt.Errorf("marshal audit export filters: %w", err)
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO audit_exports(id,organization_id,requested_by,format,status,filters,period_start,period_end,
		row_count,size_bytes,object_key,checksum,error_message,parent_export_id,created_at,completed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, export.ID, export.OrganizationID,
		export.RequestedBy, export.Format, export.Status, filters, export.PeriodStart, export.PeriodEnd, export.RowCount,
		export.SizeBytes, export.ObjectKey, export.Checksum, export.ErrorMessage, export.ParentID, export.CreatedAt, export.CompletedAt)
	if isForeignKeyViolation(err) {
		return platform.AuditExport{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.AuditExport{}, platform.ErrConflict
	}
	if err != nil {
		return platform.AuditExport{}, fmt.Errorf("create audit export: %w", err)
	}
	return export, nil
}

func scanAuditEvent(row rowScanner) (platform.AuditEvent, error) {
	var event platform.AuditEvent
	var beforeState, afterState []byte
	err := row.Scan(&event.ID, &event.OrganizationID, &event.ActorID, &event.ActorEmail, &event.Action,
		&event.ResourceType, &event.ResourceID, &event.Outcome, &event.RiskLevel, &event.Source,
		&event.Reason, &event.RequestID, &event.IPAddress, &event.UserAgent, &beforeState, &afterState,
		&event.PreviousHash, &event.IntegrityHash, &event.CreatedAt)
	if err != nil {
		return platform.AuditEvent{}, err
	}
	if err := json.Unmarshal(beforeState, &event.BeforeState); err != nil {
		return platform.AuditEvent{}, fmt.Errorf("decode audit before state: %w", err)
	}
	if err := json.Unmarshal(afterState, &event.AfterState); err != nil {
		return platform.AuditEvent{}, fmt.Errorf("decode audit after state: %w", err)
	}
	return event, nil
}

func scanAuditExport(row rowScanner) (platform.AuditExport, error) {
	var export platform.AuditExport
	var filters []byte
	err := row.Scan(&export.ID, &export.OrganizationID, &export.RequestedBy, &export.Format, &export.Status,
		&filters, &export.PeriodStart, &export.PeriodEnd, &export.RowCount, &export.SizeBytes,
		&export.ObjectKey, &export.Checksum, &export.ErrorMessage, &export.ParentID, &export.CreatedAt, &export.CompletedAt)
	if err != nil {
		return platform.AuditExport{}, err
	}
	if err := json.Unmarshal(filters, &export.Filters); err != nil {
		return platform.AuditExport{}, fmt.Errorf("decode audit export filters: %w", err)
	}
	return export, nil
}

var _ platform.AuditRepository = (*Repository)(nil)
