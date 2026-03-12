package mapper

import (
	"sort"
	"strings"

	"music-server/internal/domain"
	searchservice "music-server/internal/modules/search/service"
)

func NewSuccess() Response {
	return Response{
		Status:        "ok",
		Version:       "1.16.1",
		Type:          "music-server",
		ServerVersion: "0.1.0",
		OpenSubsonic:  true,
	}
}

func WithError(resp Response, code int, message string) Response {
	resp.Status = "failed"
	resp.Error = &Error{Code: code, Message: message}
	return resp
}

func Artists(artists []domain.Artist) *ArtistsResult {
	groups := map[string][]Artist{}
	for _, item := range artists {
		letter := "#"
		if item.Name != "" {
			letter = strings.ToUpper(string([]rune(strings.TrimSpace(item.Name))[0]))
		}
		groups[letter] = append(groups[letter], Artist{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	indexes := make([]Index, 0, len(keys))
	for _, key := range keys {
		indexes = append(indexes, Index{
			Name:    key,
			Artists: groups[key],
		})
	}

	return &ArtistsResult{Indexes: indexes}
}

func ArtistWithAlbums(artist domain.Artist, albums []domain.Album) *Artist {
	out := Artist{
		ID:         artist.ID,
		Name:       artist.Name,
		AlbumCount: len(albums),
		Albums:     make([]Album, 0, len(albums)),
	}
	for _, album := range albums {
		out.Albums = append(out.Albums, Album{
			ID:       album.ID,
			Name:     album.Title,
			ArtistID: album.ArtistID,
			Year:     album.Year,
			CoverArt: album.ID,
		})
	}
	return &out
}

func AlbumWithSongs(album domain.Album, songs []domain.Track) *Album {
	out := Album{
		ID:       album.ID,
		Name:     album.Title,
		ArtistID: album.ArtistID,
		Year:     album.Year,
		CoverArt: album.ID,
		Song:     make([]Song, 0, len(songs)),
	}
	for _, track := range songs {
		out.Song = append(out.Song, SongFromTrack(track))
	}
	return &out
}

func SongFromTrack(track domain.Track) Song {
	return Song{
		ID:       track.ID,
		Title:    track.Title,
		AlbumID:  track.AlbumID,
		ArtistID: track.ArtistID,
		Track:    track.TrackNumber,
		Duration: int(track.Duration.Seconds()),
		BitRate:  track.Bitrate,
		CoverArt: track.AlbumID,
	}
}

func AlbumListFromDomain(albums []domain.Album) *AlbumList {
	out := make([]Album, 0, len(albums))
	for _, album := range albums {
		out = append(out, Album{
			ID:       album.ID,
			Name:     album.Title,
			ArtistID: album.ArtistID,
			Year:     album.Year,
			CoverArt: album.ID,
		})
	}
	return &AlbumList{Albums: out}
}

func PlaylistsFromDomain(playlists []domain.Playlist) *Playlists {
	out := make([]Playlist, 0, len(playlists))
	for _, item := range playlists {
		out = append(out, Playlist{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	return &Playlists{Playlist: out}
}

func PlaylistWithSongs(playlist domain.Playlist, tracks []domain.Track) *Playlist {
	out := Playlist{
		ID:        playlist.ID,
		Name:      playlist.Name,
		SongCount: len(tracks),
		Song:      make([]Song, 0, len(tracks)),
	}
	for _, track := range tracks {
		out.Song = append(out.Song, SongFromTrack(track))
	}
	return &out
}

func SearchResultFromDomain(result searchservice.Result) *SearchResult3 {
	out := &SearchResult3{
		Artists: make([]Artist, 0, len(result.Artists)),
		Albums:  make([]Album, 0, len(result.Albums)),
		Songs:   make([]Song, 0, len(result.Tracks)),
	}
	for _, artist := range result.Artists {
		out.Artists = append(out.Artists, Artist{
			ID:   artist.ID,
			Name: artist.Name,
		})
	}
	for _, album := range result.Albums {
		out.Albums = append(out.Albums, Album{
			ID:       album.ID,
			Name:     album.Title,
			ArtistID: album.ArtistID,
			Year:     album.Year,
			CoverArt: album.ID,
		})
	}
	for _, track := range result.Tracks {
		out.Songs = append(out.Songs, SongFromTrack(track))
	}
	return out
}
