export type WebhookStatus = "active" | "disabled";

export type WebhookEventType =
  | "request.completed"
  | "request.failed"
  | "alert.triggered"
  | "alert.resolved"
  | "budget.threshold_reached"
  | "api_key.revoked"
  | "provider.health_changed";

export type WebhookPropertyFilter = { key: string; value: string };

export type WebhookEndpoint = {
  id: string;
  organizationId: string;
  name: string;
  status: WebhookStatus;
  destination: string;
  version: string;
  events: WebhookEventType[];
  sampleRate: number;
  includeData: boolean;
  propertyFilters: WebhookPropertyFilter[];
  signingSecretPrefix: string;
  maxAttempts: number;
  timeoutSeconds: number;
  successCount: number;
  failureCount: number;
  lastDeliveredAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type WebhookDeliveryStatus = "pending" | "delivering" | "succeeded" | "failed" | "dead_letter";

export type WebhookDelivery = {
  id: string;
  organizationId: string;
  webhookId: string;
  webhookName: string;
  eventId: string;
  eventType: WebhookEventType;
  status: WebhookDeliveryStatus;
  trigger: "event" | "test" | "retry" | "replay";
  attempt: number;
  maxAttempts: number;
  responseStatus: number | null;
  durationMs: number;
  errorMessage: string;
  nextRetryAt: string | null;
  deliveredAt: string | null;
  replayOfId: string | null;
  createdAt: string;
};

