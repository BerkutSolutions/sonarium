package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"music-server/internal/app"
	"music-server/internal/appmeta"
	albumsservice "music-server/internal/modules/albums/service"
	albumshttp "music-server/internal/modules/albums/transport/http"
	artistsservice "music-server/internal/modules/artists/service"
	artistshttp "music-server/internal/modules/artists/transport/http"
	authservice "music-server/internal/modules/auth/service"
	authhttp "music-server/internal/modules/auth/transport/http"
	covercache "music-server/internal/modules/coverart/cache"
	coverextractor "music-server/internal/modules/coverart/extractor"
	coverservice "music-server/internal/modules/coverart/service"
	coverhttp "music-server/internal/modules/coverart/transport/http"
	guihttp "music-server/internal/modules/gui/transport/http"
	librarymetadata "music-server/internal/modules/library/metadata"
	libraryrepo "music-server/internal/modules/library/repository"
	libraryscanner "music-server/internal/modules/library/scanner"
	libraryservice "music-server/internal/modules/library/service"
	libraryhttp "music-server/internal/modules/library/transport/http"
	loudnessservice "music-server/internal/modules/loudness/service"
	playerservice "music-server/internal/modules/player/service"
	playerhttp "music-server/internal/modules/player/transport/http"
	playlistsservice "music-server/internal/modules/playlists/service"
	playlistshttp "music-server/internal/modules/playlists/transport/http"
	searchservice "music-server/internal/modules/search/service"
	searchhttp "music-server/internal/modules/search/transport/http"
	sharingservice "music-server/internal/modules/sharing/service"
	sharinghttp "music-server/internal/modules/sharing/transport/http"
	streamservice "music-server/internal/modules/stream/service"
	streamhttp "music-server/internal/modules/stream/transport/http"
	subsonicservice "music-server/internal/modules/subsonic/service"
	subsonichttp "music-server/internal/modules/subsonic/transport/http"
	tracksservice "music-server/internal/modules/tracks/service"
	trackshttp "music-server/internal/modules/tracks/transport/http"
	transcodingservice "music-server/internal/modules/transcoding/service"
	waveformservice "music-server/internal/modules/waveform/service"
	waveformhttp "music-server/internal/modules/waveform/transport/http"
	"music-server/internal/platform/config"
	"music-server/internal/platform/health"
	"music-server/internal/platform/logging"
	"music-server/internal/storage/migrations"
	"music-server/internal/storage/postgres"
	"music-server/internal/storage/repositories"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	db, err := postgres.Open(cfg)
	if err != nil {
		logger.Fatal("postgres connection failed", zap.Error(err))
	}
	defer func() {
		_ = db.Close()
	}()

	if err := migrations.Up(db, migrations.ResolveDir()); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	healthService := health.NewService(postgres.ReadinessDependency{DB: db})
	artistRepository := repositories.NewArtistRepository(db)
	albumRepository := repositories.NewAlbumRepository(db)
	trackRepository := repositories.NewTrackRepository(db)
	playlistRepository := repositories.NewPlaylistRepository(db)
	libraryStateRepository := repositories.NewLibraryRepository(db)
	fileFingerprintRepository := repositories.NewFileFingerprintRepository(db)
	authRepository := authservice.NewRepository(db)
	coverCache, err := covercache.New(cfg.DataPath)
	if err != nil {
		logger.Fatal("failed to init cover cache", zap.Error(err))
	}

	artistsService := artistsservice.New(artistRepository)
	albumsService := albumsservice.New(albumRepository)
	tracksService := tracksservice.New(trackRepository)
	playlistsService := playlistsservice.New(playlistRepository)
	searchService := searchservice.New(artistRepository, albumRepository, trackRepository)
	streamService := streamservice.New(trackRepository)
	libraryRepository := libraryrepo.New(db)
	authSvc := authservice.New(authRepository)
	sharingSvc := sharingservice.New(db)
	updateChecker := appmeta.NewUpdateChecker(cfg.RepositoryURL, cfg.GitHubReleasesURL)
	coverArtService := coverservice.New(
		artistRepository,
		albumRepository,
		coverCache,
		coverextractor.New(),
	)
	dashboardService := libraryservice.NewDashboardService(libraryRepository, artistRepository, trackRepository, coverArtService, 15*time.Second)
	loudnessService := loudnessservice.New()
	playerService := playerservice.New(dashboardService)
	transcodingEngine := transcodingservice.New()
	waveformService := waveformservice.New(cfg.DataPath, trackRepository)
	scanService := libraryservice.NewScanService(
		cfg,
		logger,
		libraryscanner.NewFilesystemScanner(),
		librarymetadata.NewReader(),
		artistRepository,
		albumRepository,
		trackRepository,
		libraryStateRepository,
		fileFingerprintRepository,
		coverArtService,
		loudnessService,
		waveformService,
		libraryRepository,
	)
	libraryManagementService := libraryservice.NewManagementService(cfg, scanService, func(ctx context.Context) error {
		return db.PingContext(ctx)
	}, updateChecker)

	artistsHandler := artistshttp.NewHandler(artistsService)
	albumsHandler := albumshttp.NewHandler(albumsService)
	tracksHandler := trackshttp.NewHandler(tracksService)
	playlistsHandler := playlistshttp.NewHandler(playlistsService)
	searchHandler := searchhttp.NewHandler(searchService)
	streamHandler := streamhttp.NewHandler(logger, streamService, transcodingEngine)
	coverHandler := coverhttp.NewHandler(coverArtService)
	libraryHandler := libraryhttp.NewHandler(dashboardService, libraryManagementService)
	playerHandler := playerhttp.NewHandler(playerService)
	waveformHandler := waveformhttp.NewHandler(waveformService)
	subsonicHandler := subsonichttp.NewHandler(
		subsonicservice.NewAuthenticator(subsonicservice.AuthConfig{
			Username:   cfg.SubsonicUsername,
			Password:   cfg.SubsonicPassword,
			MinVersion: cfg.SubsonicMinVer,
		}),
		subsonicservice.New(
			artistsService,
			albumsService,
			tracksService,
			playlistsService,
			searchService,
			streamService,
			coverArtService,
		),
	)
	guiHandler := guihttp.NewHandler()
	authHandler := authhttp.NewHandler(authSvc)
	sharingHandler := sharinghttp.NewHandler(sharingSvc)

	application := app.New(
		cfg,
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
		authSvc,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting http server", zap.Int("port", cfg.Port), zap.String("env", cfg.AppEnv))
		errCh <- application.Start()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err = <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := application.Stop(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("server stopped")
}
