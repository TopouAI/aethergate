package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"slices"
	"strings"
	"time"
)

type AuditEvent struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organizationId"`
	ActorID        string         `json:"actorId"`
	ActorEmail     string         `json:"actorEmail"`
	Action         string         `json:"action"`
	ResourceType   string         `json:"resourceType"`
	ResourceID     string         `json:"resourceId"`
	Outcome        string         `json:"outcome"`
	RiskLevel      string         `json:"riskLevel"`
	Source         string         `json:"source"`
	Reason         string         `json:"reason"`
	RequestID      string         `json:"requestId"`
	IPAddress      string         `json:"ipAddress"`
	UserAgent      string         `json:"userAgent"`
	BeforeState    map[string]any `json:"beforeState"`
	AfterState     map[string]any `json:"afterState"`
	PreviousHash   string         `json:"previousHash"`
	IntegrityHash  string         `json:"integrityHash"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type AuditRetentionPolicy struct {
	OrganizationID string    `json:"organizationId"`
	RetentionDays  int       `json:"retentionDays"`
	LegalHold      bool      `json:"legalHold"`
	ExportFormat   string    `json:"exportFormat"`
	UpdatedBy      string    `json:"updatedBy"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AuditExport struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organizationId"`
	RequestedBy    string            `json:"requestedBy"`
	Format         string            `json:"format"`
	Status         string            `json:"status"`
	Filters        map[string]string `json:"filters"`
	PeriodStart    time.Time         `json:"periodStart"`
	PeriodEnd      time.Time         `json:"periodEnd"`
	RowCount       int64             `json:"rowCount"`
	SizeBytes      int64             `json:"sizeBytes"`
	ObjectKey      string            `json:"objectKey"`
	Checksum       string            `json:"checksum"`
	ErrorMessage   string            `json:"errorMessage"`
	ParentID       *string           `json:"parentId"`
	CreatedAt      time.Time         `json:"createdAt"`
	CompletedAt    *time.Time        `json:"completedAt"`
}

type AuditFilter struct {
	OrganizationID string
	Query          string
	Actor          string
	Action         string
	ResourceType   string
	ResourceID     string
	Outcome        string
	RiskLevel      string
	StartAt        *time.Time
	EndAt          *time.Time
}

type AuditExportFilter struct {
	OrganizationID string
	Status         string
	Format         string
}

type AppendAuditEventInput struct {
	OrganizationID string         `json:"organizationId"`
	ActorID        string         `json:"actorId"`
	ActorEmail     string         `json:"actorEmail"`
	Action         string         `json:"action"`
	ResourceType   string         `json:"resourceType"`
	ResourceID     string         `json:"resourceId"`
	Outcome        string         `json:"outcome"`
	RiskLevel      string         `json:"riskLevel"`
	Source         string         `json:"source"`
	Reason         string         `json:"reason"`
	RequestID      string         `json:"requestId"`
	IPAddress      string         `json:"ipAddress"`
	UserAgent      string         `json:"userAgent"`
	BeforeState    map[string]any `json:"beforeState"`
	AfterState     map[string]any `json:"afterState"`
}

type UpsertAuditRetentionInput struct {
	OrganizationID string `json:"organizationId"`
	RetentionDays  int    `json:"retentionDays"`
	LegalHold      bool   `json:"legalHold"`
	ExportFormat   string `json:"exportFormat"`
	UpdatedBy      string `json:"updatedBy"`
}

type QueueAuditExportInput struct {
	OrganizationID string            `json:"organizationId"`
	RequestedBy    string            `json:"requestedBy"`
	Format         string            `json:"format"`
	Filters        map[string]string `json:"filters"`
	PeriodStart    string            `json:"periodStart"`
	PeriodEnd      string            `json:"periodEnd"`
}

type AuditIntegrityResult struct {
	Valid          bool   `json:"valid"`
	EventCount     int    `json:"eventCount"`
	HeadHash       string `json:"headHash"`
	FirstInvalidID string `json:"firstInvalidId"`
}

type AuditRepository interface {
	Repository
	ListAuditEvents(context.Context, AuditFilter) ([]AuditEvent, error)
	LatestAuditHash(context.Context, string) (string, error)
	AppendAuditEvent(context.Context, AuditEvent) (AuditEvent, error)
	GetAuditRetentionPolicy(context.Context, string) (AuditRetentionPolicy, error)
	UpsertAuditRetentionPolicy(context.Context, AuditRetentionPolicy) (AuditRetentionPolicy, error)
	ListAuditExports(context.Context, AuditExportFilter) ([]AuditExport, error)
	GetAuditExport(context.Context, string, string) (AuditExport, error)
	CreateAuditExport(context.Context, AuditExport) (AuditExport, error)
}

type AuditService struct {
	repository AuditRepository
	now        func() time.Time
}

func NewAuditService(repository AuditRepository) *AuditService {
	return &AuditService{repository: repository, now: time.Now}
}

func (s *AuditService) List(ctx context.Context, filter AuditFilter) ([]AuditEvent, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Actor = strings.TrimSpace(filter.Actor)
	filter.Action = strings.TrimSpace(filter.Action)
	filter.ResourceType = strings.TrimSpace(filter.ResourceType)
	filter.ResourceID = strings.TrimSpace(filter.ResourceID)
	filter.Outcome = strings.TrimSpace(filter.Outcome)
	filter.RiskLevel = strings.TrimSpace(filter.RiskLevel)
	return s.repository.ListAuditEvents(ctx, filter)
}

func (s *AuditService) Append(ctx context.Context, input AppendAuditEventInput) (AuditEvent, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.ActorEmail = strings.TrimSpace(input.ActorEmail)
	input.Action = strings.TrimSpace(input.Action)
	input.ResourceType = strings.TrimSpace(input.ResourceType)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.Outcome = strings.ToLower(strings.TrimSpace(input.Outcome))
	input.RiskLevel = strings.ToLower(strings.TrimSpace(input.RiskLevel))
	input.Source = strings.TrimSpace(input.Source)
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.IPAddress = strings.TrimSpace(input.IPAddress)
	input.UserAgent = strings.TrimSpace(input.UserAgent)
	if input.ActorID == "" || input.ActorEmail == "" || input.Action == "" || input.ResourceType == "" || input.ResourceID == "" {
		return AuditEvent{}, &ValidationError{Code: "audit_identity_required", Message: "Audit actor, action, and resource identity are required."}
	}
	if !slices.Contains([]string{"success", "failure", "denied"}, input.Outcome) {
		return AuditEvent{}, &ValidationError{Code: "audit_outcome_invalid", Message: "Audit outcome must be success, failure, or denied."}
	}
	if !slices.Contains([]string{"low", "medium", "high", "critical"}, input.RiskLevel) {
		return AuditEvent{}, &ValidationError{Code: "audit_risk_invalid", Message: "Audit risk level is invalid."}
	}
	if input.IPAddress != "" && net.ParseIP(input.IPAddress) == nil {
		return AuditEvent{}, &ValidationError{Code: "audit_ip_invalid", Message: "Audit IP address is invalid."}
	}
	if !validAuditState(input.BeforeState) || !validAuditState(input.AfterState) {
		return AuditEvent{}, &ValidationError{Code: "audit_state_invalid", Message: "Audit before/after state must each be no larger than 64 KiB."}
	}
	id, err := randomIdentifier("audit_", 12)
	if err != nil {
		return AuditEvent{}, err
	}
	previous, err := s.repository.LatestAuditHash(ctx, input.OrganizationID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return AuditEvent{}, err
	}
	event := AuditEvent{
		ID: id, OrganizationID: input.OrganizationID, ActorID: input.ActorID, ActorEmail: input.ActorEmail,
		Action: input.Action, ResourceType: input.ResourceType, ResourceID: input.ResourceID,
		Outcome: input.Outcome, RiskLevel: input.RiskLevel, Source: input.Source, Reason: input.Reason,
		RequestID: input.RequestID, IPAddress: input.IPAddress, UserAgent: input.UserAgent,
		BeforeState: cloneAnyMap(input.BeforeState), AfterState: cloneAnyMap(input.AfterState),
		PreviousHash: previous, CreatedAt: s.now().UTC(),
	}
	event.IntegrityHash, err = calculateAuditHash(event)
	if err != nil {
		return AuditEvent{}, err
	}
	return s.repository.AppendAuditEvent(ctx, event)
}

func (s *AuditService) Verify(ctx context.Context, organizationID string) (AuditIntegrityResult, error) {
	events, err := s.repository.ListAuditEvents(ctx, AuditFilter{OrganizationID: defaultOrganization(organizationID)})
	if err != nil {
		return AuditIntegrityResult{}, err
	}
	slices.Reverse(events)
	previous := ""
	result := AuditIntegrityResult{Valid: true, EventCount: len(events)}
	for _, event := range events {
		expected, err := calculateAuditHash(event)
		if err != nil {
			return AuditIntegrityResult{}, err
		}
		if event.PreviousHash != previous || event.IntegrityHash != expected {
			result.Valid = false
			result.FirstInvalidID = event.ID
			return result, nil
		}
		previous = event.IntegrityHash
		result.HeadHash = previous
	}
	return result, nil
}

func (s *AuditService) Retention(ctx context.Context, organizationID string) (AuditRetentionPolicy, error) {
	organizationID = defaultOrganization(organizationID)
	policy, err := s.repository.GetAuditRetentionPolicy(ctx, organizationID)
	if errors.Is(err, ErrNotFound) {
		return AuditRetentionPolicy{OrganizationID: organizationID, RetentionDays: 365, ExportFormat: "csv", UpdatedBy: "system", UpdatedAt: s.now().UTC()}, nil
	}
	return policy, err
}

func (s *AuditService) UpsertRetention(ctx context.Context, input UpsertAuditRetentionInput) (AuditRetentionPolicy, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.ExportFormat = strings.ToLower(strings.TrimSpace(input.ExportFormat))
	input.UpdatedBy = strings.TrimSpace(input.UpdatedBy)
	if input.RetentionDays < 30 || input.RetentionDays > 2555 {
		return AuditRetentionPolicy{}, &ValidationError{Code: "audit_retention_invalid", Message: "Audit retention must be between 30 days and seven years."}
	}
	if input.ExportFormat != "csv" && input.ExportFormat != "jsonl" {
		return AuditRetentionPolicy{}, &ValidationError{Code: "audit_export_format_invalid", Message: "Audit export format must be csv or jsonl."}
	}
	if input.UpdatedBy == "" {
		input.UpdatedBy = "holden@topoai.dev"
	}
	return s.repository.UpsertAuditRetentionPolicy(ctx, AuditRetentionPolicy{
		OrganizationID: input.OrganizationID, RetentionDays: input.RetentionDays, LegalHold: input.LegalHold,
		ExportFormat: input.ExportFormat, UpdatedBy: input.UpdatedBy, UpdatedAt: s.now().UTC(),
	})
}

func (s *AuditService) ListExports(ctx context.Context, filter AuditExportFilter) ([]AuditExport, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Format = strings.TrimSpace(filter.Format)
	return s.repository.ListAuditExports(ctx, filter)
}

func (s *AuditService) QueueExport(ctx context.Context, input QueueAuditExportInput) (AuditExport, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.RequestedBy = strings.TrimSpace(input.RequestedBy)
	input.Format = strings.ToLower(strings.TrimSpace(input.Format))
	if input.RequestedBy == "" {
		input.RequestedBy = "holden@topoai.dev"
	}
	if input.Format != "csv" && input.Format != "jsonl" {
		return AuditExport{}, &ValidationError{Code: "audit_export_format_invalid", Message: "Audit export format must be csv or jsonl."}
	}
	start, startErr := time.Parse(time.RFC3339, strings.TrimSpace(input.PeriodStart))
	end, endErr := time.Parse(time.RFC3339, strings.TrimSpace(input.PeriodEnd))
	now := s.now().UTC()
	if startErr != nil || endErr != nil || !start.Before(end) || end.Sub(start) > 366*24*time.Hour || end.After(now.Add(5*time.Minute)) {
		return AuditExport{}, &ValidationError{Code: "audit_export_period_invalid", Message: "Audit export period must be ordered, no longer than 366 days, and not in the future."}
	}
	if len(input.Filters) > 10 {
		return AuditExport{}, &ValidationError{Code: "audit_export_filters_invalid", Message: "Audit export can have at most 10 filters."}
	}
	id, err := randomIdentifier("aexp_", 10)
	if err != nil {
		return AuditExport{}, err
	}
	return s.repository.CreateAuditExport(ctx, AuditExport{
		ID: id, OrganizationID: input.OrganizationID, RequestedBy: input.RequestedBy, Format: input.Format,
		Status: "queued", Filters: mapsCloneString(input.Filters), PeriodStart: start.UTC(), PeriodEnd: end.UTC(), CreatedAt: now,
	})
}

func (s *AuditService) RetryExport(ctx context.Context, organizationID, id, requestedBy string) (AuditExport, error) {
	organizationID = defaultOrganization(organizationID)
	export, err := s.repository.GetAuditExport(ctx, organizationID, strings.TrimSpace(id))
	if err != nil {
		return AuditExport{}, err
	}
	if export.Status != "failed" {
		return AuditExport{}, &ValidationError{Code: "audit_export_retry_invalid", Message: "Only failed audit exports can be retried."}
	}
	newID, err := randomIdentifier("aexp_", 10)
	if err != nil {
		return AuditExport{}, err
	}
	requestedBy = strings.TrimSpace(requestedBy)
	if requestedBy == "" {
		requestedBy = "holden@topoai.dev"
	}
	now := s.now().UTC()
	return s.repository.CreateAuditExport(ctx, AuditExport{
		ID: newID, OrganizationID: export.OrganizationID, RequestedBy: requestedBy, Format: export.Format,
		Status: "queued", Filters: mapsCloneString(export.Filters), PeriodStart: export.PeriodStart,
		PeriodEnd: export.PeriodEnd, ParentID: &export.ID, CreatedAt: now,
	})
}

func calculateAuditHash(event AuditEvent) (string, error) {
	payload := struct {
		ID, OrganizationID, ActorID, ActorEmail, Action, ResourceType, ResourceID string
		Outcome, RiskLevel, Source, Reason, RequestID, IPAddress, UserAgent       string
		BeforeState, AfterState                                                   map[string]any
		PreviousHash                                                              string
		CreatedAt                                                                 time.Time
	}{event.ID, event.OrganizationID, event.ActorID, event.ActorEmail, event.Action, event.ResourceType, event.ResourceID,
		event.Outcome, event.RiskLevel, event.Source, event.Reason, event.RequestID, event.IPAddress, event.UserAgent,
		event.BeforeState, event.AfterState, event.PreviousHash, event.CreatedAt.UTC()}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validAuditState(value map[string]any) bool {
	encoded, err := json.Marshal(value)
	return err == nil && len(encoded) <= 64*1024
}

func cloneAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	encoded, _ := json.Marshal(value)
	var clone map[string]any
	_ = json.Unmarshal(encoded, &clone)
	return clone
}

func mapsCloneString(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[strings.TrimSpace(key)] = strings.TrimSpace(item)
	}
	return result
}
