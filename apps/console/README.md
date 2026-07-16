# AetherGate Console

The HeroUI v3 enterprise management UI for organizations, workspaces, projects, members, API keys, models, providers, routing, rate limits, budgets, alerts, webhooks, scheduled reports, notification inbox/routing/escalation, Enterprise Vault, usage, audit workflows, and server-side integration diagnostics.

## Frontend baseline

- Next.js App Router
- React 19 or later
- TypeScript
- Tailwind CSS v4
- HeroUI v3 using `@heroui/react` and `@heroui/styles`

HeroUI v3 components work after importing the styles and do not require a global `HeroUIProvider`. Introduce providers only for a specific need such as locale, session state, or query state.

Official references:

- [HeroUI v3 quick start](https://heroui.com/en/docs/react/getting-started/quick-start)
- [HeroUI framework integration](https://heroui.com/en/docs/react/getting-started/frameworks)
- [HeroUI v3 Table](https://heroui.com/en/docs/react/components/table)

## Product areas

```text
app/
├── (auth)/
├── (platform)/
│   ├── overview/
│   ├── organizations/
│   ├── workspaces/
│   ├── departments/
│   ├── projects/
│   ├── applications/
│   ├── members/
│   ├── api-keys/
│   ├── models/
│   ├── usage/
│   ├── budgets/
│   ├── audit/
│   └── settings/
└── workbench/
```

This tree describes route ownership. Purpose-built application routes include `/vault`, whose HeroUI workspace provides masked metadata, rotation/disable workflows and immutable access evidence without any plaintext fallback; `/audit`, whose workspace provides tenant event search, risk/outcome filtering, before/after evidence, SHA-256 chain verification, retention/legal-hold policy, and asynchronous export/retry history; and `/developer`, which shows credential-safe LiteLLM configuration, liveness/readiness probe evidence, and reviewed repository/runtime stack placement without exposing the master key.

The Console never mutates audit history and never generates export objects in the browser. Those guarantees belong to the Go API, PostgreSQL constraints/triggers, and isolated worker boundaries.

## Interaction and table strategy

HeroUI v3 is the default component and interaction layer. Use its accessible primitives, compound components, overlays, forms, navigation, feedback, and motion behavior before adding another visual component system.

For data tables:

1. Use HeroUI Table for standard sorting, selection, resizing, pagination, loading, and request-log views.
2. Put reusable table behavior behind `packages/ui` instead of implementing it independently on every page.
3. If a requirement needs very large virtualization, pinned columns, grouped rows, pivoting, spreadsheet editing, or complex export, add a dedicated grid through the shared `DataGrid` adapter.
4. Keep HeroUI tokens and AetherGate styling around any added grid so the product does not become visually fragmented.
5. Record the grid selection and license implications in an architecture decision before adoption.

## UI rules

- Prefer server-rendered shells and data loading where it improves startup performance.
- Keep client components scoped to interaction boundaries.
- Use URL state for filters, date ranges, pagination, and shareable report views.
- Display times with the selected organization timezone and retain UTC in APIs.
- Never expose upstream provider credentials or LiteLLM master credentials to the browser.
- Treat permissions as both a UI concern and a server-enforced API rule; hiding a control is not authorization.

