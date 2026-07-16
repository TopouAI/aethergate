export type WorkspaceEnvironment = "development" | "staging" | "production" | "shared";

export type Workspace = {
  id: string;
  organizationId: string;
  name: string;
  slug: string;
  status: "active" | "suspended";
  environment: WorkspaceEnvironment;
  projects: number;
  createdAt: string;
};

export type Project = {
  id: string;
  organizationId: string;
  workspaceId: string;
  workspace: string;
  name: string;
  slug: string;
  status: "active" | "archived";
  owner: string;
  budgetUsd: number;
  monthlyCostUsd: number;
  requests: number;
  createdAt: string;
};

export type Member = {
  id: string;
  organizationId: string;
  email: string;
  displayName: string;
  status: "active" | "invited" | "suspended";
  identityProvider: "oidc" | "saml" | "password" | "invitation" | "local";
  roles: string[];
  lastActiveAt: string | null;
  createdAt: string;
};

export type ModelRecord = {
  id: string;
  provider: string;
  displayName: string;
  status: "active" | "preview" | "deprecated" | "disabled";
  contextWindow: number;
  maxOutputTokens: number;
  inputPricePerMillion: number;
  outputPricePerMillion: number;
  supportsTools: boolean;
  supportsVision: boolean;
  supportsJson: boolean;
  regions: string[];
  createdAt: string;
};
