package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string
	HTTPAddr       string
	DatabaseURL    string
	LogLevel       string
	ClipProvider   string
	ClipFixtureDir string
	ClipTimeout    time.Duration
	ClipMaxBytes   int64
	WorkerSize     int
	QueueSize      int
	MigrationDir   string
	SeedOnEmpty    bool
}

func Load() (Config, error) {
	cfg := Config{
		Env:            envOr("APP_ENV", "development"),
		HTTPAddr:       envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:    envOr("DATABASE_URL", "postgres://goreadwise:goreadwise@127.0.0.1:15233/goreadwise?sslmode=disable"),
		LogLevel:       envOr("LOG_LEVEL", "info"),
		ClipProvider:   strings.ToLower(envOr("CLIP_PROVIDER", "mock")),
		ClipFixtureDir: envOr("CLIP_FIXTURE_DIR", "testdata/clips"),
		ClipTimeout:    time.Duration(envInt("CLIP_TIMEOUT_SEC", 10)) * time.Second,
		ClipMaxBytes:   int64(envInt("CLIP_MAX_BYTES", 5*1024*1024)),
		WorkerSize:     envInt("WORKER_SIZE", 4),
		QueueSize:      envInt("QUEUE_SIZE", 64),
		MigrationDir:   envOr("MIGRATION_DIR", "migrations"),
		SeedOnEmpty:    envOr("SEED_ON_EMPTY", "true") != "false",
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.ClipProvider != "mock" && cfg.ClipProvider != "real" {
		return cfg, fmt.Errorf("CLIP_PROVIDER must be mock or real")
	}
	if cfg.WorkerSize < 1 {
		cfg.WorkerSize = 1
	}
	if cfg.QueueSize < 8 {
		cfg.QueueSize = 8
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
