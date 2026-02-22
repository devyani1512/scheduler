package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/devyani1512/scheduler/internal/entity"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

//Priority Queue(min-heap ordered by next_run)

type taskItem struct {
	task    *entity.Task
	nextRun time.Time
	index   int
}

type priorityQueue []*taskItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].nextRun.Before(pq[j].nextRun) }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*taskItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

//Interfaces

type TaskStore interface {
	UpdateTask(ctx context.Context, t *entity.Task) error
}

type TaskExecutor interface {
	Run(ctx context.Context, task *entity.Task) *entity.TaskResult
}

//Scheduler

// Scheduler is the core engine.

type Scheduler struct {
	Incoming chan *entity.Task

	store    TaskStore
	executor TaskExecutor
	logger   *zap.Logger
	parser   cron.Parser

	mu       sync.Mutex
	pq       priorityQueue
	cancelFn context.CancelFunc
}

func New(store TaskStore, exec TaskExecutor, logger *zap.Logger) *Scheduler {
	s := &Scheduler{
		Incoming: make(chan *entity.Task, 256),
		store:    store,
		executor: exec,
		logger:   logger,
		parser:   cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
	heap.Init(&s.pq)
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	childCtx, cancel := context.WithCancel(ctx)
	s.cancelFn = cancel
	go s.run(childCtx)
	s.logger.Info("scheduler started")
}

func (s *Scheduler) Stop() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
}

func (s *Scheduler) Enqueue(task *entity.Task) {
	s.Incoming <- task
}

func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.pq {
		if item.task.ID == id {
			heap.Remove(&s.pq, i)
			s.logger.Info("task removed from scheduler", zap.String("id", id))
			return
		}
	}
}

func (s *Scheduler) NextCronTime(expr string) (time.Time, error) {
	sched, err := s.parser.Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(time.Now().UTC()), nil
}

func (s *Scheduler) ValidateCron(expr string) error {
	_, err := s.parser.Parse(expr)
	return err
}

//  Core loop

func (s *Scheduler) run(ctx context.Context) {
	timer := time.NewTimer(farFuture())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return

		case task := <-s.Incoming:
			s.mu.Lock()
			heap.Push(&s.pq, &taskItem{task: task, nextRun: *task.NextRun})
			s.mu.Unlock()
			s.resetTimer(timer)

		case <-timer.C:
			s.mu.Lock()
			now := time.Now().UTC()
			//spawing all the tasks that are due at the present time
			for s.pq.Len() > 0 && !s.pq[0].nextRun.After(now) {
				item := heap.Pop(&s.pq).(*taskItem)
				go s.execute(ctx, item.task)
			}
			next := farFuture()
			if s.pq.Len() > 0 {
				next = time.Until(s.pq[0].nextRun)
			}
			s.mu.Unlock()
			safeReset(timer, next)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, task *entity.Task) {
	s.logger.Info("executing task",
		zap.String("id", task.ID),
		zap.String("name", task.Name),
		zap.String("type", string(task.Trigger.Type)),
	)

	s.executor.Run(ctx, task)

	switch task.Trigger.Type {
	case entity.TriggerOneOff:
		task.Status = entity.StatusCompleted
		task.NextRun = nil
		task.UpdatedAt = time.Now().UTC()
		if err := s.store.UpdateTask(ctx, task); err != nil {
			s.logger.Error("failed to mark task completed", zap.String("id", task.ID), zap.Error(err))
		}

	case entity.TriggerCron:
		next, err := s.NextCronTime(task.Trigger.Cron)
		if err != nil {
			s.logger.Error("invalid cron expr", zap.String("id", task.ID), zap.Error(err))
			return
		}
		task.NextRun = &next
		task.UpdatedAt = time.Now().UTC()
		if err := s.store.UpdateTask(ctx, task); err != nil {
			s.logger.Error("failed to update cron next_run", zap.String("id", task.ID), zap.Error(err))
		}
		//repeat
		s.Incoming <- task
	}
}

//  Helpers

func (s *Scheduler) resetTimer(timer *time.Timer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pq.Len() == 0 {
		return
	}
	safeReset(timer, time.Until(s.pq[0].nextRun))
}

func safeReset(t *time.Timer, d time.Duration) {
	if d < 0 {
		d = 0
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func farFuture() time.Duration {
	return 24 * time.Hour
}
