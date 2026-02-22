package main

import (
	"net/http"

	"github.com/devyani1512/scheduler/internal/api"
	"github.com/devyani1512/scheduler/internal/config"
	"github.com/devyani1512/scheduler/internal/db"
	"github.com/devyani1512/scheduler/internal/scheduler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func buildServer(cfg *config.Config, database *db.DB, sched *scheduler.Scheduler, logger *zap.Logger) *http.Server {
	handler := api.NewHandler(database, sched, logger)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))
	handler.Register(r)

	return &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}
}
func startHTTP(srv *http.Server, logger *zap.Logger) {
	go func() {
		logger.Info("server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()
}
