package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port              int
	AppEnv            string
	LogLevel          string
	MusicLibraryPath  string
	DataPath          string
	ScannerWorkers    int
	RepositoryURL     string
	GitHubReleasesURL string
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	SubsonicUsername  string
	SubsonicPassword  string
	SubsonicMinVer    string
}

func Load() (Config, error) {
	cfg := Config{
		Port:              8080,
		AppEnv:            getEnv("APP_ENV", "development"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		MusicLibraryPath:  getEnv("MUSIC_LIBRARY_PATH", "/music"),
		DataPath:          getEnv("DATA_PATH", "/data"),
		ScannerWorkers:    getEnvAsInt("SCANNER_WORKERS", 4),
		RepositoryURL:     getEnv("APP_REPOSITORY_URL", "https://github.com/BerkutSolutions/sonarium"),
		GitHubReleasesURL: getEnv("APP_GITHUB_RELEASES_API", "https://api.github.com/repos/BerkutSolutions/sonarium/releases/latest"),
		DBHost:            getEnv("DB_HOST", "postgres"),
		DBPort:            getEnvAsInt("DB_PORT", 5432),
		DBUser:            getEnv("DB_USER", "music"),
		DBPassword:        getEnv("DB_PASSWORD", "music"),
		DBName:            getEnv("DB_NAME", "music"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		SubsonicUsername:  getEnv("SUBSONIC_USERNAME", "admin"),
		SubsonicPassword:  getEnv("SUBSONIC_PASSWORD", "admin"),
		SubsonicMinVer:    getEnv("SUBSONIC_MIN_VERSION", "1.16.1"),
	}

	if rawPort, ok := os.LookupEnv("PORT"); ok && rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT value %q: %w", rawPort, err)
		}
		cfg.Port = port
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return Config{}, errors.New("PORT must be between 1 and 65535")
	}

	if cfg.AppEnv == "" {
		return Config{}, errors.New("APP_ENV cannot be empty")
	}

	if cfg.LogLevel == "" {
		return Config{}, errors.New("LOG_LEVEL cannot be empty")
	}
	if cfg.ScannerWorkers <= 0 {
		return Config{}, errors.New("SCANNER_WORKERS must be greater than 0")
	}
	if cfg.RepositoryURL == "" {
		return Config{}, errors.New("APP_REPOSITORY_URL cannot be empty")
	}
	if cfg.GitHubReleasesURL == "" {
		return Config{}, errors.New("APP_GITHUB_RELEASES_API cannot be empty")
	}
	if cfg.DBHost == "" {
		return Config{}, errors.New("DB_HOST cannot be empty")
	}
	if cfg.DBPort <= 0 || cfg.DBPort > 65535 {
		return Config{}, errors.New("DB_PORT must be between 1 and 65535")
	}
	if cfg.DBUser == "" {
		return Config{}, errors.New("DB_USER cannot be empty")
	}
	if cfg.DBName == "" {
		return Config{}, errors.New("DB_NAME cannot be empty")
	}
	if cfg.DBSSLMode == "" {
		return Config{}, errors.New("DB_SSLMODE cannot be empty")
	}
	if cfg.SubsonicUsername == "" {
		return Config{}, errors.New("SUBSONIC_USERNAME cannot be empty")
	}
	if cfg.SubsonicPassword == "" {
		return Config{}, errors.New("SUBSONIC_PASSWORD cannot be empty")
	}
	if cfg.SubsonicMinVer == "" {
		return Config{}, errors.New("SUBSONIC_MIN_VERSION cannot be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
