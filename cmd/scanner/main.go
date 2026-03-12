package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	covercache "music-server/internal/modules/coverart/cache"
	coverextractor "music-server/internal/modules/coverart/extractor"
	coverservice "music-server/internal/modules/coverart/service"
	librarymetadata "music-server/internal/modules/library/metadata"
	libraryrepo "music-server/internal/modules/library/repository"
	libraryscanner "music-server/internal/modules/library/scanner"
	libraryservice "music-server/internal/modules/library/service"
	loudnessservice "music-server/internal/modules/loudness/service"
	waveformservice "music-server/internal/modules/waveform/service"
	"music-server/internal/platform/config"
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

	artistRepository := repositories.NewArtistRepository(db)
	albumRepository := repositories.NewAlbumRepository(db)
	coverCache, err := covercache.New(cfg.DataPath)
	if err != nil {
		logger.Fatal("failed to init cover cache", zap.Error(err))
	}
	coverArtService := coverservice.New(
		artistRepository,
		albumRepository,
		coverCache,
		coverextractor.New(),
	)
	statsRepository := libraryrepo.New(db)
	waveformGenerator := waveformservice.New(cfg.DataPath, repositories.NewTrackRepository(db))
	loudnessResolver := loudnessservice.New()

	service := libraryservice.NewScanService(
		cfg,
		logger,
		libraryscanner.NewFilesystemScanner(),
		librarymetadata.NewReader(),
		artistRepository,
		albumRepository,
		repositories.NewTrackRepository(db),
		repositories.NewLibraryRepository(db),
		repositories.NewFileFingerprintRepository(db),
		coverArtService,
		loudnessResolver,
		waveformGenerator,
		statsRepository,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Scan(ctx); err != nil {
		logger.Fatal("scanner failed", zap.Error(err))
	}

	logger.Info("scanner finished successfully")
}
