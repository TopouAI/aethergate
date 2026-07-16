# AetherGate API migrations

Migrations in this directory own only the `aethergate` database. They must never read or mutate LiteLLM tables.

The foundation migration creates the enterprise tenant hierarchy, member and role bindings, model catalog, API-key metadata, provider connections, provider-health probe/event evidence, routing, rate limits, budgets, alerts, webhooks, scheduled reports and run history, recipient-scoped notifications, preferences, escalation policies, delivery evidence, Enterprise Vault, and immutable audit-event storage. Audit events include actor/resource context, JSON before/after state, a per-tenant SHA-256 forward chain, retention/legal-hold policy, export queue evidence, and retry lineage. A database trigger rejects every `UPDATE` or `DELETE` against `audit_events`; unique chain-head constraints reject competing branches.

Vault stores masked metadata separately from versioned AES-256-GCM ciphertext, nonces, and wrapped per-version data keys. Ciphertext rows are never exposed by the public repository response model. Vault access events are append-only and capture actor, workload, purpose, outcome, request, and source IP without secret values.

API-key plaintext secrets are never stored; only a 32-byte digest is persisted. Provider-health events persist routing eligibility and transition evidence; outbound probes, internal Vault resolution, report generation/delivery, external notification delivery, audit export generation, and privileged partition expiry remain worker-owned.

The down migration is intended for disposable development environments. Production rollback should normally use a forward corrective migration and a database restore plan.
