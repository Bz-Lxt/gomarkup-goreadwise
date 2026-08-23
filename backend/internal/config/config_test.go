package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("CLIP_PROVIDER", "mock")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClipProvider != "mock" || cfg.HTTPAddr == "" {
		t.Fatalf("%+v", cfg)
	}
}

func TestLoadRejectsBadProvider(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("CLIP_PROVIDER", "magic")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvIntFallback(t *testing.T) {
	os.Unsetenv("WORKER_SIZE")
	if envInt("WORKER_SIZE", 4) != 4 {
		t.Fatal("fallback")
	}
}
