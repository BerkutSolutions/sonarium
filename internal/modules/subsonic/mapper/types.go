package mapper

import "encoding/xml"

type Response struct {
	XMLName       xml.Name       `xml:"subsonic-response" json:"-"`
	Status        string         `xml:"status,attr" json:"status"`
	Version       string         `xml:"version,attr" json:"version"`
	Type          string         `xml:"type,attr" json:"type"`
	ServerVersion string         `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool           `xml:"openSubsonic,attr" json:"openSubsonic"`
	Error         *Error         `xml:"error,omitempty" json:"error,omitempty"`
	License       *License       `xml:"license,omitempty" json:"license,omitempty"`
	Artists       *ArtistsResult `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist        *Artist        `xml:"artist,omitempty" json:"artist,omitempty"`
	Album         *Album         `xml:"album,omitempty" json:"album,omitempty"`
	AlbumList     *AlbumList     `xml:"albumList,omitempty" json:"albumList,omitempty"`
	Song          *Song          `xml:"song,omitempty" json:"song,omitempty"`
	Playlists     *Playlists     `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist      *Playlist      `xml:"playlist,omitempty" json:"playlist,omitempty"`
	SearchResult3 *SearchResult3 `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
}

type Error struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type License struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

type ArtistsResult struct {
	Indexes []Index `xml:"index" json:"index"`
}

type Index struct {
	Name    string   `xml:"name,attr" json:"name"`
	Artists []Artist `xml:"artist" json:"artist"`
}

type Artist struct {
	ID         string  `xml:"id,attr" json:"id"`
	Name       string  `xml:"name,attr" json:"name"`
	AlbumCount int     `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
	Albums     []Album `xml:"album,omitempty" json:"album,omitempty"`
}

type Album struct {
	ID       string `xml:"id,attr" json:"id"`
	Name     string `xml:"name,attr" json:"name"`
	ArtistID string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Artist   string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Year     int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	CoverArt string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Song     []Song `xml:"song,omitempty" json:"song,omitempty"`
}

type AlbumList struct {
	Albums []Album `xml:"album" json:"album"`
}

type Song struct {
	ID       string `xml:"id,attr" json:"id"`
	Title    string `xml:"title,attr" json:"title"`
	Album    string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist   string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	AlbumID  string `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Track    int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year     int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Duration int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate  int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	CoverArt string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
}

type Playlists struct {
	Playlist []Playlist `xml:"playlist" json:"playlist"`
}

type Playlist struct {
	ID        string `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	SongCount int    `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	Song      []Song `xml:"entry,omitempty" json:"entry,omitempty"`
}

type SearchResult3 struct {
	Artists []Artist `xml:"artist,omitempty" json:"artist,omitempty"`
	Albums  []Album  `xml:"album,omitempty" json:"album,omitempty"`
	Songs   []Song   `xml:"song,omitempty" json:"song,omitempty"`
}
