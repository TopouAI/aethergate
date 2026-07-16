import type { LlmRequest, OverviewMetric, RequestStatus } from "@/types/observability";
import type { APIKeyRecord, Organization } from "@/types/enterprise";
import type { Member, ModelRecord, Project, Workspace, WorkspaceEnvironment } from "@/types/foundation";
import type { ProviderConnection, ProviderHealthEvent, ProviderHealthProbe, ProviderHealthSource } from "@/types/provider";
import type { RoutingPolicy, RoutingStrategy } from "@/types/routing";
import type { RateLimitDecision, RateLimitMetric, RateLimitRule, RateLimitScope } from "@/types/rate-limit";
import type { Budget, BudgetDecision, BudgetScope } from "@/types/budget";
import type { AlertEvaluation, AlertIncident, AlertMetric, AlertRule } from "@/types/alert";
import type { WebhookDelivery, WebhookEndpoint, WebhookEventType, WebhookPropertyFilter } from "@/types/webhook";
import type { ReportFormat, ReportFrequency, ReportRecipient, ReportRun, ReportSchedule, ReportStatus, ReportTemplate } from "@/types/report";
import type { InboxNotification, NotificationCategory, NotificationChannel, NotificationDelivery, NotificationDestination,
  NotificationDigest, NotificationEscalationEvaluation, NotificationEscalationPolicy, NotificationEscalationRoute,
  NotificationPreference, NotificationSeverity } from "@/types/notification";
import type { AuditEvent, AuditExport, AuditIntegrityResult, AuditOutcome, AuditRetentionPolicy, AuditRisk } from "@/types/audit";
import type { VaultAccessEvent, VaultScopeType, VaultSecret, VaultSecretKind } from "@/types/vault";
import type { LiteLLMIntegrationStatus } from "@/types/integration";

const API_BASE_URL = (process.env.NEXT_PUBLIC_AETHERGATE_API_URL ?? "http://localhost:8080/api/v1").replace(/\/$/, "");

type DataResponse<T> = { data: T };
type ListResponse<T> = { data: T[]; meta: { count: number; total?: number; source: string } };

export class ControlPlaneError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string) {
    super(message);
    this.name = "ControlPlaneError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: { Accept: "application/json", "Content-Type": "application/json", ...init?.headers },
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => null) as { error?: { code?: string; message?: string } } | null;
    throw new ControlPlaneError(response.status, payload?.error?.code ?? "request_failed", payload?.error?.message ?? `Control-plane request failed with ${response.status}.`);
  }
  return response.json() as Promise<T>;
}

function queryString(values: Record<string, string | undefined>) {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => { if (value) params.set(key, value); });
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

export async function getOverview(signal?: AbortSignal) {
  return request<DataResponse<{ metrics: OverviewMetric[]; source: string }>>("/overview", { signal });
}

export async function listRequests(filters: { q?: string; status?: "all" | RequestStatus; project?: string }, signal?: AbortSignal) {
  return request<ListResponse<LlmRequest>>(`/requests${queryString(filters)}`, { signal });
}

export async function listOrganizations(filters: { q?: string; status?: string } = {}, signal?: AbortSignal) {
  return request<ListResponse<Organization>>(`/organizations${queryString(filters)}`, { signal });
}

export async function createOrganization(input: Pick<Organization, "name" | "slug" | "plan" | "region" | "owner">) {
  return request<DataResponse<Organization>>("/organizations", { method: "POST", body: JSON.stringify(input) });
}

export async function listAPIKeys(filters: { organizationId?: string; q?: string; status?: string } = {}, signal?: AbortSignal) {
  return request<ListResponse<APIKeyRecord>>(`/api-keys${queryString(filters)}`, { signal });
}

export type CreateAPIKeyInput = {
  organizationId: string;
  name: string;
  project: string;
  models: string[];
  rpm: number;
  tpm: number;
  createdBy: string;
  expiresAt: string | null;
};

export async function createAPIKey(input: CreateAPIKeyInput) {
  return request<{ data: APIKeyRecord; secret: string; meta: { secretVisibility: "one-time" } }>("/api-keys", { method: "POST", body: JSON.stringify(input) });
}

export async function revokeAPIKey(id: string) {
  return request<DataResponse<APIKeyRecord>>(`/api-keys/${encodeURIComponent(id)}/revoke`, { method: "POST", headers: { "X-AetherGate-Actor": "holden@topoai.dev" } });
}

export async function listWorkspaces(organizationId: string, signal?: AbortSignal) {
  return request<ListResponse<Workspace>>(`/workspaces${queryString({ organizationId })}`, { signal });
}
export async function createWorkspace(input: { organizationId: string; name: string; slug: string; environment: WorkspaceEnvironment }) {
  return request<DataResponse<Workspace>>("/workspaces", { method: "POST", body: JSON.stringify(input) });
}

export async function listProjects(filters: { organizationId: string; workspaceId?: string } , signal?: AbortSignal) {
  return request<ListResponse<Project>>(`/projects${queryString(filters)}`, { signal });
}

export async function createProject(input: { organizationId: string; workspaceId: string; name: string; slug: string; owner: string; budgetUsd: number }) {
  return request<DataResponse<Project>>("/projects", { method: "POST", body: JSON.stringify(input) });
}

export async function listMembers(organizationId: string, signal?: AbortSignal) {
  return request<ListResponse<Member>>(`/members${queryString({ organizationId })}`, { signal });
}

export async function inviteMember(input: { organizationId: string; email: string; displayName: string; role: string; invitedBy: string }) {
  return request<DataResponse<Member>>("/members", { method: "POST", body: JSON.stringify(input) });
}

export async function listModels(filters: { q?: string; provider?: string; status?: string } = {}, signal?: AbortSignal) {
  return request<ListResponse<ModelRecord>>(`/models${queryString(filters)}`, { signal });
}

export type UpsertModelInput = {
  id: string;
  provider: string;
  displayName: string;
  status: ModelRecord["status"];
  contextWindow: number;
  maxOutputTokens: number;
  inputPricePerMillion: number;
  outputPricePerMillion: number;
  supportsTools: boolean;
  supportsVision: boolean;
  supportsJson: boolean;
  regions: string[];
};

export async function upsertModel(input: UpsertModelInput) {
  return request<DataResponse<ModelRecord>>("/models", { method: "POST", body: JSON.stringify(input) });
}

export async function listProviders(filters: { organizationId: string; q?: string; status?: string }, signal?: AbortSignal) {
  return request<ListResponse<ProviderConnection>>(`/providers${queryString(filters)}`, { signal });
}

export async function createProvider(input: { organizationId: string; name: string; provider: string; baseUrl: string }) {
  return request<DataResponse<ProviderConnection>>("/providers", { method: "POST", body: JSON.stringify(input) });
}

export async function listProviderHealthEvents(filters: { organizationId: string; providerId?: string; status?: string; source?: string }, signal?: AbortSignal) {
  return request<ListResponse<ProviderHealthEvent>>(`/provider-health-events${queryString(filters)}`, { signal });
}

export async function listProviderHealthProbes(filters: { organizationId: string; providerId?: string; status?: string }, signal?: AbortSignal) {
  return request<ListResponse<ProviderHealthProbe>>(`/provider-health-probes${queryString(filters)}`, { signal });
}

export async function queueProviderHealthProbe(id: string, input: { organizationId: string; region: string; model: string; requestedBy: string }) {
  return request<DataResponse<ProviderHealthProbe>>(`/providers/${encodeURIComponent(id)}/health/probes`, { method: "POST", body: JSON.stringify(input) });
}

export async function recordProviderHealth(id: string, input: { organizationId: string; probeId: string | null; source: ProviderHealthSource; success: boolean; requestCount: number; errorCount: number; averageLatencyMs: number; p95LatencyMs: number; httpStatus: number | null; message: string }) {
  return request<DataResponse<ProviderHealthEvent>>(`/providers/${encodeURIComponent(id)}/health/observations`, { method: "POST", body: JSON.stringify(input) });
}

export async function setProviderMaintenance(id: string, input: { organizationId: string; enabled: boolean; until: string | null; reason: string }) {
  return request<DataResponse<ProviderConnection>>(`/providers/${encodeURIComponent(id)}/maintenance`, { method: "POST", body: JSON.stringify(input) });
}

export type CreateRoutingPolicyInput = {
  organizationId: string;
  name: string;
  strategy: RoutingStrategy;
  modelPattern: string;
  maxRetries: number;
  requestTimeoutMs: number;
  targets: Array<{ providerId: string; model: string; priority: number; weight: number; enabled: boolean }>;
};

export async function listRoutingPolicies(filters: { organizationId: string; q?: string; status?: string }, signal?: AbortSignal) {
  return request<ListResponse<RoutingPolicy>>(`/routing-policies${queryString(filters)}`, { signal });
}

export async function createRoutingPolicy(input: CreateRoutingPolicyInput) {
  return request<DataResponse<RoutingPolicy>>("/routing-policies", { method: "POST", body: JSON.stringify(input) });
}

export async function activateRoutingPolicy(id: string, organizationId: string) {
  return request<DataResponse<RoutingPolicy>>(`/routing-policies/${encodeURIComponent(id)}/activate${queryString({ organizationId })}`, { method: "POST" });
}

export async function pauseRoutingPolicy(id: string, organizationId: string) {
  return request<DataResponse<RoutingPolicy>>(`/routing-policies/${encodeURIComponent(id)}/pause${queryString({ organizationId })}`, { method: "POST" });
}

export type CreateRateLimitInput = { organizationId: string; name: string; scopeType: RateLimitScope; scopeId: string; metric: RateLimitMetric; window: "second" | "minute" | "hour" | "day"; limit: number; burst: number; action: "reject" | "throttle" | "observe"; priority: number };

export async function listRateLimits(filters: { organizationId: string; q?: string; status?: string; scopeType?: string }, signal?: AbortSignal) {
  return request<ListResponse<RateLimitRule>>(`/rate-limits${queryString(filters)}`, { signal });
}

export async function createRateLimit(input: CreateRateLimitInput) {
  return request<DataResponse<RateLimitRule>>("/rate-limits", { method: "POST", body: JSON.stringify(input) });
}

export async function enforceRateLimit(id: string, organizationId: string) {
  return request<DataResponse<RateLimitRule>>(`/rate-limits/${encodeURIComponent(id)}/enforce${queryString({ organizationId })}`, { method: "POST" });
}

export async function disableRateLimit(id: string, organizationId: string) {
  return request<DataResponse<RateLimitRule>>(`/rate-limits/${encodeURIComponent(id)}/disable${queryString({ organizationId })}`, { method: "POST" });
}

export async function evaluateRateLimit(input: { organizationId: string; workspaceId?: string; projectId?: string; apiKeyId?: string; userId?: string; metric: RateLimitMetric; currentUsage: number; requestedUnits: number }) {
  return request<DataResponse<RateLimitDecision>>("/rate-limits/evaluate", { method: "POST", body: JSON.stringify(input) });
}

export type CreateBudgetInput = { organizationId: string; name: string; scopeType: BudgetScope; scopeId: string; period: "monthly" | "quarterly" | "annual"; limitUsd: number; warningPercent: number; criticalPercent: number; action: "alert" | "block" | "approval" };

export async function listBudgets(filters: { organizationId: string; q?: string; status?: string; scopeType?: string }, signal?: AbortSignal) {
  return request<ListResponse<Budget>>(`/budgets${queryString(filters)}`, { signal });
}

export async function createBudget(input: CreateBudgetInput) { return request<DataResponse<Budget>>("/budgets", { method: "POST", body: JSON.stringify(input) }); }

export async function activateBudget(id: string, organizationId: string) {
  return request<DataResponse<Budget>>(`/budgets/${encodeURIComponent(id)}/activate${queryString({ organizationId })}`, { method: "POST" });
}

export async function pauseBudget(id: string, organizationId: string) {
  return request<DataResponse<Budget>>(`/budgets/${encodeURIComponent(id)}/pause${queryString({ organizationId })}`, { method: "POST" });
}

export async function evaluateBudget(input: { organizationId: string; workspaceId?: string; projectId?: string; currentSpendUsd: number; proposedSpendUsd: number; elapsedPercent: number }) {
  return request<DataResponse<BudgetDecision>>("/budgets/evaluate", { method: "POST", body: JSON.stringify(input) });
}

export type CreateAlertInput = { organizationId: string; name: string; metric: AlertMetric; operator: "gt" | "gte" | "lt" | "lte"; threshold: number; window: "5m" | "15m" | "1h" | "24h"; cooldownMinutes: number; severity: "info" | "warning" | "critical"; channels: string[]; filters: Record<string, string> };

export async function listAlerts(filters: { organizationId: string; q?: string; status?: string; severity?: string }, signal?: AbortSignal) { return request<ListResponse<AlertRule>>(`/alerts${queryString(filters)}`, { signal }); }

export async function listAlertIncidents(filters: { organizationId: string; status?: string; severity?: string }, signal?: AbortSignal) { return request<ListResponse<AlertIncident>>(`/alert-incidents${queryString(filters)}`, { signal }); }

export async function createAlert(input: CreateAlertInput) { return request<DataResponse<AlertRule>>("/alerts", { method: "POST", body: JSON.stringify(input) }); }

export async function enableAlert(id: string, organizationId: string) { return request<DataResponse<AlertRule>>(`/alerts/${encodeURIComponent(id)}/enable${queryString({ organizationId })}`, { method: "POST" }); }

export async function disableAlert(id: string, organizationId: string) { return request<DataResponse<AlertRule>>(`/alerts/${encodeURIComponent(id)}/disable${queryString({ organizationId })}`, { method: "POST" }); }

export async function evaluateAlert(input: { organizationId: string; metric: AlertMetric; value: number; dimensions: Record<string, string> }) { return request<DataResponse<AlertEvaluation>>("/alerts/evaluate", { method: "POST", body: JSON.stringify(input) }); }

export type CreateWebhookInput = { organizationId: string; name: string; destination: string; events: WebhookEventType[]; sampleRate: number; includeData: boolean; propertyFilters: WebhookPropertyFilter[]; maxAttempts: number; timeoutSeconds: number };

export async function listWebhooks(filters: { organizationId: string; q?: string; status?: string; event?: string }, signal?: AbortSignal) { return request<ListResponse<WebhookEndpoint>>(`/webhooks${queryString(filters)}`, { signal }); }

export async function listWebhookDeliveries(filters: { organizationId: string; webhookId?: string; status?: string; eventType?: string }, signal?: AbortSignal) { return request<ListResponse<WebhookDelivery>>(`/webhook-deliveries${queryString(filters)}`, { signal }); }

export async function createWebhook(input: CreateWebhookInput) { return request<{ data: WebhookEndpoint; signingSecret: string; meta: { secretVisibility: "one-time"; delivery: "worker-queued" } }>("/webhooks", { method: "POST", body: JSON.stringify(input) }); }

export async function enableWebhook(id: string, organizationId: string) { return request<DataResponse<WebhookEndpoint>>(`/webhooks/${encodeURIComponent(id)}/enable${queryString({ organizationId })}`, { method: "POST" }); }

export async function disableWebhook(id: string, organizationId: string) { return request<DataResponse<WebhookEndpoint>>(`/webhooks/${encodeURIComponent(id)}/disable${queryString({ organizationId })}`, { method: "POST" }); }

export async function queueWebhookTest(id: string, organizationId: string, eventType: WebhookEventType) { return request<DataResponse<WebhookDelivery>>(`/webhooks/${encodeURIComponent(id)}/test${queryString({ organizationId })}`, { method: "POST", body: JSON.stringify({ eventType }) }); }

export async function retryWebhookDelivery(id: string, organizationId: string) { return request<DataResponse<WebhookDelivery>>(`/webhook-deliveries/${encodeURIComponent(id)}/retry${queryString({ organizationId })}`, { method: "POST" }); }

export async function replayWebhookDelivery(id: string, organizationId: string) { return request<DataResponse<WebhookDelivery>>(`/webhook-deliveries/${encodeURIComponent(id)}/replay${queryString({ organizationId })}`, { method: "POST" }); }

export type CreateReportInput = {
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
};

export async function listReports(filters: { organizationId: string; q?: string; status?: string; template?: string }, signal?: AbortSignal) {
  return request<ListResponse<ReportSchedule>>(`/reports${queryString(filters)}`, { signal });
}

export async function listReportRuns(filters: { organizationId: string; reportId?: string; status?: string; trigger?: string }, signal?: AbortSignal) {
  return request<ListResponse<ReportRun>>(`/report-runs${queryString(filters)}`, { signal });
}

export async function createReport(input: CreateReportInput) { return request<DataResponse<ReportSchedule>>("/reports", { method: "POST", body: JSON.stringify(input) }); }

export async function activateReport(id: string, organizationId: string) { return request<DataResponse<ReportSchedule>>(`/reports/${encodeURIComponent(id)}/activate${queryString({ organizationId })}`, { method: "POST" }); }

export async function pauseReport(id: string, organizationId: string) { return request<DataResponse<ReportSchedule>>(`/reports/${encodeURIComponent(id)}/pause${queryString({ organizationId })}`, { method: "POST" }); }

export async function queueReportRun(id: string, input: { organizationId: string; requestedBy: string; periodStart: string | null; periodEnd: string | null }) { return request<DataResponse<ReportRun>>(`/reports/${encodeURIComponent(id)}/run`, { method: "POST", body: JSON.stringify(input) }); }

export async function retryReportRun(id: string, organizationId: string, requestedBy: string) { return request<DataResponse<ReportRun>>(`/report-runs/${encodeURIComponent(id)}/retry${queryString({ organizationId })}`, { method: "POST", body: JSON.stringify({ requestedBy }) }); }

export async function listNotifications(filters: { organizationId: string; recipientId: string; q?: string; status?: string; category?: string; severity?: string }, signal?: AbortSignal) {
  return request<ListResponse<InboxNotification>>(`/notifications${queryString(filters)}`, { signal });
}

export async function createNotification(input: { organizationId: string; recipientId: string; category: NotificationCategory; severity: NotificationSeverity; title: string; body: string; sourceType: string; sourceId: string; actionUrl: string }) {
  return request<{ data: InboxNotification; meta: { source: string; externalDeliveries: number; dispatchBoundary: "notifications-worker" } }>("/notifications", { method: "POST", body: JSON.stringify(input) });
}

export async function markNotificationRead(id: string, organizationId: string, recipientId: string) {
  return request<DataResponse<InboxNotification>>(`/notifications/${encodeURIComponent(id)}/read${queryString({ organizationId, recipientId })}`, { method: "POST" });
}

export async function markNotificationUnread(id: string, organizationId: string, recipientId: string) {
  return request<DataResponse<InboxNotification>>(`/notifications/${encodeURIComponent(id)}/unread${queryString({ organizationId, recipientId })}`, { method: "POST" });
}

export async function archiveNotification(id: string, organizationId: string, recipientId: string) {
  return request<DataResponse<InboxNotification>>(`/notifications/${encodeURIComponent(id)}/archive${queryString({ organizationId, recipientId })}`, { method: "POST" });
}

export async function markAllNotificationsRead(organizationId: string, recipientId: string) {
  return request<DataResponse<{ updated: number }>>("/notifications/read-all", { method: "POST", body: JSON.stringify({ organizationId, recipientId }) });
}

export async function getNotificationPreference(organizationId: string, recipientId: string, signal?: AbortSignal) {
  return request<DataResponse<NotificationPreference>>(`/notification-preferences${queryString({ organizationId, recipientId })}`, { signal });
}

export async function updateNotificationPreference(input: {
  organizationId: string; recipientId: string; destinations: NotificationDestination[];
  categoryChannels: Partial<Record<NotificationCategory, NotificationChannel[]>>;
  digestFrequency: NotificationDigest; minimumSeverity: NotificationSeverity; timezone: string;
  quietHoursEnabled: boolean; quietStart: string; quietEnd: string;
}) {
  return request<DataResponse<NotificationPreference>>("/notification-preferences", { method: "PUT", body: JSON.stringify(input) });
}

export async function listNotificationPolicies(filters: { organizationId: string; q?: string; status?: string; category?: string }, signal?: AbortSignal) {
  return request<ListResponse<NotificationEscalationPolicy>>(`/notification-escalation-policies${queryString(filters)}`, { signal });
}

export async function createNotificationPolicy(input: {
  organizationId: string; name: string; status: "active" | "paused"; categories: NotificationCategory[];
  minimumSeverity: NotificationSeverity; acknowledgeWithinMinutes: number; repeatEveryMinutes: number;
  maxEscalations: number; routes: NotificationEscalationRoute[];
}) {
  return request<DataResponse<NotificationEscalationPolicy>>("/notification-escalation-policies", { method: "POST", body: JSON.stringify(input) });
}

export async function activateNotificationPolicy(id: string, organizationId: string) {
  return request<DataResponse<NotificationEscalationPolicy>>(`/notification-escalation-policies/${encodeURIComponent(id)}/activate${queryString({ organizationId })}`, { method: "POST" });
}

export async function pauseNotificationPolicy(id: string, organizationId: string) {
  return request<DataResponse<NotificationEscalationPolicy>>(`/notification-escalation-policies/${encodeURIComponent(id)}/pause${queryString({ organizationId })}`, { method: "POST" });
}

export async function evaluateNotificationEscalation(input: { organizationId: string; category: NotificationCategory; severity: NotificationSeverity; unacknowledgedMinutes: number }) {
  return request<DataResponse<NotificationEscalationEvaluation>>("/notification-escalation-policies/evaluate", { method: "POST", body: JSON.stringify(input) });
}

export async function listNotificationDeliveries(filters: { organizationId: string; recipientId: string; notificationId?: string; status?: string; channel?: string }, signal?: AbortSignal) {
  return request<ListResponse<NotificationDelivery>>(`/notification-deliveries${queryString(filters)}`, { signal });
}

export async function retryNotificationDelivery(id: string, organizationId: string) {
  return request<DataResponse<NotificationDelivery>>(`/notification-deliveries/${encodeURIComponent(id)}/retry${queryString({ organizationId })}`, { method: "POST" });
}

export async function listAuditEvents(filters: { organizationId: string; q?: string; actor?: string; action?: string; resourceType?: string; resourceId?: string; outcome?: string; riskLevel?: string; startAt?: string; endAt?: string }, signal?: AbortSignal) {
  return request<ListResponse<AuditEvent>>(`/audit-events${queryString(filters)}`, { signal });
}

export async function appendAuditEvent(input: { organizationId: string; actorId: string; actorEmail: string; action: string; resourceType: string; resourceId: string; outcome: AuditOutcome; riskLevel: AuditRisk; source: string; reason: string; requestId: string; ipAddress: string; userAgent: string; beforeState: Record<string, unknown>; afterState: Record<string, unknown> }) {
  return request<DataResponse<AuditEvent>>("/audit-events", { method: "POST", body: JSON.stringify(input) });
}

export async function verifyAuditIntegrity(organizationId: string, signal?: AbortSignal) {
  return request<DataResponse<AuditIntegrityResult>>(`/audit-events/verify${queryString({ organizationId })}`, { signal });
}

export async function getAuditRetention(organizationId: string, signal?: AbortSignal) {
  return request<DataResponse<AuditRetentionPolicy>>(`/audit-retention${queryString({ organizationId })}`, { signal });
}

export async function updateAuditRetention(input: { organizationId: string; retentionDays: number; legalHold: boolean; exportFormat: "csv" | "jsonl"; updatedBy: string }) {
  return request<DataResponse<AuditRetentionPolicy>>("/audit-retention", { method: "PUT", body: JSON.stringify(input) });
}

export async function listAuditExports(filters: { organizationId: string; status?: string; format?: string }, signal?: AbortSignal) {
  return request<ListResponse<AuditExport>>(`/audit-exports${queryString(filters)}`, { signal });
}

export async function queueAuditExport(input: { organizationId: string; requestedBy: string; format: "csv" | "jsonl"; filters: Record<string, string>; periodStart: string; periodEnd: string }) {
  return request<DataResponse<AuditExport>>("/audit-exports", { method: "POST", body: JSON.stringify(input) });
}

export async function retryAuditExport(id: string, organizationId: string, requestedBy: string) {
  return request<DataResponse<AuditExport>>(`/audit-exports/${encodeURIComponent(id)}/retry${queryString({ organizationId })}`, { method: "POST", body: JSON.stringify({ requestedBy }) });
}

export async function listVaultSecrets(filters: { organizationId: string; q?: string; kind?: string; scopeType?: string; status?: string; rotation?: string }, signal?: AbortSignal) {
  return request<ListResponse<VaultSecret>>(`/vault/secrets${queryString(filters)}`, { signal });
}

export async function createVaultSecret(input: { organizationId: string; name: string; kind: VaultSecretKind; scopeType: VaultScopeType; scopeId: string; secretValue: string; rotationIntervalDays: number; expiresAt: string; createdBy: string; requestId: string; sourceIp: string }) {
  return request<DataResponse<VaultSecret>>("/vault/secrets", { method: "POST", body: JSON.stringify(input) });
}

export async function rotateVaultSecret(id: string, input: { organizationId: string; secretValue: string; reason: string; rotatedBy: string; requestId: string; sourceIp: string }) {
  return request<DataResponse<VaultSecret>>(`/vault/secrets/${encodeURIComponent(id)}/rotate`, { method: "POST", body: JSON.stringify(input) });
}

export async function disableVaultSecret(id: string, input: { organizationId: string; reason: string; disabledBy: string; requestId: string; sourceIp: string }) {
  return request<DataResponse<VaultSecret>>(`/vault/secrets/${encodeURIComponent(id)}/disable`, { method: "POST", body: JSON.stringify(input) });
}

export async function listVaultAccessEvents(filters: { organizationId: string; secretId?: string; outcome?: string; actor?: string }, signal?: AbortSignal) {
  return request<ListResponse<VaultAccessEvent>>(`/vault/access-events${queryString(filters)}`, { signal });
}

export async function getLiteLLMIntegrationStatus(signal?: AbortSignal) {
  return request<DataResponse<LiteLLMIntegrationStatus>>("/integrations/litellm/status", { signal });
}

export async function verifyLiteLLMIntegration() {
  return request<DataResponse<LiteLLMIntegrationStatus>>("/integrations/litellm/verify", { method: "POST" });
}
