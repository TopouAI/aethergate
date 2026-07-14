# Roadmap

The roadmap is outcome-based. Dates and version numbers will be assigned after the executable scaffolds and team capacity are known.

## Phase 0 — Foundation

- [x] Product positioning and repository structure
- [x] Go backend and HeroUI v3 frontend decisions
- [x] Existing LiteLLM stack location and deployment documentation
- [ ] Import and sanitize the existing `aethergate-litellm-stack`
- [ ] Scaffold Next.js Console
- [ ] Scaffold Go API and Worker
- [ ] Establish OpenAPI and event contracts
- [ ] Add CI for documentation, frontend, Go, and deployment validation

## Phase 1 — Enterprise control plane

- [ ] Authentication and organization tenancy
- [ ] Workspaces, departments, projects, applications, and members
- [ ] Roles and permission enforcement
- [ ] LiteLLM model and virtual-key integration
- [ ] API key lifecycle, limits, model policies, and budgets
- [ ] Basic usage and cost views
- [ ] Append-oriented administration audit log

## Phase 2 — Usage intelligence

- [ ] Stable usage-event schema and idempotent ingestion
- [ ] ClickHouse deployment, retention, and recovery
- [ ] Project, application, department, and engineer analytics
- [ ] Request, token, cost, latency, reliability, and provider reports
- [ ] Alerts, anomaly detection, and exports

## Phase 3 — Metering and billing

- [ ] OpenMeter event integration
- [ ] Credits, quotas, and entitlements
- [ ] Contract pricing and budget enforcement
- [ ] Statements, reconciliation, and billing exports

## Phase 4 — Enterprise operations

- [ ] SSO and advanced identity integrations
- [ ] Advanced audit, data governance, and policy controls
- [ ] High availability and multi-region patterns
- [ ] Private deployment lifecycle, upgrades, and support tooling
