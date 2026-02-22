package entity

import (
	"encoding/json"
	"time"
)

type TriggerType string
type TaskStatus string

const (
	TriggerOneOff TriggerType = "one-off"
	TriggerCron   TriggerType = "cron"

	StatusScheduled TaskStatus = "scheduled"
	StatusCancelled TaskStatus = "cancelled"
	StatusCompleted TaskStatus = "completed"
)

type Trigger struct {
	Type        TriggerType `json:"type"`
	ScheduledAt *time.Time  `json:"schedule_at,omitempty"`
	Cron        string      `json:"cron,omitempty"`
}
type Action struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

type Task struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Trigger   Trigger    `json:"trigger"`
	Action    Action     `json:"action"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	NextRun   *time.Time `json:"next_run,omitempty"`
}
