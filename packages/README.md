# Shared Packages

Shared packages exist only for code or schemas consumed by more than one application. Keep domain logic in the owning Go service or Console feature instead of creating a generic dumping ground.

- `ui`: reusable AetherGate interface components and the complex-table boundary.
- `contracts`: OpenAPI, JSON Schema, event schemas, and generated types.
- `database`: cross-repository database conventions and migration documentation.
- `sdk`: client SDK generation and maintained helpers.
- `config`: shared linting, formatting, build, and repository configuration.

