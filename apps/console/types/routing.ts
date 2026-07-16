export type RoutingStrategy = "weighted" | "priority" | "latency";
export type RoutingPolicyStatus = "draft" | "active" | "paused";

export type RoutingTarget = {
  id: string;
  providerId: string;
  providerName: string;
  model: string;
  priority: number;
  weight: number;
  enabled: boolean;
};

export type RoutingPolicy = {
  id: string;
  organizationId: string;
  name: string;
  slug: string;
  status: RoutingPolicyStatus;
  strategy: RoutingStrategy;
  modelPattern: string;
  maxRetries: number;
  requestTimeoutMs: number;
  targets: RoutingTarget[];
  createdAt: string;
  updatedAt: string;
};
