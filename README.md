# WebTester

Security scanner and load testing tool. ZAP-backed scans, passive/BOLA checks, HTTP load testing.

## Prerequisites

- Go 1.26+
- Podman (or Docker) with compose plugin
- Redis 7+ (for auth token blacklist / rate limiting)

## Configuration

Copy example config:
```bash
cp config.example.yaml config.yaml
```

Config priority:
1. Hardcoded defaults
2. `config.yaml`
3. Environment variables (override YAML)

### Config fields (`config.yaml`)

```yaml
version: "1.0.0"
http_addr: ":9090"

database:
  path: "webtester.db"

zap:
  image: "ghcr.io/zaproxy/zaproxy:stable"
  timeout_seconds: 1800
  runtime: "podman" # "podman" or "docker"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

auth:
  access_secret: ""  # Set via AUTH_ACCESS_SECRET env var
  refresh_secret: "" # Set via AUTH_REFRESH_SECRET env var
  access_ttl: "15m"
  refresh_ttl: "168h"
```

## Run Locally

```bash
# Generate secrets
export AUTH_ACCESS_SECRET=$(openssl rand -hex 32)
export AUTH_REFRESH_SECRET=$(openssl rand -hex 32)

go run ./cmd/app
```

## Run with Podman / Docker Compose

```bash
# Using Podman Compose
podman-compose up -d

# Using Docker Compose
docker compose up -d
```

Override port or version via env:
```bash
APP_PORT=8080 APP_VERSION=1.0.0 podman-compose up -d
```

## CI/CD (GitHub Actions)

### Workflows

- **`.github/workflows/test.yml`**
  - Triggers on `push` and `pull_request` (branch: `main`).
  - Runs unit tests (`go test ./...`) on `ubuntu-latest`.
  - Runs `golangci-lint` with auto-fix and commit.

- **`.github/workflows/build.yml`** — Build Only
  - Manual trigger (`workflow_dispatch`).
  - Runs on self-hosted runner (`[self-hosted, Linux, X64]`).
  - Input: `version` (required, e.g. `1.0.0`).
  - Builds container image: `webtester:<version>`.
  - Image stored in local Podman image cache on `oong-serv`.

- **`.github/workflows/deploy.yml`** — Deploy Only
  - Manual trigger (`workflow_dispatch`).
  - Runs on self-hosted runner (`[self-hosted, Linux, X64]`).
  - Input: `version` (required, must match previously built image).
  - Verifies `webtester:<version>` exists in Podman.
  - Generates `config.yaml` from GitHub Secrets/Variables.
  - Deploys via `podman-compose up -d --remove-orphans`.
  - Health checks container and `/healthz` endpoint.

### Typical Workflow

1. **Build** → Run `Build Webtester` workflow with version `1.0.0`.
   - Creates `webtester:1.0.0` in local Podman storage.
   - No downtime, running container untouched.

2. **Deploy** → Run `Deploy Webtester` workflow with version `1.0.0`.
   - Writes `config.yaml` from GitHub Secrets/Variables.
   - Starts/updates container with `webtester:1.0.0`.

3. **Rollback** → Run `Deploy Webtester` with previous version `0.9.0`.
   - Instant rollback without rebuild.

### GitHub Secrets & Variables Setup

Navigate to **Settings → Secrets and variables → Actions**:

**Repository Secrets (Required):**
- `AUTH_ACCESS_SECRET`: 32+ character random secret (`openssl rand -hex 32`)
- `AUTH_REFRESH_SECRET`: 32+ character random secret (`openssl rand -hex 32`)
- `REDIS_PASSWORD`: Redis password (empty string if none)

**Repository Variables (Optional, defaults apply if unset):**
- `HTTP_ADDR`: Server listen address (default: `:9090`)
- `DB_PATH`: SQLite database path (default: `/data/webtester.db`)
- `REDIS_ADDR`: Redis host and port (default: `redis:6379`)
- `REDIS_DB`: Redis DB index (default: `0`)
- `ZAP_IMAGE`: ZAP container image (default: `ghcr.io/zaproxy/zaproxy:stable`)
- `ZAP_TIMEOUT_SECONDS`: Scan timeout seconds (default: `1800`)
- `ZAP_RUNTIME`: Container runtime for ZAP (default: `podman`)
- `AUTH_ACCESS_TTL`: JWT access token TTL (default: `15m`)
- `AUTH_REFRESH_TTL`: JWT refresh token TTL (default: `168h`)

## API Endpoints

### Public
- `GET  /healthz` — health check
- `GET  /api/v1/version` — app version
- `GET  /api/v1/auth/status` — check if initial user setup required
- `POST /api/v1/auth/setup` — initial admin registration
- `POST /api/v1/auth/login` — authenticate and get JWT tokens
- `POST /api/v1/auth/refresh` — refresh access token

### Protected (Bearer Token)
- `POST /api/v1/auth/logout` — revoke tokens
- `GET  /api/v1/auth/me` — current user profile
- `GET  /api/v1/scans` — list scans
- `POST /api/v1/scans` — start scan (`baseline`, `api`, `full`)
- `GET  /api/v1/scans/{scanId}` — scan status and progress
- `GET  /api/v1/scans/{scanId}/findings` — scan findings
- `GET  /api/v1/load-tests` — list load tests
- `POST /api/v1/load-tests` — start load test
- `GET  /api/v1/load-tests/{id}` — load test results
- `POST /api/v1/load-tests/{id}/cancel` — cancel running test

## Project Layout

- `cmd/app` — entrypoint
- `internal/auth` — authentication, user store, JWT, Redis token blacklist
- `internal/checks` — passive and BOLA security checks
- `internal/clients/zap` — ZAP runner client (Podman/Docker)
- `internal/configs` — YAML and environment configuration loader
- `internal/handlers` — HTTP route handlers
- `internal/loadtest` — load test execution runner
- `internal/routes` — router setup and middleware wiring
- `internal/services` — business logic layer
- `internal/stores` — SQLite storage engines
- `pkg/` — error handling, response helpers, middleware

