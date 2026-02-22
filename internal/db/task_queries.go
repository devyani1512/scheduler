package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devyani1512/scheduler/internal/dto"
	"github.com/devyani1512/scheduler/internal/entity"
)

func (d *DB) CreateTask(ctx context.Context, t *entity.Task) error {
	triggerJSON, err := json.Marshal(t.Trigger)
	if err != nil {
		return err
	}
	actionJSON, err := json.Marshal(t.Action)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO tasks (id, name, trigger, action, status, created_at, updated_at, next_run)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	_, err = d.Conn.ExecContext(ctx, query,
		t.ID, t.Name, triggerJSON, actionJSON,
		t.Status, t.CreatedAt, t.UpdatedAt, t.NextRun,
	)
	return err
}

func (d *DB) GetTask(ctx context.Context, id string) (*entity.Task, error) {
	query := `
		SELECT id, name, trigger, action, status, created_at, updated_at, next_run
		FROM tasks WHERE id = $1`

	row := d.Conn.QueryRowContext(ctx, query, id)
	return scanTask(row)
}

func (d *DB) ListTasks(ctx context.Context, pq dto.PaginationQuery) ([]*entity.Task, int, error) {
	offset := (pq.Page - 1) * pq.Limit
	args := []interface{}{}
	where := "WHERE 1=1"
	argIdx := 1

	if pq.Status != "" {
		where += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, pq.Status)
		argIdx++
	}

	var total int
	if err := d.Conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, pq.Limit, offset)
	query := fmt.Sprintf(`
		SELECT id, name, trigger, action, status, created_at, updated_at, next_run
		FROM tasks %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := d.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*entity.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

func (d *DB) UpdateTask(ctx context.Context, t *entity.Task) error {
	triggerJSON, err := json.Marshal(t.Trigger)
	if err != nil {
		return err
	}
	actionJSON, err := json.Marshal(t.Action)
	if err != nil {
		return err
	}

	query := `
		UPDATE tasks
		SET name=$1, trigger=$2, action=$3, status=$4, updated_at=$5, next_run=$6
		WHERE id=$7`

	_, err = d.Conn.ExecContext(ctx, query,
		t.Name, triggerJSON, actionJSON,
		t.Status, t.UpdatedAt, t.NextRun, t.ID,
	)
	return err
}

func (d *DB) CancelTask(ctx context.Context, id string) error {
	_, err := d.Conn.ExecContext(ctx,
		`UPDATE tasks SET status=$1, updated_at=$2 WHERE id=$3`,
		entity.StatusCancelled, time.Now().UTC(), id,
	)
	return err
}

func (d *DB) GetScheduledTasks(ctx context.Context) ([]*entity.Task, error) {
	query := `
		SELECT id, name, trigger, action, status, created_at, updated_at, next_run
		FROM tasks
		WHERE status='scheduled' AND next_run IS NOT NULL
		ORDER BY next_run ASC`

	rows, err := d.Conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*entity.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

//  scan helper

func scanTask(s scanner) (*entity.Task, error) {
	t := &entity.Task{}
	var triggerJSON, actionJSON []byte

	err := s.Scan(
		&t.ID, &t.Name, &triggerJSON, &actionJSON,
		&t.Status, &t.CreatedAt, &t.UpdatedAt, &t.NextRun,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, err
	}

	if err := json.Unmarshal(triggerJSON, &t.Trigger); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(actionJSON, &t.Action); err != nil {
		return nil, err
	}
	return t, nil
}
