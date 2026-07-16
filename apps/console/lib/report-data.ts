import { foundationOrganizationId } from "@/lib/foundation-data";
import type { ReportRun, ReportSchedule, ReportTemplate } from "@/types/report";

export const reportTemplateOptions: Array<{ value: ReportTemplate; label: string; description: string }> = [
  { value: "executive_summary", label: "Executive summary", description: "Cost, traffic, reliability, adoption, and budget posture." },
  { value: "usage_cost", label: "Usage & cost", description: "Detailed token, request, provider, model, and cost allocation." },
  { value: "reliability", label: "Reliability", description: "Latency, errors, provider health, routing, and SLO evidence." },
  { value: "adoption", label: "Adoption", description: "Active projects, teams, users, concentration, and growth." },
  { value: "raw_export", label: "Raw export", description: "Governed request-level export for downstream analysis." },
];

export const seedReports: ReportSchedule[] = [
  {
    id: "report_exec_weekly", organizationId: foundationOrganizationId, name: "Executive weekly summary",
    template: "executive_summary", status: "active", frequency: "weekly", dayOfWeek: "monday", dayOfMonth: 0,
    localTime: "10:00", timezone: "Asia/Shanghai", formats: ["xlsx", "pdf"],
    recipients: [{ channel: "email", target: "holden@topoai.dev", displayName: "Platform owner" }, { channel: "slack", target: "C_FINOPS", displayName: "#finops" }],
    filters: { environment: "production" }, includeRawData: false, lastRunAt: "2026-07-13T02:00:00Z",
    nextRunAt: "2026-07-20T02:00:00Z", createdAt: "2026-04-01T00:00:00Z", updatedAt: "2026-04-01T00:00:00Z",
  },
  {
    id: "report_finops_monthly", organizationId: foundationOrganizationId, name: "Monthly cost allocation",
    template: "usage_cost", status: "paused", frequency: "monthly", dayOfWeek: "", dayOfMonth: 1,
    localTime: "09:00", timezone: "Asia/Shanghai", formats: ["csv", "xlsx"],
    recipients: [{ channel: "email", target: "finance@topoai.dev", displayName: "Finance" }],
    filters: { cost_center: "all" }, includeRawData: true, lastRunAt: null, nextRunAt: null,
    createdAt: "2026-04-01T00:00:00Z", updatedAt: "2026-04-01T00:00:00Z",
  },
];

export const seedReportRuns: ReportRun[] = [
  {
    id: "rrun_exec_success", organizationId: foundationOrganizationId, reportId: "report_exec_weekly",
    reportName: "Executive weekly summary", status: "succeeded", trigger: "schedule", attempt: 1,
    requestedBy: "scheduler", scheduledFor: "2026-07-13T02:00:00Z", periodStart: "2026-07-06T02:00:00Z",
    periodEnd: "2026-07-13T02:00:00Z", startedAt: "2026-07-13T02:00:02Z", completedAt: "2026-07-13T02:00:18Z",
    artifactCount: 2, rowCount: 182440, sizeBytes: 2483200, deliveryStatus: "delivered",
    errorMessage: "", parentRunId: null, createdAt: "2026-07-13T02:00:00Z",
  },
  {
    id: "rrun_exec_failed", organizationId: foundationOrganizationId, reportId: "report_exec_weekly",
    reportName: "Executive weekly summary", status: "failed", trigger: "schedule", attempt: 1,
    requestedBy: "scheduler", scheduledFor: "2026-07-06T02:00:00Z", periodStart: "2026-06-29T02:00:00Z",
    periodEnd: "2026-07-06T02:00:00Z", startedAt: null, completedAt: "2026-07-06T02:00:09Z",
    artifactCount: 0, rowCount: 0, sizeBytes: 0, deliveryStatus: "failed",
    errorMessage: "Object storage upload timed out before delivery.", parentRunId: null, createdAt: "2026-07-06T02:00:00Z",
  },
];
