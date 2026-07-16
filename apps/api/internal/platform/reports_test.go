package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReportScheduleLifecycleAndTimezone(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewReportService(repository)
	now := time.Date(2026, 7, 15, 4, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	report, err := service.Create(context.Background(), CreateReportInput{
		OrganizationID: "org_topoai", Name: "Weekly reliability", Template: "reliability",
		Status: "active", Frequency: "weekly", DayOfWeek: "monday", LocalTime: "10:00",
		Timezone: "Asia/Shanghai", Formats: []string{"xlsx", "csv"},
		Recipients: []ReportRecipient{{Channel: "email", Target: "ops@topoai.dev"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	if report.NextRunAt == nil || !report.NextRunAt.Equal(expected) {
		t.Fatalf("next run=%v want=%v", report.NextRunAt, expected)
	}

	paused, err := service.Pause(context.Background(), "org_topoai", report.ID)
	if err != nil || paused.Status != "paused" || paused.NextRunAt != nil {
		t.Fatalf("pause=%+v err=%v", paused, err)
	}
	active, err := service.Activate(context.Background(), "org_topoai", report.ID)
	if err != nil || active.Status != "active" || active.NextRunAt == nil {
		t.Fatalf("activate=%+v err=%v", active, err)
	}
}

func TestReportValidationAndRunQueue(t *testing.T) {
	service := NewReportService(NewMemoryRepository())
	_, err := service.Create(context.Background(), CreateReportInput{
		Name: "Invalid", Template: "usage_cost", Frequency: "weekly", DayOfWeek: "monday",
		LocalTime: "10:00", Timezone: "UTC", Formats: []string{"xlsx"},
	})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Code != "report_recipients_required" {
		t.Fatalf("validation error=%v", err)
	}

	run, err := service.QueueRun(context.Background(), "report_exec_weekly", QueueReportRunInput{OrganizationID: "org_topoai", RequestedBy: "test@topoai.dev"})
	if err != nil || run.Status != "queued" || run.Trigger != "manual" || run.Attempt != 1 || !run.PeriodStart.Before(run.PeriodEnd) {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	retry, err := service.RetryRun(context.Background(), "org_topoai", "rrun_exec_failed", "test@topoai.dev")
	if err != nil || retry.Status != "queued" || retry.Trigger != "retry" || retry.Attempt != 2 || retry.ParentRunID == nil {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	_, err = service.RetryRun(context.Background(), "org_topoai", "rrun_exec_success", "test@topoai.dev")
	if !errors.As(err, &validation) || validation.Code != "report_retry_invalid" {
		t.Fatalf("retry validation=%v", err)
	}
}
