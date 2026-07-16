package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListReports(_ context.Context, filter ReportFilter) ([]ReportSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	items := make([]ReportSchedule, 0)
	for _, report := range r.reports {
		if report.OrganizationID != filter.OrganizationID ||
			filter.Status != "" && filter.Status != "all" && report.Status != filter.Status ||
			filter.Template != "" && filter.Template != "all" && report.Template != filter.Template {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(report.Name+" "+report.Template+" "+report.Frequency+" "+report.Timezone), query) {
			continue
		}
		items = append(items, cloneReport(report))
	}
	return items, nil
}

func (r *MemoryRepository) GetReport(_ context.Context, organizationID, id string) (ReportSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, report := range r.reports {
		if report.OrganizationID == organizationID && report.ID == id {
			return cloneReport(report), nil
		}
	}
	return ReportSchedule{}, ErrNotFound
}

func (r *MemoryRepository) CreateReport(_ context.Context, report ReportSchedule) (ReportSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, report.OrganizationID); !found {
		return ReportSchedule{}, ErrNotFound
	}
	if slices.ContainsFunc(r.reports, func(existing ReportSchedule) bool {
		return existing.OrganizationID == report.OrganizationID && strings.EqualFold(existing.Name, report.Name)
	}) {
		return ReportSchedule{}, ErrConflict
	}
	r.reports = append([]ReportSchedule{cloneReport(report)}, r.reports...)
	return cloneReport(report), nil
}

func (r *MemoryRepository) UpdateReportStatus(_ context.Context, organizationID, id, status string, nextRunAt *time.Time, updatedAt time.Time) (ReportSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.reports {
		if r.reports[index].OrganizationID == organizationID && r.reports[index].ID == id {
			r.reports[index].Status = status
			r.reports[index].NextRunAt = nextRunAt
			r.reports[index].UpdatedAt = updatedAt
			return cloneReport(r.reports[index]), nil
		}
	}
	return ReportSchedule{}, ErrNotFound
}

func (r *MemoryRepository) ListReportRuns(_ context.Context, filter ReportRunFilter) ([]ReportRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ReportRun, 0)
	for _, run := range r.reportRuns {
		if run.OrganizationID != filter.OrganizationID ||
			filter.ReportID != "" && run.ReportID != filter.ReportID ||
			filter.Status != "" && filter.Status != "all" && run.Status != filter.Status ||
			filter.Trigger != "" && filter.Trigger != "all" && run.Trigger != filter.Trigger {
			continue
		}
		items = append(items, run)
	}
	return items, nil
}

func (r *MemoryRepository) GetReportRun(_ context.Context, organizationID, id string) (ReportRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, run := range r.reportRuns {
		if run.OrganizationID == organizationID && run.ID == id {
			return run, nil
		}
	}
	return ReportRun{}, ErrNotFound
}

func (r *MemoryRepository) CreateReportRun(_ context.Context, run ReportRun) (ReportRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !slices.ContainsFunc(r.reports, func(report ReportSchedule) bool {
		return report.OrganizationID == run.OrganizationID && report.ID == run.ReportID
	}) {
		return ReportRun{}, ErrNotFound
	}
	r.reportRuns = append([]ReportRun{run}, r.reportRuns...)
	return run, nil
}

func developmentReports() []ReportSchedule {
	last := time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC)
	next := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	created := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return []ReportSchedule{
		{
			ID: "report_exec_weekly", OrganizationID: "org_topoai", Name: "Executive weekly summary",
			Template: "executive_summary", Status: "active", Frequency: "weekly", DayOfWeek: "monday",
			LocalTime: "10:00", Timezone: "Asia/Shanghai", Formats: []string{"xlsx", "pdf"},
			Recipients: []ReportRecipient{{Channel: "email", Target: "holden@topoai.dev", DisplayName: "Platform owner"}, {Channel: "slack", Target: "C_FINOPS", DisplayName: "#finops"}},
			Filters:    map[string]string{"environment": "production"}, LastRunAt: &last, NextRunAt: &next,
			CreatedAt: created, UpdatedAt: created,
		},
		{
			ID: "report_finops_monthly", OrganizationID: "org_topoai", Name: "Monthly cost allocation",
			Template: "usage_cost", Status: "paused", Frequency: "monthly", DayOfMonth: 1,
			LocalTime: "09:00", Timezone: "Asia/Shanghai", Formats: []string{"csv", "xlsx"},
			Recipients: []ReportRecipient{{Channel: "email", Target: "finance@topoai.dev", DisplayName: "Finance"}},
			Filters:    map[string]string{"cost_center": "all"}, IncludeRawData: true,
			CreatedAt: created, UpdatedAt: created,
		},
	}
}

func developmentReportRuns() []ReportRun {
	started := time.Date(2026, 7, 13, 2, 0, 2, 0, time.UTC)
	completed := time.Date(2026, 7, 13, 2, 0, 18, 0, time.UTC)
	failedAt := time.Date(2026, 7, 6, 2, 0, 9, 0, time.UTC)
	return []ReportRun{
		{
			ID: "rrun_exec_success", OrganizationID: "org_topoai", ReportID: "report_exec_weekly",
			ReportName: "Executive weekly summary", Status: "succeeded", Trigger: "schedule", Attempt: 1,
			RequestedBy: "scheduler", ScheduledFor: time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC),
			PeriodStart: time.Date(2026, 7, 6, 2, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC),
			StartedAt: &started, CompletedAt: &completed, ArtifactCount: 2, RowCount: 182440,
			SizeBytes: 2483200, DeliveryStatus: "delivered", CreatedAt: time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC),
		},
		{
			ID: "rrun_exec_failed", OrganizationID: "org_topoai", ReportID: "report_exec_weekly",
			ReportName: "Executive weekly summary", Status: "failed", Trigger: "schedule", Attempt: 1,
			RequestedBy: "scheduler", ScheduledFor: time.Date(2026, 7, 6, 2, 0, 0, 0, time.UTC),
			PeriodStart: time.Date(2026, 6, 29, 2, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 7, 6, 2, 0, 0, 0, time.UTC),
			CompletedAt: &failedAt, DeliveryStatus: "failed", ErrorMessage: "Object storage upload timed out before delivery.",
			CreatedAt: time.Date(2026, 7, 6, 2, 0, 0, 0, time.UTC),
		},
	}
}

var _ ReportRepository = (*MemoryRepository)(nil)
