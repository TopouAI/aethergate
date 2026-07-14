# Importing the Existing LiteLLM Stack

## Decision

Copy the safe, version-controlled contents of the existing `aethergate-litellm-stack` into:

```text
aethergate/deploy/compose/core/
```

Do not create an additional nested folder. The final path should be:

```text
deploy/compose/core/compose.yaml
```

not:

```text
deploy/compose/core/aethergate-litellm-stack/compose.yaml
```

The server's running copy may stay at its current location, preferably:

```text
/opt/aethergate
```

or:

```text
/opt/aethergate-litellm-stack
```

The repository directory is the reviewed configuration source. The `/opt` directory is the runtime deployment containing environment-specific secrets and data.

## What to copy

The earlier stack was described with these source files:

```text
compose.yaml
.env.example
init-env.sh
litellm-config.yaml
backup.sh
verify.sh
postgres/init/01-create-databases.sh
pgbouncer/pgbouncer.ini
```

Copy these files and any other hand-maintained, non-secret configuration required by `compose.yaml`.

Review the stack's original `README.md` and merge stack-specific commands into `deploy/compose/core/README.md`. Do not keep two conflicting operational guides. It is acceptable to preserve the original temporarily as `STACK-README.import.md` while merging it.

## What must not be copied into Git

Never commit:

```text
.env
aethergate-backend.env
secrets/*
backups/*
logs/*
postgres_data/*
redis_data/*
database dumps
TLS private keys
real API keys or provider credentials
```

The repository `.gitignore` blocks common variants, but ignore rules are only a safety net. Review files before staging.

## If the stack exists only on the server

When the repository is also available on the server, use an explicit safe copy. Replace the repository path before running:

```bash
STACK_DIR=/opt/aethergate-litellm-stack
REPO_DIR=/path/to/aethergate

rsync -av \
  --exclude='.env' \
  --exclude='*.env' \
  --exclude='secrets/' \
  --exclude='backups/' \
  --exclude='logs/' \
  --exclude='postgres_data/' \
  --exclude='redis_data/' \
  --exclude='README.md' \
  "$STACK_DIR/" "$REPO_DIR/deploy/compose/core/"
```

This deliberately excludes every `*.env`, including the generated `aethergate-backend.env`. The checked-in `.env.example` can then be copied separately after manually confirming that it contains placeholders only:

```bash
cp "$STACK_DIR/.env.example" "$REPO_DIR/deploy/compose/core/.env.example"
```

If `rsync` is unavailable, copy only the allow-listed files individually instead of copying the whole runtime directory.

## Review after import

From the repository root:

```bash
git status --short
git diff -- deploy/compose/core
```

Before staging, verify:

- `compose.yaml` uses environment variables rather than embedded passwords;
- `.env.example` contains placeholders, not working credentials;
- scripts do not print secrets unnecessarily;
- no backup, volume, log, or generated environment file is listed;
- image tags are fixed or controlled through explicit environment variables;
- `docker compose config` resolves every referenced file.

## Ongoing workflow

1. Change deployment source in `deploy/compose/core`.
2. Review and commit the change.
3. Back up the server.
4. Sync the reviewed source to the server runtime directory while preserving `.env`, secrets, backups, and volumes.
5. Run configuration validation and the stack verification script.

Do not make long-lived, undocumented edits only in `/opt`; they will drift from the repository and become difficult to reproduce.

