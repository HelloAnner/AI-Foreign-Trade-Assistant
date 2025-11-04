package store

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// CreateTodoTask enqueues a new todo query for background processing.
func (s *Store) CreateTodoTask(ctx context.Context, query string) (*domain.TodoTask, error) {
    if s == nil || s.DB == nil {
        return nil, fmt.Errorf("store not initialized")
    }
    if len(query) == 0 {
        return nil, fmt.Errorf("query is empty")
    }
    now := Now()
    res, err := s.DB.ExecContext(ctx, `
        INSERT INTO todo_tasks (query, status, created_at, updated_at) VALUES (?, 'queued', ?, ?)
    `, query, now, now)
    if err != nil {
        return nil, fmt.Errorf("创建待处理任务失败: %w", err)
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("读取任务 ID 失败: %w", err)
    }
    return s.GetTodoTask(ctx, id)
}

// GetTodoTask returns a todo task by id.
func (s *Store) GetTodoTask(ctx context.Context, id int64) (*domain.TodoTask, error) {
    if s == nil || s.DB == nil {
        return nil, fmt.Errorf("store not initialized")
    }
    row := s.DB.QueryRowContext(ctx, `
        SELECT id, query, status, last_error, customer_id, started_at, finished_at, created_at, updated_at
        FROM todo_tasks WHERE id = ?
    `, id)
    var t domain.TodoTask
    var lastErr sql.NullString
    var customerID sql.NullInt64
    var startedAt sql.NullString
    var finishedAt sql.NullString
    if err := row.Scan(&t.ID, &t.Query, &t.Status, &lastErr, &customerID, &startedAt, &finishedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
        return nil, err
    }
    if lastErr.Valid { t.LastError = lastErr.String }
    if customerID.Valid { t.CustomerID = customerID.Int64 }
    if startedAt.Valid { t.StartedAt = startedAt.String }
    if finishedAt.Valid { t.FinishedAt = finishedAt.String }
    return &t, nil
}

// ClaimNextTodo transitions the next queued todo to running.
func (s *Store) ClaimNextTodo(ctx context.Context) (*domain.TodoTask, error) {
    if s == nil || s.DB == nil {
        return nil, fmt.Errorf("store not initialized")
    }
    tx, err := s.DB.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer func() {
        if tx != nil {
            _ = tx.Rollback()
        }
    }()
    row := tx.QueryRowContext(ctx, `SELECT id, query FROM todo_tasks WHERE status = 'queued' ORDER BY id ASC LIMIT 1`)
    var id int64
    var query string
    if err := row.Scan(&id, &query); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    now := Now()
    res, err := tx.ExecContext(ctx, `
        UPDATE todo_tasks SET status = 'running', started_at = COALESCE(started_at, ?), updated_at = ?
        WHERE id = ? AND status = 'queued'
    `, now, now, id)
    if err != nil {
        return nil, err
    }
    if affected, _ := res.RowsAffected(); affected == 0 {
        return nil, nil
    }
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    tx = nil
    return s.GetTodoTask(ctx, id)
}

func (s *Store) MarkTodoCompleted(ctx context.Context, id int64, customerID int64) error {
    if s == nil || s.DB == nil {
        return fmt.Errorf("store not initialized")
    }
    now := Now()
    _, err := s.DB.ExecContext(ctx, `
        UPDATE todo_tasks SET status = 'completed', customer_id = ?, finished_at = COALESCE(finished_at, ?), updated_at = ? WHERE id = ?
    `, customerID, now, now, id)
    if err != nil {
        return fmt.Errorf("更新待处理任务失败: %w", err)
    }
    return nil
}

func (s *Store) MarkTodoFailed(ctx context.Context, id int64, errMsg string) error {
    if s == nil || s.DB == nil {
        return fmt.Errorf("store not initialized")
    }
    now := Now()
    _, err := s.DB.ExecContext(ctx, `
        UPDATE todo_tasks SET status = 'failed', last_error = ?, finished_at = COALESCE(finished_at, ?), updated_at = ? WHERE id = ?
    `, errMsg, now, now, id)
    if err != nil {
        return fmt.Errorf("标记任务失败: %w", err)
    }
    return nil
}

