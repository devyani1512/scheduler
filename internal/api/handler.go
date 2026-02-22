package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/devyani1512/scheduler/internal/dto"
	"github.com/devyani1512/scheduler/internal/entity"
	"github.com/devyani1512/scheduler/internal/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Store is the interface the handler needs from the DB layer
type Store interface {
	CreateTask(ctx context.Context, t *entity.Task) error
	GetTask(ctx context.Context, id string) (*entity.Task, error)
	ListTasks(ctx context.Context, pq dto.PaginationQuery) ([]*entity.Task, int, error)
	UpdateTask(ctx context.Context, t *entity.Task) error
	CancelTask(ctx context.Context, id string) error
	ListTaskResults(ctx context.Context, taskID string, pq dto.PaginationQuery) ([]*entity.TaskResult, int, error)
	ListAllResults(ctx context.Context, taskID string, pq dto.PaginationQuery) ([]*entity.TaskResult, int, error)
}

type Handler struct {
	store     Store
	scheduler *scheduler.Scheduler
	logger    *zap.Logger
}

func NewHandler(store Store, sched *scheduler.Scheduler, logger *zap.Logger) *Handler {
	return &Handler{store: store, scheduler: sched, logger: logger}
}

func (h *Handler) Register(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", h.Health)
		v1.POST("/tasks", h.CreateTask)
		v1.GET("/tasks", h.ListTasks)
		v1.GET("/tasks/:id", h.GetTask)
		v1.PUT("/tasks/:id", h.UpdateTask)
		v1.DELETE("/tasks/:id", h.CancelTask)
		v1.GET("/tasks/:id/results", h.ListTaskResults)
		v1.GET("/results", h.ListAllResults)
	}
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
}

// POST /tasks
func (h *Handler) CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC()
	nextRun, err := h.resolveNextRun(req.Trigger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Action.URL == "" || req.Action.Method == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action.url and action.method are required"})
		return
	}

	task := &entity.Task{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Trigger:   req.Trigger,
		Action:    req.Action,
		Status:    entity.StatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
		NextRun:   nextRun,
	}

	if err := h.store.CreateTask(c.Request.Context(), task); err != nil {
		h.logger.Error("failed to create task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	h.scheduler.Enqueue(task)
	c.JSON(http.StatusCreated, task)
}

// GET /tasks
func (h *Handler) ListTasks(c *gin.Context) {
	pq := parsePagination(c)

	tasks, total, err := h.store.ListTasks(c.Request.Context(), pq)
	if err != nil {
		h.logger.Error("failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}
	if tasks == nil {
		tasks = []*entity.Task{}
	}

	c.JSON(http.StatusOK, paginate(tasks, total, pq))
}

// GET /tasks/:id
func (h *Handler) GetTask(c *gin.Context) {
	task, err := h.store.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// PUT /tasks/:id
func (h *Handler) UpdateTask(c *gin.Context) {
	existing, err := h.store.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if existing.Status == entity.StatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot update a cancelled task"})
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Action != nil {
		existing.Action = *req.Action
	}
	if req.Trigger != nil {
		existing.Trigger = *req.Trigger
		nextRun, err := h.resolveNextRun(existing.Trigger)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		existing.NextRun = nextRun
		h.scheduler.Remove(existing.ID)
	}

	existing.UpdatedAt = time.Now().UTC()
	if err := h.store.UpdateTask(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}

	if req.Trigger != nil {
		h.scheduler.Enqueue(existing)
	}

	c.JSON(http.StatusOK, existing)
}

// DELETE /tasks/:id
func (h *Handler) CancelTask(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.store.GetTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	h.scheduler.Remove(id)
	if err := h.store.CancelTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

// GET /tasks/:id/results
func (h *Handler) ListTaskResults(c *gin.Context) {
	pq := parsePagination(c)

	results, total, err := h.store.ListTaskResults(c.Request.Context(), c.Param("id"), pq)
	if err != nil {
		h.logger.Error("failed to list results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list results"})
		return
	}
	if results == nil {
		results = []*entity.TaskResult{}
	}

	c.JSON(http.StatusOK, paginate(results, total, pq))
}

// GET /results
func (h *Handler) ListAllResults(c *gin.Context) {
	pq := parsePagination(c)
	taskID := c.Query("task_id")

	results, total, err := h.store.ListAllResults(c.Request.Context(), taskID, pq)
	if err != nil {
		h.logger.Error("failed to list all results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list results"})
		return
	}
	if results == nil {
		results = []*entity.TaskResult{}
	}

	c.JSON(http.StatusOK, paginate(results, total, pq))
}

//  Helpers 

func (h *Handler) resolveNextRun(trigger entity.Trigger) (*time.Time, error) {
	now := time.Now().UTC()
	switch trigger.Type {
	case entity.TriggerOneOff:
		if trigger.ScheduledAt == nil {
			return nil, fmt.Errorf("trigger.schedule_at is required for one-off tasks")
		}
		t := trigger.ScheduledAt.UTC()
		if t.Before(now) {
			return nil, fmt.Errorf("trigger.schedule_at must be in the future")
		}
		return &t, nil

	case entity.TriggerCron:
		if trigger.Cron == "" {
			return nil, fmt.Errorf("trigger.cron is required for cron tasks")
		}
		next, err := h.scheduler.NextCronTime(trigger.Cron)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		return &next, nil

	default:
		return nil, fmt.Errorf("trigger.type must be 'one-off' or 'cron'")
	}
}

func parsePagination(c *gin.Context) dto.PaginationQuery {
	var pq dto.PaginationQuery
	c.ShouldBindQuery(&pq)
	if pq.Page < 1 {
		pq.Page = 1
	}
	if pq.Limit < 1 || pq.Limit > 100 {
		pq.Limit = 20
	}
	return pq
}

func paginate(data interface{}, total int, pq dto.PaginationQuery) dto.PaginatedResponse {
	return dto.PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       pq.Page,
		Limit:      pq.Limit,
		TotalPages: int(math.Ceil(float64(total) / float64(pq.Limit))),
	}
}
