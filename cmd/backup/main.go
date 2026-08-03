package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"permatatex-inventory/internal/backup"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if len(os.Args) != 2 || os.Args[1] != "run" {
		logger.Error("backup command failed", slog.String("failure_class", "invalid_command"), slog.Int("exit_code", 2))
		fmt.Fprintln(os.Stderr, "usage: backup run")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	job := backup.NewJob(backup.ConfigFromEnv(os.Getenv), logger)
	err := job.Run(ctx)
	if err == nil {
		return
	}
	code := backup.ExitCode(err)
	logger.Error("backup failed", slog.String("failure_class", failureClass(err, code)), slog.Int("exit_code", code))
	os.Exit(code)
}

func failureClass(err error, code int) string {
	if errors.Is(err, backup.ErrCleanup) {
		return "cleanup"
	}
	switch code {
	case 2:
		return "configuration"
	case 3:
		return "lock_held"
	case 4:
		return "prerequisite"
	case 5:
		return "packaging"
	case 6:
		return "finalization"
	case 130:
		return "interrupted"
	default:
		return "unknown"
	}
}
