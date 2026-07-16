import type { ProviderHealthEvent, ProviderHealthProbe } from "@/types/provider";

export const seedProviderHealthEvents: ProviderHealthEvent[] = [
  {
    id: "phe_openai_01", organizationId: "org_topoai", providerId: "provider_openai_primary", providerName: "OpenAI Primary",
    probeId: null, source: "passive_telemetry", previousStatus: "healthy", status: "healthy", transition: false,
    success: true, routingEligible: true, requestCount: 184220, errorCount: 111, errorRate: 0.0603,
    averageLatencyMs: 842, p95LatencyMs: 1240, httpStatus: null, consecutiveFailures: 0,
    reason: "Passive telemetry is within routing-safe thresholds.", observedAt: "2026-07-14T06:00:00Z",
  },
  {
    id: "phe_deepseek_01", organizationId: "org_topoai", providerId: "provider_deepseek_apac", providerName: "DeepSeek APAC",
    probeId: null, source: "passive_telemetry", previousStatus: "healthy", status: "degraded", transition: true,
    success: false, routingEligible: false, requestCount: 48120, errorCount: 1049, errorRate: 2.18,
    averageLatencyMs: 1480, p95LatencyMs: 2160, httpStatus: null, consecutiveFailures: 0,
    reason: "Passive telemetry exceeded the degraded error-rate threshold.", observedAt: "2026-07-14T05:55:00Z",
  },
];

export const seedProviderHealthProbes: ProviderHealthProbe[] = [
  {
    id: "probe_openai_01", organizationId: "org_topoai", providerId: "provider_openai_primary", providerName: "OpenAI Primary",
    status: "succeeded", region: "apac", model: "gpt-5-mini", requestedBy: "system",
    requestedAt: "2026-07-14T05:59:55Z", startedAt: null, completedAt: "2026-07-14T06:00:00Z",
    eventId: "phe_openai_01", errorMessage: "",
  },
];

