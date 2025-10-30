package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// CreateCustomer inserts a new customer with associated contacts.
func (s *Store) CreateCustomer(ctx context.Context, req *domain.CreateCompanyRequest) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	if req == nil {
		return 0, fmt.Errorf("payload is nil")
	}
	if strings.TrimSpace(req.Name) == "" {
		return 0, fmt.Errorf("客户名称不能为空")
	}

	source := req.SourceJSON
	if len(source) == 0 {
		source = []byte("{}")
	}

	now := Now()
	var customerID int64
	err := s.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO customers (name, website, country, grade, grade_reason, summary, source_json, created_at, updated_at)
             VALUES (?, ?, ?, 'unknown', '', ?, ?, ?, ?)`,
			req.Name,
			req.Website,
			req.Country,
			req.Summary,
			string(source),
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("插入客户失败: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("读取客户 ID 失败: %w", err)
		}
		customerID = id
		if err := insertContactsTx(ctx, tx, customerID, req.Contacts); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return customerID, nil
}

// ReplaceContacts rewrites the contacts for a customer.
func (s *Store) ReplaceContacts(ctx context.Context, customerID int64, contacts []domain.Contact) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return fmt.Errorf("invalid customer id")
	}
	return s.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM contacts WHERE customer_id = ?`, customerID); err != nil {
			return fmt.Errorf("清理旧联系人失败: %w", err)
		}
		return insertContactsTx(ctx, tx, customerID, contacts)
	})
}

// GetCustomer fetches a customer by id.
func (s *Store) GetCustomer(ctx context.Context, id int64) (*domain.Customer, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.DB.QueryRowContext(ctx, `SELECT id, name, website, country, grade, grade_reason, summary, source_json, created_at, updated_at FROM customers WHERE id = ?`, id)
	var (
		customer domain.Customer
		source   sql.NullString
	)
	if err := row.Scan(
		&customer.ID,
		&customer.Name,
		&customer.Website,
		&customer.Country,
		&customer.Grade,
		&customer.GradeReason,
		&customer.Summary,
		&source,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("未找到客户记录")
		}
		return nil, fmt.Errorf("查询客户失败: %w", err)
	}
	if source.Valid {
		customer.SourceJSON = json.RawMessage(source.String)
	}
	return &customer, nil
}

// ListContacts returns contacts of a customer.
func (s *Store) ListContacts(ctx context.Context, customerID int64) ([]domain.Contact, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT name, title, email, phone, source, is_key FROM contacts WHERE customer_id = ? ORDER BY is_key DESC, id ASC`,
		customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询联系人失败: %w", err)
	}
	defer rows.Close()

	contacts := make([]domain.Contact, 0)
	for rows.Next() {
		var c domain.Contact
		var isKey int
		if err := rows.Scan(&c.Name, &c.Title, &c.Email, &c.Phone, &c.Source, &isKey); err != nil {
			return nil, fmt.Errorf("解析联系人失败: %w", err)
		}
		c.IsKey = isKey == 1
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历联系人失败: %w", err)
	}
	return contacts, nil
}

// UpdateCustomerGrade writes the final grade and optional reason.
func (s *Store) UpdateCustomerGrade(ctx context.Context, id int64, grade, reason string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if id <= 0 {
		return fmt.Errorf("invalid customer id")
	}
	now := Now()
	res, err := s.DB.ExecContext(ctx, `UPDATE customers SET grade = ?, grade_reason = ?, updated_at = ? WHERE id = ?`, grade, reason, now, id)
	if err != nil {
		return fmt.Errorf("更新客户评级失败: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("客户不存在或未更新")
	}
	return nil
}

// SaveAnalysis upserts the analysis record for a customer.
func (s *Store) SaveAnalysis(ctx context.Context, customerID int64, content domain.AnalysisContent) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	now := Now()
	var analysisID int64
	err := s.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM analyses WHERE customer_id = ?`, customerID); err != nil {
			return fmt.Errorf("清理旧分析失败: %w", err)
		}
		res, err := tx.ExecContext(ctx,
			`INSERT INTO analyses (customer_id, core_business, pain_points, my_entry_points, full_report, created_at, updated_at)
             VALUES (?, ?, ?, ?, ?, ?, ?)`,
			customerID,
			content.CoreBusiness,
			content.PainPoints,
			content.MyEntryPoints,
			content.FullReport,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("写入分析报告失败: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("读取分析 ID 失败: %w", err)
		}
		analysisID = id
		return nil
	})
	if err != nil {
		return 0, err
	}
	return analysisID, nil
}

// GetLatestAnalysis returns the most recent analysis for a customer if exists.
func (s *Store) GetLatestAnalysis(ctx context.Context, customerID int64) (*domain.AnalysisResponse, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, core_business, pain_points, my_entry_points, full_report FROM analyses WHERE customer_id = ? ORDER BY id DESC LIMIT 1`,
		customerID,
	)
	var resp domain.AnalysisResponse
	if err := row.Scan(&resp.AnalysisID, &resp.CoreBusiness, &resp.PainPoints, &resp.MyEntryPoints, &resp.FullReport); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("尚未生成切入点分析")
		}
		return nil, fmt.Errorf("查询分析报告失败: %w", err)
	}
	return &resp, nil
}

// InsertEmailDraft creates a draft email record.
func (s *Store) InsertEmailDraft(ctx context.Context, customerID int64, emailType string, draft domain.EmailDraft, status string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	if emailType == "" {
		emailType = "initial"
	}
	if status == "" {
		status = "draft"
	}
	now := Now()
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO emails (customer_id, type, subject, body, status, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		customerID,
		emailType,
		draft.Subject,
		draft.Body,
		status,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("写入邮件草稿失败: %w", err)
	}
	emailID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取邮件 ID 失败: %w", err)
	}
	return emailID, nil
}

// GetEmail fetches an email record by id.
func (s *Store) GetEmail(ctx context.Context, emailID int64) (*domain.EmailRecord, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, customer_id, type, subject, body, status, sent_at, created_at, updated_at FROM emails WHERE id = ?`,
		emailID,
	)
	var (
		record domain.EmailRecord
		sent   sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.CustomerID,
		&record.Type,
		&record.Subject,
		&record.Body,
		&record.Status,
		&sent,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("未找到邮件记录")
		}
		return nil, fmt.Errorf("查询邮件失败: %w", err)
	}
	if sent.Valid {
		record.SentAt = sent.String
	}
	return &record, nil
}

// SaveInitialFollowup persists the initial follow-up record referencing the draft email.
func (s *Store) SaveInitialFollowup(ctx context.Context, customerID int64, emailID int64, notes string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	now := Now()
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO followups (customer_id, initial_email_id, notes, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?)`,
		customerID,
		emailID,
		notes,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("写入首次跟进记录失败: %w", err)
	}
	followupID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取跟进记录 ID 失败: %w", err)
	}
	return followupID, nil
}

// insertContactsTx inserts contacts within an existing transaction.
func insertContactsTx(ctx context.Context, tx *sql.Tx, customerID int64, contacts []domain.Contact) error {
	if len(contacts) == 0 {
		return nil
	}
	now := Now()
	for _, c := range contacts {
		if strings.TrimSpace(c.Email) == "" && strings.TrimSpace(c.Name) == "" {
			continue
		}
		isKey := 0
		if c.IsKey {
			isKey = 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO contacts (customer_id, name, title, email, phone, source, is_key, created_at, updated_at)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			customerID,
			c.Name,
			c.Title,
			c.Email,
			c.Phone,
			c.Source,
			isKey,
			now,
			now,
		); err != nil {
			return fmt.Errorf("写入联系人失败: %w", err)
		}
	}
	return nil
}
