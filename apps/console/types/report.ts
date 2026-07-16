export type ReportTemplate = "executive_summary" | "usage_cost" | "reliability" | "adoption" | "raw_export";
export type ReportStatus = "active" | "paused";
export type ReportFrequency = "daily" | "weekly" | "monthly";
export type ReportFormat = "csv" | "xlsx" | "pdf";
export type ReportRecipientChannel = "email" | "slack";

export type ReportRecipient = {
  channel: ReportRecipientChannel;
  target: string;
  displayName: string;
};

export type ReportSchedule = {
  id: string;
  organizationId: string;
  name: string;
  template: ReportTemplate;
  status: ReportStatus;
  frequency: ReportFrequency;
  dayOfWeek: string;
  dayOfMonth: number;
  localTime: string;
  timezone: string;
  formats: ReportFormat[];
  recipients: ReportRecipient[];
  filters: Record<string, string>;
  includeRawData: boolean;
  lastRunAt: string | null;
  nextRunAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type ReportRunStatus = "queued" | "running" | "succeeded" | "failed" | "cancelled";
export type ReportRunTrigger = "schedule" | "manual" | "retry";
export type ReportDeliveryStatus = "pending" | "delivered" | "partial" | "failed";

export type ReportRun = {
  id: string;
  organizationId: string;
  reportId: string;
  reportName: string;
  status: ReportRunStatus;
  trigger: ReportRunTrigger;
  attempt: number;
  requestedBy: string;
  scheduledFor: string;
  periodStart: string;
  periodEnd: string;
  startedAt: string | null;
  completedAt: string | null;
  artifactCount: number;
  rowCount: number;
  sizeBytes: number;
  deliveryStatus: ReportDeliveryStatus;
  errorMessage: string;
  parentRunId: string | null;
  createdAt: string;
};
