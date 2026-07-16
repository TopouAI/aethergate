# Helicone to AetherGate Feature Parity Matrix

## Purpose

This document is the authoritative migration ledger for bringing Helicone's product capabilities into AetherGate while preserving AetherGate's enterprise gateway identity model and Go service architecture.

Helicone is used as an Apache-2.0 product and interaction reference. AetherGate is not intended to remain a fork: its control plane, tenancy, authorization, billing boundaries, and gateway integrations are owned by AetherGate.

## Reference baseline

- Repository: `https://github.com/Helicone/helicone`
- Audited commit: `4df16a30ab79bc6f31e4b3a29aca179d767db878`
- Baseline date: 2026-07-14
- Audited surfaces: `web/pages`, authenticated sidebar and feature flags, Jawn controllers/managers, worker/gateway paths, Supabase migrations, ClickHouse migrations, and deployment manifests

Re-audit the upstream repository before each major release. Any newly discovered upstream capability must be added here before implementation work is considered complete.

## Status model

| Status | Meaning |
| --- | --- |
| `Baseline` | Capability is identified and mapped, but implementation has not started. |
| `Scaffolded` | A navigable product surface or service boundary exists; it is not production-complete. |
| `In progress` | One or more required layers are being implemented. |
| `Implemented` | All required layers exist and targeted tests pass. |
| `Verified` | End-to-end behavior, authorization, persistence, failure states, and deployment evidence pass. |

`Scaffolded` must never be interpreted as migrated or production-ready.

## Product capability matrix

### Gateway and traffic control

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| OpenAI-compatible gateway | Worker/gateway proxy and control-plane routes | LiteLLM data plane plus Go control plane | `In progress` | Sanitized server configuration plus liveness/readiness diagnostics and a HeroUI integration gate exist; real streaming/non-streaming traffic, cancellation, AetherGate/virtual-key enforcement, routing, usage attribution, and provider-failure evidence remain. |
| Multi-provider routing | Provider, proxy, model registry, provider-status domains | Providers, Models, routing policies | `In progress` | Ordered routes, weighted routes, fallbacks, retries, and health state are enforced. |
| Provider credentials | Vault and provider controllers | Enterprise Vault | `In progress` | Metadata-only API, per-version AES-256-GCM envelope encryption, rotation/disable, internal-only resolution, immutable access evidence, PostgreSQL schema, and HeroUI workspace exist; external KMS/key-ring rewrap, live worker/provider integration, auth, and database verification remain. |
| Model catalog | Model registry, comparison, pricing | Models | `In progress` | Aliases, capabilities, context limits, prices, regions, and lifecycle state are managed. |
| Response streaming | Proxy/worker stream path | LiteLLM data plane | `Baseline` | SSE chunks, cancellation, token accounting, and errors are preserved. |
| Request caching | Cache pages and cache metadata | Cache | `Scaffolded` | Exact/provider and semantic cache policies expose hit, savings, and invalidation behavior. |
| Rate limits | Rate-limit controllers and settings | Rate limits | `In progress` | RPM, TPM, concurrency, user/key/project hierarchy, and override behavior pass. |
| Budgets and credits | Credits, wallet, billing and Stripe domains | Budgets, Billing | `In progress` | Hierarchical limits, thresholds, forecast, enforcement, and reconciliation pass. |
| Request feedback and scores | Feedback/score paths and request enrichment | Requests, Evaluators | `Baseline` | Human and programmatic scores are attached, queried, and audited. |
| Gateway metadata | Helicone headers, properties, user/session identifiers | Gateway ingestion contract | `Baseline` | AetherGate headers and OpenTelemetry context are normalized without losing compatibility. |

### Observability and analytics

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| Usage dashboard | Dashboard and stats pages | Dashboard | `Scaffolded` | Real request, cost, token, latency, error, model, provider, and project data replace fixtures. |
| Request explorer | Requests pages and request controller | Requests | `In progress` | Server-side filters, saved views, pagination, export, raw payload, scores, and error detail pass. |
| Request detail | Request drawer/detail views | Requests | `In progress` | Prompt, response, timing, tokens, cost, metadata, properties, and trace links persist correctly. |
| Sessions | Sessions pages and session controller | Sessions | `Scaffolded` | Session trees, aggregated cost/latency, search, and request navigation pass. |
| Traces and agents | Trace/agent managers and span views | Traces | `Scaffolded` | OpenTelemetry spans, tool calls, critical path, inputs/outputs, and scores render correctly. |
| Users | Users pages and user controller | Users | `Scaffolded` | Identity aggregation, usage, cost, retention, and request drill-down pass. |
| Custom properties | Properties pages/controller | Properties | `Scaffolded` | Property discovery, value ranking, segmentation, filtering, and saved segments pass. |
| Cache analytics | Cache pages | Cache | `Scaffolded` | Hit rate, cost/time saved, trends, and request-level evidence agree with gateway events. |
| HQL / SQL analytics | HQL and Helicone SQL surfaces | HQL | `Scaffolded` | Governed query execution, schema browser, limits, history, saved queries, and export pass. |
| Metrics API | Metrics controllers and stats API | Analytics API | `Baseline` | Stable dimensions, time buckets, percentiles, and cost semantics are contract-tested. |
| Data export | Request and analytics exports | Requests, Reports, Developer | `Baseline` | CSV/JSON export is authorized, streamed, reproducible, and audit logged. |
| Data retention | Organization/settings controls | Settings | `Scaffolded` | Retention policies apply per tenant and propagate to operational and analytics stores. |

### Prompt engineering and evaluation

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| Prompt management | Prompt and Prompt 2025 controllers/pages | Prompts | `Scaffolded` | Templates, variables, tools, versions, tags, deployment state, and audit history pass. |
| Prompt execution history | Prompt run pages and request linkage | Prompts, Requests | `Baseline` | Every production/test run resolves to an immutable prompt version. |
| Datasets | Dataset controller/pages | Datasets | `Scaffolded` | Manual rows, request import, versions, CSV import/export, and lineage pass. |
| Playground | Playground controller/pages | Playground | `Scaffolded` | Multi-model runs, variables, tools, response formats, streaming, and save-to-prompt pass. |
| Evaluators | Evaluator/eval controllers/pages | Evaluators | `Scaffolded` | LLM judge, deterministic, and isolated code evaluators expose schemas and run history. |
| Online evaluation | Evaluator assignment and request scoring | Evaluators, Requests | `Baseline` | Sampling, asynchronous runs, retries, costs, and request scores pass. |
| Experiments | Experiment and Experiment V2 domains | Experiments | `Scaffolded` | Dataset cases, variants, evaluators, comparisons, cost, latency, and reproducibility pass. |
| Fine-tuning datasets | Fine-tuning pages and partner integration | Fine-tuning | `Scaffolded` | Validation, redaction, partner export, job state, and model comparison pass. |
| Model comparison | Model comparison manager/surfaces | Models, Playground, Experiments | `Baseline` | Quality, cost, latency, and reliability compare over the same reproducible sample. |

### Reliability and operations

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| Alerts | Alert controllers/pages/settings | Alerts | `In progress` | Conditions, filters, windows, cooldowns, channels, incidents, and recovery events pass. |
| Webhooks | Webhook controllers/pages/settings | Webhooks | `In progress` | Signed delivery, subscriptions, replay protection, retries, tests, and history pass. |
| Provider health | Provider status manager | Providers | `In progress` | Active probes and passive telemetry drive routing-safe health states. |
| Scheduled reports | Reports/settings surfaces | Reports | `In progress` | Templates, recipients, schedules, XLSX/CSV, tenant time zones, and history pass. |
| Notifications | Alerts and product notification surfaces | Notifications | `In progress` | Inbox, read/archive state, quiet hours, digests, personal channel routing, escalation dry-runs, worker delivery evidence, and retries pass. |
| Audit trail | Organization and administration events | Audit | `In progress` | Append-only tenant events, SHA-256 chain verification, actor/action/resource/risk/outcome filters, retention/legal hold, export queue evidence, and retries exist; automatic emission across all administrative domains, worker execution, auth, and live PostgreSQL verification remain. |
| Data reconciliation | Usage/cost/credits domains | Billing, Reports | `Baseline` | Gateway, provider, ClickHouse, and OpenMeter totals reconcile within documented tolerances. |

### Identity, tenancy, and enterprise administration

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| Authentication | Auth pages and Supabase auth | Identity service integration | `Baseline` | Login, logout, recovery, session expiry, MFA hooks, and secure cookie behavior pass. |
| Organizations | Organization controllers/settings | Organizations | `In progress` | Tenant lifecycle, ownership, domains, region, status, and policy inheritance pass. |
| Members and invitations | Members/settings pages | Members | `In progress` | Invite, accept, revoke, role changes, expiration, and access review pass. |
| RBAC | Organization membership and feature access | Policy package in Go API | `In progress` | Every UI and API action enforces explicit resource/action permissions. |
| Workspaces | Organization/project-like grouping plus AetherGate extension | Workspaces | `In progress` | Environment/team partitioning, membership, projects, budgets, and policies pass. |
| Projects/applications | Request properties plus AetherGate enterprise model | Projects | `In progress` | Ownership, application mapping, budget, keys, policies, and usage attribution pass. |
| API keys | API-key controllers/settings | API keys | `In progress` | Issue, one-time reveal, hash storage, scope, expiration, rotation, revoke, and audit pass. |
| SSO and connections | Settings/connections/enterprise portal | Settings | `Scaffolded` | OIDC/SAML configuration, domain claims, provisioning hooks, and break-glass access pass. |
| Billing administration | Billing, credits, Stripe surfaces | Billing | `Scaffolded` | Contracts, credits, price books, statements, adjustments, and roles pass. |
| Regional and retention policy | Organization settings | Organizations, Settings | `Baseline` | Data residency and retention constraints are enforced by deployment and storage topology. |

### Developer and integration experience

| Capability | Helicone reference | AetherGate destination | Current status | Acceptance boundary |
| --- | --- | --- | --- | --- |
| Onboarding | Onboarding pages and integration guides | Onboarding flow | `Baseline` | First org, project, provider, key, test request, and observed request form one guided flow. |
| REST APIs | Jawn controllers and public APIs | Go OpenAPI service | `Baseline` | Versioned OpenAPI, stable errors, pagination, idempotency, and auth contracts pass. |
| SDK compatibility | OpenAI-compatible proxy patterns | Gateway and generated SDKs | `Baseline` | OpenAI clients work by changing base URL/key; AetherGate SDKs cover management APIs. |
| MCP integration | Helicone data/MCP capability | Developer | `Scaffolded` | Read-only analytics and governed operational tools expose auditable MCP contracts. |
| API diagnostics | Developer and request inspection | Developer | `In progress` | Go and HeroUI surfaces return credential-safe LiteLLM liveness/readiness status with redirect rejection and normalized errors; real routing, rate-limit, virtual-key, provider, and traffic diagnostics remain. |
| Admin operations | Admin functions and enterprise portal | Protected operator console | `Baseline` | Cross-tenant operations are isolated, justified, audited, and unavailable to tenant admins. |

### AetherGate enterprise extensions

These are required product capabilities even though they are not strict Helicone parity items.

| Capability | Current status | Acceptance boundary |
| --- | --- | --- |
| Organization → workspace → project → application hierarchy | `Scaffolded` | Stable IDs, ownership, lifecycle, policy inheritance, and reporting dimensions pass. |
| Enterprise model access policy | `Scaffolded` | Allow/deny policy resolves deterministically across tenant scopes and gateway routes. |
| Hierarchical budgets and quotas | `Scaffolded` | Organization, workspace, project, application, key, and user scopes reconcile and enforce. |
| Contract price books | `Scaffolded` | Effective-dated provider and customer pricing produce reproducible cost and charge amounts. |
| Enterprise adoption intelligence | `Scaffolded` | Adoption, active projects, team trends, concentration, and unused allocations are explainable. |
| Immutable security audit | `In progress` | Actor, resource, before/after state, reason, IP/request context, hash-chain integrity, export evidence, and retention policy are implemented at the control-plane boundary; end-to-end automatic emission and worker/storage verification remain. |
| Private deployment operations | `Baseline` | Single-server and scalable topologies have backup, restore, upgrade, rollback, and verification evidence. |

## Console route coverage

The HeroUI v3 shell registers the following information architecture. Dashboard, Requests, Organizations, API Keys, Workspaces, Projects, Members, Models, Providers and Provider Health, Routing, Rate limits, Budgets, Alerts, Webhooks, Scheduled Reports, Notifications, Enterprise Vault, Audit Trail, and Developer Integration Diagnostics currently have purpose-built interfaces; unfinished routes intentionally show migration status instead of a false finished screen.

| Area | Routes |
| --- | --- |
| Observe | `/dashboard`, `/requests`, `/sessions`, `/traces`, `/users`, `/properties`, `/cache`, `/hql` |
| Improve | `/prompts`, `/datasets`, `/playground`, `/evaluators`, `/experiments`, `/fine-tuning` |
| Operate | `/models`, `/providers`, `/rate-limits`, `/alerts`, `/webhooks`, `/reports` |
| Enterprise | `/organizations`, `/workspaces`, `/projects`, `/members`, `/api-keys`, `/budgets`, `/billing`, `/vault`, `/audit`, `/settings`, `/notifications`, `/developer` |

## Definition of done for each row

A capability can move to `Verified` only when all applicable checks pass:

1. Domain model and database migrations are reviewed and reversible.
2. Go API contracts and stable error semantics are documented in OpenAPI.
3. Tenant isolation and resource/action authorization tests pass.
4. Console loading, empty, populated, error, permission-denied, and narrow-screen states pass.
5. Audit, telemetry, rate limiting, and idempotency behavior are implemented where relevant.
6. Automated unit, integration, contract, and end-to-end tests pass in CI.
7. Deployment, backup/restore, upgrade, and rollback paths are documented and exercised where relevant.
8. Fixtures and mock responses have been removed from production execution paths.

## Delivery sequence

1. **Foundation:** tenancy, RBAC, projects/applications, providers, models, keys, LiteLLM integration, request ingestion, cost normalization, and basic dashboard.
2. **Observability:** production Requests, Sessions, Traces, Users, Properties, Cache, analytics API, and export.
3. **Improvement loop:** Prompts, Datasets, Playground, Evaluators, Experiments, and Fine-tuning.
4. **Operations:** routing policy, rate limits, budgets, alerts, webhooks, reports, notifications, vault, and audit.
5. **Enterprise hardening:** SSO, price books, reconciliation, residency/retention, operator console, MCP/SDKs, and scalable deployment validation.

The sequence controls implementation order, not scope. Every row in this document remains part of the committed migration target.
