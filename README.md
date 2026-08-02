# Itema

A self-hosted inventory management app. Track physical items across hierarchical locations.
Built with Go + SQLite backend and a React frontend. Runs in a single Docker container.

---

## Prerequisites

| Tool | Min Version | Install |
|---|---|---|
| Go | 1.22+ | https://go.dev/dl/ |
| Node.js | 20+ | https://nodejs.org/ |
| Git for Windows | any | https://git-scm.com/download/win |
| Air (Go hot-reload) | latest | `go install github.com/air-verse/air@latest` |

---

## Quick Start (Local Development)

**First-time setup** (run once after cloning):

```bash
# Install frontend dependencies
npm install --prefix frontend
```

**Start dev servers:**

```powershell
# Windows (PowerShell)
.\make.ps1 dev
```

```bash
# Git Bash / macOS / Linux
make dev
```

- Go API: http://localhost:8080
- React UI: http://localhost:5173

---

## Available Commands

| Windows (PowerShell) | Git Bash | Description |
|---|---|---|
| `.\make.ps1 dev` | `make dev` | Start dev servers with hot-reload |
| `.\make.ps1 build` | `make build` | Build production binary |
| `.\make.ps1 test-fast` | `make test-fast` | Run Go integration tests (~30s) |
| `.\make.ps1 test` | `make test` | Run full test suite |
| `.\make.ps1 docker` | `make docker` | Build Docker image |
| `.\make.ps1 help` | `make help` | List all targets |

---

## Docker Deployment

```yaml
services:
  itema:
    image: itema:latest
    container_name: itema
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - APP_PORT=8080
    restart: unless-stopped
```

```bash
docker compose up -d
```

Access at `http://your-nas-ip:8080`
