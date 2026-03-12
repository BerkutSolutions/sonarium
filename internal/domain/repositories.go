package domain

import "context"

type ArtistRepository interface {
	List(ctx context.Context) ([]Artist, error)
	GetByID(ctx context.Context, id string) (Artist, error)
	GetByName(ctx context.Context, name string) (Artist, error)
	Search(ctx context.Context, query string) ([]Artist, error)
	Upsert(ctx context.Context, artist Artist) error
}

type AlbumRepository interface {
	List(ctx context.Context) ([]Album, error)
	GetByID(ctx context.Context, id string) (Album, error)
	GetByTitleAndArtistID(ctx context.Context, title, artistID string) (Album, error)
	ListByArtistID(ctx context.Context, artistID string) ([]Album, error)
	Search(ctx context.Context, query string) ([]Album, error)
	Upsert(ctx context.Context, album Album) error
}

type TrackRepository interface {
	List(ctx context.Context) ([]Track, error)
	GetByID(ctx context.Context, id string) (Track, error)
	GetByFilePath(ctx context.Context, filePath string) (Track, error)
	ListByAlbumID(ctx context.Context, albumID string) ([]Track, error)
	ListByArtistID(ctx context.Context, artistID string) ([]Track, error)
	Search(ctx context.Context, query string) ([]Track, error)
	Upsert(ctx context.Context, track Track) error
}

type PlaylistRepository interface {
	List(ctx context.Context) ([]Playlist, error)
	ListAccessible(ctx context.Context, userID string) ([]Playlist, error)
	GetByID(ctx context.Context, id string) (Playlist, error)
	GetAccessibleByID(ctx context.Context, id, userID, shareToken string) (Playlist, error)
	Create(ctx context.Context, playlist Playlist) (Playlist, error)
	Delete(ctx context.Context, id string) error
	Rename(ctx context.Context, id, name string) error
	Update(ctx context.Context, id, name, description string) error
	AddTrack(ctx context.Context, item PlaylistTrack) error
	RemoveTrack(ctx context.Context, playlistID string, trackID string) error
	ListTracks(ctx context.Context, playlistID string) ([]Track, error)
	CanEdit(ctx context.Context, id, userID string) (bool, error)
	IsOwner(ctx context.Context, id, userID string) (bool, error)
}

type LibraryRepository interface {
	Get(ctx context.Context) (Library, error)
	Save(ctx context.Context, library Library) error
	UpdateLastScanAt(ctx context.Context) error
}
