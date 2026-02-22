package main

import (
	"github.com/devyani1512/scheduler/internal/config"
	"github.com/devyani1512/scheduler/internal/db"
	"go.uber.org/zap"
)

func connectDB(cfg *config.Config, logger *zap.Logger) *db.DB {
	database, err := db.New(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	return database
}
