export type OrganizationStatus = "active" | "suspended" | "provisioning";

export type Organization = {
  id: string;
  name: string;
  slug: string;
  status: OrganizationStatus;
  plan: "Open Source" | "Enterprise" | "Evaluation";
  region: string;
  workspaces: number;
  projects: number;
  members: number;
  monthlyCostUsd: number;
  budgetUsd: number;
  requests: number;
  owner: string;
  createdAt: string;
};

export type APIKeyStatus = "active" | "revoked" | "expired";

export type APIKeyRecord = {
  id: string;
  name: string;
  prefix: string;
  project: string;
  status: APIKeyStatus;
  models: string[];
  rpm: number;
  tpm: number;
  spendUsd: number;
  createdBy: string;
  createdAt: string;
  lastUsedAt: string | null;
  expiresAt: string | null;
};
