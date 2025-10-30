package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// CreateScheduledTask inserts a new scheduled follow-up task.
func (s *Store) CreateScheduledTask(ctx context.Context, customerID, contextEmailID int64, dueAt time.Time) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	now := Now()
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO scheduled_tasks (customer_id, due_at, status, context_email_id, created_at, updated_at)
         VALUES (?, ?, 'scheduled', ?, ?, ?)`,
		customerID,
		dueAt.UTC().Format(time.RFC3339),
		contextEmailID,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("创建自动跟进任务失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取任务 ID 失败: %w", err)
	}
	return id, nil
}

// ListScheduledTasks retrieves tasks filtered by status.
func (s *Store) ListScheduledTasks(ctx context.Context, status string) ([]domain.ScheduledTask, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	query := `SELECT id, customer_id, due_at, status, last_error, context_email_id, generated_email_id, created_at, updated_at FROM scheduled_tasks`
	var rows *sql.Rows
	var err error
	if status != "" {
		query += " WHERE status = ?"
		rows, err = s.DB.QueryContext(ctx, query, status)
	} else {
		rows, err = s.DB.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.ScheduledTask, 0)
	for rows.Next() {
		var task domain.ScheduledTask
		var dueAt string
		var lastError sql.NullString
		var generated sql.NullInt64
		if err := rows.Scan(
			&task.ID,
			&task.CustomerID,
			&dueAt,
			&task.Status,
			&lastError,
			&task.ContextEmailID,
			&generated,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("解析任务失败: %w", err)
		}
		task.DueAt = dueAt
		if lastError.Valid {
			task.LastError = lastError.String
		}
		if generated.Valid {
			task.GeneratedEmailID = generated.Int64
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历任务失败: %w", err)
	}
	return tasks, nil
}

// FetchDueTasks returns tasks that should run now.
func (s *Store) FetchDueTasks(ctx context.Context, limit int) ([]domain.ScheduledTask, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, customer_id, due_at, status, last_error, context_email_id, generated_email_id, created_at, updated_at
         FROM scheduled_tasks
         WHERE status = 'scheduled' AND due_at <= ?
         ORDER BY due_at ASC
         LIMIT ?`,
		time.Now().UTC().Format(time.RFC3339),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询到期任务失败: %w", err)
	}
	defer rows.Close()

	var tasks []domain.ScheduledTask
	for rows.Next() {
		var task domain.ScheduledTask
		var dueAt string
		var lastError sql.NullString
		var generated sql.NullInt64
		if err := rows.Scan(
			&task.ID,
			&task.CustomerID,
			&dueAt,
			&task.Status,
			&lastError,
			&task.ContextEmailID,
			&generated,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("解析任务失败: %w", err)
		}
		task.DueAt = dueAt
		if lastError.Valid {
			task.LastError = lastError.String
		}
		if generated.Valid {
			task.GeneratedEmailID = generated.Int64
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历任务失败: %w", err)
	}
	return tasks, nil
}

// UpdateTaskStatus updates status, generated email id, and last error if provided.
func (s *Store) UpdateTaskStatus(ctx context.Context, taskID int64, status string, generatedEmailID sql.NullInt64, lastError sql.NullString) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	now := Now()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE scheduled_tasks SET status = ?, generated_email_id = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		status,
		generatedEmailID,
		lastError,
		now,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	return nil
}

// MarkTaskRunning sets status to running if still scheduled.
func (s *Store) MarkTaskRunning(ctx context.Context, taskID int64) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	res, err := s.DB.ExecContext(ctx,
		`UPDATE scheduled_tasks SET status = 'running', updated_at = ? WHERE id = ? AND status = 'scheduled'`,
		Now(),
		taskID,
	)
	if err != nil {
		return fmt.Errorf("锁定任务失败: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("任务状态已变更")
	}
	return nil
}

// GetTask returns a scheduled task by id.
func (s *Store) GetTask(ctx context.Context, taskID int64) (*domain.ScheduledTask, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, customer_id, due_at, status, last_error, context_email_id, generated_email_id, created_at, updated_at
         FROM scheduled_tasks WHERE id = ?`,
		taskID,
	)
	var (
		task      domain.ScheduledTask
		dueAt     string
		lastErr   sql.NullString
		generated sql.NullInt64
	)
	if err := row.Scan(
		&task.ID,
		&task.CustomerID,
		&dueAt,
		&task.Status,
		&lastErr,
		&task.ContextEmailID,
		&generated,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("未找到任务")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	task.DueAt = dueAt
	if lastErr.Valid {
		task.LastError = lastErr.String
	}
	if generated.Valid {
		task.GeneratedEmailID = generated.Int64
	}
	return &task, nil
}
