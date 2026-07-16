package platform

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/mail"
	"slices"
	"strings"
	"time"
)

var (
	notificationCategories = []string{"alert", "budget", "provider", "report", "access", "security", "platform"}
	notificationSeverities = []string{"info", "warning", "critical"}
	notificationChannels   = []string{"in_app", "email", "slack", "teams", "webhook"}
	notificationDigests    = []string{"realtime", "hourly", "daily", "weekly"}
)

type Notification struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	RecipientID    string     `json:"recipientId"`
	Category       string     `json:"category"`
	Severity       string     `json:"severity"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	SourceType     string     `json:"sourceType"`
	SourceID       string     `json:"sourceId"`
	ActionURL      string     `json:"actionUrl"`
	Status         string     `json:"status"`
	ReadAt         *time.Time `json:"readAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type NotificationDestination struct {
	Channel     string `json:"channel"`
	Target      string `json:"target"`
	DisplayName string `json:"displayName"`
}

type NotificationPreference struct {
	OrganizationID    string                    `json:"organizationId"`
	RecipientID       string                    `json:"recipientId"`
	Destinations      []NotificationDestination `json:"destinations"`
	CategoryChannels  map[string][]string       `json:"categoryChannels"`
	DigestFrequency   string                    `json:"digestFrequency"`
	MinimumSeverity   string                    `json:"minimumSeverity"`
	Timezone          string                    `json:"timezone"`
	QuietHoursEnabled bool                      `json:"quietHoursEnabled"`
	QuietStart        string                    `json:"quietStart"`
	QuietEnd          string                    `json:"quietEnd"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
}

type NotificationEscalationRoute struct {
	Level        int    `json:"level"`
	DelayMinutes int    `json:"delayMinutes"`
	Channel      string `json:"channel"`
	Target       string `json:"target"`
	DisplayName  string `json:"displayName"`
}

type NotificationEscalationPolicy struct {
	ID                       string                        `json:"id"`
	OrganizationID           string                        `json:"organizationId"`
	Name                     string                        `json:"name"`
	Status                   string                        `json:"status"`
	Categories               []string                      `json:"categories"`
	MinimumSeverity          string                        `json:"minimumSeverity"`
	AcknowledgeWithinMinutes int                           `json:"acknowledgeWithinMinutes"`
	RepeatEveryMinutes       int                           `json:"repeatEveryMinutes"`
	MaxEscalations           int                           `json:"maxEscalations"`
	Routes                   []NotificationEscalationRoute `json:"routes"`
	CreatedAt                time.Time                     `json:"createdAt"`
	UpdatedAt                time.Time                     `json:"updatedAt"`
}

type NotificationDelivery struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	NotificationID string     `json:"notificationId"`
	Notification   string     `json:"notification"`
	RecipientID    string     `json:"recipientId"`
	Channel        string     `json:"channel"`
	Target         string     `json:"target"`
	DisplayName    string     `json:"displayName"`
	Status         string     `json:"status"`
	Attempt        int        `json:"attempt"`
	AvailableAt    time.Time  `json:"availableAt"`
	DeliveredAt    *time.Time `json:"deliveredAt"`
	ErrorMessage   string     `json:"errorMessage"`
	ParentID       *string    `json:"parentId"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type NotificationFilter struct {
	OrganizationID string
	RecipientID    string
	Query          string
	Status         string
	Category       string
	Severity       string
}

type NotificationDeliveryFilter struct {
	OrganizationID string
	RecipientID    string
	NotificationID string
	Status         string
	Channel        string
}

type NotificationPolicyFilter struct {
	OrganizationID string
	Query          string
	Status         string
	Category       string
}

type CreateNotificationInput struct {
	OrganizationID string `json:"organizationId"`
	RecipientID    string `json:"recipientId"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	SourceType     string `json:"sourceType"`
	SourceID       string `json:"sourceId"`
	ActionURL      string `json:"actionUrl"`
}

type UpsertNotificationPreferenceInput struct {
	OrganizationID    string                    `json:"organizationId"`
	RecipientID       string                    `json:"recipientId"`
	Destinations      []NotificationDestination `json:"destinations"`
	CategoryChannels  map[string][]string       `json:"categoryChannels"`
	DigestFrequency   string                    `json:"digestFrequency"`
	MinimumSeverity   string                    `json:"minimumSeverity"`
	Timezone          string                    `json:"timezone"`
	QuietHoursEnabled bool                      `json:"quietHoursEnabled"`
	QuietStart        string                    `json:"quietStart"`
	QuietEnd          string                    `json:"quietEnd"`
}

type CreateNotificationEscalationPolicyInput struct {
	OrganizationID           string                        `json:"organizationId"`
	Name                     string                        `json:"name"`
	Status                   string                        `json:"status"`
	Categories               []string                      `json:"categories"`
	MinimumSeverity          string                        `json:"minimumSeverity"`
	AcknowledgeWithinMinutes int                           `json:"acknowledgeWithinMinutes"`
	RepeatEveryMinutes       int                           `json:"repeatEveryMinutes"`
	MaxEscalations           int                           `json:"maxEscalations"`
	Routes                   []NotificationEscalationRoute `json:"routes"`
}

type EvaluateNotificationEscalationInput struct {
	OrganizationID        string `json:"organizationId"`
	Category              string `json:"category"`
	Severity              string `json:"severity"`
	UnacknowledgedMinutes int    `json:"unacknowledgedMinutes"`
}

type NotificationEscalationMatch struct {
	PolicyID   string                        `json:"policyId"`
	PolicyName string                        `json:"policyName"`
	Routes     []NotificationEscalationRoute `json:"routes"`
}

type NotificationEscalationEvaluation struct {
	Matched bool                          `json:"matched"`
	Matches []NotificationEscalationMatch `json:"matches"`
}

type NotificationDispatch struct {
	Notification Notification
	Deliveries   []NotificationDelivery
}

type NotificationRepository interface {
	Repository
	ListNotifications(context.Context, NotificationFilter) ([]Notification, error)
	GetNotification(context.Context, string, string, string) (Notification, error)
	CreateNotificationDispatch(context.Context, Notification, []NotificationDelivery) (NotificationDispatch, error)
	UpdateNotificationStatus(context.Context, string, string, string, string, *time.Time, time.Time) (Notification, error)
	MarkAllNotificationsRead(context.Context, string, string, time.Time) (int64, error)
	GetNotificationPreference(context.Context, string, string) (NotificationPreference, error)
	UpsertNotificationPreference(context.Context, NotificationPreference) (NotificationPreference, error)
	ListNotificationEscalationPolicies(context.Context, NotificationPolicyFilter) ([]NotificationEscalationPolicy, error)
	GetNotificationEscalationPolicy(context.Context, string, string) (NotificationEscalationPolicy, error)
	CreateNotificationEscalationPolicy(context.Context, NotificationEscalationPolicy) (NotificationEscalationPolicy, error)
	UpdateNotificationEscalationPolicyStatus(context.Context, string, string, string, time.Time) (NotificationEscalationPolicy, error)
	ListNotificationDeliveries(context.Context, NotificationDeliveryFilter) ([]NotificationDelivery, error)
	GetNotificationDelivery(context.Context, string, string) (NotificationDelivery, error)
	CreateNotificationDelivery(context.Context, NotificationDelivery) (NotificationDelivery, error)
}

type NotificationService struct {
	repository NotificationRepository
	now        func() time.Time
}

func NewNotificationService(repository NotificationRepository) *NotificationService {
	return &NotificationService{repository: repository, now: time.Now}
}

func (s *NotificationService) List(ctx context.Context, filter NotificationFilter) ([]Notification, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.RecipientID = defaultNotificationRecipient(filter.RecipientID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Severity = strings.TrimSpace(filter.Severity)
	return s.repository.ListNotifications(ctx, filter)
}

func (s *NotificationService) ListDeliveries(ctx context.Context, filter NotificationDeliveryFilter) ([]NotificationDelivery, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.RecipientID = defaultNotificationRecipient(filter.RecipientID)
	filter.NotificationID = strings.TrimSpace(filter.NotificationID)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Channel = strings.TrimSpace(filter.Channel)
	return s.repository.ListNotificationDeliveries(ctx, filter)
}

func (s *NotificationService) Create(ctx context.Context, input CreateNotificationInput) (NotificationDispatch, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.RecipientID = defaultNotificationRecipient(input.RecipientID)
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Severity = strings.ToLower(strings.TrimSpace(input.Severity))
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.ActionURL = strings.TrimSpace(input.ActionURL)
	if !slices.Contains(notificationCategories, input.Category) {
		return NotificationDispatch{}, &ValidationError{Code: "notification_category_invalid", Message: "Notification category is invalid."}
	}
	if !slices.Contains(notificationSeverities, input.Severity) {
		return NotificationDispatch{}, &ValidationError{Code: "notification_severity_invalid", Message: "Notification severity is invalid."}
	}
	if input.Title == "" || len(input.Title) > 180 || input.Body == "" || len(input.Body) > 4000 {
		return NotificationDispatch{}, &ValidationError{Code: "notification_content_invalid", Message: "Notification title and body are required and must fit within their limits."}
	}
	if input.ActionURL != "" && !strings.HasPrefix(input.ActionURL, "/") {
		return NotificationDispatch{}, &ValidationError{Code: "notification_action_invalid", Message: "Notification action URL must be an application-relative path."}
	}
	now := s.now().UTC()
	id, err := randomIdentifier("note_", 10)
	if err != nil {
		return NotificationDispatch{}, err
	}
	notification := Notification{
		ID: id, OrganizationID: input.OrganizationID, RecipientID: input.RecipientID,
		Category: input.Category, Severity: input.Severity, Title: input.Title, Body: input.Body,
		SourceType: input.SourceType, SourceID: input.SourceID, ActionURL: input.ActionURL,
		Status: "unread", CreatedAt: now, UpdatedAt: now,
	}
	preference, err := s.Preference(ctx, input.OrganizationID, input.RecipientID)
	if err != nil {
		return NotificationDispatch{}, err
	}
	deliveries, err := s.buildDeliveries(notification, preference, now)
	if err != nil {
		return NotificationDispatch{}, err
	}
	return s.repository.CreateNotificationDispatch(ctx, notification, deliveries)
}

func (s *NotificationService) MarkRead(ctx context.Context, organizationID, recipientID, id string) (Notification, error) {
	now := s.now().UTC()
	return s.repository.UpdateNotificationStatus(ctx, defaultOrganization(organizationID), defaultNotificationRecipient(recipientID), strings.TrimSpace(id), "read", &now, now)
}

func (s *NotificationService) MarkUnread(ctx context.Context, organizationID, recipientID, id string) (Notification, error) {
	return s.repository.UpdateNotificationStatus(ctx, defaultOrganization(organizationID), defaultNotificationRecipient(recipientID), strings.TrimSpace(id), "unread", nil, s.now().UTC())
}

func (s *NotificationService) Archive(ctx context.Context, organizationID, recipientID, id string) (Notification, error) {
	now := s.now().UTC()
	return s.repository.UpdateNotificationStatus(ctx, defaultOrganization(organizationID), defaultNotificationRecipient(recipientID), strings.TrimSpace(id), "archived", &now, now)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, organizationID, recipientID string) (int64, error) {
	return s.repository.MarkAllNotificationsRead(ctx, defaultOrganization(organizationID), defaultNotificationRecipient(recipientID), s.now().UTC())
}

func (s *NotificationService) Preference(ctx context.Context, organizationID, recipientID string) (NotificationPreference, error) {
	organizationID = defaultOrganization(organizationID)
	recipientID = defaultNotificationRecipient(recipientID)
	preference, err := s.repository.GetNotificationPreference(ctx, organizationID, recipientID)
	if errors.Is(err, ErrNotFound) {
		return defaultNotificationPreference(organizationID, recipientID, s.now().UTC()), nil
	}
	return preference, err
}

func (s *NotificationService) UpsertPreference(ctx context.Context, input UpsertNotificationPreferenceInput) (NotificationPreference, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.RecipientID = defaultNotificationRecipient(input.RecipientID)
	input.DigestFrequency = strings.ToLower(strings.TrimSpace(input.DigestFrequency))
	input.MinimumSeverity = strings.ToLower(strings.TrimSpace(input.MinimumSeverity))
	input.Timezone = strings.TrimSpace(input.Timezone)
	input.QuietStart = strings.TrimSpace(input.QuietStart)
	input.QuietEnd = strings.TrimSpace(input.QuietEnd)
	if !slices.Contains(notificationDigests, input.DigestFrequency) {
		return NotificationPreference{}, &ValidationError{Code: "notification_digest_invalid", Message: "Notification digest frequency is invalid."}
	}
	if !slices.Contains(notificationSeverities, input.MinimumSeverity) {
		return NotificationPreference{}, &ValidationError{Code: "notification_severity_invalid", Message: "Notification minimum severity is invalid."}
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return NotificationPreference{}, &ValidationError{Code: "notification_timezone_invalid", Message: "Notification timezone must be a valid IANA timezone."}
	}
	if input.QuietStart == "" {
		input.QuietStart = "22:00"
	}
	if input.QuietEnd == "" {
		input.QuietEnd = "08:00"
	}
	if _, err := parseClock(input.QuietStart); err != nil {
		return NotificationPreference{}, &ValidationError{Code: "notification_quiet_hours_invalid", Message: "Quiet hours must use HH:MM in 24-hour format."}
	}
	if _, err := parseClock(input.QuietEnd); err != nil || input.QuietHoursEnabled && input.QuietStart == input.QuietEnd {
		return NotificationPreference{}, &ValidationError{Code: "notification_quiet_hours_invalid", Message: "Quiet hours must use distinct HH:MM values."}
	}
	destinations, err := normalizeNotificationDestinations(input.RecipientID, input.Destinations)
	if err != nil {
		return NotificationPreference{}, err
	}
	routes, err := normalizeNotificationCategoryChannels(input.CategoryChannels, destinations)
	if err != nil {
		return NotificationPreference{}, err
	}
	return s.repository.UpsertNotificationPreference(ctx, NotificationPreference{
		OrganizationID: input.OrganizationID, RecipientID: input.RecipientID, Destinations: destinations,
		CategoryChannels: routes, DigestFrequency: input.DigestFrequency, MinimumSeverity: input.MinimumSeverity,
		Timezone: input.Timezone, QuietHoursEnabled: input.QuietHoursEnabled,
		QuietStart: input.QuietStart, QuietEnd: input.QuietEnd, UpdatedAt: s.now().UTC(),
	})
}

func (s *NotificationService) ListPolicies(ctx context.Context, filter NotificationPolicyFilter) ([]NotificationEscalationPolicy, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Category = strings.TrimSpace(filter.Category)
	return s.repository.ListNotificationEscalationPolicies(ctx, filter)
}

func (s *NotificationService) CreatePolicy(ctx context.Context, input CreateNotificationEscalationPolicyInput) (NotificationEscalationPolicy, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.MinimumSeverity = strings.ToLower(strings.TrimSpace(input.MinimumSeverity))
	if input.Name == "" {
		return NotificationEscalationPolicy{}, &ValidationError{Code: "notification_policy_name_required", Message: "Escalation policy name is required."}
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "paused" {
		return NotificationEscalationPolicy{}, &ValidationError{Code: "notification_policy_status_invalid", Message: "Escalation policy status must be active or paused."}
	}
	if !slices.Contains(notificationSeverities, input.MinimumSeverity) {
		return NotificationEscalationPolicy{}, &ValidationError{Code: "notification_severity_invalid", Message: "Escalation minimum severity is invalid."}
	}
	categories, err := normalizeNotificationCategories(input.Categories)
	if err != nil {
		return NotificationEscalationPolicy{}, err
	}
	if input.AcknowledgeWithinMinutes < 1 || input.AcknowledgeWithinMinutes > 1440 || input.RepeatEveryMinutes < 0 || input.RepeatEveryMinutes > 1440 || input.MaxEscalations < 1 || input.MaxEscalations > 5 {
		return NotificationEscalationPolicy{}, &ValidationError{Code: "notification_escalation_timing_invalid", Message: "Escalation acknowledgement, repeat, or maximum level is invalid."}
	}
	routes, err := normalizeNotificationEscalationRoutes(input.Routes, input.MaxEscalations)
	if err != nil {
		return NotificationEscalationPolicy{}, err
	}
	id, err := randomIdentifier("npol_", 9)
	if err != nil {
		return NotificationEscalationPolicy{}, err
	}
	now := s.now().UTC()
	return s.repository.CreateNotificationEscalationPolicy(ctx, NotificationEscalationPolicy{
		ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Status: input.Status,
		Categories: categories, MinimumSeverity: input.MinimumSeverity,
		AcknowledgeWithinMinutes: input.AcknowledgeWithinMinutes, RepeatEveryMinutes: input.RepeatEveryMinutes,
		MaxEscalations: input.MaxEscalations, Routes: routes, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *NotificationService) ActivatePolicy(ctx context.Context, organizationID, id string) (NotificationEscalationPolicy, error) {
	return s.repository.UpdateNotificationEscalationPolicyStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "active", s.now().UTC())
}

func (s *NotificationService) PausePolicy(ctx context.Context, organizationID, id string) (NotificationEscalationPolicy, error) {
	return s.repository.UpdateNotificationEscalationPolicyStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "paused", s.now().UTC())
}

func (s *NotificationService) EvaluateEscalation(ctx context.Context, input EvaluateNotificationEscalationInput) (NotificationEscalationEvaluation, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Severity = strings.ToLower(strings.TrimSpace(input.Severity))
	if !slices.Contains(notificationCategories, input.Category) || !slices.Contains(notificationSeverities, input.Severity) || input.UnacknowledgedMinutes < 0 {
		return NotificationEscalationEvaluation{}, &ValidationError{Code: "notification_escalation_input_invalid", Message: "Escalation evaluation input is invalid."}
	}
	policies, err := s.repository.ListNotificationEscalationPolicies(ctx, NotificationPolicyFilter{OrganizationID: input.OrganizationID, Status: "active", Category: input.Category})
	if err != nil {
		return NotificationEscalationEvaluation{}, err
	}
	matches := make([]NotificationEscalationMatch, 0)
	for _, policy := range policies {
		if severityRank(input.Severity) < severityRank(policy.MinimumSeverity) || input.UnacknowledgedMinutes < policy.AcknowledgeWithinMinutes {
			continue
		}
		routes := make([]NotificationEscalationRoute, 0)
		for _, route := range policy.Routes {
			if route.Level <= policy.MaxEscalations && route.DelayMinutes <= input.UnacknowledgedMinutes-policy.AcknowledgeWithinMinutes {
				routes = append(routes, route)
			}
		}
		if len(routes) > 0 {
			matches = append(matches, NotificationEscalationMatch{PolicyID: policy.ID, PolicyName: policy.Name, Routes: routes})
		}
	}
	return NotificationEscalationEvaluation{Matched: len(matches) > 0, Matches: matches}, nil
}

func (s *NotificationService) RetryDelivery(ctx context.Context, organizationID, deliveryID string) (NotificationDelivery, error) {
	organizationID = defaultOrganization(organizationID)
	delivery, err := s.repository.GetNotificationDelivery(ctx, organizationID, strings.TrimSpace(deliveryID))
	if err != nil {
		return NotificationDelivery{}, err
	}
	if delivery.Status != "failed" {
		return NotificationDelivery{}, &ValidationError{Code: "notification_delivery_retry_invalid", Message: "Only failed notification deliveries can be retried."}
	}
	id, err := randomIdentifier("ndel_", 10)
	if err != nil {
		return NotificationDelivery{}, err
	}
	now := s.now().UTC()
	return s.repository.CreateNotificationDelivery(ctx, NotificationDelivery{
		ID: id, OrganizationID: delivery.OrganizationID, NotificationID: delivery.NotificationID,
		Notification: delivery.Notification, RecipientID: delivery.RecipientID, Channel: delivery.Channel,
		Target: delivery.Target, DisplayName: delivery.DisplayName, Status: "queued", Attempt: delivery.Attempt + 1,
		AvailableAt: now, ParentID: &delivery.ID, CreatedAt: now,
	})
}

func (s *NotificationService) buildDeliveries(notification Notification, preference NotificationPreference, now time.Time) ([]NotificationDelivery, error) {
	channels := preference.CategoryChannels[notification.Category]
	deliveries := make([]NotificationDelivery, 0)
	for _, channel := range channels {
		if channel == "in_app" {
			continue
		}
		for _, destination := range preference.Destinations {
			if destination.Channel != channel {
				continue
			}
			id, err := randomIdentifier("ndel_", 10)
			if err != nil {
				return nil, err
			}
			status := "queued"
			availableAt := now
			errorMessage := ""
			if severityRank(notification.Severity) < severityRank(preference.MinimumSeverity) {
				status = "suppressed"
				errorMessage = "Below recipient severity threshold."
			} else {
				availableAt = nextNotificationDeliveryTime(now, preference)
				if availableAt.After(now.Add(time.Second)) {
					status = "deferred"
				}
			}
			deliveries = append(deliveries, NotificationDelivery{
				ID: id, OrganizationID: notification.OrganizationID, NotificationID: notification.ID,
				Notification: notification.Title, RecipientID: notification.RecipientID, Channel: channel,
				Target: destination.Target, DisplayName: destination.DisplayName, Status: status, Attempt: 1,
				AvailableAt: availableAt, ErrorMessage: errorMessage, CreatedAt: now,
			})
		}
	}
	return deliveries, nil
}

func normalizeNotificationDestinations(recipientID string, values []NotificationDestination) ([]NotificationDestination, error) {
	items := make([]NotificationDestination, 0, len(values)+1)
	seen := make(map[string]struct{})
	hasInApp := false
	for _, destination := range values {
		destination.Channel = strings.ToLower(strings.TrimSpace(destination.Channel))
		destination.Target = strings.TrimSpace(destination.Target)
		destination.DisplayName = strings.TrimSpace(destination.DisplayName)
		if !slices.Contains(notificationChannels, destination.Channel) || destination.Target == "" {
			return nil, &ValidationError{Code: "notification_destination_invalid", Message: "Every notification destination requires a supported channel and target."}
		}
		if destination.Channel == "email" {
			address, err := mail.ParseAddress(destination.Target)
			if err != nil || !strings.EqualFold(address.Address, destination.Target) {
				return nil, &ValidationError{Code: "notification_destination_invalid", Message: "Notification email destination is invalid."}
			}
		}
		if destination.Channel == "in_app" {
			destination.Target = recipientID
			hasInApp = true
		}
		key := destination.Channel + "\x00" + strings.ToLower(destination.Target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, destination)
	}
	if !hasInApp {
		items = append([]NotificationDestination{{Channel: "in_app", Target: recipientID, DisplayName: "AetherGate inbox"}}, items...)
	}
	if len(items) > 20 {
		return nil, &ValidationError{Code: "notification_destination_limit", Message: "A recipient can have at most 20 notification destinations."}
	}
	return items, nil
}

func normalizeNotificationCategoryChannels(values map[string][]string, destinations []NotificationDestination) (map[string][]string, error) {
	if len(values) == 0 {
		return nil, &ValidationError{Code: "notification_routes_required", Message: "At least one category channel route is required."}
	}
	available := make(map[string]bool)
	for _, destination := range destinations {
		available[destination.Channel] = true
	}
	items := make(map[string][]string, len(values))
	for category, channels := range values {
		category = strings.ToLower(strings.TrimSpace(category))
		if !slices.Contains(notificationCategories, category) {
			return nil, &ValidationError{Code: "notification_route_category_invalid", Message: "Notification route category is invalid."}
		}
		normalized := make([]string, 0, len(channels))
		for _, channel := range channels {
			channel = strings.ToLower(strings.TrimSpace(channel))
			if !slices.Contains(notificationChannels, channel) || !available[channel] {
				return nil, &ValidationError{Code: "notification_route_channel_invalid", Message: "Notification route channel has no configured destination."}
			}
			if !slices.Contains(normalized, channel) {
				normalized = append(normalized, channel)
			}
		}
		items[category] = normalized
	}
	return items, nil
}

func normalizeNotificationCategories(values []string) ([]string, error) {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !slices.Contains(notificationCategories, value) {
			return nil, &ValidationError{Code: "notification_policy_category_invalid", Message: "Escalation policy category is invalid."}
		}
		if !slices.Contains(items, value) {
			items = append(items, value)
		}
	}
	if len(items) == 0 {
		return nil, &ValidationError{Code: "notification_policy_categories_required", Message: "Escalation policy requires at least one category."}
	}
	return items, nil
}

func normalizeNotificationEscalationRoutes(values []NotificationEscalationRoute, maxLevel int) ([]NotificationEscalationRoute, error) {
	items := make([]NotificationEscalationRoute, 0, len(values))
	seen := make(map[string]struct{})
	hasLevelOne := false
	for _, route := range values {
		route.Channel = strings.ToLower(strings.TrimSpace(route.Channel))
		route.Target = strings.TrimSpace(route.Target)
		route.DisplayName = strings.TrimSpace(route.DisplayName)
		if route.Level < 1 || route.Level > maxLevel || route.DelayMinutes < 0 || route.DelayMinutes > 1440 || !slices.Contains(notificationChannels, route.Channel) || route.Target == "" {
			return nil, &ValidationError{Code: "notification_escalation_route_invalid", Message: "Escalation route level, delay, channel, or target is invalid."}
		}
		if route.Level == 1 && route.DelayMinutes == 0 {
			hasLevelOne = true
		}
		key := fmt.Sprintf("%d\x00%s\x00%s", route.Level, route.Channel, strings.ToLower(route.Target))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, route)
	}
	if !hasLevelOne {
		return nil, &ValidationError{Code: "notification_escalation_route_required", Message: "Escalation policy requires a level-one route with zero delay."}
	}
	return items, nil
}

func defaultNotificationPreference(organizationID, recipientID string, now time.Time) NotificationPreference {
	routes := make(map[string][]string, len(notificationCategories))
	for _, category := range notificationCategories {
		routes[category] = []string{"in_app"}
	}
	return NotificationPreference{
		OrganizationID: organizationID, RecipientID: recipientID,
		Destinations:     []NotificationDestination{{Channel: "in_app", Target: recipientID, DisplayName: "AetherGate inbox"}},
		CategoryChannels: routes, DigestFrequency: "realtime", MinimumSeverity: "info", Timezone: "UTC",
		QuietStart: "22:00", QuietEnd: "08:00", UpdatedAt: now,
	}
}

func defaultNotificationRecipient(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "holden@topoai.dev"
}

func severityRank(value string) int { return slices.Index(notificationSeverities, value) }

func parseClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func nextNotificationDeliveryTime(now time.Time, preference NotificationPreference) time.Time {
	location, err := time.LoadLocation(preference.Timezone)
	if err != nil {
		return now
	}
	local := now.In(location)
	candidate := local
	if preference.DigestFrequency == "hourly" {
		candidate = time.Date(local.Year(), local.Month(), local.Day(), local.Hour()+1, 0, 0, 0, location)
	} else if preference.DigestFrequency == "daily" {
		candidate = nextLocalClock(local, time.Monday, false)
	} else if preference.DigestFrequency == "weekly" {
		candidate = nextLocalClock(local, time.Monday, true)
	}
	if preference.QuietHoursEnabled {
		quietEnd := quietHoursEnd(local, preference.QuietStart, preference.QuietEnd)
		if quietEnd.After(candidate) {
			candidate = quietEnd
		}
	}
	return candidate.UTC()
}

func nextLocalClock(local time.Time, weekday time.Weekday, weekly bool) time.Time {
	if !weekly {
		candidate := time.Date(local.Year(), local.Month(), local.Day(), 9, 0, 0, 0, local.Location())
		if !candidate.After(local) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate
	}
	delta := (int(weekday) - int(local.Weekday()) + 7) % 7
	candidate := time.Date(local.Year(), local.Month(), local.Day()+delta, 9, 0, 0, 0, local.Location())
	if !candidate.After(local) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

func quietHoursEnd(local time.Time, startText, endText string) time.Time {
	start, startErr := parseClock(startText)
	end, endErr := parseClock(endText)
	if startErr != nil || endErr != nil || start == end {
		return local
	}
	minute := local.Hour()*60 + local.Minute()
	within := false
	endDay := 0
	if start < end {
		within = minute >= start && minute < end
	} else {
		within = minute >= start || minute < end
		if minute >= start {
			endDay = 1
		}
	}
	if !within {
		return local
	}
	return time.Date(local.Year(), local.Month(), local.Day()+endDay, end/60, end%60, 0, 0, local.Location())
}

func cloneNotificationPreference(preference NotificationPreference) NotificationPreference {
	preference.Destinations = slices.Clone(preference.Destinations)
	preference.CategoryChannels = maps.Clone(preference.CategoryChannels)
	for category, channels := range preference.CategoryChannels {
		preference.CategoryChannels[category] = slices.Clone(channels)
	}
	return preference
}

func cloneNotificationPolicy(policy NotificationEscalationPolicy) NotificationEscalationPolicy {
	policy.Categories = slices.Clone(policy.Categories)
	policy.Routes = slices.Clone(policy.Routes)
	return policy
}
