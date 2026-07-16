package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListNotifications(_ context.Context, filter NotificationFilter) ([]Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.ToLower(filter.Query)
	items := make([]Notification, 0)
	for _, notification := range r.notifications {
		if notification.OrganizationID != filter.OrganizationID || notification.RecipientID != filter.RecipientID ||
			filter.Status != "" && filter.Status != "all" && notification.Status != filter.Status ||
			filter.Category != "" && filter.Category != "all" && notification.Category != filter.Category ||
			filter.Severity != "" && filter.Severity != "all" && notification.Severity != filter.Severity {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(notification.Title+" "+notification.Body+" "+notification.SourceType+" "+notification.SourceID), needle) {
			continue
		}
		items = append(items, notification)
	}
	return items, nil
}

func (r *MemoryRepository) GetNotification(_ context.Context, organizationID, recipientID, id string) (Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, notification := range r.notifications {
		if notification.OrganizationID == organizationID && notification.RecipientID == recipientID && notification.ID == id {
			return notification, nil
		}
	}
	return Notification{}, ErrNotFound
}

func (r *MemoryRepository) CreateNotificationDispatch(_ context.Context, notification Notification, deliveries []NotificationDelivery) (NotificationDispatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, notification.OrganizationID); !found {
		return NotificationDispatch{}, ErrNotFound
	}
	if slices.ContainsFunc(r.notifications, func(existing Notification) bool { return existing.ID == notification.ID }) {
		return NotificationDispatch{}, ErrConflict
	}
	r.notifications = append([]Notification{notification}, r.notifications...)
	r.notificationDeliveries = append(slices.Clone(deliveries), r.notificationDeliveries...)
	return NotificationDispatch{Notification: notification, Deliveries: slices.Clone(deliveries)}, nil
}

func (r *MemoryRepository) UpdateNotificationStatus(_ context.Context, organizationID, recipientID, id, status string, readAt *time.Time, updatedAt time.Time) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.notifications {
		if r.notifications[index].OrganizationID == organizationID && r.notifications[index].RecipientID == recipientID && r.notifications[index].ID == id {
			r.notifications[index].Status = status
			r.notifications[index].ReadAt = readAt
			r.notifications[index].UpdatedAt = updatedAt
			return r.notifications[index], nil
		}
	}
	return Notification{}, ErrNotFound
}

func (r *MemoryRepository) MarkAllNotificationsRead(_ context.Context, organizationID, recipientID string, readAt time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for index := range r.notifications {
		if r.notifications[index].OrganizationID == organizationID && r.notifications[index].RecipientID == recipientID && r.notifications[index].Status == "unread" {
			r.notifications[index].Status = "read"
			r.notifications[index].ReadAt = &readAt
			r.notifications[index].UpdatedAt = readAt
			count++
		}
	}
	return count, nil
}

func (r *MemoryRepository) GetNotificationPreference(_ context.Context, organizationID, recipientID string) (NotificationPreference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, preference := range r.notificationPreferences {
		if preference.OrganizationID == organizationID && preference.RecipientID == recipientID {
			return cloneNotificationPreference(preference), nil
		}
	}
	return NotificationPreference{}, ErrNotFound
}

func (r *MemoryRepository) UpsertNotificationPreference(_ context.Context, preference NotificationPreference) (NotificationPreference, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, preference.OrganizationID); !found {
		return NotificationPreference{}, ErrNotFound
	}
	for index := range r.notificationPreferences {
		if r.notificationPreferences[index].OrganizationID == preference.OrganizationID && r.notificationPreferences[index].RecipientID == preference.RecipientID {
			r.notificationPreferences[index] = cloneNotificationPreference(preference)
			return cloneNotificationPreference(preference), nil
		}
	}
	r.notificationPreferences = append([]NotificationPreference{cloneNotificationPreference(preference)}, r.notificationPreferences...)
	return cloneNotificationPreference(preference), nil
}

func (r *MemoryRepository) ListNotificationEscalationPolicies(_ context.Context, filter NotificationPolicyFilter) ([]NotificationEscalationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.ToLower(filter.Query)
	items := make([]NotificationEscalationPolicy, 0)
	for _, policy := range r.notificationEscalationPolicies {
		if policy.OrganizationID != filter.OrganizationID ||
			filter.Status != "" && filter.Status != "all" && policy.Status != filter.Status ||
			filter.Category != "" && filter.Category != "all" && !slices.Contains(policy.Categories, filter.Category) {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(policy.Name+" "+strings.Join(policy.Categories, " ")+" "+policy.MinimumSeverity), needle) {
			continue
		}
		items = append(items, cloneNotificationPolicy(policy))
	}
	return items, nil
}

func (r *MemoryRepository) GetNotificationEscalationPolicy(_ context.Context, organizationID, id string) (NotificationEscalationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.notificationEscalationPolicies {
		if policy.OrganizationID == organizationID && policy.ID == id {
			return cloneNotificationPolicy(policy), nil
		}
	}
	return NotificationEscalationPolicy{}, ErrNotFound
}

func (r *MemoryRepository) CreateNotificationEscalationPolicy(_ context.Context, policy NotificationEscalationPolicy) (NotificationEscalationPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, policy.OrganizationID); !found {
		return NotificationEscalationPolicy{}, ErrNotFound
	}
	if slices.ContainsFunc(r.notificationEscalationPolicies, func(existing NotificationEscalationPolicy) bool {
		return existing.OrganizationID == policy.OrganizationID && strings.EqualFold(existing.Name, policy.Name)
	}) {
		return NotificationEscalationPolicy{}, ErrConflict
	}
	r.notificationEscalationPolicies = append([]NotificationEscalationPolicy{cloneNotificationPolicy(policy)}, r.notificationEscalationPolicies...)
	return cloneNotificationPolicy(policy), nil
}

func (r *MemoryRepository) UpdateNotificationEscalationPolicyStatus(_ context.Context, organizationID, id, status string, updatedAt time.Time) (NotificationEscalationPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.notificationEscalationPolicies {
		if r.notificationEscalationPolicies[index].OrganizationID == organizationID && r.notificationEscalationPolicies[index].ID == id {
			r.notificationEscalationPolicies[index].Status = status
			r.notificationEscalationPolicies[index].UpdatedAt = updatedAt
			return cloneNotificationPolicy(r.notificationEscalationPolicies[index]), nil
		}
	}
	return NotificationEscalationPolicy{}, ErrNotFound
}

func (r *MemoryRepository) ListNotificationDeliveries(_ context.Context, filter NotificationDeliveryFilter) ([]NotificationDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]NotificationDelivery, 0)
	for _, delivery := range r.notificationDeliveries {
		if delivery.OrganizationID != filter.OrganizationID || delivery.RecipientID != filter.RecipientID ||
			filter.NotificationID != "" && delivery.NotificationID != filter.NotificationID ||
			filter.Status != "" && filter.Status != "all" && delivery.Status != filter.Status ||
			filter.Channel != "" && filter.Channel != "all" && delivery.Channel != filter.Channel {
			continue
		}
		items = append(items, delivery)
	}
	return items, nil
}

func (r *MemoryRepository) GetNotificationDelivery(_ context.Context, organizationID, id string) (NotificationDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, delivery := range r.notificationDeliveries {
		if delivery.OrganizationID == organizationID && delivery.ID == id {
			return delivery, nil
		}
	}
	return NotificationDelivery{}, ErrNotFound
}

func (r *MemoryRepository) CreateNotificationDelivery(_ context.Context, delivery NotificationDelivery) (NotificationDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !slices.ContainsFunc(r.notifications, func(notification Notification) bool {
		return notification.OrganizationID == delivery.OrganizationID && notification.ID == delivery.NotificationID
	}) {
		return NotificationDelivery{}, ErrNotFound
	}
	r.notificationDeliveries = append([]NotificationDelivery{delivery}, r.notificationDeliveries...)
	return delivery, nil
}

func developmentNotifications() []Notification {
	now := time.Date(2026, 7, 15, 2, 30, 0, 0, time.UTC)
	read := time.Date(2026, 7, 14, 8, 10, 0, 0, time.UTC)
	return []Notification{
		{ID: "note_provider_offline", OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "provider", Severity: "critical", Title: "Azure East provider is offline", Body: "Three active probes failed and the provider was removed from eligible routing targets.", SourceType: "provider_health", SourceID: "provider_azure_east", ActionURL: "/providers", Status: "unread", CreatedAt: now, UpdatedAt: now},
		{ID: "note_budget_warning", OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "budget", Severity: "warning", Title: "Engineering Copilot reached 80% budget", Body: "Current spend is $7,984 of the $10,000 monthly project budget.", SourceType: "budget", SourceID: "budget_engineering", ActionURL: "/budgets", Status: "unread", CreatedAt: now.Add(-45 * time.Minute), UpdatedAt: now.Add(-45 * time.Minute)},
		{ID: "note_report_ready", OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "report", Severity: "info", Title: "Executive weekly summary is ready", Body: "XLSX and PDF artifacts were generated and delivered to two approved recipients.", SourceType: "report_run", SourceID: "rrun_exec_success", ActionURL: "/reports", Status: "read", ReadAt: &read, CreatedAt: now.Add(-18 * time.Hour), UpdatedAt: read},
		{ID: "note_access_change", OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev", Category: "access", Severity: "warning", Title: "Administrator role granted", Body: "li.ming@topoai.dev was granted Administrator access by the organization owner.", SourceType: "member_role", SourceID: "member_li_ming", ActionURL: "/members", Status: "unread", CreatedAt: now.Add(-22 * time.Hour), UpdatedAt: now.Add(-22 * time.Hour)},
	}
}

func developmentNotificationPreferences() []NotificationPreference {
	updated := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return []NotificationPreference{{
		OrganizationID: "org_topoai", RecipientID: "holden@topoai.dev",
		Destinations: []NotificationDestination{
			{Channel: "in_app", Target: "holden@topoai.dev", DisplayName: "AetherGate inbox"},
			{Channel: "email", Target: "holden@topoai.dev", DisplayName: "Work email"},
			{Channel: "slack", Target: "C_PLATFORM_OPS", DisplayName: "#platform-ops"},
		},
		CategoryChannels: map[string][]string{
			"alert": {"in_app", "email", "slack"}, "budget": {"in_app", "email"},
			"provider": {"in_app", "slack"}, "report": {"in_app", "email"},
			"access": {"in_app", "email"}, "security": {"in_app", "email", "slack"}, "platform": {"in_app"},
		},
		DigestFrequency: "realtime", MinimumSeverity: "info", Timezone: "Asia/Shanghai",
		QuietHoursEnabled: true, QuietStart: "22:00", QuietEnd: "08:00", UpdatedAt: updated,
	}}
}

func developmentNotificationEscalationPolicies() []NotificationEscalationPolicy {
	created := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return []NotificationEscalationPolicy{{
		ID: "npol_critical_ops", OrganizationID: "org_topoai", Name: "Critical platform escalation", Status: "active",
		Categories: []string{"alert", "budget", "provider", "security"}, MinimumSeverity: "critical",
		AcknowledgeWithinMinutes: 15, RepeatEveryMinutes: 15, MaxEscalations: 3,
		Routes: []NotificationEscalationRoute{
			{Level: 1, DelayMinutes: 0, Channel: "slack", Target: "C_PLATFORM_OPS", DisplayName: "#platform-ops"},
			{Level: 2, DelayMinutes: 15, Channel: "email", Target: "oncall@topoai.dev", DisplayName: "Platform on-call"},
			{Level: 3, DelayMinutes: 30, Channel: "teams", Target: "vault://notifications/teams/leadership", DisplayName: "AI leadership"},
		},
		CreatedAt: created, UpdatedAt: created,
	}}
}

func developmentNotificationDeliveries() []NotificationDelivery {
	delivered := time.Date(2026, 7, 15, 2, 30, 4, 0, time.UTC)
	created := time.Date(2026, 7, 15, 2, 30, 0, 0, time.UTC)
	return []NotificationDelivery{
		{ID: "ndel_provider_slack", OrganizationID: "org_topoai", NotificationID: "note_provider_offline", Notification: "Azure East provider is offline", RecipientID: "holden@topoai.dev", Channel: "slack", Target: "C_PLATFORM_OPS", DisplayName: "#platform-ops", Status: "delivered", Attempt: 1, AvailableAt: created, DeliveredAt: &delivered, CreatedAt: created},
		{ID: "ndel_budget_email_failed", OrganizationID: "org_topoai", NotificationID: "note_budget_warning", Notification: "Engineering Copilot reached 80% budget", RecipientID: "holden@topoai.dev", Channel: "email", Target: "holden@topoai.dev", DisplayName: "Work email", Status: "failed", Attempt: 1, AvailableAt: created.Add(-45 * time.Minute), ErrorMessage: "SMTP relay timed out before acknowledgement.", CreatedAt: created.Add(-45 * time.Minute)},
	}
}

var _ NotificationRepository = (*MemoryRepository)(nil)
