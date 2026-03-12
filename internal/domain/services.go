package domain

import "context"

type ArtistService interface {
	ListArtists(ctx context.Context) ([]Artist, error)
	GetArtistByID(ctx context.Context, id string) (Artist, error)
	SearchArtists(ctx context.Context, query string) ([]Artist, error)
}

type AlbumService interface {
	ListAlbums(ctx context.Context) ([]Album, error)
	GetAlbumByID(ctx context.Context, id string) (Album, error)
	ListAlbumsByArtist(ctx context.Context, artistID string) ([]Album, error)
	SearchAlbums(ctx context.Context, query string) ([]Album, error)
}

type TrackService interface {
	ListTracks(ctx context.Context) ([]Track, error)
	GetTrackByID(ctx context.Context, id string) (Track, error)
	ListTracksByAlbum(ctx context.Context, albumID string) ([]Track, error)
	ListTracksByArtist(ctx context.Context, artistID string) ([]Track, error)
	SearchTracks(ctx context.Context, query string) ([]Track, error)
}

type PlaylistService interface {
	ListPlaylists(ctx context.Context) ([]Playlist, error)
	GetPlaylistByID(ctx context.Context, id string) (Playlist, error)
	CreatePlaylist(ctx context.Context, playlist Playlist) (Playlist, error)
	AddTrackToPlaylist(ctx context.Context, playlistID string, trackID string) error
	RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error
	ListPlaylistTracks(ctx context.Context, playlistID string) ([]Track, error)
}

type LibraryService interface {
	GetLibrary(ctx context.Context) (Library, error)
	UpdateLibraryRootPath(ctx context.Context, rootPath string) (Library, error)
	RefreshLibrary(ctx context.Context) (Library, error)
}
