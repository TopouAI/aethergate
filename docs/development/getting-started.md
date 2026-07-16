# Development Guide

Step-by-step instructions for running the AetherGate Console, API, and supporting infrastructure on a local workstation. See the [repository layout](../../README.md#repository-layout) for how these pieces fit together.

This is a public, version-controlled document (linked from the root README). Keep real hostnames, IP addresses, and credentials out of it — use placeholders like `YOUR_SERVER_IP` and keep the real values only in your own ignored `.env` files, exactly as the [environment rules](#environment-rules) below describe.

## Prerequisites

| Tool | Version | Check |
| --- | --- | --- |
| Node.js | 20 or newer | `node -v` |
| npm | 11 or newer | `npm -v` |
| Go | 1.26.4 or newer | `go version` |
| Docker Desktop with Compose v2 | latest | `docker compose version` (only for the [local infrastructure stack](#optional-run-the-infrastructure-stack-locally)) |

On Windows, a plain `go version` can fail with "not recognized" even when Go is installed, if its `bin` directory was never added to the user or machine `PATH` (common when Go was installed manually rather than through the official installer's default option, or when only an IDE's Go SDK setting points at it). If that happens: add Go's `bin` directory to `PATH`, call the full path to `go.exe` for CLI commands, or rely on your IDE's own Go SDK configuration (see [GoLand](#option-b-goland-run-configuration) below), which doesn't need `PATH` at all.

## Daily startup sequence

This is the routine for working against a shared PostgreSQL/LiteLLM stack such as the one described in [`docs/deployment/server-foundation.md`](../deployment/server-foundation.md). If you'd rather run everything on your own machine instead, skip to [Quick start without a database](#quick-start-without-a-database) or [running the infrastructure stack locally](#optional-run-the-infrastructure-stack-locally).

### 1. Open the SSH tunnel

PostgreSQL and PgBouncer are not exposed publicly by that deployment (see [`deploy/compose/core/README.md`](../../deploy/compose/core/README.md)), so reach them through a tunnel. Keep a dedicated PowerShell window open for the whole session:

```powershell
ssh -N `
  -p 22 `
  -o ExitOnForwardFailure=yes `
  -o ServerAliveInterval=30 `
  -L 6432:127.0.0.1:6432 `
  -L 5433:127.0.0.1:5433 `
  root@YOUR_SERVER_IP
```

Replace `YOUR_SERVER_IP` and the login user with your actual deployment target — keep those in your own notes or password manager, not in chat transcripts or committed files. `-N` opens the tunnel without a remote shell; `6432` is the pooled PgBouncer port and `5433` is the direct PostgreSQL port (see step 2). Leave the window running until you're done for the day.

### 2. Apply database migrations (first run, and after any schema change)

From the repository root:

```powershell
Get-Content apps/api/.env | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
        Set-Item "Env:$($matches[1].Trim())" $matches[2].Trim()
    }
}

$env:AETHERGATE_DATABASE_URL = $env:AETHERGATE_DIRECT_DATABASE_URL
go run ./apps/api/cmd/migrate
```

This loads `apps/api/.env` into the current shell, then points the migration at the direct PostgreSQL port instead of the pooled one — migrations should never run through PgBouncer's transaction pool. `AETHERGATE_DIRECT_DATABASE_URL` isn't read by the application itself; it's a convenience variable you keep alongside `AETHERGATE_DATABASE_URL` in your own `.env` purely so this snippet has something to swap in. Success prints:

```text
AetherGate database is up to date
```

You only need to repeat this after a migration is added, not on every startup.

### 3. Start the Go API

#### Option A: command line

In its own terminal, from the repository root (after loading `apps/api/.env` the same way as above, without overriding `AETHERGATE_DATABASE_URL`):

```powershell
go run ./apps/api/cmd/server
```

#### Option B: GoLand Run Configuration

1. Create a **Go Build** run configuration.
2. **Run kind**: Directory. **Directory**: `apps/api/cmd/server` under the repository root.
3. **Working directory**: the repository root.
4. **Environment**: point the `Environment files` field at `apps/api/.env` if your GoLand version has one; otherwise open the `Environment variables` editor and paste in the non-comment lines from that file.
5. Run or Debug the configuration.

Either way, the server listens at `http://localhost:8080` by default. Verify it:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

This should return `status: ok`. Use the pooled URL (port `6432`) here, not the direct migration URL (port `5433`) — the running API should always go through PgBouncer.

### 4. Start the Console

In another terminal:

```powershell
npm install
npm run dev
```

(`npm install` is only needed the first time, or after dependencies change.) Open `http://localhost:3000` — use `localhost`, not `127.0.0.1`: the API's CORS policy currently allows only `http://localhost:3000` as an origin.

### Current state at a glance

| Component | How it's running |
| --- | --- |
| SSH tunnel | Keep the PowerShell window from step 1 open |
| Go API, `:8080` | GoLand Run/Debug, or `go run` |
| Console, `:3000` | `npm run dev` |
| LiteLLM, `:4000` | Remote Docker stack |
| PostgreSQL / PgBouncer | Reached through the SSH tunnel |

## Quick start without a database

For quick Console or API work that doesn't need persistence, skip the tunnel and migrations entirely:

```powershell
npm install
npm run dev
```

```powershell
go run ./apps/api/cmd/server
```

With no `AETHERGATE_DATABASE_URL` set, the server logs `using development memory repository` and keeps all state in process memory — nothing persists across restarts.

## API reference: endpoints and environment variables

Core endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /api/v1/overview`
- `GET /api/v1/requests`
- `GET /api/v1/requests/{requestID}`

The API also exposes organization, API-key, workspace, project, member, model, provider-health, routing, rate-limit, budget, alert, webhook, report, notification, audit, and Vault endpoints — see [`apps/api/README.md`](../../apps/api/README.md) for the complete list.

| Variable | Purpose |
| --- | --- |
| `AETHERGATE_API_ADDR` | HTTP listen address; defaults to `:8080`. |
| `AETHERGATE_DATABASE_URL` | PostgreSQL/PgBouncer connection string. Empty selects the in-memory development repository. |
| `AETHERGATE_AUTO_MIGRATE` | Set to `true` only in controlled development environments to migrate at startup. Prefer the explicit `migrate` command. |
| `AETHERGATE_VAULT_KEK` | Required for persistent Vault writes: standard Base64 encoding of exactly 32 random bytes, unique per environment. Never expose it to the Console or commit it. |
| `LITELLM_BASE_URL` | Internal absolute HTTP(S) URL for LiteLLM, e.g. `http://127.0.0.1:4000` for a tunneled or local stack. |
| `LITELLM_MASTER_KEY` | Optional server-only bearer credential for LiteLLM health probes. Never expose it through `NEXT_PUBLIC_*`, logs, or Git. |

The Console reads its own API base URL from `apps/console/.env`:

```text
NEXT_PUBLIC_AETHERGATE_API_URL=http://localhost:8080/api/v1
```

That is also the built-in default, so only create the file if you need to point the Console at a different API instance.

`apps/worker` has no runnable entry point yet; it currently only documents its planned responsibilities in [`apps/worker/README.md`](../../apps/worker/README.md).

## Optional: run the infrastructure stack locally

[`deploy/compose/core`](../../deploy/compose/core/README.md) is the reviewed source for the LiteLLM, PostgreSQL 17, PgBouncer, and Redis stack that also backs the [server deployment](../deployment/server-foundation.md). The same Compose file runs directly on a workstation instead of a remote server — in that case skip that guide's `/opt` upload and SSH-tunnel steps, which apply only to the remote scenario.

`init-env.sh`, `backup.sh`, and `verify.sh` are bash scripts. On Windows, run them from Git Bash (installed with Git for Windows) or WSL; they will not run directly in PowerShell.

```bash
cd deploy/compose/core
./init-env.sh          # generates .env, aethergate-backend.env, secrets/pgbouncer_users.txt with random credentials
docker compose config --quiet
docker compose pull
docker compose up -d
```

Prefer to fill in values yourself instead? Copy `.env.example` to `.env` and replace every `CHANGE_ME` rather than running `init-env.sh`.

Local ports (from the generated `.env` defaults):

- LiteLLM: `http://localhost:4000` (UI at `/ui`; read the generated `UI_USERNAME` / `UI_PASSWORD` from `.env`)
- PostgreSQL (direct): `127.0.0.1:5433`
- PgBouncer: `127.0.0.1:6432`

Point the API at this stack using those host-mapped ports, not the Docker-internal hostnames in the generated `aethergate-backend.env` (`pgbouncer`, `postgres`, `litellm` only resolve for a container joining the same Compose network).

Never commit the generated `.env`, `aethergate-backend.env`, or `secrets/pgbouncer_users.txt` — all three are gitignored and must stay that way.

## Verify your changes

Run before opening a pull request:

```powershell
npm run typecheck
npm run lint
npm run build
go test ./apps/api/...
go vet ./apps/api/...
```

## Environment rules

- Store local values in ignored `.env` files next to the app that reads them (`apps/console/.env`, `apps/api/.env`, `deploy/compose/core/.env`).
- Only `deploy/compose/core` ships a checked-in `.env.example` today. `apps/console` and `apps/api` don't have one yet, so set their variables directly using the tables above until an example file is added.
- Never copy production secrets, master keys, or database dumps into local configuration, and never paste real hostnames, IP addresses, or credentials into chat, issues, or committed files. If a real secret is ever exposed that way, treat it as compromised and rotate it.
- Use an isolated development database and LiteLLM master key; never point local development at the production stack.
- Prefer the reviewed Compose source in `deploy/compose/core` over ad hoc containers.

## Frontend

The Console uses HeroUI v3 with React 19+ and Tailwind CSS v4. Follow [`apps/console/README.md`](../../apps/console/README.md) for interaction and complex-table boundaries.

## Backend

The API and Worker use Go under the `github.com/topoai/aethergate` module. Follow the domain boundaries in [`apps/api/README.md`](../../apps/api/README.md) and [`apps/worker/README.md`](../../apps/worker/README.md).

## Documentation as part of development

Changes to configuration, ports, environment variables, migrations, service ownership, or operational behavior must update the corresponding documentation in the same pull request.
