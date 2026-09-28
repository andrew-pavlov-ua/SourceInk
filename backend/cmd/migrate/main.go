package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sourceink/backend/internal/config"
	"sourceink/backend/internal/database"
)

func main() {
	command := "status"
	var args []string
	if len(os.Args) > 1 {
		command = os.Args[1]
		args = os.Args[2:]
	}

	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fail(err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db, command, args...); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
