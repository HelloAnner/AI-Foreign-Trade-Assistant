package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// CreateAutomationJob enqueues a new automation workflow for a customer.
func (s *Store) CreateAutomationJob(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer id")
	}
	now := Now()
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO automation_jobs (customer_id, status, stage, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?)`,
		customerID,
		domain.AutomationStatusQueued,
		domain.AutomationStagePending,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("创建自动化任务失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("读取自动化任务 ID 失败: %w", err)
	}
	return s.GetAutomationJob(ctx, id)
}

// GetAutomationJob returns full job detail by id.
func (s *Store) GetAutomationJob(ctx context.Context, jobID int64) (*domain.AutomationJob, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, customer_id, status, stage, last_error, started_at, finished_at, created_at, updated_at
         FROM automation_jobs WHERE id = ?`,
		jobID,
	)
	return scanAutomationJob(row)
}

// GetLatestAutomationJob returns the latest automation job for a customer.
func (s *Store) GetLatestAutomationJob(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer id")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, customer_id, status, stage, last_error, started_at, finished_at, created_at, updated_at
         FROM automation_jobs
         WHERE customer_id = ?
         ORDER BY id DESC
         LIMIT 1`,
		customerID,
	)
	job, err := scanAutomationJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return job, nil
}

// GetActiveAutomationJob returns the latest queued or running job for a customer.
func (s *Store) GetActiveAutomationJob(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer id")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, customer_id, status, stage, last_error, started_at, finished_at, created_at, updated_at
         FROM automation_jobs
         WHERE customer_id = ? AND status IN (?, ?)
         ORDER BY id DESC
         LIMIT 1`,
		customerID,
		domain.AutomationStatusQueued,
		domain.AutomationStatusRunning,
	)
	job, err := scanAutomationJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return job, nil
}

// ClaimNextAutomationJob transitions the next queued job to running.
func (s *Store) ClaimNextAutomationJob(ctx context.Context) (*domain.AutomationJob, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				fmt.Printf("automation tx rollback error: %v\n", rbErr)
			}
		}
	}()

	row := tx.QueryRowContext(ctx,
		`SELECT id FROM automation_jobs
         WHERE status = ?
         ORDER BY id ASC
         LIMIT 1`,
		domain.AutomationStatusQueued,
	)
	var jobID int64
	if err := row.Scan(&jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	now := Now()
	res, err := tx.ExecContext(ctx,
		`UPDATE automation_jobs
         SET status = ?, stage = CASE WHEN stage = ? THEN ? ELSE stage END,
             started_at = COALESCE(started_at, ?), updated_at = ?
         WHERE id = ? AND status = ?`,
		domain.AutomationStatusRunning,
		domain.AutomationStagePending,
		domain.AutomationStageGrading,
		now,
		now,
		jobID,
		domain.AutomationStatusQueued,
	)
	if err != nil {
		return nil, fmt.Errorf("claim automation job: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return s.GetAutomationJob(ctx, jobID)
}

// UpdateAutomationJobStage updates the current stage of the job.
func (s *Store) UpdateAutomationJobStage(ctx context.Context, jobID int64, stage string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = domain.AutomationStagePending
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE automation_jobs
         SET stage = ?, updated_at = ?
         WHERE id = ?`,
		stage,
		Now(),
		jobID,
	)
	if err != nil {
		return fmt.Errorf("更新自动化任务阶段失败: %w", err)
	}
	return nil
}

// MarkAutomationJobCompleted marks a job as completed with the final stage.
func (s *Store) MarkAutomationJobCompleted(ctx context.Context, jobID int64, stage string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = domain.AutomationStageCompleted
	}
	now := Now()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE automation_jobs
         SET status = ?, stage = ?, finished_at = COALESCE(finished_at, ?), updated_at = ?
         WHERE id = ?`,
		domain.AutomationStatusCompleted,
		stage,
		now,
		now,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("标记自动化任务完成失败: %w", err)
	}
	return nil
}

// MarkAutomationJobStopped sets the job to completed with stopped stage.
func (s *Store) MarkAutomationJobStopped(ctx context.Context, jobID int64, reason string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	now := Now()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE automation_jobs
         SET status = ?, stage = ?, last_error = ?, finished_at = COALESCE(finished_at, ?), updated_at = ?
         WHERE id = ?`,
		domain.AutomationStatusCompleted,
		domain.AutomationStageStopped,
		strings.TrimSpace(reason),
		now,
		now,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("标记自动化任务结束失败: %w", err)
	}
	return nil
}

// MarkAutomationJobFailed records job failure with error detail.
func (s *Store) MarkAutomationJobFailed(ctx context.Context, jobID int64, stage string, errMsg string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = domain.AutomationStagePending
	}
	now := Now()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE automation_jobs
         SET status = ?, stage = ?, last_error = ?, finished_at = COALESCE(finished_at, ?), updated_at = ?
         WHERE id = ?`,
		domain.AutomationStatusFailed,
		stage,
		strings.TrimSpace(errMsg),
		now,
		now,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("标记自动化任务失败: %w", err)
	}
	return nil
}

func scanAutomationJob(row *sql.Row) (*domain.AutomationJob, error) {
	var job domain.AutomationJob
	var (
		lastError  sql.NullString
		startedAt  sql.NullString
		finishedAt sql.NullString
	)
	if err := row.Scan(
		&job.ID,
		&job.CustomerID,
		&job.Status,
		&job.Stage,
		&lastError,
		&startedAt,
		&finishedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastError.Valid {
		job.LastError = lastError.String
	}
	if startedAt.Valid {
		job.StartedAt = startedAt.String
	}
	if finishedAt.Valid {
		job.FinishedAt = finishedAt.String
	}
	return &job, nil
}

// DeleteAutomationJob removes a job permanently once processed.
func (s *Store) DeleteAutomationJob(ctx context.Context, jobID int64) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if jobID <= 0 {
		return nil
	}
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM automation_jobs WHERE id = ?`, jobID); err != nil {
		return fmt.Errorf("删除自动化任务失败: %w", err)
	}
	return nil
}
