import { foundationOrganizationId } from "@/lib/foundation-data";
import type { RateLimitRule } from "@/types/rate-limit";

export const seedRateLimits: RateLimitRule[] = [
  { id: "limit_org_tokens", organizationId: foundationOrganizationId, name: "Organization token ceiling", status: "enforced", scopeType: "organization", scopeId: foundationOrganizationId, metric: "tokens", window: "minute", limit: 2000000, burst: 200000, action: "reject", priority: 100, matchedRequests: 482340, limitedRequests: 184, createdAt: "2026-05-01T00:00:00Z", updatedAt: "2026-05-01T00:00:00Z" },
  { id: "limit_engineering_requests", organizationId: foundationOrganizationId, name: "Engineering request budget", status: "enforced", scopeType: "workspace", scopeId: "ws_engineering", metric: "requests", window: "minute", limit: 1200, burst: 120, action: "throttle", priority: 200, matchedRequests: 251571, limitedRequests: 39, createdAt: "2026-05-01T00:00:00Z", updatedAt: "2026-05-01T00:00:00Z" },
  { id: "limit_copilot_observe", organizationId: foundationOrganizationId, name: "Copilot concurrency preview", status: "draft", scopeType: "project", scopeId: "project_engineering_copilot", metric: "concurrency", window: "second", limit: 80, burst: 10, action: "observe", priority: 300, matchedRequests: 0, limitedRequests: 0, createdAt: "2026-05-01T00:00:00Z", updatedAt: "2026-05-01T00:00:00Z" },
];
