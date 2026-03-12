# Music API

Base path: `/api`

All responses use JSON envelope:

```json
{
  "data": {}
}
```

Errors use platform error envelope:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "details"
  }
}
```

## Query Parameters

List endpoints support:
- `limit` (integer, optional)
- `offset` (integer, optional)
- `sort` (module-specific, optional)

## Artists

- `GET /api/artists`
  - sort: `name`, `created_at`
- `GET /api/artists/{id}`

## Albums

- `GET /api/albums`
  - sort: `name`, `year`, `created_at`
- `GET /api/albums/{id}`
- `GET /api/artists/{id}/albums`
  - sort: `name`, `year`, `created_at`

## Tracks

- `GET /api/tracks`
  - sort: `name`, `created_at`
- `GET /api/tracks/{id}`
- `GET /api/albums/{id}/tracks`
  - sort: `name`, `created_at`

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

## Streaming

- `GET /api/stream/{track_id}`
  - supports HTTP range requests
  - supports optional transcoding query:
  - `format=opus|aac|mp3`
  - `bitrate=96|128|192|320`

## Cover Art

- `GET /api/covers/album/{album_id}`
  - returns original album cover image
  - returns placeholder image when album has no cover
- `GET /api/covers/artist/{artist_id}`
  - returns original artist cover image
  - returns placeholder image when artist has no cover
- `GET /api/covers/album/{album_id}/thumb/{size}`
  - supported sizes: `64`, `128`, `256`
  - returns cached/generated thumbnail

## Smart Library

- `GET /api/library/home?limit=12`
  - returns:
  - `recent_albums`
  - `recent_tracks`
  - `continue_listening`
  - `random_albums`
  - `favorites` (`albums`, `artists`, `tracks`)
- `GET /api/library/random-albums?limit=12`
- `POST /api/library/favorites/tracks/{track_id}/toggle`
- `POST /api/library/favorites/albums/{album_id}/toggle`
- `POST /api/library/favorites/artists/{artist_id}/toggle`

## Smart Player

- `GET /api/player/state`
  - returns current playback session state
- `POST /api/player/state`
  - replace playback state snapshot
- `POST /api/player/queue/replace`
  - body:
  ```json
  {
    "queue": [{"track_id":"...","title":"...","artist":"...","duration":180,"cover_ref":"album_id"}],
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

## Waveform

- `GET /api/tracks/{id}/waveform`
  - returns:
  ```json
  {
    "data": {
      "track_id": "uuid",
      "amplitude": [0, 12, 18, 9, ...]
    }
  }
  ```

## Subsonic Compatibility Layer

Base path: `/rest`

Supported formats:
- `f=json`
- `f=xml` (adapter supports XML output path)

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
- Subsonic layer is adapter-only and maps to internal services.
- `stream.view` uses the existing streaming service (no separate stream engine).
- `getCoverArt.view` uses the shared cover art service/cache (no dedicated Subsonic cover store).
