package http

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	albumshttp "music-server/internal/modules/albums/transport/http"
	artistshttp "music-server/internal/modules/artists/transport/http"
	authservice "music-server/internal/modules/auth/service"
	authhttp "music-server/internal/modules/auth/transport/http"
	coverhttp "music-server/internal/modules/coverart/transport/http"
	guihttp "music-server/internal/modules/gui/transport/http"
	libraryhttp "music-server/internal/modules/library/transport/http"
	playerhttp "music-server/internal/modules/player/transport/http"
	playlistshttp "music-server/internal/modules/playlists/transport/http"
	searchhttp "music-server/internal/modules/search/transport/http"
	sharinghttp "music-server/internal/modules/sharing/transport/http"
	streamhttp "music-server/internal/modules/stream/transport/http"
	subsonichttp "music-server/internal/modules/subsonic/transport/http"
	trackshttp "music-server/internal/modules/tracks/transport/http"
	waveformhttp "music-server/internal/modules/waveform/transport/http"
	"music-server/internal/platform/health"
)

func NewRouter(
	logger *zap.Logger,
	healthService *health.Service,
	streamHandler *streamhttp.Handler,
	artistsHandler *artistshttp.Handler,
	albumsHandler *albumshttp.Handler,
	tracksHandler *trackshttp.Handler,
	playlistsHandler *playlistshttp.Handler,
	searchHandler *searchhttp.Handler,
	coverHandler *coverhttp.Handler,
	libraryHandler *libraryhttp.Handler,
	playerHandler *playerhttp.Handler,
	waveformHandler *waveformhttp.Handler,
	subsonicHandler *subsonichttp.Handler,
	guiHandler *guihttp.Handler,
	authHandler *authhttp.Handler,
	sharingHandler *sharinghttp.Handler,
	authService *authservice.Service,
) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestID)
	r.Use(middleware.RealIP)
	r.Use(PanicRecovery(logger))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(requestLogger(logger))

	r.Get("/healthz", healthService.HealthzHandler)
	r.Get("/readyz", healthService.ReadyzHandler)
	r.Route("/api", func(api chi.Router) {
		api.Get("/auth/status", authHandler.Status)
		api.Post("/auth/login", authHandler.Login)
		api.Post("/auth/register", authHandler.Register)
		api.With(AuthRequired(authService)).Post("/auth/logout", authHandler.Logout)
		api.With(AuthRequired(authService)).Get("/auth/users", authHandler.ListUsers)
		api.With(AuthRequired(authService)).Post("/auth/users/{user_id}/active", authHandler.SetUserActive)
		api.With(AuthRequired(authService)).Post("/auth/users/{user_id}/delete", authHandler.DeleteUser)
		api.With(AuthRequired(authService)).Post("/auth/settings/registration", authHandler.SetRegistrationOpen)
		api.With(AuthRequired(authService)).Get("/auth/users/lookup", authHandler.ListShareableUsers)
		api.With(AuthRequired(authService)).Get("/auth/profile/{user_id}", authHandler.GetProfile)
		api.With(AuthRequired(authService)).Post("/auth/profile/update", authHandler.UpdateProfile)
		api.With(AuthRequired(authService)).Post("/auth/profile/password", authHandler.ChangePassword)

		api.Group(func(api chi.Router) {
			api.Use(AuthRequired(authService))

			api.Get("/stream/{track_id}", streamHandler.StreamTrack)

			api.Get("/artists", artistsHandler.ListArtists)
			api.Get("/artists/{id}", artistsHandler.GetArtist)
			api.Get("/artists/{id}/albums", albumsHandler.ListArtistAlbums)

			api.Get("/albums", albumsHandler.ListAlbums)
			api.Get("/albums/{id}", albumsHandler.GetAlbum)
			api.Get("/albums/{id}/tracks", tracksHandler.ListAlbumTracks)

			api.Get("/tracks", tracksHandler.ListTracks)
			api.Get("/tracks/{id}", tracksHandler.GetTrack)
			api.Get("/tracks/{id}/waveform", waveformHandler.GetTrackWaveform)

			api.Get("/playlists", playlistsHandler.ListPlaylists)
			api.Get("/playlists/{id}", playlistsHandler.GetPlaylist)
			api.Post("/playlists", playlistsHandler.CreatePlaylist)
			api.Post("/playlists/{id}/rename", playlistsHandler.RenamePlaylist)
			api.Post("/playlists/{id}/update", playlistsHandler.UpdatePlaylist)
			api.Delete("/playlists/{id}", playlistsHandler.DeletePlaylist)
			api.Post("/playlists/{id}/tracks", playlistsHandler.AddTrack)
			api.Delete("/playlists/{id}/tracks/{track_id}", playlistsHandler.RemoveTrack)

			api.Get("/search", searchHandler.Search)
			api.Get("/shares/received", sharingHandler.ListReceivedShares)
			api.Get("/shares/{entity_type}/{entity_id}", sharingHandler.ListEntityShares)
			api.Post("/shares/{entity_type}/{entity_id}/users", sharingHandler.ShareWithUser)
			api.Post("/shares/{entity_type}/{entity_id}/public", sharingHandler.SetPublicShare)
			api.Delete("/shares/{share_id}", sharingHandler.DeleteShare)

			api.Get("/covers/album/{album_id}", coverHandler.AlbumCover)
			api.Get("/covers/artist/{artist_id}", coverHandler.ArtistCover)
			api.Get("/covers/album/{album_id}/thumb/{size}", coverHandler.AlbumThumb)

			api.Get("/library/home", libraryHandler.Home)
			api.Get("/library/random-albums", libraryHandler.RandomAlbums)
			api.Get("/library/artist-album-counts", libraryHandler.ArtistAlbumCounts)
			api.Post("/library/scan", libraryHandler.Scan)
			api.Get("/library/scan/status", libraryHandler.ScanStatus)
			api.Post("/library/upload", libraryHandler.Upload)
			api.Post("/library/favorites/tracks/{track_id}/toggle", libraryHandler.ToggleFavoriteTrack)
			api.Post("/library/favorites/albums/{album_id}/toggle", libraryHandler.ToggleFavoriteAlbum)
			api.Post("/library/favorites/artists/{artist_id}/toggle", libraryHandler.ToggleFavoriteArtist)
			api.Post("/library/artists/{artist_id}/update", libraryHandler.UpdateArtist)
			api.Post("/library/artists/{artist_id}/cover", libraryHandler.UpdateArtistCover)
			api.Post("/library/artists/{artist_id}/delete", libraryHandler.DeleteArtist)
			api.Post("/library/tracks/{track_id}/delete", libraryHandler.DeleteTrack)
			api.Post("/library/tracks/{track_id}/rename", libraryHandler.RenameTrack)
			api.Post("/library/tracks/{track_id}/update", libraryHandler.UpdateTrack)
			api.Post("/library/albums/{album_id}/delete", libraryHandler.DeleteAlbum)
			api.Post("/library/albums/{album_id}/rename", libraryHandler.RenameAlbum)
			api.Post("/library/albums/{album_id}/update", libraryHandler.UpdateAlbum)
			api.Post("/library/albums/{album_id}/merge", libraryHandler.MergeAlbum)
			api.Post("/library/albums/create", libraryHandler.CreateAlbum)
			api.Get("/settings", libraryHandler.Settings)
			api.Post("/settings/updates/check", libraryHandler.CheckUpdates)

			api.Get("/player/state", playerHandler.GetState)
			api.Post("/player/state", playerHandler.SetState)
			api.Post("/player/queue/replace", playerHandler.ReplaceQueue)
			api.Post("/player/queue/append", playerHandler.AppendQueue)
			api.Post("/player/queue/remove", playerHandler.RemoveQueueItem)
			api.Post("/player/queue/clear", playerHandler.ClearQueue)
			api.Post("/player/queue/move", playerHandler.MoveQueueItem)
			api.Post("/player/queue/shuffle", playerHandler.ShuffleQueue)
			api.Post("/player/played", playerHandler.Played)
		})
	})

	r.Route("/rest", func(rest chi.Router) {
		subsonicHandler.Mount(rest)
	})

	r.Handle("/static/*", http.StripPrefix("/static/", guiHandler.StaticFiles()))
	r.Get("/", guiHandler.ServeShell)
	r.Get("/artists", guiHandler.ServeShell)
	r.Get("/artists/{id}", guiHandler.ServeShell)
	r.Get("/albums", guiHandler.ServeShell)
	r.Get("/albums/{id}", guiHandler.ServeShell)
	r.Get("/tracks", guiHandler.ServeShell)
	r.Get("/tracks/{id}", guiHandler.ServeShell)
	r.Get("/playlists", guiHandler.ServeShell)
	r.Get("/playlists/{id}", guiHandler.ServeShell)
	r.Get("/search", guiHandler.ServeShell)
	r.Get("/library", guiHandler.ServeShell)
	r.Get("/settings", guiHandler.ServeShell)
	r.Get("/users", guiHandler.ServeShell)
	r.Get("/profile", guiHandler.ServeShell)
	r.Get("/profile/{id}", guiHandler.ServeShell)
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.NotFound(w, req)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/static/") || strings.HasPrefix(req.URL.Path, "/rest/") {
			http.NotFound(w, req)
			return
		}
		if ext := path.Ext(req.URL.Path); ext != "" {
			http.NotFound(w, req)
			return
		}
		guiHandler.ServeShell(w, req)
	})

	return r
}
