package main

import (
	"context"
	"net/http"
	"time"

	"github.com/devyani1512/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

func shutdown(srv *http.Server, sched *scheduler.Scheduler, cancel context.CancelFunc, logger *zap.Logger) {
	logger.Info("shutting down...")

	cancel()
	sched.Stop()

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", zap.Error(err))
	}

	logger.Info("shutdown complete")
}
