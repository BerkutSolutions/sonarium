# Repository Structure

## Top-Level Layout

```text
.
├── cmd/
│   └── app/
│       └── main.go                 # Composition root (wiring only)
├── internal/
│   ├── modules/
│   │   ├── library/
│   │   ├── artists/
│   │   ├── albums/
│   │   ├── tracks/
│   │   ├── playlists/
│   │   ├── search/
│   │   ├── metadata/
│   │   ├── stream/
│   │   ├── player/
│   │   ├── subsonic/
│   │   ├── auth/
│   │   ├── users/
│   │   ├── ui/
│   │   └── health/
│   └── platform/
│       ├── config/
│       ├── logging/
│       ├── httpserver/
│       ├── persistence/
│       │   ├── sqlite/
│       │   ├── postgres/
│       │   └── migrations/
│       ├── cache/
│       └── filesystem/
├── pkg/
│   └── <optional-public-libraries>/
├── web/
│   ├── src/                        # Frontend source (later)
│   └── dist/                       # Frontend build output (generated)
├── deploy/
│   ├── compose/
│   │   ├── docker-compose.yml
│   │   └── .env.example
│   ├── portainer/
│   │   └── stack.yml
│   └── docker/
│       └── Dockerfile
├── docs/
│   ├── architecture.md
│   ├── modules.md
│   ├── repository_structure.md
│   └── docker_strategy.md
├── LOCAL_AI_RULES.md
├── Makefile
├── Dockerfile                      # Optional root shortcut (can mirror deploy/docker)
├── docker-compose.yml              # Optional root shortcut (can mirror deploy/compose)
└── .env.example
```

## Directory Roles

### `cmd/`
Application entrypoints only. `main.go` should perform wiring and startup orchestration; no domain logic.

### `internal/`
Private application code.
- `internal/modules/`: domain modules (bounded contexts)
- `internal/platform/`: infrastructure and technical capabilities used by modules

### `pkg/`
Optional exported reusable packages. Keep minimal; do not place app-specific domain logic here.

### `web/`
Frontend source and built assets. Backend serves built assets; runtime does not depend on CDN.

### `deploy/`
Deployment artifacts for Docker Compose/Portainer.
- Compose files
- Portainer stack definitions
- Docker build files

### `docs/`
Architecture, module contracts, deployment strategy, and decision records.

## Root Files

### `Dockerfile`
Multi-stage build for backend (and optionally UI build stage).

### `docker-compose.yml`
Local/prod-like stack definition with named volumes, env vars, and health checks.

### `.env.example`
Template for runtime configuration.

### `Makefile`
Standardized developer commands (build, test, compose-up/down, lint, migrations).

## File Size Guidance

Target file size: ~300-400 LOC where practical.
If larger files emerge, split by responsibility (handler/service/repository/query).

## Import and Boundary Guidance

- Avoid cross-importing module internals.
- Define explicit interfaces for allowed interactions.
- Keep transport DTOs separate from domain models.
- Keep SQL and persistence details in `internal/platform/persistence`.
