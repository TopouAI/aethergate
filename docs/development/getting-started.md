# Development Guide

## Current state

The repository currently defines the product, architecture, deployment source location, and application boundaries. Executable Console and Go scaffolds will be added next.

## Planned local toolchain

- Go for `apps/api` and `apps/worker`
- Node.js and pnpm for `apps/console` and TypeScript packages
- Docker with Docker Compose for LiteLLM, PostgreSQL, PgBouncer, and Redis
- PostgreSQL migration tooling selected with the Go data-access approach

Exact tool versions should be pinned in repository-managed version files when the executable scaffolds are introduced.

## Environment rules

- Start from checked-in `.env.example` files.
- Store local values in ignored `.env` files.
- Never copy production secrets into local development configuration.
- Use an isolated development database and LiteLLM master key.
- Prefer the reviewed Compose source in `deploy/compose/core` over ad hoc containers.

## Frontend

The Console uses HeroUI v3 with React 19+ and Tailwind CSS v4. Follow [`apps/console/README.md`](../../apps/console/README.md) for interaction and complex-table boundaries.

## Backend

The API and Worker use Go. Follow the domain boundaries in [`apps/api/README.md`](../../apps/api/README.md) and [`apps/worker/README.md`](../../apps/worker/README.md). The repository import path and Go workspace will be added when the GitHub organization path is confirmed.

## Documentation as part of development

Changes to configuration, ports, environment variables, migrations, service ownership, or operational behavior must update the corresponding documentation in the same pull request.

