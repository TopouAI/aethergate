package platform

import (
	"context"
	"fmt"
	"maps"
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	reportTemplates   = []string{"executive_summary", "usage_cost", "reliability", "adoption", "raw_export"}
	reportFrequencies = []string{"daily", "weekly", "monthly"}
	reportFormats     = []string{"csv", "xlsx", "pdf"}
	reportChannels    = []string{"email", "slack"}
	weekdays          = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
)

type ReportRecipient struct {
	Channel     string `json:"channel"`
	Target      string `json:"target"`
	DisplayName string `json:"displayName"`
}

type ReportSchedule struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organizationId"`
	Name           string            `json:"name"`
	Template       string            `json:"template"`
	Status         string            `json:"status"`
	Frequency      string            `json:"frequency"`
	DayOfWeek      string            `json:"dayOfWeek"`
	DayOfMonth     int               `json:"dayOfMonth"`
	LocalTime      string            `json:"localTime"`
	Timezone       string            `json:"timezone"`
	Formats        []string          `json:"formats"`
	Recipients     []ReportRecipient `json:"recipients"`
	Filters        map[string]string `json:"filters"`
	IncludeRawData bool              `json:"includeRawData"`
	LastRunAt      *time.Time        `json:"lastRunAt"`
	NextRunAt      *time.Time        `json:"nextRunAt"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type ReportRun struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	ReportID       string     `json:"reportId"`
	ReportName     string     `json:"reportName"`
	Status         string     `json:"status"`
	Trigger        string     `json:"trigger"`
	Attempt        int        `json:"attempt"`
	RequestedBy    string     `json:"requestedBy"`
	ScheduledFor   time.Time  `json:"scheduledFor"`
	PeriodStart    time.Time  `json:"periodStart"`
	PeriodEnd      time.Time  `json:"periodEnd"`
	StartedAt      *time.Time `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt"`
	ArtifactCount  int        `json:"artifactCount"`
	RowCount       int64      `json:"rowCount"`
	SizeBytes      int64      `json:"sizeBytes"`
	DeliveryStatus string     `json:"deliveryStatus"`
	ErrorMessage   string     `json:"errorMessage"`
	ParentRunID    *string    `json:"parentRunId"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type ReportFilter struct {
	OrganizationID string
	Query          string
	Status         string
	Template       string
}

type ReportRunFilter struct {
	OrganizationID string
	ReportID       string
	Status         string
	Trigger        string
}

type CreateReportInput struct {
	OrganizationID string            `json:"organizationId"`
	Name           string            `json:"name"`
	Template       string            `json:"template"`
	Status         string            `json:"status"`
	Frequency      string            `json:"frequency"`
	DayOfWeek      string            `json:"dayOfWeek"`
	DayOfMonth     int               `json:"dayOfMonth"`
	LocalTime      string            `json:"localTime"`
	Timezone       string            `json:"timezone"`
	Formats        []string          `json:"formats"`
	Recipients     []ReportRecipient `json:"recipients"`
	Filters        map[string]string `json:"filters"`
	IncludeRawData bool              `json:"includeRawData"`
}

type QueueReportRunInput struct {
	OrganizationID string  `json:"organizationId"`
	RequestedBy    string  `json:"requestedBy"`
	PeriodStart    *string `json:"periodStart"`
	PeriodEnd      *string `json:"periodEnd"`
}

type ReportRepository interface {
	Repository
	ListReports(context.Context, ReportFilter) ([]ReportSchedule, error)
	GetReport(context.Context, string, string) (ReportSchedule, error)
	CreateReport(context.Context, ReportSchedule) (ReportSchedule, error)
	UpdateReportStatus(context.Context, string, string, string, *time.Time, time.Time) (ReportSchedule, error)
	ListReportRuns(context.Context, ReportRunFilter) ([]ReportRun, error)
	GetReportRun(context.Context, string, string) (ReportRun, error)
	CreateReportRun(context.Context, ReportRun) (ReportRun, error)
}

type ReportService struct {
	repository ReportRepository
	now        func() time.Time
}

func NewReportService(repository ReportRepository) *ReportService {
	return &ReportService{repository: repository, now: time.Now}
}

func (s *ReportService) List(ctx context.Context, filter ReportFilter) ([]ReportSchedule, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Template = strings.TrimSpace(filter.Template)
	return s.repository.ListReports(ctx, filter)
}

func (s *ReportService) ListRuns(ctx context.Context, filter ReportRunFilter) ([]ReportRun, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.ReportID = strings.TrimSpace(filter.ReportID)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Trigger = strings.TrimSpace(filter.Trigger)
	return s.repository.ListReportRuns(ctx, filter)
}

func (s *ReportService) Create(ctx context.Context, input CreateReportInput) (ReportSchedule, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.Template = strings.TrimSpace(input.Template)
	input.Status = strings.TrimSpace(input.Status)
	input.Frequency = strings.TrimSpace(input.Frequency)
	input.DayOfWeek = strings.ToLower(strings.TrimSpace(input.DayOfWeek))
	input.LocalTime = strings.TrimSpace(input.LocalTime)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.Name == "" {
		return ReportSchedule{}, &ValidationError{Code: "report_name_required", Message: "Report name is required."}
	}
	if !slices.Contains(reportTemplates, input.Template) {
		return ReportSchedule{}, &ValidationError{Code: "report_template_invalid", Message: "Report template is invalid."}
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "paused" {
		return ReportSchedule{}, &ValidationError{Code: "report_status_invalid", Message: "Report status must be active or paused."}
	}
	if !slices.Contains(reportFrequencies, input.Frequency) {
		return ReportSchedule{}, &ValidationError{Code: "report_frequency_invalid", Message: "Report frequency must be daily, weekly, or monthly."}
	}
	if input.LocalTime == "" {
		input.LocalTime = "10:00"
	}
	if _, err := time.Parse("15:04", input.LocalTime); err != nil {
		return ReportSchedule{}, &ValidationError{Code: "report_time_invalid", Message: "Report local time must use HH:MM in 24-hour format."}
	}
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return ReportSchedule{}, &ValidationError{Code: "report_timezone_invalid", Message: "Report timezone must be a valid IANA timezone."}
	}
	if input.Frequency == "weekly" {
		if !slices.Contains(weekdays, input.DayOfWeek) {
			return ReportSchedule{}, &ValidationError{Code: "report_weekday_invalid", Message: "Weekly reports require a valid day of week."}
		}
	} else {
		input.DayOfWeek = ""
	}
	if input.Frequency == "monthly" {
		if input.DayOfMonth < 1 || input.DayOfMonth > 28 {
			return ReportSchedule{}, &ValidationError{Code: "report_month_day_invalid", Message: "Monthly reports must run on day 1 through 28."}
		}
	} else {
		input.DayOfMonth = 0
	}

	formats, err := normalizeReportFormats(input.Formats)
	if err != nil {
		return ReportSchedule{}, err
	}
	recipients, err := normalizeReportRecipients(input.Recipients)
	if err != nil {
		return ReportSchedule{}, err
	}
	filters, err := normalizeReportFilters(input.Filters)
	if err != nil {
		return ReportSchedule{}, err
	}
	id, err := randomIdentifier("report_", 9)
	if err != nil {
		return ReportSchedule{}, err
	}
	now := s.now().UTC()
	report := ReportSchedule{
		ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Template: input.Template,
		Status: input.Status, Frequency: input.Frequency, DayOfWeek: input.DayOfWeek,
		DayOfMonth: input.DayOfMonth, LocalTime: input.LocalTime, Timezone: input.Timezone,
		Formats: formats, Recipients: recipients, Filters: filters, IncludeRawData: input.IncludeRawData,
		CreatedAt: now, UpdatedAt: now,
	}
	if report.Status == "active" {
		next, err := nextReportRun(now, report)
		if err != nil {
			return ReportSchedule{}, err
		}
		report.NextRunAt = &next
	}
	return s.repository.CreateReport(ctx, report)
}

func (s *ReportService) Activate(ctx context.Context, organizationID, id string) (ReportSchedule, error) {
	organizationID = defaultOrganization(organizationID)
	report, err := s.repository.GetReport(ctx, organizationID, strings.TrimSpace(id))
	if err != nil {
		return ReportSchedule{}, err
	}
	now := s.now().UTC()
	next, err := nextReportRun(now, report)
	if err != nil {
		return ReportSchedule{}, err
	}
	return s.repository.UpdateReportStatus(ctx, organizationID, report.ID, "active", &next, now)
}

func (s *ReportService) Pause(ctx context.Context, organizationID, id string) (ReportSchedule, error) {
	return s.repository.UpdateReportStatus(ctx, defaultOrganization(organizationID), strings.TrimSpace(id), "paused", nil, s.now().UTC())
}

func (s *ReportService) QueueRun(ctx context.Context, reportID string, input QueueReportRunInput) (ReportRun, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	report, err := s.repository.GetReport(ctx, input.OrganizationID, strings.TrimSpace(reportID))
	if err != nil {
		return ReportRun{}, err
	}
	input.RequestedBy = strings.TrimSpace(input.RequestedBy)
	if input.RequestedBy == "" {
		input.RequestedBy = "holden@topoai.dev"
	}
	now := s.now().UTC()
	start, end, err := reportPeriod(report.Frequency, now, input.PeriodStart, input.PeriodEnd)
	if err != nil {
		return ReportRun{}, err
	}
	return s.createRun(ctx, report, "manual", 1, input.RequestedBy, start, end, nil, now)
}

func (s *ReportService) RetryRun(ctx context.Context, organizationID, runID, requestedBy string) (ReportRun, error) {
	organizationID = defaultOrganization(organizationID)
	run, err := s.repository.GetReportRun(ctx, organizationID, strings.TrimSpace(runID))
	if err != nil {
		return ReportRun{}, err
	}
	if run.Status != "failed" {
		return ReportRun{}, &ValidationError{Code: "report_retry_invalid", Message: "Only failed report runs can be retried."}
	}
	report, err := s.repository.GetReport(ctx, organizationID, run.ReportID)
	if err != nil {
		return ReportRun{}, err
	}
	requestedBy = strings.TrimSpace(requestedBy)
	if requestedBy == "" {
		requestedBy = "holden@topoai.dev"
	}
	now := s.now().UTC()
	return s.createRun(ctx, report, "retry", run.Attempt+1, requestedBy, run.PeriodStart, run.PeriodEnd, &run.ID, now)
}

func (s *ReportService) createRun(ctx context.Context, report ReportSchedule, trigger string, attempt int, requestedBy string, periodStart, periodEnd time.Time, parentRunID *string, now time.Time) (ReportRun, error) {
	id, err := randomIdentifier("rrun_", 10)
	if err != nil {
		return ReportRun{}, err
	}
	return s.repository.CreateReportRun(ctx, ReportRun{
		ID: id, OrganizationID: report.OrganizationID, ReportID: report.ID, ReportName: report.Name,
		Status: "queued", Trigger: trigger, Attempt: attempt, RequestedBy: requestedBy,
		ScheduledFor: now, PeriodStart: periodStart, PeriodEnd: periodEnd,
		DeliveryStatus: "pending", ParentRunID: parentRunID, CreatedAt: now,
	})
}

func normalizeReportFormats(values []string) ([]string, error) {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || slices.Contains(items, value) {
			continue
		}
		if !slices.Contains(reportFormats, value) {
			return nil, &ValidationError{Code: "report_format_invalid", Message: "Report formats must be csv, xlsx, or pdf."}
		}
		items = append(items, value)
	}
	if len(items) == 0 {
		return nil, &ValidationError{Code: "report_formats_required", Message: "At least one report format is required."}
	}
	return items, nil
}

func normalizeReportRecipients(values []ReportRecipient) ([]ReportRecipient, error) {
	items := make([]ReportRecipient, 0, len(values))
	seen := make(map[string]struct{})
	for _, recipient := range values {
		recipient.Channel = strings.ToLower(strings.TrimSpace(recipient.Channel))
		recipient.Target = strings.TrimSpace(recipient.Target)
		recipient.DisplayName = strings.TrimSpace(recipient.DisplayName)
		if !slices.Contains(reportChannels, recipient.Channel) || recipient.Target == "" {
			return nil, &ValidationError{Code: "report_recipient_invalid", Message: "Each report recipient requires an email or Slack channel and a target."}
		}
		if recipient.Channel == "email" {
			address, err := mail.ParseAddress(recipient.Target)
			if err != nil || !strings.EqualFold(address.Address, recipient.Target) {
				return nil, &ValidationError{Code: "report_recipient_invalid", Message: "One or more report email recipients are invalid."}
			}
		}
		key := recipient.Channel + "\x00" + strings.ToLower(recipient.Target)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, recipient)
	}
	if len(items) == 0 {
		return nil, &ValidationError{Code: "report_recipients_required", Message: "At least one report recipient is required."}
	}
	if len(items) > 50 {
		return nil, &ValidationError{Code: "report_recipients_limit", Message: "A report can have at most 50 recipients."}
	}
	return items, nil
}

func normalizeReportFilters(values map[string]string) (map[string]string, error) {
	if len(values) > 20 {
		return nil, &ValidationError{Code: "report_filters_limit", Message: "A report can have at most 20 filters."}
	}
	items := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, &ValidationError{Code: "report_filter_invalid", Message: "Report filter keys and values are required."}
		}
		items[key] = value
	}
	return items, nil
}

func reportPeriod(frequency string, now time.Time, startText, endText *string) (time.Time, time.Time, error) {
	if (startText == nil) != (endText == nil) {
		return time.Time{}, time.Time{}, &ValidationError{Code: "report_period_invalid", Message: "Custom report periods require both start and end."}
	}
	if startText != nil {
		start, startErr := time.Parse(time.RFC3339, strings.TrimSpace(*startText))
		end, endErr := time.Parse(time.RFC3339, strings.TrimSpace(*endText))
		if startErr != nil || endErr != nil || !start.Before(end) || end.Sub(start) > 366*24*time.Hour || end.After(now.Add(5*time.Minute)) {
			return time.Time{}, time.Time{}, &ValidationError{Code: "report_period_invalid", Message: "Report period must be valid, ordered, no longer than 366 days, and not in the future."}
		}
		return start.UTC(), end.UTC(), nil
	}
	start := now.Add(-24 * time.Hour)
	if frequency == "weekly" {
		start = now.Add(-7 * 24 * time.Hour)
	}
	if frequency == "monthly" {
		start = now.AddDate(0, -1, 0)
	}
	return start, now, nil
}

func nextReportRun(now time.Time, report ReportSchedule) (time.Time, error) {
	location, err := time.LoadLocation(report.Timezone)
	if err != nil {
		return time.Time{}, &ValidationError{Code: "report_timezone_invalid", Message: "Report timezone must be a valid IANA timezone."}
	}
	parts := strings.Split(report.LocalTime, ":")
	if len(parts) != 2 {
		return time.Time{}, &ValidationError{Code: "report_time_invalid", Message: "Report local time must use HH:MM in 24-hour format."}
	}
	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, &ValidationError{Code: "report_time_invalid", Message: "Report local time must use HH:MM in 24-hour format."}
	}
	localNow := now.In(location)
	switch report.Frequency {
	case "daily":
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, location)
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate.UTC(), nil
	case "weekly":
		target := slices.Index(weekdays, report.DayOfWeek)
		if target < 0 {
			return time.Time{}, &ValidationError{Code: "report_weekday_invalid", Message: "Weekly reports require a valid day of week."}
		}
		delta := (target - int(localNow.Weekday()) + 7) % 7
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+delta, hour, minute, 0, 0, location)
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		return candidate.UTC(), nil
	case "monthly":
		if report.DayOfMonth < 1 || report.DayOfMonth > 28 {
			return time.Time{}, &ValidationError{Code: "report_month_day_invalid", Message: "Monthly reports must run on day 1 through 28."}
		}
		candidate := time.Date(localNow.Year(), localNow.Month(), report.DayOfMonth, hour, minute, 0, 0, location)
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 1, 0)
		}
		return candidate.UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported report frequency %q", report.Frequency)
	}
}

func cloneReport(report ReportSchedule) ReportSchedule {
	report.Formats = slices.Clone(report.Formats)
	report.Recipients = slices.Clone(report.Recipients)
	report.Filters = maps.Clone(report.Filters)
	return report
}
