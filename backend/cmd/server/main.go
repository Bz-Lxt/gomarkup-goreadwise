package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goreadwise/internal/clip"
	"goreadwise/internal/clock"
	"goreadwise/internal/config"
	"goreadwise/internal/handler"
	"goreadwise/internal/logger"
	"goreadwise/internal/seed"
	"goreadwise/internal/service"
	"goreadwise/internal/store"
	"goreadwise/internal/worker"
)

func main() {
	if err := run(); err != nil {
		logger.L().Error("fatal", "err", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger.Init(cfg.LogLevel, os.Stdout)
	logger.L().Info("boot", "tz", clock.Now().Format(time.RFC3339), "clip", cfg.ClipProvider)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx, cfg.MigrationDir); err != nil {
		return err
	}

	hub := handler.NewHub()
	graphSvc := &service.GraphService{DB: db}
	cardSvc := &service.CardService{DB: db, Hub: hub}

	pool := worker.New(ctx, db, cfg.WorkerSize, cfg.QueueSize, cardSvc.ApplyAsync)
	defer pool.Close()
	cardSvc.Queue = pool
	pool.Recover(ctx)

	var provider clip.Provider
	if cfg.ClipProvider == "real" {
		rp := clip.NewReal(cfg.ClipTimeout, cfg.ClipMaxBytes)
		provider = rp
	} else {
		provider = clip.MockProvider{Dir: cfg.ClipFixtureDir}
	}
	clipSvc := &service.ClipService{Cards: cardSvc, Provider: provider, Mode: cfg.ClipProvider}
	rebuilder := &service.Rebuilder{DB: db, Cards: cardSvc, Graph: graphSvc}

	if cfg.SeedOnEmpty {
		if err := seed.Maybe(ctx, db, cardSvc); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: handler.NewRouter(handler.Deps{
			Cards: cardSvc, Graph: graphSvc, Clip: clipSvc, DB: db,
			Pool: pool, Hub: hub, ClipMode: cfg.ClipProvider, Started: clock.Now(), Rebuilder: rebuilder,
		}),
		ReadHeaderTimeout: 8 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.L().Info("listen", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
