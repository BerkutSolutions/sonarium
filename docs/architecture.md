# Architecture: Self-Hosted Music Streaming Platform

## 1. System Overview

This project is a **container-first modular monolith** written in Go. It runs as one deployable backend service with strict internal module boundaries. The design targets safe AI-assisted evolution, Subsonic compatibility, and efficient local audio streaming.

Core goals:
- single runtime service, modular internals
- strict separation: domain logic vs transports (HTTP/Subsonic/UI)
- Docker-first operation (Compose + Portainer)
- persistence abstraction around a PostgreSQL runtime backend
- extension-ready for metadata providers, transcoding, multi-user, and external APIs

## 2. Architectural Style

### Modular Monolith
- One process/service: `app`
- Internal modules as bounded contexts
- Cross-module interaction through explicit service interfaces
- No cyclic dependencies

### Layering (inside each module)
1. **Model (domain entities/value objects)**
2. **Service (business use-cases)**
3. **Repository Interface (persistence contracts)**
4. **Transport Adapters (HTTP/Subsonic/internal adapters only)**

Rule: transports call services; services call repository interfaces; infrastructure implements repositories.

## 3. High-Level Runtime Components

- **API Server**: REST/JSON endpoints for first-party UI and integrations
- **Subsonic Adapter**: compatibility endpoints mapped to internal services
- **Streaming Handler**: range-aware file streaming with low-memory I/O
- **Background Workers** (in same process): library scan, metadata refresh, cache warming
- **Storage Adapters**: PostgreSQL adapter, filesystem adapters, cache adapter

## 4. Dependency Direction

Allowed direction:
- `transport -> service -> repository interface`
- `infra(repository impl) -> repository interface`
- `module A service -> module B service interface` (no direct repository access across modules)

Forbidden:
- transport importing infrastructure
- repository implementation used as domain type
- direct module-to-module data store coupling
- cyclic module references

## 5. Module Boundary Contract

Each module owns:
- domain models for its context
- service interfaces + implementations
- repository interface(s)
- transport adapters specific to that module

Shared/common utilities must remain generic and live outside domain modules (for example logging, config parsing, database wiring), never holding business rules.

## 6. Request Flow (Example)

1. HTTP/Subsonic endpoint receives request
2. Adapter validates transport-level input and maps into service command/query
3. Service executes business logic and calls repository interfaces
4. Repository implementation persists/loads data from PostgreSQL
5. Response DTO returned via adapter

Business rules never exist in handlers.

## 7. Persistence Strategy

- Runtime database: PostgreSQL
- Repository abstraction is kept to preserve modularity and future backend flexibility
- Repository pattern isolates SQL and storage concerns
- Migrations are versioned and deterministic (startup migration step)

Migration policy:
- SQL migration files with monotonic version numbers
- forward-only in production
- rollback scripts optional for development
- app refuses startup on dirty/partial migration state

## 8. Streaming Architecture

Streaming module responsibilities:
- HTTP range request parsing and validation
- partial-content (206) responses
- efficient chunked file reads (`io.Copy`-style streaming, no full-file buffering)
- content type + bitrate metadata exposure

Interaction with library module:
- library module resolves canonical track identity and path access policy
- stream module requests file locator info from library service
- stream module does transport/byte streaming only

## 9. Subsonic Compatibility Layer

Subsonic endpoints are an **adapter layer** only:
- map Subsonic request/response contract to internal service calls
- reuse existing modules (`library`, `tracks`, `albums`, `artists`, `playlists`, `auth`, `users`, `stream`)
- do not define or dictate core domain models

This ensures Subsonic support can evolve independently from internal architecture.

Implemented baseline compatibility:
- base path: `/rest`
- required protocol params: `u`, `t`, `s`, `v`, `c`, `f`
- token auth support: `t = md5(password + salt)`
- supported endpoints:
  - `ping.view`, `getLicense.view`
  - `getArtists.view`, `getArtist.view`
  - `getAlbum.view`, `getAlbumList.view`, `getSong.view`
  - `getPlaylists.view`, `getPlaylist.view`
  - `search3.view`, `getCoverArt.view`, `stream.view`

Response formats:
- JSON by default
- XML rendering path is supported by the adapter renderer

Error mapping:
- internal errors are translated into Subsonic-compatible error payloads
- internal platform error objects are not exposed directly

## 10. UI Integration Modes

Two supported modes:

1. **Integrated assets mode**
- frontend build artifacts copied into backend image
- Go serves static files from local filesystem/embed
- simplest single-container deployment

2. **Bundled separate frontend artifact mode**
- UI built in dedicated build stage/container
- final runtime image includes generated assets
- still no external CDN/runtime dependency

Runtime rule: UI must function without internet access (except user-configured metadata providers).

## 19. Web UI

The project ships an embedded Web UI served by the Go backend itself:
- static assets live under `gui/static`
- assets are embedded via `embed.FS` (`gui/assets.go`)
- UI routes are served directly by backend:
  - `/`
  - `/artists`
  - `/albums`
  - `/tracks`
  - `/playlists`
  - `/search`
  - `/library`
  - `/settings`
  - `/users`
  - `/profile`
- SPA fallback keeps deep links like `/genres`, `/profile/{id}` and entity detail pages routable through the same shell
- local JS/CSS only, no CDN runtime dependencies

Frontend architecture:
- vanilla JavaScript modules under `gui/static/js`
- sidebar + content + bottom player layout
- content pages loaded dynamically by client-side router
- player streams audio via existing backend endpoint `/api/stream/{track_id}`

Design constraint:
- UI is an adapter over backend APIs and must not contain server business logic

## 20. Cover Art System

Cover art is implemented as a dedicated infrastructure module: `internal/modules/coverart`.

Purpose:
- extract album/artist art during library scan (not on request path)
- persist originals and thumbnails in cache storage
- serve reusable assets to REST, Subsonic, and Web UI

Sources (priority order):
- album directory images: `cover.jpg`, `folder.jpg`, `front.jpg` (also jpeg/png variants)
- embedded tags from audio metadata (`MP3`/`FLAC`)
- generated placeholder image if art is absent (for REST/UI use-cases)

Cache layout:
- `${DATA_PATH}/covers/original`
- `${DATA_PATH}/covers/thumb/64`
- `${DATA_PATH}/covers/thumb/128`
- `${DATA_PATH}/covers/thumb/256`

Integration rules:
- scanner extracts and attaches cover path to `albums.cover_path` and `artists.cover_path`
- no per-request extraction is allowed
- REST endpoints (`/api/covers/...`) and Subsonic `getCoverArt.view` read from cache-backed paths
- Subsonic remains an adapter and does not define cover storage semantics

## 21. Smart Player

Smart Player is implemented as a platform module (`internal/modules/player`) plus lightweight UI state management.

Core model:
- `current_track_id`
- compact `queue` items (`track_id`, `title`, `artist`, `duration`, `cover_ref`)
- `queue_position`
- `is_playing`
- `shuffle_enabled`
- `repeat_mode` (`off`, `one`, `all`)
- `volume`
- `current_time_seconds`
- `context_type`, `context_id`

Performance approach:
- queue is built once from the current context (album/artist/playlist/search/list)
- `next/previous` operate on queue snapshot only
- no full-library re-fetch on playback actions
- queue persistence is compact and bounded for large libraries

Persistence strategy:
- primary state persisted client-side in `localStorage`
- server exposes minimal state/queue contract (`/api/player/*`) for synchronization and future server-side session persistence
- architecture keeps player transport independent from stream/domain modules

Scalability safeguards:
- queue stores minimal metadata only (no full track payload duplication)
- persisted queue is windowed/capped in browser storage to avoid degradation on very large collections

## 22. Smart Library

Smart Library extends the backend with user-centric discovery blocks and history-aware data:
- Recently Added (from `albums.created_at`)
- Recently Played (from `play_history`)
- Continue Listening (partial playback progress)
- Random Albums (index-driven random key strategy)
- Favorites (`tracks`, `albums`, `artists`)

Data model additions:
- `user_favorite_tracks`
- `user_favorite_albums`
- `user_favorite_artists`
- `play_history` (`track_id`, `played_at`, `position_seconds`, `context_type`, `context_id`)

Home endpoint:
- `GET /api/library/home`
- built by dedicated library repository/service layer
- protected by short-lived in-process cache (10-30s TTL)

## 23. Fast Scanner

Scanner incremental behavior uses file fingerprints to avoid expensive metadata re-reads:
- tracked attributes: `file_path`, `file_size`, `modified_time`, `fingerprint_hash`
- unchanged fingerprint => metadata read is skipped
- changed fingerprint => metadata, cover, waveform, loudness and DB upserts are executed

This keeps repeated scans dramatically faster on stable libraries.

## 24. Waveform System

Waveform module generates lightweight amplitude previews:
- generation occurs during scan (and lazily on-demand if missing)
- cache path: `${DATA_PATH}/waveforms/*.json`
- transport endpoint: `GET /api/tracks/{id}/waveform`
- UI uses waveform data for richer progress visualization

## 25. Loudness Normalization

Loudness module resolves ReplayGain-compatible values:
- reads embedded metadata when available
- falls back to lightweight heuristic if tags are absent
- persisted fields:
  - `tracks.replay_gain_track`
  - `tracks.replay_gain_album`

This keeps normalization metadata available for future player and transcoding policies.

## 26. Transcoding Engine

Transcoding module provides on-demand conversion for streaming:
- powered by ffmpeg inside existing server container
- requested via stream query params (`format`, `bitrate`)
- no additional runtime services required

Supported targets:
- `opus`
- `aac`
- `mp3`

The default stream path remains direct file serving with HTTP range support; transcoding is activated only when requested.

## 11. Configuration Model

Environment-variable driven config only (with optional `.env` loading in local dev):
- server bind/port
- database driver + DSN/path
- library root path
- cache path/settings
- auth/session settings
- feature flags (subsonic, metadata providers, transcoding)

Invalid config must fail fast at startup.

## 12. Docker and Deployment Baseline

- single primary backend container
- volumes: music library, database, cache
- stateless image; mutable data only in mounted volumes
- health endpoint for container health checks
- compatible with Docker Compose and Portainer stacks

## 13. Future Extension Points

Planned extensibility seams:
- metadata provider interface registry
- transcoding pipeline abstraction
- pluggable auth strategies
- user/tenant policy layer for multi-user evolution
- external API adapters as transport modules over existing services

## 14. Non-Goals for Initial Architecture Phase

- no microservices split
- no direct external object storage dependency required
- no runtime CDN dependency for UI assets
- no code generation of domain from Subsonic schema

## 15. Domain Layer

The project includes a dedicated `internal/domain` layer as the canonical model for core music concepts:
- `Artist`
- `Album`
- `Track`
- `Playlist` and `PlaylistTrack`
- `Library`

Rules for the domain layer:
- no HTTP, SQL, ORM, router, or infrastructure dependencies
- only Go standard library and local domain packages
- contains entity definitions, domain invariants/validation, and service/repository interfaces
- acts as the contract consumed by platform, adapters, and future modules

Usage policy:
- transport layers map incoming requests to domain-level operations
- infrastructure implements domain repository interfaces
- adapters (including future Subsonic compatibility) must never redefine internal domain ownership

## 16. Persistence Layer

Persistence is implemented in `internal/storage` and split into:
- `internal/storage/postgres`: PostgreSQL connection management, ping checks, and pool configuration
- `internal/storage/migrations`: goose migration runner and SQL migration files
- `internal/storage/repositories`: `database/sql` implementations of domain repository interfaces

Persistence rules:
- repository interfaces are defined in `internal/domain`
- storage adapters implement those interfaces without changing domain contracts
- SQL details stay inside storage packages only
- startup sequence runs migrations first, then starts HTTP server

Current storage backend:
- PostgreSQL (primary runtime database)
- goose for schema versioning and controlled schema evolution

## 17. Library Scanner

Library scanner is implemented as `internal/modules/library` with three sub-packages:
- `scanner`: filesystem walk and audio file discovery
- `metadata`: audio tag parsing via `github.com/dhowden/tag`
- `service`: scan pipeline orchestration and persistence coordination

Supported file formats:
- `.mp3`
- `.flac`
- `.ogg`
- `.m4a`

Fingerprint strategy (incremental scan):
- fingerprint key: file path
- fingerprint attributes: file size + modification time
- unchanged files are skipped on subsequent scans
- changed or new files are parsed and upserted

Worker pool:
- metadata extraction runs in parallel worker goroutines
- worker count is configured via `SCANNER_WORKERS`
- repository writes are coordinated in the scan service

Persistence integration:
- scanner updates `Artist`, `Album`, and `Track` through repository interfaces
- scanner updates `Library.LastScanAt` on successful completion
- file fingerprints are stored in PostgreSQL for incremental behavior

## 18. Streaming Engine

Streaming is implemented in `internal/modules/stream`:
- `service`: resolves track metadata/path via repository and validates file availability
- `transport`: HTTP handler for streaming endpoint

Primary endpoint:
- `GET /api/stream/{track_id}`

Behavior:
- fetch track by ID from `TrackRepository`
- open file path stored in database
- stream audio bytes using `http.ServeContent`
- support HTTP range requests for seek/partial playback

Content type handling:
- `.mp3` -> `audio/mpeg`
- `.flac` -> `audio/flac`
- `.ogg` -> `audio/ogg`
- `.m4a` -> `audio/mp4`

Error handling:
- missing track -> not found JSON error
- missing file on disk -> not found JSON error
- internal stream failure -> internal JSON error

Security model:
- endpoint never accepts filesystem path from request
- streaming path is resolved only from trusted repository data
