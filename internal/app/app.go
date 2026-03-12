package app

import (
	"context"
	"fmt"
	"net/http"

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
	"music-server/internal/platform/config"
	"music-server/internal/platform/health"
	httptransport "music-server/internal/platform/http"
)

type App struct {
	server *http.Server
}

func New(
	cfg config.Config,
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
) *App {
	router := httptransport.NewRouter(
		logger,
		healthService,
		streamHandler,
		artistsHandler,
		albumsHandler,
		tracksHandler,
		playlistsHandler,
		searchHandler,
		coverHandler,
		libraryHandler,
		playerHandler,
		waveformHandler,
		subsonicHandler,
		guiHandler,
		authHandler,
		sharingHandler,
		authService,
	)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	return &App{server: server}
}

func (a *App) Start() error {
	return a.server.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
