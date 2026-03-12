# Docker Strategy

## 1. Deployment Objectives

- Entire system runs in containers.
- Deployable via Docker Compose and Portainer without modification.
- No runtime external dependencies for core operation (music serving, DB, UI assets).

## 2. Containers

Primary service:
- `music-app` (Go backend, includes API, Subsonic adapter, stream endpoints, health endpoints, optional static UI serving)

Optional sidecars (future, not required initially):
- none required for baseline

## 3. Volumes

Persistent volumes required:
- `music_library` -> mounted read-only/read-write (configurable) path for audio files
- `music_data` -> database storage (SQLite file and app state)
- `music_cache` -> metadata/transcode/cache files

Rules:
- container image must remain stateless
- no important state in writable container layer

## 4. Environment Configuration

All runtime config via environment variables.
Recommended variables:
- `APP_ENV`
- `APP_HTTP_ADDR`
- `APP_HTTP_PORT`
- `APP_DB_DRIVER` (`sqlite` or `postgres`)
- `APP_DB_DSN` (PostgreSQL DSN when used)
- `APP_DB_PATH` (SQLite path)
- `APP_MUSIC_ROOT`
- `APP_CACHE_DIR`
- `APP_LOG_LEVEL`
- `APP_ENABLE_SUBSONIC`
- `APP_ENABLE_TRANSCODING` (future)

Startup behavior:
- validate required env vars
- run migrations before serving traffic
- fail fast on invalid configuration

## 5. Compose / Portainer Compatibility

Compose requirements:
- explicit named volumes
- explicit container healthcheck hitting readiness endpoint
- restart policy
- network definition (default bridge acceptable)
- `.env` support

Portainer requirements:
- provide stack file in `deploy/portainer/stack.yml`
- avoid unsupported Compose features
- keep volume paths and env vars configurable from stack UI

## 6. Build Strategy

Use multi-stage Docker build:
1. Go build stage (static binary preferred when practical)
2. optional frontend build stage
3. minimal runtime image with binary + UI assets

No runtime package installation in final container.

## 7. Database Strategy in Containers

Default SQLite:
- DB file on mounted `music_data` volume
- appropriate file permissions on container startup

Optional PostgreSQL:
- external Postgres container or managed instance
- selected by env var switch
- same repository interfaces used regardless of backend

## 8. Migration Strategy

- versioned SQL migrations stored in repository
- migrations executed at startup before app begins serving
- if migration fails, container exits non-zero
- migration status logged clearly

## 9. Streaming Runtime Requirements

- support HTTP range requests (`206 Partial Content`)
- stream from mounted library volume
- avoid full-file buffering
- tune read buffer size for stable memory profile

## 10. Security Baseline

- run as non-root user in container when possible
- expose only required ports
- do not embed secrets in image
- use `.env`/Portainer environment management for secrets

## 11. Operational Observability

- structured logs to stdout/stderr
- health endpoints:
  - liveness: process up
  - readiness: DB + required storage available
- optional metrics endpoint reserved for future additions

## 12. Persistence Layer

Database stack:
- PostgreSQL container (`postgres:16-alpine`)
- persistent named volume `postgres_data`
- application connects through environment variables:
  - `DB_HOST`
  - `DB_PORT`
  - `DB_USER`
  - `DB_PASSWORD`
  - `DB_NAME`
  - `DB_SSLMODE`

Schema management:
- goose migrations live in `internal/storage/migrations`
- application startup runs migrations before serving HTTP traffic
- migration and runtime flow:
  1. open PostgreSQL connection
  2. run `goose up`
  3. start HTTP server

Repository pattern:
- domain repository interfaces are implemented in `internal/storage/repositories`
- all persistence access is via `database/sql` (no ORM)
- readiness endpoint validates database availability via ping
