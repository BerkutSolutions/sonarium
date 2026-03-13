# Music API

Base path: `/api`

Most `/api/*` endpoints require an authenticated session unless explicitly noted otherwise.

All successful responses use a JSON envelope:

```json
{
  "data": {}
}
```

Errors use the platform error envelope:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "details"
  }
}
```

## Query Parameters

List endpoints commonly support:
- `limit` (integer, optional)
- `offset` (integer, optional)
- `sort` (module-specific, optional)

## Auth

Public endpoints:
- `GET /api/auth/status`
- `POST /api/auth/login`
- `POST /api/auth/register`

Authenticated endpoints:
- `POST /api/auth/logout`
- `GET /api/auth/users`
- `POST /api/auth/users/{user_id}/active`
- `POST /api/auth/users/{user_id}/delete`
- `POST /api/auth/settings/registration`
- `GET /api/auth/users/lookup`
- `GET /api/auth/profile/{user_id}`
- `POST /api/auth/profile/update`
- `POST /api/auth/profile/password`

## Artists

- `GET /api/artists`
  - sort: `name`, `created_at`
- `GET /api/artists/{id}`
- `GET /api/artists/{id}/albums`
  - sort: `name`, `year`, `created_at`

## Albums

- `GET /api/albums`
  - sort: `name`, `year`, `created_at`
- `GET /api/albums/{id}`
- `GET /api/albums/{id}/tracks`
  - sort: `name`, `created_at`

## Tracks

- `GET /api/tracks`
  - sort: `name`, `created_at`
- `GET /api/tracks/{id}`
- `GET /api/tracks/{id}/waveform`

## Playlists

- `GET /api/playlists`
  - sort: `name`, `created_at`
- `GET /api/playlists/{id}`
  - returns playlist details with tracks
- `POST /api/playlists`
  - body:
  ```json
  {
    "name": "My Playlist"
  }
  ```
- `POST /api/playlists/{id}/rename`
- `POST /api/playlists/{id}/update`
- `DELETE /api/playlists/{id}`
- `POST /api/playlists/{id}/tracks`
  - body:
  ```json
  {
    "track_id": "uuid",
    "position": 1
  }
  ```
- `DELETE /api/playlists/{id}/tracks/{track_id}`

## Search

- `GET /api/search?q=test`
  - optional: `limit`, `offset`, `sort`
  - sort: `name`, `year`, `created_at`
  - response:
  ```json
  {
    "data": {
      "artists": [],
      "albums": [],
      "tracks": []
    }
  }
  ```

## Sharing

- `GET /api/shares/received`
- `GET /api/shares/{entity_type}/{entity_id}`
- `POST /api/shares/{entity_type}/{entity_id}/users`
- `POST /api/shares/{entity_type}/{entity_id}/public`
- `DELETE /api/shares/{share_id}`

## Streaming

- `GET /api/stream/{track_id}`
  - supports HTTP range requests
  - supports optional transcoding query params:
  - `format=opus|aac|mp3`
  - `bitrate=96|128|192|320`

## Cover Art

- `GET /api/covers/album/{album_id}`
  - returns original album cover image
  - returns placeholder image when the album has no cover
- `GET /api/covers/artist/{artist_id}`
  - returns original artist cover image
  - returns placeholder image when the artist has no cover
- `GET /api/covers/album/{album_id}/thumb/{size}`
  - supported sizes: `64`, `128`, `256`
  - returns cached or generated thumbnail

## Library

- `GET /api/library/home?limit=12`
  - returns `recent_albums`, `recent_tracks`, `continue_listening`, `random_albums`, `favorites`
- `GET /api/library/random-albums?limit=12`
- `GET /api/library/artist-album-counts`
- `POST /api/library/scan`
- `GET /api/library/scan/status`
- `POST /api/library/upload`
- `POST /api/library/favorites/tracks/{track_id}/toggle`
- `POST /api/library/favorites/albums/{album_id}/toggle`
- `POST /api/library/favorites/artists/{artist_id}/toggle`
- `POST /api/library/artists/{artist_id}/update`
- `POST /api/library/artists/{artist_id}/cover`
- `POST /api/library/artists/{artist_id}/delete`
- `POST /api/library/tracks/{track_id}/delete`
- `POST /api/library/tracks/{track_id}/rename`
- `POST /api/library/tracks/{track_id}/update`
- `POST /api/library/albums/{album_id}/delete`
- `POST /api/library/albums/{album_id}/rename`
- `POST /api/library/albums/{album_id}/update`
- `POST /api/library/albums/{album_id}/merge`
- `POST /api/library/albums/create`

## Settings

- `GET /api/settings`
- `POST /api/settings/updates/check`
- `POST /api/settings/updates/auto`
- `GET /api/settings/storage`
- `POST /api/settings/library/delete-all`
- `POST /api/settings/upload-concurrency`

## Smart Player

- `GET /api/player/state`
- `POST /api/player/state`
- `POST /api/player/queue/replace`
  - body:
  ```json
  {
    "queue": [
      {
        "track_id": "...",
        "title": "...",
        "artist": "...",
        "duration": 180,
        "cover_ref": "album_id"
      }
    ],
    "queue_position": 0,
    "context_type": "album",
    "context_id": "album_uuid"
  }
  ```
- `POST /api/player/queue/append`
  - body: `{"items":[...]}`
- `POST /api/player/queue/remove`
  - body: `{"index":1}`
- `POST /api/player/queue/clear`
- `POST /api/player/queue/move`
  - body: `{"from":3,"to":1}`
- `POST /api/player/queue/shuffle`
  - body: `{"enabled":true}`
- `POST /api/player/played`
  - body:
  ```json
  {
    "track_id": "uuid",
    "position_seconds": 0,
    "context_type": "album",
    "context_id": "album_uuid"
  }
  ```

## Subsonic Compatibility Layer

Base path: `/rest`

Supported formats:
- `f=json`
- `f=xml`

Required protocol parameters:
- `u` (username)
- `t` (token, md5(password + salt))
- `s` (salt)
- `v` (protocol version)
- `c` (client name)
- `f` (response format)

Supported endpoints:
- `GET /rest/ping.view`
- `GET /rest/getLicense.view`
- `GET /rest/getArtists.view`
- `GET /rest/getArtist.view?id=...`
- `GET /rest/getAlbum.view?id=...`
- `GET /rest/getAlbumList.view`
- `GET /rest/getSong.view?id=...`
- `GET /rest/getPlaylists.view`
- `GET /rest/getPlaylist.view?id=...`
- `GET /rest/search3.view?query=...`
- `GET /rest/getCoverArt.view?id=...`
- `GET /rest/stream.view?id=...`

Notes:
- Subsonic is an adapter layer over internal services.
- `stream.view` uses the same streaming service as `/api/stream/{track_id}`.
- `getCoverArt.view` uses the shared cover cache and placeholder logic.
