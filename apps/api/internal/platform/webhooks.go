package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"
)

var webhookEventTypes = []string{
	"request.completed",
	"request.failed",
	"alert.triggered",
	"alert.resolved",
	"budget.threshold_reached",
	"api_key.revoked",
	"provider.health_changed",
}

type WebhookPropertyFilter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WebhookEndpoint struct {
	ID                  string                  `json:"id"`
	OrganizationID      string                  `json:"organizationId"`
	Name                string                  `json:"name"`
	Status              string                  `json:"status"`
	Destination         string                  `json:"destination"`
	Version             string                  `json:"version"`
	Events              []string                `json:"events"`
	SampleRate          float64                 `json:"sampleRate"`
	IncludeData         bool                    `json:"includeData"`
	PropertyFilters     []WebhookPropertyFilter `json:"propertyFilters"`
	SigningSecretPrefix string                  `json:"signingSecretPrefix"`
	MaxAttempts         int                     `json:"maxAttempts"`
	TimeoutSeconds      int                     `json:"timeoutSeconds"`
	SuccessCount        int64                   `json:"successCount"`
	FailureCount        int64                   `json:"failureCount"`
	LastDeliveredAt     *time.Time              `json:"lastDeliveredAt"`
	CreatedAt           time.Time               `json:"createdAt"`
	UpdatedAt           time.Time               `json:"updatedAt"`
	SigningSecretDigest [32]byte                `json:"-"`
	SecretReference     string                  `json:"-"`
}

type WebhookDelivery struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	WebhookID      string     `json:"webhookId"`
	WebhookName    string     `json:"webhookName"`
	EventID        string     `json:"eventId"`
	EventType      string     `json:"eventType"`
	Status         string     `json:"status"`
	Trigger        string     `json:"trigger"`
	Attempt        int        `json:"attempt"`
	MaxAttempts    int        `json:"maxAttempts"`
	ResponseStatus *int       `json:"responseStatus"`
	DurationMS     int        `json:"durationMs"`
	ErrorMessage   string     `json:"errorMessage"`
	NextRetryAt    *time.Time `json:"nextRetryAt"`
	DeliveredAt    *time.Time `json:"deliveredAt"`
	ReplayOfID     *string    `json:"replayOfId"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type CreatedWebhook struct {
	Record        WebhookEndpoint `json:"data"`
	SigningSecret string          `json:"signingSecret"`
}

type WebhookFilter struct {
	OrganizationID string
	Query          string
	Status         string
	Event          string
}

type WebhookDeliveryFilter struct {
	OrganizationID string
	WebhookID      string
	Status         string
	EventType      string
}

type CreateWebhookInput struct {
	OrganizationID  string                  `json:"organizationId"`
	Name            string                  `json:"name"`
	Destination     string                  `json:"destination"`
	Events          []string                `json:"events"`
	SampleRate      float64                 `json:"sampleRate"`
	IncludeData     bool                    `json:"includeData"`
	PropertyFilters []WebhookPropertyFilter `json:"propertyFilters"`
	MaxAttempts     int                     `json:"maxAttempts"`
	TimeoutSeconds  int                     `json:"timeoutSeconds"`
}

type WebhookTestInput struct {
	EventType string `json:"eventType"`
}

type WebhookRepository interface {
	Repository
	ListWebhookEndpoints(context.Context, WebhookFilter) ([]WebhookEndpoint, error)
	GetWebhookEndpoint(context.Context, string, string) (WebhookEndpoint, error)
	CreateWebhookEndpoint(context.Context, WebhookEndpoint) (WebhookEndpoint, error)
	UpdateWebhookEndpointStatus(context.Context, string, string, string, time.Time) (WebhookEndpoint, error)
	ListWebhookDeliveries(context.Context, WebhookDeliveryFilter) ([]WebhookDelivery, error)
	GetWebhookDelivery(context.Context, string, string) (WebhookDelivery, error)
	CreateWebhookDelivery(context.Context, WebhookDelivery) (WebhookDelivery, error)
}

type WebhookService struct {
	repository WebhookRepository
	now        func() time.Time
}

func NewWebhookService(repository WebhookRepository) *WebhookService {
	return &WebhookService{repository: repository, now: time.Now}
}

func (s *WebhookService) List(ctx context.Context, filter WebhookFilter) ([]WebhookEndpoint, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	return s.repository.ListWebhookEndpoints(ctx, filter)
}

func (s *WebhookService) ListDeliveries(ctx context.Context, filter WebhookDeliveryFilter) ([]WebhookDelivery, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	return s.repository.ListWebhookDeliveries(ctx, filter)
}

func (s *WebhookService) Create(ctx context.Context, input CreateWebhookInput) (CreatedWebhook, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.Destination = strings.TrimSpace(input.Destination)
	if input.Name == "" {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_name_required", Message: "Webhook name is required."}
	}
	if err := validateWebhookDestination(input.Destination); err != nil {
		return CreatedWebhook{}, err
	}
	input.Events = slices.Compact(input.Events)
	if len(input.Events) == 0 {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_events_required", Message: "At least one webhook event is required."}
	}
	for index := range input.Events {
		input.Events[index] = strings.TrimSpace(input.Events[index])
		if !slices.Contains(webhookEventTypes, input.Events[index]) {
			return CreatedWebhook{}, &ValidationError{Code: "webhook_event_invalid", Message: "One or more webhook events are invalid."}
		}
	}
	if input.SampleRate == 0 {
		input.SampleRate = 100
	}
	if input.SampleRate < 0.1 || input.SampleRate > 100 {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_sample_rate_invalid", Message: "Sample rate must be between 0.1 and 100 percent."}
	}
	if input.MaxAttempts == 0 {
		input.MaxAttempts = 5
	}
	if input.MaxAttempts < 1 || input.MaxAttempts > 10 {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_attempts_invalid", Message: "Max attempts must be between 1 and 10."}
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 10
	}
	if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 30 {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_timeout_invalid", Message: "Timeout must be between 1 and 30 seconds."}
	}
	if len(input.PropertyFilters) > 20 {
		return CreatedWebhook{}, &ValidationError{Code: "webhook_filters_invalid", Message: "A webhook can have at most 20 property filters."}
	}
	for index := range input.PropertyFilters {
		input.PropertyFilters[index].Key = strings.TrimSpace(input.PropertyFilters[index].Key)
		input.PropertyFilters[index].Value = strings.TrimSpace(input.PropertyFilters[index].Value)
		if input.PropertyFilters[index].Key == "" || input.PropertyFilters[index].Value == "" {
			return CreatedWebhook{}, &ValidationError{Code: "webhook_filters_invalid", Message: "Property filter keys and values are required."}
		}
	}

	id, err := randomIdentifier("wh_", 9)
	if err != nil {
		return CreatedWebhook{}, err
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return CreatedWebhook{}, err
	}
	secret := "whsec_" + base64.RawURLEncoding.EncodeToString(secretBytes)
	now := s.now().UTC()
	record := WebhookEndpoint{
		ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Status: "active",
		Destination: input.Destination, Version: "2026-07-15", Events: slices.Clone(input.Events),
		SampleRate: input.SampleRate, IncludeData: input.IncludeData,
		PropertyFilters: slices.Clone(input.PropertyFilters), SigningSecretPrefix: secret[:14],
		MaxAttempts: input.MaxAttempts, TimeoutSeconds: input.TimeoutSeconds,
		CreatedAt: now, UpdatedAt: now, SigningSecretDigest: sha256.Sum256([]byte(secret)),
		SecretReference: "vault://aethergate/webhooks/" + id,
	}
	created, err := s.repository.CreateWebhookEndpoint(ctx, record)
	if err != nil {
		return CreatedWebhook{}, err
	}
	return CreatedWebhook{Record: created, SigningSecret: secret}, nil
}

func (s *WebhookService) Enable(ctx context.Context, organizationID, id string) (WebhookEndpoint, error) {
	return s.repository.UpdateWebhookEndpointStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "active", s.now().UTC())
}

func (s *WebhookService) Disable(ctx context.Context, organizationID, id string) (WebhookEndpoint, error) {
	return s.repository.UpdateWebhookEndpointStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "disabled", s.now().UTC())
}

func (s *WebhookService) QueueTest(ctx context.Context, organizationID, id string, input WebhookTestInput) (WebhookDelivery, error) {
	organizationID = defaultOrganization(organizationID)
	endpoint, err := s.repository.GetWebhookEndpoint(ctx, organizationID, strings.TrimSpace(id))
	if err != nil {
		return WebhookDelivery{}, err
	}
	if endpoint.Status != "active" {
		return WebhookDelivery{}, ErrInactive
	}
	input.EventType = strings.TrimSpace(input.EventType)
	if input.EventType == "" {
		input.EventType = endpoint.Events[0]
	}
	if !slices.Contains(endpoint.Events, input.EventType) {
		return WebhookDelivery{}, &ValidationError{Code: "webhook_test_event_invalid", Message: "The test event is not subscribed by this webhook."}
	}
	eventID, err := randomIdentifier("evt_test_", 9)
	if err != nil {
		return WebhookDelivery{}, err
	}
	return s.queueDelivery(ctx, endpoint, eventID, input.EventType, "test", 1, endpoint.MaxAttempts, nil)
}

func (s *WebhookService) Retry(ctx context.Context, organizationID, deliveryID string) (WebhookDelivery, error) {
	organizationID = defaultOrganization(organizationID)
	delivery, err := s.repository.GetWebhookDelivery(ctx, organizationID, strings.TrimSpace(deliveryID))
	if err != nil {
		return WebhookDelivery{}, err
	}
	if delivery.Status != "failed" && delivery.Status != "dead_letter" {
		return WebhookDelivery{}, &ValidationError{Code: "webhook_retry_invalid", Message: "Only failed or dead-letter deliveries can be retried."}
	}
	if delivery.Attempt >= delivery.MaxAttempts {
		return WebhookDelivery{}, &ValidationError{Code: "webhook_retry_exhausted", Message: "This delivery has exhausted its configured attempts; use replay instead."}
	}
	endpoint, err := s.repository.GetWebhookEndpoint(ctx, organizationID, delivery.WebhookID)
	if err != nil {
		return WebhookDelivery{}, err
	}
	if endpoint.Status != "active" {
		return WebhookDelivery{}, ErrInactive
	}
	return s.queueDelivery(ctx, endpoint, delivery.EventID, delivery.EventType, "retry", delivery.Attempt+1, delivery.MaxAttempts, &delivery.ID)
}

func (s *WebhookService) Replay(ctx context.Context, organizationID, deliveryID string) (WebhookDelivery, error) {
	organizationID = defaultOrganization(organizationID)
	delivery, err := s.repository.GetWebhookDelivery(ctx, organizationID, strings.TrimSpace(deliveryID))
	if err != nil {
		return WebhookDelivery{}, err
	}
	if delivery.Status == "pending" || delivery.Status == "delivering" {
		return WebhookDelivery{}, &ValidationError{Code: "webhook_replay_invalid", Message: "A pending delivery cannot be replayed."}
	}
	endpoint, err := s.repository.GetWebhookEndpoint(ctx, organizationID, delivery.WebhookID)
	if err != nil {
		return WebhookDelivery{}, err
	}
	if endpoint.Status != "active" {
		return WebhookDelivery{}, ErrInactive
	}
	eventID, err := randomIdentifier("evt_replay_", 9)
	if err != nil {
		return WebhookDelivery{}, err
	}
	return s.queueDelivery(ctx, endpoint, eventID, delivery.EventType, "replay", 1, endpoint.MaxAttempts, &delivery.ID)
}

func (s *WebhookService) queueDelivery(ctx context.Context, endpoint WebhookEndpoint, eventID, eventType, trigger string, attempt, maxAttempts int, replayOfID *string) (WebhookDelivery, error) {
	id, err := randomIdentifier("whd_", 10)
	if err != nil {
		return WebhookDelivery{}, err
	}
	now := s.now().UTC()
	return s.repository.CreateWebhookDelivery(ctx, WebhookDelivery{
		ID: id, OrganizationID: endpoint.OrganizationID, WebhookID: endpoint.ID, WebhookName: endpoint.Name,
		EventID: eventID, EventType: eventType, Status: "pending", Trigger: trigger,
		Attempt: attempt, MaxAttempts: maxAttempts, ReplayOfID: replayOfID, CreatedAt: now,
	})
}

func validateWebhookDestination(destination string) error {
	parsed, err := url.ParseRequestURI(destination)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return &ValidationError{Code: "webhook_destination_invalid", Message: "Destination must be an absolute HTTP or HTTPS URL without credentials or a fragment."}
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" {
		return &ValidationError{Code: "webhook_destination_invalid", Message: "Destination must use HTTPS, except for local development endpoints."}
	}
	host := parsed.Hostname()
	address := net.ParseIP(host)
	if strings.EqualFold(host, "localhost") || address != nil && address.IsLoopback() {
		return nil
	}
	return &ValidationError{Code: "webhook_destination_insecure", Message: "Non-local webhook destinations must use HTTPS."}
}
