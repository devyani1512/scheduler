package entity

import (
	"encoding/json"
	"time"
)

type TaskResult struct {
	ID              string          `json:"id"`
	TaskID          string          `json:"task_id"`
	RunAt           time.Time       `json:"run_at"`
	StatusCode      int             `json:"status_code"`
	Success         bool            `json:"success"`
	ResponseHeaders json.RawMessage `json:"response_headers"`
	ResponseBody    string          `json:"response_body"`
	ErrorMessage    *string         `json:"error_message"`
	DurationMs      int64           `json:"duration_ms"`
	CreatedAt       time.Time       `json:"created_at"`
}
