package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/config"
	"sourceink/backend/internal/database"
	"sourceink/backend/internal/httpserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if cfg.AutoMigrate {
		if err := database.RunMigrations(ctx, db, "up"); err != nil {
			logger.Error("run migrations", "error", err)
			os.Exit(1)
		}
	}
	if err := auth.EnsureDevelopmentAdmin(
		ctx,
		db,
		cfg.Environment,
		cfg.DevAdminEmail,
		cfg.DevAdminUsername,
		cfg.DevAdminPassword,
	); err != nil {
		logger.Error("seed development admin", "error", err)
		os.Exit(1)
	}
	if cfg.Environment == "development" {
		logger.Info("development admin ready", "email", cfg.DevAdminEmail, "username", cfg.DevAdminUsername)
	}

	handler, articlesWorker, err := httpserver.New(cfg, db, logger)
	if err != nil {
		logger.Error("create HTTP server", "error", err)
		os.Exit(1)
	}
	logger.Info("initial article sync started")
	initialArticles, err := articlesWorker.SyncOnce(ctx)
	if err != nil {
		logger.Error("initial article sync failed", "error", err)
	} else {
		logger.Info("initial article sync completed", "articles", len(initialArticles))
	}

	go func() {
		logger.Info("articles worker started")
		for ctx.Err() == nil {
			if err := articlesWorker.Start(ctx); err != nil && ctx.Err() == nil {
				logger.Error("article sync failed; worker will retry", "error", err)
				continue
			}
			return
		}
	}()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Info("HTTP server listening", "address", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}
