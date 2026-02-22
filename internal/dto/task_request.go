package dto

import "github.com/devyani1512/scheduler/internal/entity"

type CreateTaskRequest struct {
	Name    string         `json:"name" binding:"required"`
	Trigger entity.Trigger `json:"trigger" binding:"required"`
	Action  entity.Action  `json:"action" binding:"required"`
}

type UpdateTaskRequest struct {
	Name    *string         `json:"name"`
	Trigger *entity.Trigger `json:"trigger"`
	Action  *entity.Action  `json:"action"`
}
