package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) GetProvider(ctx context.Context, organizationID, id string) (platform.ProviderConnection, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id,organization_id,name,provider,base_url,status,credential_state,model_count,
		       p95_latency_ms,success_rate,last_checked_at,created_at,routing_eligible,health_source,
		       health_reason,error_rate,request_count_24h,average_latency_ms,consecutive_failures,
		       last_transition_at,maintenance_until,maintenance_reason
		FROM provider_connections WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id)
	provider, err := scanProviderConnection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ProviderConnection{}, platform.ErrNotFound
	}
	return provider, err
}

func (r *Repository) ListProviderHealthEvents(ctx context.Context, filter platform.ProviderHealthFilter) ([]platform.ProviderHealthEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id,e.organization_id,e.provider_id,p.name,e.probe_id,e.source,e.previous_status,e.status,
		       e.is_transition,e.success,e.routing_eligible,e.request_count,e.error_count,e.error_rate,
		       e.average_latency_ms,e.p95_latency_ms,e.http_status,e.consecutive_failures,e.reason,e.observed_at
		FROM provider_health_events e JOIN provider_connections p ON p.id=e.provider_id
		WHERE e.organization_id=$1
		  AND ($2='' OR e.provider_id=$2)
		  AND ($3='' OR $3='all' OR e.status=$3)
		  AND ($4='' OR $4='all' OR e.source=$4)
		ORDER BY e.observed_at DESC`, filter.OrganizationID, filter.ProviderID, filter.Status, filter.Source)
	if err != nil {
		return nil, fmt.Errorf("list provider health events: %w", err)
	}
	defer rows.Close()
	items := make([]platform.ProviderHealthEvent, 0)
	for rows.Next() {
		var event platform.ProviderHealthEvent
		if err := rows.Scan(&event.ID, &event.OrganizationID, &event.ProviderID, &event.ProviderName, &event.ProbeID,
			&event.Source, &event.PreviousStatus, &event.Status, &event.Transition, &event.Success,
			&event.RoutingEligible, &event.RequestCount, &event.ErrorCount, &event.ErrorRate,
			&event.AverageLatencyMS, &event.P95LatencyMS, &event.HTTPStatus, &event.ConsecutiveFailures,
			&event.Reason, &event.ObservedAt); err != nil {
			return nil, fmt.Errorf("scan provider health event: %w", err)
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

func (r *Repository) ListProviderHealthProbes(ctx context.Context, filter platform.ProviderHealthProbeFilter) ([]platform.ProviderHealthProbe, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT q.id,q.organization_id,q.provider_id,p.name,q.status,q.region,q.model,q.requested_by,
		       q.requested_at,q.started_at,q.completed_at,q.event_id,q.error_message
		FROM provider_health_probes q JOIN provider_connections p ON p.id=q.provider_id
		WHERE q.organization_id=$1 AND ($2='' OR q.provider_id=$2) AND ($3='' OR $3='all' OR q.status=$3)
		ORDER BY q.requested_at DESC`, filter.OrganizationID, filter.ProviderID, filter.Status)
	if err != nil {
		return nil, fmt.Errorf("list provider health probes: %w", err)
	}
	defer rows.Close()
	items := make([]platform.ProviderHealthProbe, 0)
	for rows.Next() {
		var probe platform.ProviderHealthProbe
		if err := rows.Scan(&probe.ID, &probe.OrganizationID, &probe.ProviderID, &probe.ProviderName,
			&probe.Status, &probe.Region, &probe.Model, &probe.RequestedBy, &probe.RequestedAt,
			&probe.StartedAt, &probe.CompletedAt, &probe.EventID, &probe.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan provider health probe: %w", err)
		}
		items = append(items, probe)
	}
	return items, rows.Err()
}

func (r *Repository) CreateProviderHealthProbe(ctx context.Context, probe platform.ProviderHealthProbe) (platform.ProviderHealthProbe, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO provider_health_probes(id,organization_id,provider_id,status,region,model,requested_by,requested_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, probe.ID, probe.OrganizationID, probe.ProviderID,
		probe.Status, probe.Region, probe.Model, probe.RequestedBy, probe.RequestedAt)
	if isForeignKeyViolation(err) {
		return platform.ProviderHealthProbe{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.ProviderHealthProbe{}, fmt.Errorf("create provider health probe: %w", err)
	}
	return probe, nil
}

func (r *Repository) RecordProviderHealth(ctx context.Context, provider platform.ProviderConnection, event platform.ProviderHealthEvent) (platform.ProviderHealthEvent, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.ProviderHealthEvent{}, fmt.Errorf("begin provider health transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE provider_connections SET status=$3,routing_eligible=$4,health_source=$5,health_reason=$6,
		       error_rate=$7,request_count_24h=$8,average_latency_ms=$9,p95_latency_ms=$10,
		       success_rate=$11,consecutive_failures=$12,last_checked_at=$13,last_transition_at=$14
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, provider.OrganizationID, provider.ID,
		provider.Status, provider.RoutingEligible, provider.HealthSource, provider.HealthReason,
		provider.ErrorRate, provider.RequestCount24H, provider.AverageLatencyMS, provider.P95LatencyMS,
		provider.SuccessRate, provider.ConsecutiveFailures, provider.LastCheckedAt, provider.LastTransitionAt)
	if err != nil {
		return platform.ProviderHealthEvent{}, fmt.Errorf("update provider health: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.ProviderHealthEvent{}, platform.ErrNotFound
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_health_events(
			id,organization_id,provider_id,probe_id,source,previous_status,status,is_transition,success,
			routing_eligible,request_count,error_count,error_rate,average_latency_ms,p95_latency_ms,
			http_status,consecutive_failures,reason,observed_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		event.ID, event.OrganizationID, event.ProviderID, event.ProbeID, event.Source, event.PreviousStatus,
		event.Status, event.Transition, event.Success, event.RoutingEligible, event.RequestCount,
		event.ErrorCount, event.ErrorRate, event.AverageLatencyMS, event.P95LatencyMS, event.HTTPStatus,
		event.ConsecutiveFailures, event.Reason, event.ObservedAt)
	if isForeignKeyViolation(err) {
		return platform.ProviderHealthEvent{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.ProviderHealthEvent{}, fmt.Errorf("insert provider health event: %w", err)
	}
	if event.ProbeID != nil {
		probeStatus := "succeeded"
		errorMessage := ""
		if !event.Success {
			probeStatus = "failed"
			errorMessage = event.Reason
		}
		command, err = tx.Exec(ctx, `
			UPDATE provider_health_probes SET status=$3,completed_at=$4,event_id=$5,error_message=$6
			WHERE organization_id=$1 AND id=$2 AND provider_id=$7`, event.OrganizationID, *event.ProbeID,
			probeStatus, event.ObservedAt, event.ID, errorMessage, event.ProviderID)
		if err != nil {
			return platform.ProviderHealthEvent{}, fmt.Errorf("complete provider probe: %w", err)
		}
		if command.RowsAffected() == 0 {
			return platform.ProviderHealthEvent{}, platform.ErrNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.ProviderHealthEvent{}, fmt.Errorf("commit provider health: %w", err)
	}
	return event, nil
}

func (r *Repository) UpdateProviderMaintenance(ctx context.Context, provider platform.ProviderConnection) (platform.ProviderConnection, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE provider_connections SET status=$3,routing_eligible=$4,health_source=$5,health_reason=$6,
		       maintenance_until=$7,maintenance_reason=$8,last_transition_at=$9
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, provider.OrganizationID, provider.ID,
		provider.Status, provider.RoutingEligible, provider.HealthSource, provider.HealthReason,
		provider.MaintenanceUntil, provider.MaintenanceReason, provider.LastTransitionAt)
	if err != nil {
		return platform.ProviderConnection{}, fmt.Errorf("update provider maintenance: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.ProviderConnection{}, platform.ErrNotFound
	}
	return provider, nil
}

func scanProviderConnection(row rowScanner) (platform.ProviderConnection, error) {
	var provider platform.ProviderConnection
	err := row.Scan(&provider.ID, &provider.OrganizationID, &provider.Name, &provider.Provider,
		&provider.BaseURL, &provider.Status, &provider.CredentialState, &provider.Models,
		&provider.P95LatencyMS, &provider.SuccessRate, &provider.LastCheckedAt, &provider.CreatedAt,
		&provider.RoutingEligible, &provider.HealthSource, &provider.HealthReason, &provider.ErrorRate,
		&provider.RequestCount24H, &provider.AverageLatencyMS, &provider.ConsecutiveFailures,
		&provider.LastTransitionAt, &provider.MaintenanceUntil, &provider.MaintenanceReason)
	return provider, err
}

var _ platform.ProviderHealthRepository = (*Repository)(nil)
