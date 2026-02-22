package main

import (
	"context"
	"time"

	"github.com/devyani1512/scheduler/internal/db"
	"github.com/devyani1512/scheduler/internal/entity"
	"github.com/devyani1512/scheduler/internal/scheduler"
	"go.uber.org/zap"
)

func recoverTasks(ctx context.Context, store *db.DB, sched *scheduler.Scheduler, logger *zap.Logger) {
	tasks, err := store.GetScheduledTasks(ctx)
	if err != nil {
		logger.Error("failed to recover tasks from db", zap.Error(err))
		return
	}

	now := time.Now().UTC()
	count := 0
	for _, t := range tasks {
		if t.NextRun == nil {
			continue
		}
		// one-off tasks that already passed their schedule_at → enqueue immediately
		if t.Trigger.Type == entity.TriggerOneOff && t.NextRun.Before(now) {
			t2 := t
			go func() {
				time.Sleep(100 * time.Millisecond)
				sched.Enqueue(t2)
			}()
		} else {
			sched.Enqueue(t)
		}
		count++
	}

	logger.Info("recovered tasks from database", zap.Int("count", count))
}
