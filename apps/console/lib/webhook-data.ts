import type { WebhookDelivery, WebhookEndpoint, WebhookEventType } from "@/types/webhook";

export const webhookEventOptions: Array<{ value: WebhookEventType; label: string }> = [
  { value: "request.completed", label: "Request completed" },
  { value: "request.failed", label: "Request failed" },
  { value: "alert.triggered", label: "Alert triggered" },
  { value: "alert.resolved", label: "Alert resolved" },
  { value: "budget.threshold_reached", label: "Budget threshold reached" },
  { value: "api_key.revoked", label: "API key revoked" },
  { value: "provider.health_changed", label: "Provider health changed" },
];

export const seedWebhooks: WebhookEndpoint[] = [
  {
    id: "wh_prod_events", organizationId: "org_topoai", name: "Production event bus", status: "active",
    destination: "https://events.topoai.dev/aethergate", version: "2026-07-15",
    events: ["request.completed", "request.failed", "alert.triggered"], sampleRate: 100, includeData: true,
    propertyFilters: [{ key: "environment", value: "production" }], signingSecretPrefix: "whsec_xD9m2Q",
    maxAttempts: 5, timeoutSeconds: 10, successCount: 18342, failureCount: 17,
    lastDeliveredAt: "2026-07-14T06:42:00Z", createdAt: "2026-05-14T02:00:00Z", updatedAt: "2026-05-14T02:00:00Z",
  },
  {
    id: "wh_finops", organizationId: "org_topoai", name: "FinOps automation", status: "disabled",
    destination: "https://finance.topoai.dev/hooks/usage", version: "2026-07-15",
    events: ["budget.threshold_reached"], sampleRate: 100, includeData: false, propertyFilters: [],
    signingSecretPrefix: "whsec_7Kp2aN", maxAttempts: 8, timeoutSeconds: 15, successCount: 94, failureCount: 3,
    lastDeliveredAt: null, createdAt: "2026-05-14T02:00:00Z", updatedAt: "2026-05-14T02:00:00Z",
  },
];

export const seedWebhookDeliveries: WebhookDelivery[] = [
  {
    id: "whd_success_01", organizationId: "org_topoai", webhookId: "wh_prod_events", webhookName: "Production event bus",
    eventId: "evt_req_01", eventType: "request.completed", status: "succeeded", trigger: "event", attempt: 1,
    maxAttempts: 5, responseStatus: 202, durationMs: 184, errorMessage: "", nextRetryAt: null,
    deliveredAt: "2026-07-14T06:42:01Z", replayOfId: null, createdAt: "2026-07-14T06:42:00Z",
  },
  {
    id: "whd_failed_02", organizationId: "org_topoai", webhookId: "wh_prod_events", webhookName: "Production event bus",
    eventId: "evt_alert_07", eventType: "alert.triggered", status: "failed", trigger: "event", attempt: 2,
    maxAttempts: 5, responseStatus: 503, durationMs: 10012, errorMessage: "Destination returned HTTP 503.",
    nextRetryAt: "2026-07-14T06:51:00Z", deliveredAt: null, replayOfId: null, createdAt: "2026-07-14T06:41:00Z",
  },
  {
    id: "whd_dead_03", organizationId: "org_topoai", webhookId: "wh_prod_events", webhookName: "Production event bus",
    eventId: "evt_req_legacy", eventType: "request.failed", status: "dead_letter", trigger: "event", attempt: 5,
    maxAttempts: 5, responseStatus: null, durationMs: 30000, errorMessage: "Delivery timed out after all configured attempts.",
    nextRetryAt: null, deliveredAt: null, replayOfId: null, createdAt: "2026-07-13T22:20:00Z",
  },
];

