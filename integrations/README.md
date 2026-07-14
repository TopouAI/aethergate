# Integrations

Adapters and operational notes for systems that AetherGate does not own. Each integration must expose an internal interface so external API changes do not leak throughout the product.

- `litellm`: model gateway and administrative API.
- `clickhouse`: high-volume usage analytics, introduced in phase two.
- `openmeter`: metering and billing events, introduced in phase three.

