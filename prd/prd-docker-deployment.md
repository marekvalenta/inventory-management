# PRD: Docker Deployment — InventoryManagement

> **Status:** Draft v1.0
> **Scope:** Multi-stage Dockerfile, docker-compose with Watchtower auto-updates, GitHub Container Registry publishing, health check, data persistence, NAS-ready deployment guide.
> **Backlog Position:** #10

---

## 0. Cross-PRD Consistency Report

### PRDs Reviewed
- `prd-overarching-architecture.md` — Single-container deployment model, volume mounts, env vars, image size target (~20-40 MB), health endpoint, docker-compose skeleton.
- `prd-project-setup.md` — `make docker` target, `make build` pipeline (React → Go), repo structure, CGO-free SQLite driver.
- `prd-database-schema.md` — SQLite WAL mode, migration runner at startup, `/data/inventory.db` file path, auto-seed root location.
- `prd-backend-architecture.md` — Go serves embedded React SPA, chi router, `/api/v1/health` endpoint, middleware timeout.
- `prd-frontend-architecture.md` — React build outputs to `frontend/dist/`, Vite proxy, TanStack Query.
- `prd-testing.md` — `make test-fast` validation gate, seed data, local dev run modes.

### Conflicts & Resolutions

| # | Conflict | PRDs Involved | Resolution |
|---|---|---|---|
| - | No conflicts detected. This PRD is consistent with all prior PRDs. | - | - |

### Confirmed Alignments
- Volume mount `/data` → `inventory.db` — consistent with overarching §7.2, database PRD TR-2.
- Environment variables (`APP_PORT`, `APP_NAME`, `DATA_DIR`, `LOG_LEVEL`) — consistent with overarching §7.1.
- `make docker` calls `docker build -t inventory-management:latest .` — consistent with project-setup FR-10.
- `make build` compiles React then Go binary at `bin/server` — prerequisite for Docker build.
- CGO-free build (`modernc.org/sqlite`, `CGO_ENABLED=0`) — consistent with overarching §3.1.
- Health endpoint `/api/v1/health` returns `{"status":"ok"}` — consistent with overarching §5.2, project-setup FR-5.
- Single container, no external DB or reverse proxy — consistent with overarching §3.4.
- Scope does not contradict any non-goal in overarching PRD.

---

## 1. Overview & Problem Statement

This PRD defines how InventoryManagement is packaged, distributed, and deployed as a Docker container. The goal is a single-command deployment (`docker compose up -d`) that brings up a fully working application with automatic database setup, health monitoring, and optional hands-free updates via Watchtower.

### Core Deliverables
1. **Multi-stage Dockerfile** — builds React frontend + Go binary in separate stages, produces a minimal Alpine runtime image.
2. **docker-compose.yml** — orchestrates the app container with persistent data volume, environment configuration, and Watchtower for auto-updates.
3. **GitHub Container Registry publishing** — GitHub Actions workflow that builds and pushes the image to `ghcr.io/marekvalenta/inventory-management` on push to main.
4. **Health check** — Docker-native health monitoring via the `/api/v1/health` endpoint.
5. **Data persistence** — Volume mount ensures zero data loss on container restart or upgrade.
6. **Deployment guide** — Step-by-step instructions in the README for cloning, building, and deploying.

---

## 2. Goals & Measurable Success Metrics

| Goal | Metric |
|---|---|
| Image size | Final Alpine image < 25 MB |
| Idle memory | Container idle < 50 MB RAM |
| Build time | `make docker` completes in < 2 minutes on modern hardware |
| Single-command deploy | `docker compose up -d` brings up healthy app in < 10 seconds |
| Health monitoring | Docker reports container as healthy within 30s of startup |
| Zero data loss | Container restart/rebuild preserves all data in mounted volume |
| Auto-update | Watchtower detects new GHCR image and restarts container within polling interval |
| CI publishing | Push to main → image published to GHCR within 5 minutes |

---

## 3. Critical Risks & Technical Assumptions

| Risk | Mitigation |
|---|---|
| Alpine `wget` missing for healthcheck | `wget` is included in the Alpine base image by default. Verify in Dockerfile. |
| Go binary links against musl (Alpine uses musl, not glibc) | Build with `CGO_ENABLED=0` — produces a statically linked binary that runs on any Linux kernel regardless of libc. |
| SQLite WAL files not cleaned up on SIGTERM | Go server handles OS signals (SIGTERM, SIGINT) and performs graceful shutdown with DB close, which triggers WAL checkpoint. |
| GHCR push fails due to missing permissions | GitHub Actions uses `GITHUB_TOKEN` with default `packages: write` scope. Public repo — no extra configuration needed. |
| Watchtower pulls broken image | Watchtower only updates to the same tag (default: `latest`). Pin to `:latest` or a specific version. Rollback is manual. |
| Port 8080 conflicts on host | docker-compose `ports` mapping is configurable (e.g., `"9090:8080"`). Document how to change it. |
| `/data` directory permissions | Docker creates the volume mount directory with root ownership. The container runs as root by default (acceptable for single-user NAS). If running rootless, document `user` directive. |
| Multi-stage build caching inefficiency | Order stages for optimal layer caching: npm dependencies → React build → Go dependencies → Go build → runtime. `COPY` only changed files. |

---

## 4. User Stories & Acceptance Criteria

### US-001: Build Docker Image Locally
**Description:** As a developer, I want to build the production Docker image with a single command.

**Acceptance Criteria:**
- [ ] `make docker` builds the image and tags it `inventory-management:latest`.
- [ ] `make build` runs first (React build + Go compile) before the Docker build.
- [ ] Final image size is < 25 MB (verified via `docker images`).
- [ ] Image contains only the Go binary and necessary runtime files — no Go toolchain, no node_modules, no source code.
- [ ] Build uses Docker layer caching — subsequent builds for unchanged layers complete in < 30s.
- [ ] Typecheck / build / test suite passes.

### US-002: Deploy with docker-compose
**Description:** As a user, I want to start the full application stack with a single compose command.

**Acceptance Criteria:**
- [ ] `docker compose up -d` starts the inventory container and Watchtower container.
- [ ] Inventory container exposes port 8080 (configurable via `APP_PORT` env var).
- [ ] `/data` directory is mounted as a volume — database persists across container recreations.
- [ ] Environment variables (`APP_NAME`, `APP_PORT`, `DATA_DIR`, `LOG_LEVEL`) are configurable in the compose file or `.env` file.
- [ ] `restart: unless-stopped` ensures the container survives Docker daemon restarts.
- [ ] `docker compose down` stops and removes both containers. Data volume is NOT removed.
- [ ] Typecheck / build / test suite passes.

### US-003: Health Check Monitoring
**Description:** As an operator, I want Docker to automatically detect and restart unhealthy containers.

**Acceptance Criteria:**
- [ ] Dockerfile includes a `HEALTHCHECK` directive that calls `wget -qO- http://localhost:8080/api/v1/health` every 30s.
- [ ] Container is marked as `healthy` within 30s of startup (after migrations + HTTP server ready).
- [ ] If the health endpoint becomes unreachable, Docker marks the container `unhealthy` and eventually restarts it (depending on restart policy).
- [ ] `docker ps` shows health status for the container.
- [ ] Typecheck / build / test suite passes.

### US-004: Automatic Updates via Watchtower
**Description:** As a NAS user, I want the application to update itself automatically when a new image is published.

**Acceptance Criteria:**
- [ ] docker-compose includes a `watchtower` service that monitors only the `inventory` container.
- [ ] Watchtower polls GHCR every 24 hours by default (configurable via `WATCHTOWER_POLL_INTERVAL`).
- [ ] On detecting a new `:latest` image, Watchtower pulls it, gracefully stops the old container, and starts the new one.
- [ ] Old images are cleaned up (`WATCHTOWER_CLEANUP=true`) to prevent disk bloat on the NAS.
- [ ] Watchtower sends notifications on update (optional — configured via `WATCHTOWER_NOTIFICATIONS` env var, default: none in v1).
- [ ] Typecheck / build / test suite passes.

### US-005: Publish Image to GitHub Container Registry
**Description:** As a maintainer, I want Docker images automatically published to GHCR on every push to the main branch so users can pull the latest version.

**Acceptance Criteria:**
- [ ] GitHub Actions workflow (`.github/workflows/docker-publish.yml`) triggers on push to `main`.
- [ ] Workflow checks out code, logs into GHCR, builds the multi-stage Docker image, and pushes it with tags `latest` and the commit SHA.
- [ ] Image is published at `ghcr.io/marekvalenta/inventory-management:latest`.
- [ ] Workflow uses `GITHUB_TOKEN` for authentication (no personal access token required).
- [ ] Workflow fails cleanly if build or push fails — error visible in GitHub Actions UI.
- [ ] Typecheck / build / test suite passes.

### US-006: Data Persistence Across Container Lifecycle
**Description:** As a user, I want all my inventory data to survive container recreation, upgrades, and system reboots.

**Acceptance Criteria:**
- [ ] SQLite database file is stored at `/data/inventory.db` inside the container.
- [ ] `/data` is mounted from a Docker volume or host directory declared in docker-compose.
- [ ] `docker compose down && docker compose up -d` — all previous data is intact.
- [ ] `docker compose pull && docker compose up -d` (Watchtower upgrade path) — all data intact.
- [ ] On first run, migrations auto-run and root location is auto-seeded.
- [ ] On subsequent runs, existing database is detected and migrations are skipped (already applied).
- [ ] Typecheck / build / test suite passes.

### US-007: Clear Deployment Documentation
**Description:** As a user, I want step-by-step instructions in the README for deploying the application with Docker.

**Acceptance Criteria:**
- [ ] README includes a **Docker Deployment** section with:
  - Prerequisites (Docker + Docker Compose installed).
  - Option 1: Pull from GHCR (`docker compose up -d` with pre-built image).
  - Option 2: Build locally (`make docker && docker compose up -d`).
  - How to customize environment variables (`.env` file or direct compose edits).
  - How to access the app (`http://<host-ip>:8080`).
  - How to stop/update/check logs.
  - Volume backup note ("back up the `./data` directory").
- [ ] README is concise — deployment section under 40 lines.
- [ ] Sample `docker-compose.yml` included so users can copy-paste.

---

## 5. Functional & Technical Requirements

### 5.1 Multi-Stage Dockerfile

**FR-1:** The Dockerfile uses three build stages:

```
Stage 1: node-builder  (node:20-alpine)
  → npm ci --prefix frontend
  → npm run build --prefix frontend
  → Output: frontend/dist/

Stage 2: go-builder     (golang:1.22-alpine)
  → COPY --from=node-builder frontend/dist/ /build/frontend/dist/
  → COPY go.mod go.sum ./
  → RUN go mod download
  → COPY . .
  → RUN CGO_ENABLED=0 GOOS=linux go build -o /build/server ./cmd/server
  → Output: /build/server

Stage 3: runtime        (alpine:latest)
  → COPY --from=go-builder /build/server /app/server
  → EXPOSE 8080
  → HEALTHCHECK ...
  → ENTRYPOINT ["/app/server"]
```

**FR-2:** Full `Dockerfile`:

```dockerfile
# Stage 1: Build React frontend
FROM node:20-alpine AS node-builder

WORKDIR /build

COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN npm ci --prefix frontend

COPY frontend/ ./frontend/
RUN npm run build --prefix frontend

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS go-builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=node-builder /build/frontend/dist/ ./frontend/dist/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=go-builder /build/server .

RUN mkdir -p /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["./server"]
```

**FR-3:** Build flags explanation:
- `CGO_ENABLED=0` — produces a fully static binary with no C library dependency.
- `GOOS=linux` — target OS (explicit, though Alpine builder is already Linux).
- `GOARCH=amd64` — target architecture.
- `-ldflags="-s -w"` — strips debug symbols and DWARF data, reducing binary size by ~30%.

**FR-4:** Runtime stage additions:
- `ca-certificates` — enables TLS for potential future features (HTTPS outbound calls).
- `tzdata` — timezone support for correct `CURRENT_TIMESTAMP` in SQLite.
- `/data` directory created for the SQLite volume mount point.

**FR-5:** `.dockerignore` file to exclude unnecessary files from the build context:

```
# Go
/bin/
/tmp/
*.exe
*.test
*.out
go.work
go.work.sum

# Node / Frontend
frontend/node_modules/
frontend/.vite/

# SQLite
*.db
*.db-shm
*.db-wal
/data/

# Git
.git/
.gitignore
.gitattributes

# Docs
*.md
prd/
AGENTS.md

# IDE
.vscode/
.idea/

# E2E
e2e/node_modules/

# OS
.DS_Store
Thumbs.db
desktop.ini
```

### 5.2 docker-compose.yml

**FR-6:** `docker-compose.yml` at the project root:

```yaml
services:
  inventory:
    image: ghcr.io/marekvalenta/inventory-management:latest
    container_name: inventory
    ports:
      - "${APP_PORT:-8080}:8080"
    volumes:
      - ./data:/data
    environment:
      - APP_PORT=8080
      - APP_NAME=${APP_NAME:-Inventory}
      - DATA_DIR=/data
      - LOG_LEVEL=${LOG_LEVEL:-info}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

  watchtower:
    image: containrrr/watchtower:latest
    container_name: watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_POLL_INTERVAL=86400
      - WATCHTOWER_INCLUDE_STOPPED=false
      - WATCHTOWER_NOTIFICATIONS=none
    restart: unless-stopped
```

**FR-7:** Environment variables used in compose:
- `APP_PORT`: Host port mapping. Default `8080`. Change to `9090:8080` if 8080 is in use.
- `APP_NAME`: Display name in the UI. Default `"Inventory"`. User overrides via `.env` file.
- `LOG_LEVEL`: Logging verbosity. Default `"info"`. Options: `debug`, `info`, `warn`, `error`.
- `WATCHTOWER_POLL_INTERVAL`: Seconds between registry polls. Default 86400 (24 hours).
- `WATCHTOWER_CLEANUP`: Remove old images after update. Prevents disk bloat.
- `WATCHTOWER_INCLUDE_STOPPED`: Only monitors running containers.
- `WATCHTOWER_NOTIFICATIONS`: Set to `none` in v1. User can configure email/Slack/webhook later.

**FR-8:** Optional `.env` file template for user customization (not committed — listed in `.gitignore`):

```bash
# InventoryManagement — User Settings
APP_NAME=My Inventory
APP_PORT=8080
LOG_LEVEL=info
```

**FR-9:** Watchtower is scoped to monitor only the `inventory` container. This is implicitly handled because Watchtower by default monitors all containers on the host. In v1, this is acceptable since the compose file only has two containers and Watchtower won't update itself unless configured to. For stricter scoping in v2, add `command: --label-enable` to Watchtower and `labels: com.centurylinklabs.watchtower.enable=true` to the inventory service.

### 5.3 GitHub Actions CI/CD

**FR-10:** `.github/workflows/docker-publish.yml`:

```yaml
name: Publish Docker Image

on:
  push:
    branches: [main]
  workflow_dispatch:

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=raw,value=latest
            type=sha,prefix=

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

**FR-11:** GitHub Actions uses:
- `docker/build-push-action@v6` — the official Docker build action with BuildKit support.
- `cache-from/cache-to: type=gha` — GitHub Actions cache for Docker layers, speeding up subsequent builds.
- `secrets.GITHUB_TOKEN` — auto-generated token with `packages: write` scope for public repos.
- Two tags pushed: `latest` and the short commit SHA (e.g., `abc1234`).
- `workflow_dispatch` trigger — allows manual runs from the GitHub Actions UI.

### 5.4 Makefile Integration

**FR-12:** Update `make docker` target in the Makefile:

```makefile
## docker: Build production Docker image
docker: build
	docker build -t inventory-management:latest .
```

The existing target already does this — confirm it still works and add `--platform linux/amd64` if the build host is ARM:

```makefile
## docker: Build production Docker image
docker: build
	docker build --platform linux/amd64 -t inventory-management:latest .
```

**FR-13:** `make docker` depends on `make build` — the `build` target must succeed first (compiles React + Go binary at `bin/server`). This ensures the `Dockerfile` COPY commands find the expected files.

### 5.5 README Updates

**FR-14:** Replace the existing "Docker Deployment" section in `README.md` with:

```markdown
## Docker Deployment

### Option 1: Pull from GitHub Container Registry (Recommended)

Create `docker-compose.yml`:

[Content of docker-compose.yml from FR-6]

Run:
```bash
docker compose up -d
```

Access at `http://localhost:8080` (or your NAS IP).

### Option 2: Build Locally

```bash
git clone https://github.com/marekvalenta/inventory-management.git
cd inventory-management
make docker
docker compose up -d
```

### Managing the App

```bash
# View logs
docker compose logs -f

# Stop
docker compose down

# Update to latest image
docker compose pull && docker compose up -d

# Back up your data
cp -r ./data ./backup-$(date +%Y%m%d)
```

**Data is stored in `./data/inventory.db`.** Back up this directory before major upgrades. Mount it on NAS persistent storage.
```

**FR-15:** Keep the existing README quick-start section for local development unchanged. The Docker section is appended separately.

### 5.6 Image Size & Performance Targets

**FR-16:** Expected image layer sizes:

| Layer | Size |
|---|---|
| alpine:latest base | ~3 MB |
| ca-certificates + tzdata | ~3 MB |
| Go binary (stripped) | ~12 MB |
| **Total** | **~18-20 MB** |

**FR-17:** Container resource limits (optional, documented in compose comments):

```yaml
# Uncomment to set resource limits
# deploy:
#   resources:
#     limits:
#       memory: 128M
#       cpus: '0.5'
```

Not enforced by default — NAS users can enable if needed.

---

## 6. Edge Cases & Failure Modes

| Scenario | System Behavior |
|---|---|
| `/data` directory is read-only or unwriteable | Go server fails to open/create SQLite database. Process exits with fatal error. Docker restarts container (depending on restart policy). Error visible in `docker logs`. |
| Port 8080 already in use on host | Docker fails to start the container with "port is already allocated" error. User must change `APP_PORT` in `.env` file. |
| GHCR image pull fails (network down, registry unreachable) | On first deploy: `docker compose up` fails with pull error. On Watchtower update: Watchtower logs the error and retries on next poll interval. Existing container keeps running. |
| Watchtower pulls a broken image | New container starts with broken image. Healthcheck eventually marks it unhealthy. User must manually roll back: `docker compose down && docker pull ghcr.io/marekvalenta/inventory-management:<previous-sha> && docker compose up -d`. |
| Multiple containers write to same SQLite file | SQLite WAL mode supports concurrent readers but serialized writers. If user accidentally runs two containers against the same volume, the second container's writes wait on the lock. Application remains functional but may see `database is locked` errors (5s timeout). |
| Container receives SIGTERM during active write | Go server's graceful shutdown handler drains in-flight requests before closing the DB connection. SQLite WAL checkpoint runs on close. No data corruption. |
| NAS volume runs out of disk space | SQLite write fails. API returns 500 with "database or disk is full" error. Go server remains running (returns errors, does not crash). User must free space. |
| Docker daemon not running | `docker compose up -d` fails immediately with "Cannot connect to the Docker daemon" error. User starts Docker daemon and retries. |
| `docker compose` plugin not installed (old Docker) | User needs `docker-compose` (standalone, with hyphen). Document both command forms in README. |
| Alpine base image CVE reported | Rebuild with `docker build --no-cache` to pull the latest Alpine patch. Watchtower handles this automatically if the image is rebuilt and pushed. |
| Go binary compiled for wrong architecture | Build fails at container startup with "exec format error". Fixed by ensuring `GOARCH=amd64` in the Dockerfile. |

---

## 7. Non-Goals & Scope Boundaries

- **CI/CD beyond GHCR push:** No automated testing in CI, no staging environments, no deployment to production from CI. CI only builds and publishes the image.
- **Multi-architecture builds (arm64, armv7):** amd64 only in v1. ARM users must build locally with `GOARCH=arm64`.
- **Docker Swarm / Kubernetes:** docker-compose only. No stack files, no Helm charts.
- **HTTPS / TLS termination:** No built-in TLS. Users place a reverse proxy (Nginx, Caddy, Traefik) in front if HTTPS is needed.
- **Database backup automation:** No scheduled backup container or sidecar. Users manually back up `./data/` as documented.
- **Rootless container:** Container runs as root (default). Rootless mode can be added with `user: "1000:1000"` in compose — documented but not default.
- **Log aggregation:** Stdout/stderr only. `docker compose logs` is the log viewer. No log shipping to external services.
- **Resource limits enforcement:** Optional, documented but not enabled by default.
- **Docker secrets:** Environment variables are plaintext in the compose file. No secret management for single-user NAS.
- **Watchtower notification configuration:** Default is `none`. Users configure email/Slack/webhook notifications themselves.
- **Prometheus metrics endpoint:** Not in v1. `/api/v1/health` is binary healthy/unhealthy only.

---

## 8. Open Questions & Deferred Items

| # | Question | Status |
|---|---|---|
| OQ-1 | Should docker-compose include a `depends_on` with healthcheck condition between Watchtower and inventory? | Deferred — Watchtower monitors containers, not the other way around. No dependency needed. Inventory starts independently. |
| OQ-2 | Should `make docker` push to GHCR automatically? | Deferred — GHCR push is a CI concern (GitHub Actions), not a local Makefile concern. Local `make docker` builds for local testing only. |
| OQ-3 | Should the image include a default `docker-compose.yml` inside the image (mounted as read-only for reference)? | Deferred — compose file is distributed via the repo README, not inside the image. |
| OQ-4 | Should Watchtower be scoped with labels to only monitor the inventory container? | Deferred to v2 — in a two-container compose, implicit scoping is acceptable. If user runs other containers on the same host, they should add `--label-enable` scoping. |
| OQ-5 | Should there be a `docker-compose.override.yml` example for development (mounting source code, enabling debug)? | Deferred — local development uses `make dev` (Air + Vite), not Docker. |
| OQ-6 | What happens if Watchtower updates itself mid-cycle? | Watchtower by default does not monitor itself. The compose file scopes it to watch only the `inventory` container via implicit behavior. If the user wants self-update, they configure it manually. |
