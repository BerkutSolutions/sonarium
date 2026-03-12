package service

import (
	"context"
	"database/sql"
	"errors"

	"music-server/internal/domain"
	albumsservice "music-server/internal/modules/albums/service"
	artistsservice "music-server/internal/modules/artists/service"
	coverservice "music-server/internal/modules/coverart/service"
	playlistsservice "music-server/internal/modules/playlists/service"
	searchservice "music-server/internal/modules/search/service"
	streamservice "music-server/internal/modules/stream/service"
	tracksservice "music-server/internal/modules/tracks/service"
)

var ErrCoverArtNotFound = errors.New("cover art not found")

type Service struct {
	artists   *artistsservice.Service
	albums    *albumsservice.Service
	tracks    *tracksservice.Service
	playlists *playlistsservice.Service
	search    *searchservice.Service
	stream    *streamservice.Service
	covers    *coverservice.Service
}

func New(
	artists *artistsservice.Service,
	albums *albumsservice.Service,
	tracks *tracksservice.Service,
	playlists *playlistsservice.Service,
	search *searchservice.Service,
	stream *streamservice.Service,
	covers *coverservice.Service,
) *Service {
	return &Service{
		artists:   artists,
		albums:    albums,
		tracks:    tracks,
		playlists: playlists,
		search:    search,
		stream:    stream,
		covers:    covers,
	}
}

func (s *Service) ListArtists(ctx context.Context) ([]domain.Artist, error) {
	return s.artists.List(ctx, artistsservice.ListParams{SortBy: "name"})
}

func (s *Service) GetArtist(ctx context.Context, id string) (domain.Artist, []domain.Album, error) {
	artist, err := s.artists.GetByID(ctx, id)
	if err != nil {
		return domain.Artist{}, nil, err
	}
	albums, err := s.albums.ListByArtistID(ctx, id, albumsservice.ListParams{SortBy: "year"})
	if err != nil {
		return domain.Artist{}, nil, err
	}
	return artist, albums, nil
}

func (s *Service) GetAlbum(ctx context.Context, id string) (domain.Album, []domain.Track, error) {
	album, err := s.albums.GetByID(ctx, id)
	if err != nil {
		return domain.Album{}, nil, err
	}
	tracks, err := s.tracks.ListByAlbumID(ctx, id, tracksservice.ListParams{SortBy: "name"})
	if err != nil {
		return domain.Album{}, nil, err
	}
	return album, tracks, nil
}

func (s *Service) GetAlbumList(ctx context.Context, size, offset int) ([]domain.Album, error) {
	return s.albums.List(ctx, albumsservice.ListParams{
		Limit:  size,
		Offset: offset,
		SortBy: "created_at",
	})
}

func (s *Service) GetSong(ctx context.Context, id string) (domain.Track, error) {
	return s.tracks.GetByID(ctx, id)
}

func (s *Service) GetPlaylists(ctx context.Context) ([]domain.Playlist, error) {
	return s.playlists.List(ctx, nil, playlistsservice.ListParams{SortBy: "name"})
}

func (s *Service) GetPlaylist(ctx context.Context, id string) (domain.Playlist, []domain.Track, error) {
	playlist, err := s.playlists.GetByID(ctx, nil, id, "")
	if err != nil {
		return domain.Playlist{}, nil, err
	}
	tracks, err := s.playlists.ListTracks(ctx, nil, id, "")
	if err != nil {
		return domain.Playlist{}, nil, err
	}
	return playlist, tracks, nil
}

func (s *Service) Search3(ctx context.Context, query string, limit, offset int) (searchservice.Result, error) {
	return s.search.Search(ctx, searchservice.Params{
		Query:  query,
		Limit:  limit,
		Offset: offset,
		SortBy: "name",
	})
}

func (s *Service) ResolveCoverArt(ctx context.Context, id string) (string, string, error) {
	if s.covers == nil {
		return "", "", ErrCoverArtNotFound
	}

	if path, mimeType, err := s.covers.AlbumOriginal(ctx, id, false); err == nil {
		return path, mimeType, nil
	}

	track, trackErr := s.tracks.GetByID(ctx, id)
	if trackErr != nil {
		if errors.Is(trackErr, sql.ErrNoRows) {
			return "", "", ErrCoverArtNotFound
		}
		return "", "", trackErr
	}

	path, mimeType, err := s.covers.AlbumOriginal(ctx, track.AlbumID, false)
	if err != nil {
		return "", "", ErrCoverArtNotFound
	}
	return path, mimeType, nil
}

func (s *Service) ResolveStream(ctx context.Context, trackID string) (streamservice.StreamableTrack, error) {
	return s.stream.ResolveTrack(ctx, trackID)
}
