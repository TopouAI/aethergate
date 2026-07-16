import type { LucideIcon } from "lucide-react";
import {
  Activity,
  AlertTriangle,
  Archive,
  BarChart3,
  BellRing,
  Blocks,
  BookOpenText,
  Bot,
  Box,
  Braces,
  Building2,
  ChartNoAxesCombined,
  CircleDollarSign,
  Database,
  FlaskConical,
  GitBranch,
  Gauge,
  KeyRound,
  LayoutDashboard,
  ListTree,
  LockKeyhole,
  Network,
  Play,
  ScrollText,
  Settings2,
  ShieldCheck,
  Tags,
  TestTube2,
  Users,
  Webhook,
  Workflow,
} from "lucide-react";

export type NavigationItem = {
  name: string;
  href: string;
  icon: LucideIcon;
  description: string;
  source: "Helicone" | "AetherGate" | "Shared";
  capabilities: string[];
};

export type NavigationGroup = {
  name: string;
  items: NavigationItem[];
};

export const navigationGroups: NavigationGroup[] = [
  {
    name: "Observe",
    items: [
      { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard, description: "Unified traffic, cost, latency, quality, and provider health overview.", source: "Shared", capabilities: ["Time ranges", "Cost and token KPIs", "Model mix", "Project activity", "Data export"] },
      { name: "Requests", href: "/requests", icon: Activity, description: "Inspect, filter, compare, and debug every model request and response.", source: "Helicone", capabilities: ["Advanced filters", "Request detail", "Prompt and response rendering", "Scores", "Properties", "Export"] },
      { name: "Sessions", href: "/sessions", icon: ListTree, description: "Group multi-step application and agent activity into inspectable sessions.", source: "Helicone", capabilities: ["Session timeline", "Request tree", "Cost rollup", "Latency rollup", "Session search"] },
      { name: "Traces", href: "/traces", icon: Workflow, description: "Trace agent, tool, and model spans across distributed AI workflows.", source: "Helicone", capabilities: ["Span tree", "Tool calls", "Critical path", "Inputs and outputs", "Trace scores"] },
      { name: "Users", href: "/users", icon: Users, description: "Analyze usage and quality by end user or engineer.", source: "Helicone", capabilities: ["User ranking", "Active users", "User request history", "Cost per user", "Retention"] },
      { name: "Properties", href: "/properties", icon: Tags, description: "Segment traffic with custom request properties and dimensions.", source: "Helicone", capabilities: ["Property discovery", "Top values", "Trend comparison", "Request drill-down", "Saved segments"] },
      { name: "Cache", href: "/cache", icon: Archive, description: "Measure semantic and provider cache effectiveness and savings.", source: "Helicone", capabilities: ["Hit rate", "Cost saved", "Time saved", "Cache trends", "Cached request drill-down"] },
      { name: "HQL", href: "/hql", icon: Braces, description: "Query observability and enterprise usage data with a governed analytics language.", source: "Helicone", capabilities: ["Query editor", "Schema browser", "Saved queries", "Result export", "Execution history"] },
    ],
  },
  {
    name: "Improve",
    items: [
      { name: "Prompts", href: "/prompts", icon: ScrollText, description: "Version, test, deploy, and audit prompt templates.", source: "Helicone", capabilities: ["Prompt versions", "Variables", "Tools", "Tags", "Production deployment", "Run history"] },
      { name: "Datasets", href: "/datasets", icon: Database, description: "Curate production requests and examples into reusable datasets.", source: "Helicone", capabilities: ["Request import", "Manual rows", "Versioned datasets", "CSV import/export", "Experiment linkage"] },
      { name: "Playground", href: "/playground", icon: Play, description: "Compare models, prompts, tools, and parameters in an interactive workbench.", source: "Helicone", capabilities: ["Multi-model runs", "Prompt variables", "Tool definitions", "Response formats", "Save as prompt"] },
      { name: "Evaluators", href: "/evaluators", icon: TestTube2, description: "Define deterministic, LLM, and code-based quality evaluators.", source: "Helicone", capabilities: ["LLM-as-judge", "Python evaluator", "Score schemas", "Test evaluator", "Run history"] },
      { name: "Experiments", href: "/experiments", icon: FlaskConical, description: "Run prompt and model variants against datasets and evaluators.", source: "Helicone", capabilities: ["Variants", "Test cases", "Evaluator runs", "Result comparison", "Cost and latency"] },
      { name: "Fine-tuning", href: "/fine-tuning", icon: Bot, description: "Prepare and export high-quality production datasets for tuning partners.", source: "Helicone", capabilities: ["Training set builder", "Partner export", "Validation", "Job status", "Model comparison"] },
    ],
  },
  {
    name: "Operate",
    items: [
      { name: "Models", href: "/models", icon: Blocks, description: "Manage the enterprise model catalog, aliases, pricing, and permissions.", source: "Shared", capabilities: ["Model registry", "Aliases", "Pricing", "Capabilities", "Access policy"] },
      { name: "Providers", href: "/providers", icon: Network, description: "Configure upstream credentials, deployments, routing, and health.", source: "Shared", capabilities: ["Provider vault", "Deployment pools", "Fallbacks", "Health status", "Cost overrides"] },
      { name: "Rate limits", href: "/rate-limits", icon: Gauge, description: "Control RPM, TPM, concurrency, and scoped usage limits.", source: "Shared", capabilities: ["Rules", "Scope hierarchy", "Limited requests", "Overrides", "Dry run"] },
      { name: "Routing", href: "/routing", icon: GitBranch, description: "Author ordered, weighted, latency-aware, and fallback routing policies.", source: "AetherGate", capabilities: ["Weighted pools", "Priority fallback", "Health gates", "Retries", "Timeouts"] },
      { name: "Alerts", href: "/alerts", icon: AlertTriangle, description: "Detect budget, reliability, latency, and usage anomalies.", source: "Helicone", capabilities: ["Metric conditions", "Filters", "Channels", "Cooldown", "Incident history"] },
      { name: "Webhooks", href: "/webhooks", icon: Webhook, description: "Deliver signed lifecycle, usage, and alert events to enterprise systems.", source: "Helicone", capabilities: ["Subscriptions", "Signing secrets", "Retries", "Test delivery", "Delivery history"] },
      { name: "Reports", href: "/reports", icon: BarChart3, description: "Schedule and export management, engineering, and finance reports.", source: "Shared", capabilities: ["Templates", "Schedules", "Recipients", "CSV/XLSX", "Delivery history"] },
    ],
  },
  {
    name: "Enterprise",
    items: [
      { name: "Organizations", href: "/organizations", icon: Building2, description: "Manage customer organizations, tenancy, lifecycle, and policy inheritance.", source: "AetherGate", capabilities: ["Tenant lifecycle", "Domains", "Status", "Data region", "Policy inheritance"] },
      { name: "Workspaces", href: "/workspaces", icon: Box, description: "Partition teams, environments, and AI programs inside an organization.", source: "AetherGate", capabilities: ["Environment scopes", "Members", "Projects", "Budgets", "Model policy"] },
      { name: "Projects", href: "/projects", icon: ChartNoAxesCombined, description: "Attribute model usage and governance to enterprise initiatives.", source: "AetherGate", capabilities: ["Applications", "Owners", "Budget", "Keys", "Usage intelligence"] },
      { name: "Members", href: "/members", icon: Users, description: "Manage identities, invitations, roles, and enterprise access.", source: "Shared", capabilities: ["Invitations", "Roles", "Teams", "SSO state", "Access review"] },
      { name: "API keys", href: "/api-keys", icon: KeyRound, description: "Issue, scope, rotate, revoke, and audit gateway credentials.", source: "Shared", capabilities: ["Virtual keys", "Model scope", "Rate limits", "Expiration", "Rotation", "Usage"] },
      { name: "Budgets", href: "/budgets", icon: CircleDollarSign, description: "Set and enforce budgets across organizations, workspaces, and projects.", source: "AetherGate", capabilities: ["Budget hierarchy", "Thresholds", "Forecast", "Enforcement", "Notifications"] },
      { name: "Billing", href: "/billing", icon: CircleDollarSign, description: "Manage credits, contract pricing, statements, and reconciliation.", source: "Shared", capabilities: ["Credits", "Contracts", "Price books", "Statements", "Reconciliation"] },
      { name: "Vault", href: "/vault", icon: LockKeyhole, description: "Protect provider credentials and sensitive gateway configuration.", source: "Helicone", capabilities: ["Provider keys", "Encryption", "Rotation", "Access audit", "Secret references"] },
      { name: "Audit", href: "/audit", icon: ShieldCheck, description: "Review immutable administrative and security events.", source: "AetherGate", capabilities: ["Actor history", "Resource history", "Export", "Retention", "Integrity evidence"] },
      { name: "Settings", href: "/settings", icon: Settings2, description: "Configure organization defaults, SSO, connections, and integrations.", source: "Shared", capabilities: ["Organization", "SSO", "Connections", "Notifications", "Data retention"] },
      { name: "Notifications", href: "/notifications", icon: BellRing, description: "Central inbox for alerts, reports, access changes, and platform notices.", source: "AetherGate", capabilities: ["Inbox", "Preferences", "Escalations", "Read state", "Delivery channels"] },
      { name: "Developer", href: "/developer", icon: BookOpenText, description: "API documentation, SDKs, MCP, examples, and integration diagnostics.", source: "Shared", capabilities: ["OpenAPI", "SDKs", "MCP", "Examples", "API diagnostics"] },
    ],
  },
];

export const navigationItems = navigationGroups.flatMap((group) => group.items);

export function getNavigationItem(pathname: string) {
  return navigationItems.find((item) => item.href === pathname);
}
