# AetherGate API

The Go control-plane service for enterprise identity, organization structure, model governance, key lifecycle, budgets, usage queries, and LiteLLM administration.

## Proposed Go layout

```text
apps/api/
├── cmd/aethergate-api/          # Process entry point
├── internal/
│   ├── platform/                # Startup, configuration, HTTP, telemetry
│   ├── identity/                # Authentication and principals
│   ├── organization/
│   ├── workspace/
│   ├── department/
│   ├── project/
│   ├── application/
│   ├── membership/
│   ├── apikey/
│   ├── modelpolicy/
│   ├── budget/
│   ├── usage/
│   ├── audit/
│   └── litellm/                 # LiteLLM Admin API adapter
├── migrations/
└── tests/
```

This layout documents intended package boundaries and will be materialized when the Go module and repository import path are selected.

## Responsibilities

- Own AetherGate enterprise data in the `aethergate` PostgreSQL database.
- Call LiteLLM through an adapter; do not write LiteLLM internal tables directly.
- Create and revoke LiteLLM virtual keys through supported APIs.
- Enforce organization, workspace, project, and member authorization.
- Serve OpenAPI-described APIs consumed by the Console and SDKs.
- Record security-relevant administration in an append-oriented audit log.
- Query analytics stores through explicit interfaces rather than coupling domain packages to ClickHouse.

## Database connections

- `DATABASE_URL`: normal runtime access through PgBouncer transaction pooling.
- `DIRECT_URL`: direct PostgreSQL access for schema migrations and operations that require a session.

The API must not use the LiteLLM database account. The two applications share a PostgreSQL instance during the single-server phase but use separate databases and users.

