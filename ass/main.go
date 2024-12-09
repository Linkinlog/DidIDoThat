package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var logger *slog.Logger

func init() {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, slogOpts))
}

func main() {
	conn, err := pgxpool.New(context.Background(), databaseURL())
	if err != nil {
		logger.Error("Unable to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	if isFirstRun() {
		if err := createTables(context.Background(), conn); err != nil {
			logger.Error("Unable to create tables", "error", err.Error())
			os.Exit(1)
		}
	}

	if err := startHTTP(8008, conn); err != nil {
		logger.Error("Unable to start HTTP server", "error", err.Error())
		os.Exit(1)
	}
}
