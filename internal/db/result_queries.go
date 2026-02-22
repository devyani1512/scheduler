package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/devyani1512/scheduler/internal/dto"
	"github.com/devyani1512/scheduler/internal/entity"
)

func (d *DB) CreateTaskResult(ctx context.Context, r *entity.TaskResult) error {
	headersJSON, err := json.Marshal(r.ResponseHeaders)
	if err != nil {
		headersJSON = []byte("{}")
	}

	query := `
		INSERT INTO task_results
			(id, task_id, run_at, status_code, success, response_headers, response_body, error_message, duration_ms, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

	_, err = d.Conn.ExecContext(ctx, query,
		r.ID, r.TaskID, r.RunAt, r.StatusCode, r.Success,
		headersJSON, r.ResponseBody, r.ErrorMessage, r.DurationMs, r.CreatedAt,
	)
	return err
}

func (d *DB) ListTaskResults(ctx context.Context, taskID string, pq dto.PaginationQuery) ([]*entity.TaskResult, int, error) {
	offset := (pq.Page - 1) * pq.Limit
	args := []interface{}{taskID}
	argIdx := 2
	where := "WHERE task_id=$1" 

	if pq.Status == "success" {
		where += fmt.Sprintf(" AND success=$%d", argIdx) 
		args = append(args, true)
		argIdx++
	} else if pq.Status == "failed" {
		where += fmt.Sprintf(" AND success=$%d", argIdx) 
		args = append(args, false)
		argIdx++
	}

	var total int
	if err := d.Conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_results "+where, args...).Scan(&total); err != nil {
		return nil, 0, err 
	}

	args = append(args, pq.Limit, offset)
	query := fmt.Sprintf(`
		SELECT id, task_id, run_at, status_code, success, response_headers,
		       response_body, error_message, duration_ms, created_at
		FROM task_results %s ORDER BY run_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := d.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*entity.TaskResult
	for rows.Next() {
		r, err := scanResult(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, total, rows.Err()
}


func (d *DB) ListAllResults(ctx context.Context, taskID string, pq dto.PaginationQuery) ([]*entity.TaskResult, int, error) {
	offset := (pq.Page - 1) * pq.Limit
	args := []interface{}{}
	argIdx := 1
	where := "WHERE 1=1"

	if taskID != "" {
		where += fmt.Sprintf(" AND task_id=$%d", argIdx)
		args = append(args, taskID)
		argIdx++
	}

	var total int
	if err := d.Conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_results "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, pq.Limit, offset)
	query := fmt.Sprintf(`
		SELECT id, task_id, run_at, status_code, success, response_headers,
		       response_body, error_message, duration_ms, created_at
		FROM task_results %s ORDER BY run_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := d.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*entity.TaskResult
	for rows.Next() {
		r, err := scanResult(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, total, rows.Err()
}

func scanResult(s scanner) (*entity.TaskResult, error) {
	r := &entity.TaskResult{}
	var headersJSON []byte

	err := s.Scan(
		&r.ID, &r.TaskID, &r.RunAt, &r.StatusCode, &r.Success,
		&headersJSON, &r.ResponseBody, &r.ErrorMessage, &r.DurationMs, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.ResponseHeaders = headersJSON
	return r, nil
}
