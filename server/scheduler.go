package main

import (
	"github.com/devyani1512/scheduler/internal/db"
	"github.com/devyani1512/scheduler/internal/executor"
	"github.com/devyani1512/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

func buildScheduler(database *db.DB, logger *zap.Logger) *scheduler.Scheduler {
	exec := executor.New(database, logger)
	return scheduler.New(database, exec, logger)
}
