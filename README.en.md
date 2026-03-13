# Berkut Solutions - Sonarium

<p align="center">
  <img src="gui/static/logo.png" alt="Sonarium logo" width="220">
</p>

[Russian version](README.md)

Sonarium is a self-hosted music platform for local libraries: streaming, smart player UX, collections, collaboration, and a built-in web interface with no SaaS dependency.

Current version: `1.0.1`

## Overview

Sonarium runs through Docker Compose, stores data in dedicated Docker volumes, indexes local music, extracts metadata and cover art, and provides a full web UI with deep-link pages for albums, artists, tracks, playlists, genres, users, and profiles.

It is designed for:
- local self-hosted usage
- zero-trust auth with sessions
- collaborative playlist access and sharing
- large personal music libraries

## Key Features

- Dedicated pages for:
  - albums
  - artists
  - tracks
  - playlists
  - genres
  - user profiles
- Built-in smart player:
  - queue
  - drag-and-drop reorder
  - shuffle / repeat
  - waveform and progress UI
  - persistent playback state
- Local library management:
  - directory scan
  - single-file and folder upload
  - metadata and cover extraction
  - genres, favorites, duplicate search
  - inline editing and merge flows for library entities
- Collaboration:
  - public share links
  - playlist access roles: listener / editor
  - user profile viewing
  - "shared with me" view
- Administration:
  - first user becomes admin
  - user management
  - registration policy control
  - update checks in settings
  - storage usage overview
  - upload concurrency setting
- Compatibility:
  - REST API
  - Subsonic adapter (`/rest`)

## Screenshots

Current repository screenshots:

- `gui/static/screen1.png`
- `gui/static/screen2.png`
- `gui/static/screen3.png`

![Screenshot 1](gui/static/screen1.png)
![Screenshot 2](gui/static/screen2.png)
![Screenshot 3](gui/static/screen3.png)

## Quick Start

1. Copy env file:

```bash
cp .env.example .env
```

2. Start the stack:

```bash
docker compose up -d --build
```

3. Open:

```text
http://localhost:8080
```

4. Create the first user. That account becomes admin.

## Docker Volumes

Runtime data is stored in named Docker volumes:

- `postgres_data` - PostgreSQL
- `soundhub_data` - app data, thumbnails, service data
- `soundhub_music` - music library

Inspect:

```bash
docker volume ls
```

Full stack removal including data:

```bash
docker compose down -v
```

## Documentation

- Docs index: [docs/README.md](docs/README.md)
- Russian docs: [docs/ru/README.md](docs/ru/README.md)
- English docs: [docs/eng/README.md](docs/eng/README.md)

Core documents:
- Architecture: [docs/architecture.md](docs/architecture.md)
- API: [docs/api.md](docs/api.md)
- Docker strategy: [docs/docker_strategy.md](docs/docker_strategy.md)
- Modules: [docs/modules.md](docs/modules.md)
- Repository structure: [docs/repository_structure.md](docs/repository_structure.md)

## Stack

- Go
- PostgreSQL
- Docker / Docker Compose
- Vanilla JS UI
- FFmpeg
- Goose migrations

## License

[LICENSE](LICENSE)
