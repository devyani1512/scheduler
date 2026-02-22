package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/devyani1512/scheduler/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	logger := buildLogger(cfg.LogLevel)
	defer logger.Sync()
	database := connectDB(cfg, logger)
	defer database.Close()

	sched := buildScheduler(database, logger)
	srv := buildServer(cfg, database, sched, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recoverTasks(ctx, database, sched, logger)
	sched.Start(ctx)
	startHTTP(srv, logger)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdown(srv, sched, cancel, logger)
}
