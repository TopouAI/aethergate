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

func (r *Repository) ListReports(ctx context.Context, filter platform.ReportFilter) ([]platform.ReportSchedule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id,organization_id,name,template,status,frequency,day_of_week,day_of_month,
		       local_time,timezone,formats,recipients,filters,include_raw_data,last_run_at,
		       next_run_at,created_at,updated_at
		FROM report_schedules
		WHERE organization_id=$1 AND deleted_at IS NULL
		  AND ($2='' OR $2='all' OR status=$2)
		  AND ($3='' OR $3='all' OR template=$3)
		  AND ($4='' OR lower(name) LIKE '%'||lower($4)||'%' OR lower(template) LIKE '%'||lower($4)||'%'
		       OR lower(frequency) LIKE '%'||lower($4)||'%' OR lower(timezone) LIKE '%'||lower($4)||'%')
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Status, filter.Template, filter.Query)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()
	items := make([]platform.ReportSchedule, 0)
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, report)
	}
	return items, rows.Err()
}

func (r *Repository) GetReport(ctx context.Context, organizationID, id string) (platform.ReportSchedule, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id,organization_id,name,template,status,frequency,day_of_week,day_of_month,
		       local_time,timezone,formats,recipients,filters,include_raw_data,last_run_at,
		       next_run_at,created_at,updated_at
		FROM report_schedules
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ReportSchedule{}, platform.ErrNotFound
	}
	return report, err
}

func (r *Repository) CreateReport(ctx context.Context, report platform.ReportSchedule) (platform.ReportSchedule, error) {
	recipients, err := json.Marshal(report.Recipients)
	if err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("marshal report recipients: %w", err)
	}
	filters, err := json.Marshal(report.Filters)
	if err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("marshal report filters: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO report_schedules(
			id,organization_id,name,template,status,frequency,day_of_week,day_of_month,
			local_time,timezone,formats,recipients,filters,include_raw_data,last_run_at,
			next_run_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		report.ID, report.OrganizationID, report.Name, report.Template, report.Status, report.Frequency,
		report.DayOfWeek, report.DayOfMonth, report.LocalTime, report.Timezone, report.Formats,
		recipients, filters, report.IncludeRawData, report.LastRunAt, report.NextRunAt,
		report.CreatedAt, report.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.ReportSchedule{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.ReportSchedule{}, platform.ErrConflict
	}
	if err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("create report: %w", err)
	}
	return report, nil
}

func (r *Repository) UpdateReportStatus(ctx context.Context, organizationID, id, status string, nextRunAt *time.Time, updatedAt time.Time) (platform.ReportSchedule, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE report_schedules SET status=$3,next_run_at=$4,updated_at=$5
		WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`,
		organizationID, id, status, nextRunAt, updatedAt)
	if err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("update report status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return platform.ReportSchedule{}, platform.ErrNotFound
	}
	return r.GetReport(ctx, organizationID, id)
}

func (r *Repository) ListReportRuns(ctx context.Context, filter platform.ReportRunFilter) ([]platform.ReportRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT x.id,x.organization_id,x.report_id,r.name,x.status,x.trigger_type,x.attempt,x.requested_by,
		       x.scheduled_for,x.period_start,x.period_end,x.started_at,x.completed_at,x.artifact_count,
		       x.row_count,x.size_bytes,x.delivery_status,x.error_message,x.parent_run_id,x.created_at
		FROM report_runs x JOIN report_schedules r ON r.id=x.report_id
		WHERE x.organization_id=$1
		  AND ($2='' OR x.report_id=$2)
		  AND ($3='' OR $3='all' OR x.status=$3)
		  AND ($4='' OR $4='all' OR x.trigger_type=$4)
		ORDER BY x.created_at DESC`, filter.OrganizationID, filter.ReportID, filter.Status, filter.Trigger)
	if err != nil {
		return nil, fmt.Errorf("list report runs: %w", err)
	}
	defer rows.Close()
	items := make([]platform.ReportRun, 0)
	for rows.Next() {
		run, err := scanReportRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, run)
	}
	return items, rows.Err()
}

func (r *Repository) GetReportRun(ctx context.Context, organizationID, id string) (platform.ReportRun, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT x.id,x.organization_id,x.report_id,r.name,x.status,x.trigger_type,x.attempt,x.requested_by,
		       x.scheduled_for,x.period_start,x.period_end,x.started_at,x.completed_at,x.artifact_count,
		       x.row_count,x.size_bytes,x.delivery_status,x.error_message,x.parent_run_id,x.created_at
		FROM report_runs x JOIN report_schedules r ON r.id=x.report_id
		WHERE x.organization_id=$1 AND x.id=$2`, organizationID, id)
	run, err := scanReportRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ReportRun{}, platform.ErrNotFound
	}
	return run, err
}

func (r *Repository) CreateReportRun(ctx context.Context, run platform.ReportRun) (platform.ReportRun, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO report_runs(
			id,organization_id,report_id,status,trigger_type,attempt,requested_by,scheduled_for,
			period_start,period_end,started_at,completed_at,artifact_count,row_count,size_bytes,
			delivery_status,error_message,parent_run_id,created_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		run.ID, run.OrganizationID, run.ReportID, run.Status, run.Trigger, run.Attempt, run.RequestedBy,
		run.ScheduledFor, run.PeriodStart, run.PeriodEnd, run.StartedAt, run.CompletedAt,
		run.ArtifactCount, run.RowCount, run.SizeBytes, run.DeliveryStatus, run.ErrorMessage,
		run.ParentRunID, run.CreatedAt)
	if isForeignKeyViolation(err) {
		return platform.ReportRun{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.ReportRun{}, platform.ErrConflict
	}
	if err != nil {
		return platform.ReportRun{}, fmt.Errorf("create report run: %w", err)
	}
	return run, nil
}

func scanReport(row rowScanner) (platform.ReportSchedule, error) {
	var report platform.ReportSchedule
	var recipients []byte
	var filters []byte
	err := row.Scan(
		&report.ID, &report.OrganizationID, &report.Name, &report.Template, &report.Status,
		&report.Frequency, &report.DayOfWeek, &report.DayOfMonth, &report.LocalTime, &report.Timezone,
		&report.Formats, &recipients, &filters, &report.IncludeRawData, &report.LastRunAt,
		&report.NextRunAt, &report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		return platform.ReportSchedule{}, err
	}
	if err := json.Unmarshal(recipients, &report.Recipients); err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("decode report recipients: %w", err)
	}
	if err := json.Unmarshal(filters, &report.Filters); err != nil {
		return platform.ReportSchedule{}, fmt.Errorf("decode report filters: %w", err)
	}
	return report, nil
}

func scanReportRun(row rowScanner) (platform.ReportRun, error) {
	var run platform.ReportRun
	err := row.Scan(
		&run.ID, &run.OrganizationID, &run.ReportID, &run.ReportName, &run.Status, &run.Trigger,
		&run.Attempt, &run.RequestedBy, &run.ScheduledFor, &run.PeriodStart, &run.PeriodEnd,
		&run.StartedAt, &run.CompletedAt, &run.ArtifactCount, &run.RowCount, &run.SizeBytes,
		&run.DeliveryStatus, &run.ErrorMessage, &run.ParentRunID, &run.CreatedAt,
	)
	return run, err
}

var _ platform.ReportRepository = (*Repository)(nil)
