# AetherGate OpenAPI

The OpenAPI contracts describe currently implemented Go HTTP behavior:

- [`aethergate.yaml`](./aethergate.yaml) covers system health and observability bootstrap endpoints.
- [`enterprise.yaml`](./enterprise.yaml) covers organization and API-key lifecycle endpoints.
- [`foundation.yaml`](./foundation.yaml) covers workspace, project, member, and model catalog lifecycle endpoints.
- [`providers.yaml`](./providers.yaml) covers non-secret provider registry, probe queueing, active/passive health evidence, maintenance windows, and routing eligibility.
- [`routing.yaml`](./routing.yaml) covers routing-policy authoring, targets, and health-gated activation.
- [`rate-limits.yaml`](./rate-limits.yaml) covers hierarchical rule lifecycle and deterministic dry-run decisions.
- [`budgets.yaml`](./budgets.yaml) covers hierarchical budgets, thresholds, forecasts, and dry-run decisions.
- [`alerts.yaml`](./alerts.yaml) covers alert-rule lifecycle, incident history, and deterministic dry-run evaluation.
- [`webhooks.yaml`](./webhooks.yaml) covers subscriptions, one-time signing secrets, queued tests, delivery evidence, retries, and replay.
- [`reports.yaml`](./reports.yaml) covers timezone-aware schedules, recipients, output formats, worker-queued runs, delivery evidence, and retries.
- [`notifications.yaml`](./notifications.yaml) covers recipient-scoped inbox state, personal channel routing, quiet hours and digests, escalation policies, worker-queued external delivery, and retries.
- [`audit.yaml`](./audit.yaml) covers immutable event append/search, SHA-256 chain verification, retention/legal hold, and worker-queued export evidence and retries.
- [`vault.yaml`](./vault.yaml) covers metadata-only envelope-encrypted secret creation, rotation, disablement, and immutable access evidence; plaintext resolution is intentionally absent from public HTTP.
- [`integrations.yaml`](./integrations.yaml) covers credential-safe LiteLLM configuration state and server-side liveness/readiness diagnostics without key, body, or database exposure.

Add an endpoint only when its handler exists. Planned product capabilities belong in the feature-parity matrix until implementation starts.

The next contract step is to pin an OpenAPI validator as a reviewed repository dependency, run it in CI, and generate the Console client and Go conformance checks from the same sources.
