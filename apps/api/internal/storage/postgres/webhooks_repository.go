package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListWebhookEndpoints(ctx context.Context, filter platform.WebhookFilter) ([]platform.WebhookEndpoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.id,w.organization_id,w.name,w.status,w.destination,w.version,w.events,
		       w.sample_rate,w.include_data,w.property_filters,w.signing_secret_prefix,
		       w.max_attempts,w.timeout_seconds,
		       (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status='succeeded'),
		       (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status IN ('failed','dead_letter')),
		       (SELECT max(d.delivered_at) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status='succeeded'),
		       w.created_at,w.updated_at,w.signing_secret_digest,w.secret_reference
		FROM webhook_endpoints w
		WHERE w.organization_id=$1 AND w.deleted_at IS NULL
		  AND ($2='' OR $2='all' OR w.status=$2)
		  AND ($3='' OR $3='all' OR $3=ANY(w.events))
		  AND ($4='' OR lower(w.name) LIKE '%'||lower($4)||'%' OR lower(w.destination) LIKE '%'||lower($4)||'%' OR EXISTS (SELECT 1 FROM unnest(w.events) event WHERE lower(event) LIKE '%'||lower($4)||'%'))
		ORDER BY w.created_at DESC`, filter.OrganizationID, filter.Status, filter.Event, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list webhook endpoints: %w", err)
	}
	defer rows.Close()
	items := make([]platform.WebhookEndpoint, 0)
	for rows.Next() {
		endpoint, err := scanWebhookEndpoint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, endpoint)
	}
	return items, rows.Err()
}

func (r *Repository) GetWebhookEndpoint(ctx context.Context, organizationID, id string) (platform.WebhookEndpoint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT w.id,w.organization_id,w.name,w.status,w.destination,w.version,w.events,
		       w.sample_rate,w.include_data,w.property_filters,w.signing_secret_prefix,
		       w.max_attempts,w.timeout_seconds,
		       (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status='succeeded'),
		       (SELECT count(*) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status IN ('failed','dead_letter')),
		       (SELECT max(d.delivered_at) FROM webhook_deliveries d WHERE d.webhook_id=w.id AND d.status='succeeded'),
		       w.created_at,w.updated_at,w.signing_secret_digest,w.secret_reference
		FROM webhook_endpoints w WHERE w.organization_id=$1 AND w.id=$2 AND w.deleted_at IS NULL`, organizationID, id)
	endpoint, err := scanWebhookEndpoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.WebhookEndpoint{}, platform.ErrNotFound
	}
	return endpoint, err
}

func (r *Repository) CreateWebhookEndpoint(ctx context.Context, endpoint platform.WebhookEndpoint) (platform.WebhookEndpoint, error) {
	filters, err := json.Marshal(endpoint.PropertyFilters)
	if err != nil {
		return platform.WebhookEndpoint{}, fmt.Errorf("marshal webhook filters: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO webhook_endpoints(
			id,organization_id,name,status,destination,version,events,sample_rate,include_data,
			property_filters,signing_secret_prefix,signing_secret_digest,secret_reference,
			max_attempts,timeout_seconds,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		endpoint.ID, endpoint.OrganizationID, endpoint.Name, endpoint.Status, endpoint.Destination,
		endpoint.Version, endpoint.Events, endpoint.SampleRate, endpoint.IncludeData, filters,
		endpoint.SigningSecretPrefix, endpoint.SigningSecretDigest[:], endpoint.SecretReference,
		endpoint.MaxAttempts, endpoint.TimeoutSeconds, endpoint.CreatedAt, endpoint.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.WebhookEndpoint{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.WebhookEndpoint{}, platform.ErrConflict
	}
	if err != nil {
		return platform.WebhookEndpoint{}, fmt.Errorf("create webhook endpoint: %w", err)
	}
	return endpoint, nil
}

func (r *Repository) UpdateWebhookEndpointStatus(ctx context.Context, organizationID, id, status string, updatedAt time.Time) (platform.WebhookEndpoint, error) {
	command, err := r.pool.Exec(ctx, `UPDATE webhook_endpoints SET status=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id, status, updatedAt)
	if err != nil {
		return platform.WebhookEndpoint{}, fmt.Errorf("update webhook status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.WebhookEndpoint{}, platform.ErrNotFound
	}
	return r.GetWebhookEndpoint(ctx, organizationID, id)
}

func (r *Repository) ListWebhookDeliveries(ctx context.Context, filter platform.WebhookDeliveryFilter) ([]platform.WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id,d.organization_id,d.webhook_id,w.name,d.event_id,d.event_type,d.status,d.trigger_type,
		       d.attempt,d.max_attempts,d.response_status,d.duration_ms,d.error_message,d.next_retry_at,
		       d.delivered_at,d.replay_of_id,d.created_at
		FROM webhook_deliveries d JOIN webhook_endpoints w ON w.id=d.webhook_id
		WHERE d.organization_id=$1
		  AND ($2='' OR d.webhook_id=$2)
		  AND ($3='' OR $3='all' OR d.status=$3)
		  AND ($4='' OR $4='all' OR d.event_type=$4)
		ORDER BY d.created_at DESC`, filter.OrganizationID, filter.WebhookID, filter.Status, filter.EventType)
	if err != nil {
		return nil, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()
	items := make([]platform.WebhookDelivery, 0)
	for rows.Next() {
		delivery, err := scanWebhookDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, delivery)
	}
	return items, rows.Err()
}

func (r *Repository) GetWebhookDelivery(ctx context.Context, organizationID, id string) (platform.WebhookDelivery, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT d.id,d.organization_id,d.webhook_id,w.name,d.event_id,d.event_type,d.status,d.trigger_type,
		       d.attempt,d.max_attempts,d.response_status,d.duration_ms,d.error_message,d.next_retry_at,
		       d.delivered_at,d.replay_of_id,d.created_at
		FROM webhook_deliveries d JOIN webhook_endpoints w ON w.id=d.webhook_id
		WHERE d.organization_id=$1 AND d.id=$2`, organizationID, id)
	delivery, err := scanWebhookDelivery(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.WebhookDelivery{}, platform.ErrNotFound
	}
	return delivery, err
}

func (r *Repository) CreateWebhookDelivery(ctx context.Context, delivery platform.WebhookDelivery) (platform.WebhookDelivery, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO webhook_deliveries(
			id,organization_id,webhook_id,event_id,event_type,status,trigger_type,attempt,max_attempts,
			response_status,duration_ms,error_message,next_retry_at,delivered_at,replay_of_id,created_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		delivery.ID, delivery.OrganizationID, delivery.WebhookID, delivery.EventID, delivery.EventType,
		delivery.Status, delivery.Trigger, delivery.Attempt, delivery.MaxAttempts, delivery.ResponseStatus,
		delivery.DurationMS, delivery.ErrorMessage, delivery.NextRetryAt, delivery.DeliveredAt,
		delivery.ReplayOfID, delivery.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.WebhookDelivery{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.WebhookDelivery{}, platform.ErrConflict
	}
	if err != nil {
		return platform.WebhookDelivery{}, fmt.Errorf("create webhook delivery: %w", err)
	}
	return delivery, nil
}

func scanWebhookEndpoint(row rowScanner) (platform.WebhookEndpoint, error) {
	var endpoint platform.WebhookEndpoint
	var filters []byte
	var digest []byte
	err := row.Scan(
		&endpoint.ID, &endpoint.OrganizationID, &endpoint.Name, &endpoint.Status, &endpoint.Destination,
		&endpoint.Version, &endpoint.Events, &endpoint.SampleRate, &endpoint.IncludeData, &filters,
		&endpoint.SigningSecretPrefix, &endpoint.MaxAttempts, &endpoint.TimeoutSeconds,
		&endpoint.SuccessCount, &endpoint.FailureCount, &endpoint.LastDeliveredAt,
		&endpoint.CreatedAt, &endpoint.UpdatedAt, &digest, &endpoint.SecretReference,
	)
	if err != nil {
		return platform.WebhookEndpoint{}, err
	}
	if len(digest) != len(endpoint.SigningSecretDigest) {
		return platform.WebhookEndpoint{}, fmt.Errorf("webhook signing digest has invalid length")
	}
	copy(endpoint.SigningSecretDigest[:], digest)
	if err := json.Unmarshal(filters, &endpoint.PropertyFilters); err != nil {
		return platform.WebhookEndpoint{}, fmt.Errorf("decode webhook filters: %w", err)
	}
	return endpoint, nil
}

func scanWebhookDelivery(row rowScanner) (platform.WebhookDelivery, error) {
	var delivery platform.WebhookDelivery
	err := row.Scan(
		&delivery.ID, &delivery.OrganizationID, &delivery.WebhookID, &delivery.WebhookName,
		&delivery.EventID, &delivery.EventType, &delivery.Status, &delivery.Trigger,
		&delivery.Attempt, &delivery.MaxAttempts, &delivery.ResponseStatus, &delivery.DurationMS,
		&delivery.ErrorMessage, &delivery.NextRetryAt, &delivery.DeliveredAt,
		&delivery.ReplayOfID, &delivery.CreatedAt,
	)
	return delivery, err
}

var _ platform.WebhookRepository = (*Repository)(nil)
