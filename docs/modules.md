# Module Catalog

This document defines mandatory bounded contexts and their responsibilities.

## Domain Layer

Core domain contracts are centralized in `internal/domain`.

Purpose:
- define canonical entities shared by bounded contexts
- define service interfaces for business capabilities
- define repository interfaces for persistence abstraction
- enforce baseline invariants through domain validation methods

Isolation rules:
- no dependency on HTTP, SQL, or infrastructure packages
- no framework-specific types in domain entities or interfaces
- adapters and storage implementations must depend on domain contracts, not the reverse

## Cross-Module Rules

- Every module must contain at least:
  - models
  - service layer
  - repository interface(s)
  - transport adapters when exposed externally
- Modules communicate through service interfaces, not direct DB access.
- Shared DTOs between modules are forbidden unless explicitly defined as public contracts.

## Required Modules

### library
- Responsibility: source-of-truth for local media inventory and file indexing
- Models: library item, scan state, file location metadata
- Services: scan, rescan, path resolution, availability checks
- Repository interfaces: library index persistence, scan checkpoint state
- Adapters: HTTP admin scan endpoints
  - Scanner internals:
  - `scanner`: filesystem traversal for supported formats (`mp3`, `flac`, `ogg`, `m4a`)
  - `metadata`: tag extraction (`artist`, `album`, `title`, `track`, `year`, `duration`, cover art)
  - `service`: orchestration, fingerprint checks, worker pool scheduling, repository upserts
  - Smart Library extensions:
  - `repository`: home dashboard aggregations (`recent`, `continue`, `favorites`, `random`)
  - `service`: cached home composition (short TTL), favorites toggles, play-history ingestion
  - `transport/http`: `/api/library/home`, `/api/library/random-albums`, favorites toggle endpoints

### artists
- Responsibility: artist domain representation and retrieval
- Models: artist, artist stats/relations
- Services: list/get artists, aggregate from tracks/albums
- Repository interfaces: artist read/write/query storage
- Adapters: `GET /api/artists`, `GET /api/artists/{id}`

### albums
- Responsibility: album entities and album-level metadata
- Models: album, release info, artwork references
- Services: list/get albums, album-track aggregation
- Repository interfaces: album persistence and queries
- Adapters: `GET /api/albums`, `GET /api/albums/{id}`, `GET /api/artists/{id}/albums`

### tracks
- Responsibility: track domain and playback-related track metadata
- Models: track, codec metadata, duration, replay fields
- Services: list/get tracks, track lookup by ID/path key
- Repository interfaces: track persistence and lookup
- Adapters: `GET /api/tracks`, `GET /api/tracks/{id}`, `GET /api/albums/{id}/tracks`

### playlists
- Responsibility: user/system playlist lifecycle
- Models: playlist, playlist entry ordering, ownership
- Services: create/update/delete playlists, reorder tracks
- Repository interfaces: playlist persistence
- Adapters: `GET /api/playlists`, `POST /api/playlists`, `POST /api/playlists/{id}/tracks`, `DELETE /api/playlists/{id}/tracks/{track_id}`

### search
- Responsibility: cross-domain search orchestration
- Models: search query, indexed token metadata, result page
- Services: federated search across artists/albums/tracks/playlists
- Repository interfaces: search index/query abstraction
- Adapters: `GET /api/search?q=...`

### metadata
- Responsibility: metadata enrichment orchestration
- Models: provider result, match confidence, enrichment job
- Services: fetch/merge metadata, schedule refresh
- Repository interfaces: metadata cache/state
- Adapters: provider adapters (e.g. local tags, external providers)

### stream
- Responsibility: byte streaming and range delivery
- Models: stream request context, byte range, delivery info
- Services: authorize + resolve stream source + serve ranges
- Repository interfaces: optional stream session/audit persistence
- Adapters: `/stream/*`, Subsonic stream endpoint adapter
  - Engine:
  - endpoint: `GET /api/stream/{track_id}`
  - file resolution via `TrackRepository`
  - range support via `http.ServeContent`
  - content type mapping for `mp3`, `flac`, `ogg`, `m4a`
  - optional on-demand transcoding adapter (via ffmpeg) with `format` + `bitrate` request parameters

### coverart
- Responsibility: unified cover art extraction, caching, and delivery for all clients
- Models: cached original image path, thumbnail variants, placeholder asset
- Services: resolve/attach cover to album/artist, load original, load thumbnail
- Repository interfaces: reuse existing `ArtistRepository` and `AlbumRepository` (cover path fields)
- Adapters: REST cover endpoints and Subsonic `getCoverArt.view`
  - Subpackages:
  - `extractor`: embedded tag extraction (`MP3`/`FLAC`) + directory image discovery (`cover.jpg`, `folder.jpg`, `front.jpg`)
  - `cache`: persistent files under `DATA_PATH/covers/{original,thumb}`
  - `service`: scanner-time extraction and cache binding to domain entities
  - `transport/http`: `/api/covers/...` handlers

### player
- Responsibility: smart playback state and queue orchestration (adapter-safe, reusable)
- Models: playback session state, compact queue item, repeat mode, playback context
- Services: queue replace/append/remove/clear/move, next/previous rules, shuffle/repeat behavior
- Repository interfaces: not required in current stage (in-memory contract; prepared for persistence later)
- Adapters: `/api/player/*`
  - `GET /api/player/state`
  - `POST /api/player/state`
  - `POST /api/player/queue/replace`
  - `POST /api/player/queue/append`
  - `POST /api/player/queue/remove`
  - `POST /api/player/queue/clear`
  - `POST /api/player/queue/move`
  - `POST /api/player/queue/shuffle`
  - Context model examples:
  - `album:{album_id}`
  - `artist:{artist_id}`
  - `playlist:{playlist_id}`
  - `search:{query_hash}`
  - Play-history integration:
  - `POST /api/player/played` writes playback events for Smart Library dashboards

### waveform
- Responsibility: waveform preview generation and retrieval for UI progress visualization
- Models: per-track amplitude series (JSON cache)
- Services: analyze audio bytes, generate compact amplitude array, cache under `DATA_PATH/waveforms`
- Adapters: `GET /api/tracks/{id}/waveform`

### loudness
- Responsibility: replay gain / loudness normalization metadata resolution
- Models: `track_gain`, `album_gain`
- Services: resolve from existing metadata tags, fallback heuristic where missing
- Integration: scanner stores gains into `tracks.replay_gain_track` and `tracks.replay_gain_album`

### transcoding
- Responsibility: adaptive streaming formats for constrained clients/networks
- Services: on-demand ffmpeg pipeline (`FLAC/WAV` and others to `opus`, `aac`, `mp3`)
- Integration: stream transport uses transcoder only when request asks for `format` or `bitrate`

### subsonic
- Responsibility: compatibility API adapter only
- Models: Subsonic DTOs only (adapter-layer contracts)
- Services: request mapping and response projection
- Repository interfaces: none mandatory; use internal services
- Adapters: `/rest/*` Subsonic-compatible endpoints
  - Supported endpoints:
  - `ping.view`, `getLicense.view`
  - `getArtists.view`, `getArtist.view`
  - `getAlbum.view`, `getAlbumList.view`, `getSong.view`
  - `getPlaylists.view`, `getPlaylist.view`
  - `search3.view`, `getCoverArt.view`, `stream.view`
  - Auth strategy: Subsonic token validation (`t = md5(password + salt)`) isolated from internal auth model

### auth
- Responsibility: authentication and session/token policies
- Models: credential policy, token/session, auth claims
- Services: login, token issuance/validation, permission checks
- Repository interfaces: auth/session store
- Adapters: login/logout/auth-check endpoints

### users
- Responsibility: user lifecycle and preferences
- Models: user, role, settings, library scopes
- Services: user CRUD, role assignment, preference updates
- Repository interfaces: user store
- Adapters: user management endpoints

### ui
- Responsibility: serving frontend artifacts and UI bootstrap routes
- Models: UI build info, asset manifest representation
- Services: asset resolution and cache policy
- Repository interfaces: none usually required
- Adapters: static asset handlers + SPA fallback
  - Routes:
  - `/`
  - `/artists`
  - `/albums`
  - `/tracks`
  - `/playlists`
  - `/search`
  - Static files: `/static/*` served from embedded assets

### health
- Responsibility: liveness/readiness/reporting
- Models: health status summary
- Services: dependency checks and status composition
- Repository interfaces: none required
- Adapters: `/health/live`, `/health/ready`

## Suggested Internal Module Template

For each module under `internal/modules/<module>`:
- `models/`
- `service/`
- `repository/` (interfaces only)
- `transport/http/` (if REST)
- `transport/subsonic/` (only where needed)

Infrastructure implementations belong outside module domain directories (for example `internal/platform/persistence/...`).
