export type ProviderStatus = "configuring" | "healthy" | "degraded" | "offline" | "maintenance";
export type CredentialState = "missing" | "configured" | "rotating";
export type ProviderHealthSource = "manual" | "active_probe" | "passive_telemetry";
export type ProviderProbeStatus = "queued" | "running" | "succeeded" | "failed" | "cancelled";

export type ProviderConnection = {
  id: string;
  organizationId: string;
  name: string;
  provider: string;
  baseUrl: string;
  status: ProviderStatus;
  credentialState: CredentialState;
  models: number;
  p95LatencyMs: number;
  successRate: number;
  lastCheckedAt: string | null;
  routingEligible: boolean;
  healthSource: ProviderHealthSource;
  healthReason: string;
  errorRate: number;
  requestCount24h: number;
  averageLatencyMs: number;
  consecutiveFailures: number;
  lastTransitionAt: string | null;
  maintenanceUntil: string | null;
  maintenanceReason: string;
  createdAt: string;
};

export type ProviderHealthEvent = {
  id: string;
  organizationId: string;
  providerId: string;
  providerName: string;
  probeId: string | null;
  source: ProviderHealthSource;
  previousStatus: ProviderStatus;
  status: ProviderStatus;
  transition: boolean;
  success: boolean;
  routingEligible: boolean;
  requestCount: number;
  errorCount: number;
  errorRate: number;
  averageLatencyMs: number;
  p95LatencyMs: number;
  httpStatus: number | null;
  consecutiveFailures: number;
  reason: string;
  observedAt: string;
};

export type ProviderHealthProbe = {
  id: string;
  organizationId: string;
  providerId: string;
  providerName: string;
  status: ProviderProbeStatus;
  region: string;
  model: string;
  requestedBy: string;
  requestedAt: string;
  startedAt: string | null;
  completedAt: string | null;
  eventId: string | null;
  errorMessage: string;
};
