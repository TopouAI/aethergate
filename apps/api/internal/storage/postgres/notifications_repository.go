package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

func (r *Repository) ListNotifications(ctx context.Context, filter platform.NotificationFilter) ([]platform.Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id,organization_id,recipient_id,category,severity,title,body,source_type,source_id,
		       action_url,status,read_at,created_at,updated_at
		FROM notifications
		WHERE organization_id=$1 AND recipient_id=$2
		  AND ($3='' OR $3='all' OR status=$3)
		  AND ($4='' OR $4='all' OR category=$4)
		  AND ($5='' OR $5='all' OR severity=$5)
		  AND ($6='' OR lower(title) LIKE '%'||lower($6)||'%' OR lower(body) LIKE '%'||lower($6)||'%'
		       OR lower(source_type) LIKE '%'||lower($6)||'%' OR lower(source_id) LIKE '%'||lower($6)||'%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.RecipientID, filter.Status, filter.Category, filter.Severity, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	items := make([]platform.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, notification)
	}
	return items, rows.Err()
}

func (r *Repository) GetNotification(ctx context.Context, organizationID, recipientID, id string) (platform.Notification, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id,organization_id,recipient_id,category,severity,title,body,source_type,source_id,
		       action_url,status,read_at,created_at,updated_at
		FROM notifications WHERE organization_id=$1 AND recipient_id=$2 AND id=$3`, organizationID, recipientID, id)
	notification, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.Notification{}, platform.ErrNotFound
	}
	return notification, err
}

func (r *Repository) CreateNotificationDispatch(ctx context.Context, notification platform.Notification, deliveries []platform.NotificationDelivery) (platform.NotificationDispatch, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.NotificationDispatch{}, fmt.Errorf("begin notification dispatch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications(id,organization_id,recipient_id,category,severity,title,body,source_type,source_id,action_url,status,read_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		notification.ID, notification.OrganizationID, notification.RecipientID, notification.Category,
		notification.Severity, notification.Title, notification.Body, notification.SourceType,
		notification.SourceID, notification.ActionURL, notification.Status, notification.ReadAt,
		notification.CreatedAt, notification.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.NotificationDispatch{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.NotificationDispatch{}, platform.ErrConflict
	}
	if err != nil {
		return platform.NotificationDispatch{}, fmt.Errorf("insert notification: %w", err)
	}
	for _, delivery := range deliveries {
		if err := insertNotificationDelivery(ctx, tx, delivery); err != nil {
			return platform.NotificationDispatch{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.NotificationDispatch{}, fmt.Errorf("commit notification dispatch: %w", err)
	}
	return platform.NotificationDispatch{Notification: notification, Deliveries: deliveries}, nil
}

func (r *Repository) UpdateNotificationStatus(ctx context.Context, organizationID, recipientID, id, status string, readAt *time.Time, updatedAt time.Time) (platform.Notification, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE notifications SET status=$4,read_at=$5,updated_at=$6
		WHERE organization_id=$1 AND recipient_id=$2 AND id=$3`, organizationID, recipientID, id, status, readAt, updatedAt)
	if err != nil {
		return platform.Notification{}, fmt.Errorf("update notification status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.Notification{}, platform.ErrNotFound
	}
	return r.GetNotification(ctx, organizationID, recipientID, id)
}

func (r *Repository) MarkAllNotificationsRead(ctx context.Context, organizationID, recipientID string, readAt time.Time) (int64, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE notifications SET status='read',read_at=$3,updated_at=$3
		WHERE organization_id=$1 AND recipient_id=$2 AND status='unread'`, organizationID, recipientID, readAt)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return command.RowsAffected(), nil
}

func (r *Repository) GetNotificationPreference(ctx context.Context, organizationID, recipientID string) (platform.NotificationPreference, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT organization_id,recipient_id,destinations,category_channels,digest_frequency,minimum_severity,
		       timezone,quiet_hours_enabled,quiet_start,quiet_end,updated_at
		FROM notification_preferences WHERE organization_id=$1 AND recipient_id=$2`, organizationID, recipientID)
	preference, err := scanNotificationPreference(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.NotificationPreference{}, platform.ErrNotFound
	}
	return preference, err
}

func (r *Repository) UpsertNotificationPreference(ctx context.Context, preference platform.NotificationPreference) (platform.NotificationPreference, error) {
	destinations, err := json.Marshal(preference.Destinations)
	if err != nil {
		return platform.NotificationPreference{}, fmt.Errorf("marshal notification destinations: %w", err)
	}
	routes, err := json.Marshal(preference.CategoryChannels)
	if err != nil {
		return platform.NotificationPreference{}, fmt.Errorf("marshal notification category channels: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO notification_preferences(organization_id,recipient_id,destinations,category_channels,digest_frequency,minimum_severity,timezone,quiet_hours_enabled,quiet_start,quiet_end,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(organization_id,recipient_id) DO UPDATE SET
			destinations=EXCLUDED.destinations,category_channels=EXCLUDED.category_channels,
			digest_frequency=EXCLUDED.digest_frequency,minimum_severity=EXCLUDED.minimum_severity,
			timezone=EXCLUDED.timezone,quiet_hours_enabled=EXCLUDED.quiet_hours_enabled,
			quiet_start=EXCLUDED.quiet_start,quiet_end=EXCLUDED.quiet_end,updated_at=EXCLUDED.updated_at`,
		preference.OrganizationID, preference.RecipientID, destinations, routes, preference.DigestFrequency,
		preference.MinimumSeverity, preference.Timezone, preference.QuietHoursEnabled,
		preference.QuietStart, preference.QuietEnd, preference.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.NotificationPreference{}, platform.ErrNotFound
	}
	if err != nil {
		return platform.NotificationPreference{}, fmt.Errorf("upsert notification preference: %w", err)
	}
	return preference, nil
}

func (r *Repository) ListNotificationEscalationPolicies(ctx context.Context, filter platform.NotificationPolicyFilter) ([]platform.NotificationEscalationPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id,organization_id,name,status,categories,minimum_severity,acknowledge_within_minutes,
		       repeat_every_minutes,max_escalations,routes,created_at,updated_at
		FROM notification_escalation_policies
		WHERE organization_id=$1 AND deleted_at IS NULL
		  AND ($2='' OR $2='all' OR status=$2)
		  AND ($3='' OR $3='all' OR $3=ANY(categories))
		  AND ($4='' OR lower(name) LIKE '%'||lower($4)||'%' OR lower(minimum_severity) LIKE '%'||lower($4)||'%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Status, filter.Category, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list notification escalation policies: %w", err)
	}
	defer rows.Close()
	items := make([]platform.NotificationEscalationPolicy, 0)
	for rows.Next() {
		policy, err := scanNotificationEscalationPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, policy)
	}
	return items, rows.Err()
}

func (r *Repository) GetNotificationEscalationPolicy(ctx context.Context, organizationID, id string) (platform.NotificationEscalationPolicy, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id,organization_id,name,status,categories,minimum_severity,acknowledge_within_minutes,
		       repeat_every_minutes,max_escalations,routes,created_at,updated_at
		FROM notification_escalation_policies
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id)
	policy, err := scanNotificationEscalationPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.NotificationEscalationPolicy{}, platform.ErrNotFound
	}
	return policy, err
}

func (r *Repository) CreateNotificationEscalationPolicy(ctx context.Context, policy platform.NotificationEscalationPolicy) (platform.NotificationEscalationPolicy, error) {
	routes, err := json.Marshal(policy.Routes)
	if err != nil {
		return platform.NotificationEscalationPolicy{}, fmt.Errorf("marshal notification escalation routes: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO notification_escalation_policies(id,organization_id,name,status,categories,minimum_severity,
			acknowledge_within_minutes,repeat_every_minutes,max_escalations,routes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, policy.ID, policy.OrganizationID, policy.Name,
		policy.Status, policy.Categories, policy.MinimumSeverity, policy.AcknowledgeWithinMinutes,
		policy.RepeatEveryMinutes, policy.MaxEscalations, routes, policy.CreatedAt, policy.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.NotificationEscalationPolicy{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.NotificationEscalationPolicy{}, platform.ErrConflict
	}
	if err != nil {
		return platform.NotificationEscalationPolicy{}, fmt.Errorf("create notification escalation policy: %w", err)
	}
	return policy, nil
}

func (r *Repository) UpdateNotificationEscalationPolicyStatus(ctx context.Context, organizationID, id, status string, updatedAt time.Time) (platform.NotificationEscalationPolicy, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE notification_escalation_policies SET status=$3,updated_at=$4
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id, status, updatedAt)
	if err != nil {
		return platform.NotificationEscalationPolicy{}, fmt.Errorf("update notification escalation policy: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.NotificationEscalationPolicy{}, platform.ErrNotFound
	}
	return r.GetNotificationEscalationPolicy(ctx, organizationID, id)
}

func (r *Repository) ListNotificationDeliveries(ctx context.Context, filter platform.NotificationDeliveryFilter) ([]platform.NotificationDelivery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id,d.organization_id,d.notification_id,n.title,n.recipient_id,d.channel,d.target,d.display_name,
		       d.status,d.attempt,d.available_at,d.delivered_at,d.error_message,d.parent_delivery_id,d.created_at
		FROM notification_deliveries d JOIN notifications n ON n.id=d.notification_id
		WHERE d.organization_id=$1 AND n.recipient_id=$2
		  AND ($3='' OR d.notification_id=$3)
		  AND ($4='' OR $4='all' OR d.status=$4)
		  AND ($5='' OR $5='all' OR d.channel=$5)
		ORDER BY d.created_at DESC`, filter.OrganizationID, filter.RecipientID, filter.NotificationID, filter.Status, filter.Channel)
	if err != nil {
		return nil, fmt.Errorf("list notification deliveries: %w", err)
	}
	defer rows.Close()
	items := make([]platform.NotificationDelivery, 0)
	for rows.Next() {
		delivery, err := scanNotificationDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, delivery)
	}
	return items, rows.Err()
}

func (r *Repository) GetNotificationDelivery(ctx context.Context, organizationID, id string) (platform.NotificationDelivery, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT d.id,d.organization_id,d.notification_id,n.title,n.recipient_id,d.channel,d.target,d.display_name,
		       d.status,d.attempt,d.available_at,d.delivered_at,d.error_message,d.parent_delivery_id,d.created_at
		FROM notification_deliveries d JOIN notifications n ON n.id=d.notification_id
		WHERE d.organization_id=$1 AND d.id=$2`, organizationID, id)
	delivery, err := scanNotificationDelivery(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.NotificationDelivery{}, platform.ErrNotFound
	}
	return delivery, err
}

func (r *Repository) CreateNotificationDelivery(ctx context.Context, delivery platform.NotificationDelivery) (platform.NotificationDelivery, error) {
	if err := insertNotificationDelivery(ctx, r.pool, delivery); err != nil {
		return platform.NotificationDelivery{}, err
	}
	return delivery, nil
}

type notificationDeliveryExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertNotificationDelivery(ctx context.Context, execer notificationDeliveryExecer, delivery platform.NotificationDelivery) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO notification_deliveries(id,organization_id,notification_id,channel,target,display_name,status,
			attempt,available_at,delivered_at,error_message,parent_delivery_id,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, delivery.ID, delivery.OrganizationID,
		delivery.NotificationID, delivery.Channel, delivery.Target, delivery.DisplayName, delivery.Status,
		delivery.Attempt, delivery.AvailableAt, delivery.DeliveredAt, delivery.ErrorMessage, delivery.ParentID, delivery.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insert notification delivery: %w", err)
	}
	return nil
}

func scanNotification(row rowScanner) (platform.Notification, error) {
	var notification platform.Notification
	err := row.Scan(&notification.ID, &notification.OrganizationID, &notification.RecipientID, &notification.Category,
		&notification.Severity, &notification.Title, &notification.Body, &notification.SourceType,
		&notification.SourceID, &notification.ActionURL, &notification.Status, &notification.ReadAt,
		&notification.CreatedAt, &notification.UpdatedAt)
	return notification, err
}

func scanNotificationPreference(row rowScanner) (platform.NotificationPreference, error) {
	var preference platform.NotificationPreference
	var destinations, routes []byte
	err := row.Scan(&preference.OrganizationID, &preference.RecipientID, &destinations, &routes,
		&preference.DigestFrequency, &preference.MinimumSeverity, &preference.Timezone,
		&preference.QuietHoursEnabled, &preference.QuietStart, &preference.QuietEnd, &preference.UpdatedAt)
	if err != nil {
		return platform.NotificationPreference{}, err
	}
	if err := json.Unmarshal(destinations, &preference.Destinations); err != nil {
		return platform.NotificationPreference{}, fmt.Errorf("decode notification destinations: %w", err)
	}
	if err := json.Unmarshal(routes, &preference.CategoryChannels); err != nil {
		return platform.NotificationPreference{}, fmt.Errorf("decode notification category channels: %w", err)
	}
	return preference, nil
}

func scanNotificationEscalationPolicy(row rowScanner) (platform.NotificationEscalationPolicy, error) {
	var policy platform.NotificationEscalationPolicy
	var routes []byte
	err := row.Scan(&policy.ID, &policy.OrganizationID, &policy.Name, &policy.Status, &policy.Categories,
		&policy.MinimumSeverity, &policy.AcknowledgeWithinMinutes, &policy.RepeatEveryMinutes,
		&policy.MaxEscalations, &routes, &policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return platform.NotificationEscalationPolicy{}, err
	}
	if err := json.Unmarshal(routes, &policy.Routes); err != nil {
		return platform.NotificationEscalationPolicy{}, fmt.Errorf("decode notification escalation routes: %w", err)
	}
	return policy, nil
}

func scanNotificationDelivery(row rowScanner) (platform.NotificationDelivery, error) {
	var delivery platform.NotificationDelivery
	err := row.Scan(&delivery.ID, &delivery.OrganizationID, &delivery.NotificationID, &delivery.Notification,
		&delivery.RecipientID, &delivery.Channel, &delivery.Target, &delivery.DisplayName, &delivery.Status,
		&delivery.Attempt, &delivery.AvailableAt, &delivery.DeliveredAt, &delivery.ErrorMessage,
		&delivery.ParentID, &delivery.CreatedAt)
	return delivery, err
}

var _ platform.NotificationRepository = (*Repository)(nil)
