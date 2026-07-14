# AetherGate Worker

Go processes for asynchronous work that should not extend control-plane API latency.

Planned jobs include:

- LiteLLM usage-event ingestion and normalization;
- ClickHouse batch writes and aggregation refreshes;
- OpenMeter event publication;
- report generation and exports;
- budget and anomaly notifications;
- reconciliation, retention, and scheduled maintenance.

Workers must use idempotency keys, bounded retries, dead-letter handling, and observable job status. A request identifier and organization context should remain traceable from the gateway event through analytics and billing.

