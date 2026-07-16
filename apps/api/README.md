# AetherGate API

The AetherGate control-plane API is a Go service with a domain service layer, an in-memory development repository, and a PostgreSQL repository implemented with pgx.

## Run locally

Memory-backed development mode:

```powershell
go run ./apps/api/cmd/server
```

PostgreSQL-backed mode:

```powershell
$env:AETHERGATE_DATABASE_URL = "postgresql://aethergate_user:password@127.0.0.1:6432/aethergate?sslmode=disable"
go run ./apps/api/cmd/migrate
go run ./apps/api/cmd/server
```

The server listens on `:8080` by default. Override it with `AETHERGATE_API_ADDR`.

| Variable | Purpose |
| --- | --- |
| `AETHERGATE_API_ADDR` | HTTP listen address; defaults to `:8080`. |
| `AETHERGATE_DATABASE_URL` | PostgreSQL/PgBouncer connection string. Empty selects the in-memory development repository. |
| `AETHERGATE_AUTO_MIGRATE` | Set to `true` only in controlled development environments to migrate at startup. Prefer the explicit migrate command. |
| `AETHERGATE_VAULT_KEK` | Required for persistent Vault writes: standard Base64 encoding of exactly 32 random bytes. Never expose it to the Console or commit it. |
| `LITELLM_BASE_URL` | Internal absolute HTTP(S) URL for LiteLLM, normally `http://litellm:4000` in Compose. Credentials, query strings, and fragments are rejected. |
| `LITELLM_MASTER_KEY` | Optional server-only bearer credential for LiteLLM health probes. Never expose it through `NEXT_PUBLIC_*`, logs, API responses, or Git. |

pgx is configured without connection-local prepared statement assumptions so the runtime is compatible with PgBouncer transaction pooling.

## Development endpoints

System and observability:

- `GET /healthz`
- `GET /readyz`
- `GET /api/v1/overview`
- `GET /api/v1/requests`
- `GET /api/v1/requests/{requestID}`

Enterprise control plane:

- `GET|POST /api/v1/organizations`
- `GET /api/v1/organizations/{organizationID}`
- `GET|POST /api/v1/api-keys`
- `POST /api/v1/api-keys/{apiKeyID}/revoke`
- `GET /api/v1/provider-health-events`
- `GET /api/v1/provider-health-probes`
- `POST /api/v1/providers/{providerID}/health/probes`
- `POST /api/v1/providers/{providerID}/health/observations`
- `POST /api/v1/providers/{providerID}/maintenance`
- `GET|POST /api/v1/workspaces`
- `GET|POST /api/v1/projects`
- `GET|POST /api/v1/members`
- `GET|POST /api/v1/models`
- `GET|POST /api/v1/providers`
- `GET|POST /api/v1/routing-policies`
- `POST /api/v1/routing-policies/{policyID}/activate`
- `POST /api/v1/routing-policies/{policyID}/pause`
- `GET|POST /api/v1/rate-limits`
- `POST /api/v1/rate-limits/evaluate`
- `POST /api/v1/rate-limits/{ruleID}/enforce`
- `POST /api/v1/rate-limits/{ruleID}/disable`
- `GET|POST /api/v1/budgets`
- `POST /api/v1/budgets/evaluate`
- `POST /api/v1/budgets/{budgetID}/activate`
- `POST /api/v1/budgets/{budgetID}/pause`
- `GET|POST /api/v1/alerts`
- `POST /api/v1/alerts/evaluate`
- `POST /api/v1/alerts/{alertID}/enable`
- `POST /api/v1/alerts/{alertID}/disable`
- `GET /api/v1/alert-incidents`
- `GET|POST /api/v1/webhooks`
- `POST /api/v1/webhooks/{webhookID}/enable`
- `POST /api/v1/webhooks/{webhookID}/disable`
- `POST /api/v1/webhooks/{webhookID}/test`
- `GET /api/v1/webhook-deliveries`
- `POST /api/v1/webhook-deliveries/{deliveryID}/retry`
- `POST /api/v1/webhook-deliveries/{deliveryID}/replay`
- `GET|POST /api/v1/reports`
- `POST /api/v1/reports/{reportID}/activate`
- `POST /api/v1/reports/{reportID}/pause`
- `POST /api/v1/reports/{reportID}/run`
- `GET /api/v1/report-runs`
- `POST /api/v1/report-runs/{runID}/retry`
- `GET|POST /api/v1/notifications`
- `POST /api/v1/notifications/read-all`
- `POST /api/v1/notifications/{notificationID}/read|unread|archive`
- `GET|PUT /api/v1/notification-preferences`
- `GET|POST /api/v1/notification-escalation-policies`
- `POST /api/v1/notification-escalation-policies/evaluate`
- `POST /api/v1/notification-escalation-policies/{policyID}/activate|pause`
- `GET /api/v1/notification-deliveries`
- `POST /api/v1/notification-deliveries/{deliveryID}/retry`
- `GET|POST /api/v1/audit-events`
- `GET /api/v1/audit-events/verify`
- `GET|PUT /api/v1/audit-retention`
- `GET|POST /api/v1/audit-exports`
- `POST /api/v1/audit-exports/{exportID}/retry`
- `GET|POST /api/v1/vault/secrets`
- `GET /api/v1/vault/secrets/{secretID}`
- `POST /api/v1/vault/secrets/{secretID}/rotate`
- `POST /api/v1/vault/secrets/{secretID}/disable`
- `GET /api/v1/vault/access-events`

- `GET /api/v1/integrations/litellm/status`
- `POST /api/v1/integrations/litellm/verify`

API-key creation exposes the generated secret exactly once. The repository persists only a 32-byte digest, and serialization tests ensure the digest never enters public API responses.
Webhook creation likewise reveals its signing secret once. Endpoint records expose only a non-secret prefix and a Vault reference contract; test, retry, and replay calls enqueue delivery records for the isolated webhook worker instead of performing outbound network access in the control plane.
Provider active-probe calls follow the same isolation rule: the control plane stores probe intent and observation evidence, while the future provider-health worker owns outbound requests. Passive observations and worker results compute health with a three-failure active-probe debounce, maintenance suppression, and an explicit `routingEligible` decision. Routing activation also requires configured credentials.
Report schedule calls calculate the next run in the configured IANA timezone and persist worker jobs. The control plane does not query analytics stores, create CSV/XLSX/PDF artifacts, upload files, or deliver email/Slack messages; those operations belong to the isolated Reports Worker.

Notification creation always produces a recipient-scoped inbox item. Personal severity thresholds, category routes, quiet hours, and digest settings determine whether external delivery records are queued, deferred, or suppressed. The control plane never performs email, Slack, Teams, or webhook requests; the isolated Notifications Worker claims eligible records, resolves server-side connector references, enforces idempotency, and writes delivery evidence.

Audit events are tenant-scoped append-only records linked by a SHA-256 forward hash chain. The API supports actor/action/resource/risk/outcome/time filtering and full-chain verification, but exposes no update or delete endpoint. Retention and legal-hold configuration are control-plane policy; a future privileged partition-retention worker owns physical expiry. Export requests and parent-linked retries are evidence records for the isolated Audit Export Worker, which will generate checksummed objects outside the API process.

Vault creation and rotation use a fresh 256-bit data-encryption key per version, AES-256-GCM authenticated encryption, and a separately GCM-wrapped data key under `AETHERGATE_VAULT_KEK`. Tenant, secret ID, and version are authenticated additional data. Public HTTP responses contain only masked metadata; plaintext resolution is an internal Go service boundary requiring actor/workload/purpose/request context and append-only access evidence. Persistent writes fail closed when the KEK is absent or invalid. The current `env-v1` KEK is a single-key boundary: do not replace it until a reviewed key-ring rewrap procedure exists.

LiteLLM diagnostics read sanitized server configuration and probe only `/health/liveliness` and `/health/readiness`. The Go server attaches the optional bearer key, rejects redirects, bounds discarded response bodies, and returns only status, latency, and normalized error evidence. The control plane never reads or mutates LiteLLM internal database tables. A successful diagnostic is not evidence that streaming, cancellation, virtual keys, routing, usage attribution, or provider failure behavior has passed real-stack integration testing.

## Migrations and owned data

Migrations live under [`internal/storage/postgres/migrations`](./internal/storage/postgres/migrations/README.md). They own only the `aethergate` database and create:

- organizations, workspaces, projects, and members;
- system roles and scoped role bindings;
- model catalog and model capabilities;
- API-key metadata and allowed-model relations;
- provider registry, active-probe queue, observed-health events, maintenance state, and routing eligibility;
- routing policies and provider targets;
- hierarchical rate-limit rules and budgets;
- alert rules and alert-incident history;
- webhook endpoints, event subscriptions, delivery attempts, and replay lineage;
- report schedules, timezone-aware next-run state, worker jobs, generation evidence, artifacts metadata, delivery state, and retry lineage;
- recipient-scoped inbox items, personal routing preferences, escalation policies, and external delivery evidence;
- Vault secret metadata, encrypted versions/data keys, rotation state, and immutable access evidence;
- immutable chained audit events, retention/legal-hold policy, and export/retry evidence.

The service must use supported LiteLLM APIs and must never query or mutate LiteLLM internal tables.

## Verification

```powershell
go test ./apps/api/...
go vet ./apps/api/...
```

OpenAPI contracts are under [`packages/contracts/openapi`](../../packages/contracts/openapi/README.md).

Responses marked `development-seed` or `development-memory` are bootstrap data. PostgreSQL storage does not by itself prove production readiness: identity middleware, enforced RBAC, automatic audit emission across every administrative domain, external KMS/key-ring and rewrap operations, live Vault resolution by isolated workers, provider-health, webhook, reports, notifications, audit-export and retention workers, backup/restore evidence, and a real PostgreSQL integration run must also pass.
