# Contributing to AetherGate

Thank you for helping build AetherGate.

## Before opening a change

1. Check existing issues and discussions to avoid duplicate work.
2. For a material architecture or product change, open a proposal before implementation.
3. Keep the open-source edition independently deployable and useful.
4. Never include credentials, customer data, production logs, database dumps, or generated `.env` files.

## Repository boundaries

- `apps/console`: user-facing Next.js application.
- `apps/api`: Go control-plane API and domain rules.
- `apps/worker`: Go background processing and usage ingestion.
- `packages`: shared contracts, UI boundaries, SDKs, and repository configuration.
- `deploy`: reviewed deployment source only, never live secrets or runtime data.
- `docs`: product, architecture, development, deployment, and roadmap documentation.

## Pull requests

- Keep changes focused and explain the user or operator outcome.
- Add or update tests for behavioral changes.
- Update documentation when behavior, configuration, or deployment steps change.
- Call out migrations, compatibility concerns, and security implications.
- Use clear English in code and public APIs. Chinese documentation may be maintained alongside English documentation.

Detailed build and test commands will be added as the application scaffolds become executable.

