package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/devyani1512/scheduler/internal/entity"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ResultStore interface {
	CreateTaskResult(ctx context.Context, r *entity.TaskResult) error
}

type Executor struct {
	store  ResultStore
	client *http.Client
	logger *zap.Logger
}

func New(store ResultStore, logger *zap.Logger) *Executor {
	return &Executor{
		store:  store,
		logger: logger,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *Executor) Run(ctx context.Context, task *entity.Task) *entity.TaskResult {
	runAt := time.Now().UTC()
	start := time.Now()

	statusCode, respHeaders, respBody, execErr := e.doHTTP(ctx, task.Action)

	result := &entity.TaskResult{
		ID:           uuid.New().String(),
		TaskID:       task.ID,
		RunAt:        runAt,
		StatusCode:   statusCode,
		Success:      execErr == nil && statusCode >= 200 && statusCode < 300,
		ResponseBody: respBody,
		DurationMs:   time.Since(start).Milliseconds(),
		CreatedAt:    time.Now().UTC(),
	}

	if headersJSON, err := json.Marshal(respHeaders); err == nil {
		result.ResponseHeaders = headersJSON
	} else {
		result.ResponseHeaders = []byte("{}")
	}

	if execErr != nil {
		msg := execErr.Error()
		result.ErrorMessage = &msg
	}

	if err := e.store.CreateTaskResult(ctx, result); err != nil {
		e.logger.Error("failed to persist task result",
			zap.String("task_id", task.ID),
			zap.Error(err),
		)
	}

	e.logger.Info("task executed",
		zap.String("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Int("status_code", statusCode),
		zap.Bool("success", result.Success),
		zap.Int64("duration_ms", result.DurationMs),
	)

	return result
}

func (e *Executor) doHTTP(ctx context.Context, action entity.Action) (int, map[string]string, string, error) {
	var bodyReader io.Reader
	//converting json body given by users to a reader - bytes the http library can read from
	if len(action.Payload) > 0 && string(action.Payload) != "null" {
		bodyReader = strings.NewReader(string(action.Payload))
	}
	//passing context so that if it shut downs mid-request, this call gets cancelled
	req, err := http.NewRequestWithContext(ctx, action.Method, action.URL, bodyReader)
	if err != nil {
		return 0, nil, "", fmt.Errorf("creating request: %w", err)
	}

	for k, v := range action.Headers {
		req.Header.Set(k, v)
	}
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	//sending request
	resp, err := e.client.Do(req)
	if err != nil {
		return 0, nil, "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB cap

	headers := make(map[string]string)
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}

	return resp.StatusCode, headers, string(bodyBytes), nil
}
