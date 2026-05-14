package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/bootstrap"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/config"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/logger"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/migrator"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize configuration
	cfg := config.Load()

	// Initialize structured logger
	log, closeLog, err := logger.New(logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Environment,
		Dir:         cfg.LogDir,
	})
	if err != nil {
		// Fallback to simple stdout logger if we can't open log files
		log = logger.NewSimple(cfg.LogLevel, cfg.Environment)
		log.Error("Failed to initialize file logger, falling back to stdout", "error", err)
	} else {
		defer closeLog()
	}
	slog.SetDefault(log)

	slog.Info("Starting application", "env", cfg.Environment, "port", cfg.Port)
	// initialize db
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	slog.Info("Successfully connected to database")

	// Run database migrations
	if err := migrator.Run(cfg.DatabaseURL, "file://migrations"); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize dependency injection container
	container := bootstrap.NewContainer(cfg, log, db)



	// Initialize HTTP server
	srv := server.New(cfg, container.Router())

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exiting")
}
