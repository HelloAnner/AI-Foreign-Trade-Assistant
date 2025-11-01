package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// CustomerListFilter defines filters for listing customers.
type CustomerListFilter struct {
	Grade   string
	Country string
	Search  string
	Sort    string
	Status  string
	Limit   int
	Offset  int
}

// ListCustomers returns lightweight summaries for the management table.
func (s *Store) ListCustomers(ctx context.Context, filter CustomerListFilter) ([]domain.CustomerSummary, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}

	baseQuery := `
        SELECT
            c.id,
            c.name,
            c.country,
            COALESCE(NULLIF(upper(c.grade), ''), 'UNKNOWN') AS grade,
            c.updated_at,
            COALESCE(MAX(f.updated_at), '') AS last_followup_at,
            COALESCE(MIN(CASE WHEN st.status = 'scheduled' THEN st.due_at END), '') AS next_followup_at
        FROM customers c
        LEFT JOIN followups f ON f.customer_id = c.id
        LEFT JOIN scheduled_tasks st ON st.customer_id = c.id
    `

	conditions := make([]string, 0)
	args := make([]any, 0)

	if grade := strings.TrimSpace(filter.Grade); grade != "" {
		conditions = append(conditions, "upper(c.grade) = ?")
		args = append(args, strings.ToUpper(grade))
	}
	if country := strings.TrimSpace(filter.Country); country != "" {
		conditions = append(conditions, "c.country = ?")
		args = append(args, country)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		conditions = append(conditions, "(lower(c.name) LIKE ? OR lower(c.website) LIKE ?)")
		args = append(args, like, like)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " GROUP BY c.id"

	orderBy := " ORDER BY c.updated_at DESC"
	switch filter.Sort {
	case "name_asc":
		orderBy = " ORDER BY c.name COLLATE NOCASE ASC"
	case "name_desc":
		orderBy = " ORDER BY c.name COLLATE NOCASE DESC"
	case "grade_desc":
		orderBy = " ORDER BY grade DESC, c.updated_at DESC"
	case "grade_asc":
		orderBy = " ORDER BY grade ASC, c.updated_at DESC"
	case "last_followup_asc":
		orderBy = " ORDER BY (last_followup_at = ''), last_followup_at ASC"
	case "last_followup_desc":
		orderBy = " ORDER BY (last_followup_at = ''), last_followup_at DESC"
	case "next_followup_asc":
		orderBy = " ORDER BY (next_followup_at = ''), next_followup_at ASC"
	case "next_followup_desc":
		orderBy = " ORDER BY (next_followup_at = ''), next_followup_at DESC"
	}
	// SQLite does not understand NULLS LAST, so emulate by coalescing.
	query := baseQuery + orderBy
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询客户列表失败: %w", err)
	}
	defer rows.Close()

	summaries := make([]domain.CustomerSummary, 0)
	now := time.Now().UTC()

	for rows.Next() {
		var (
			summary domain.CustomerSummary
			grade   sql.NullString
			updated sql.NullString
			last    sql.NullString
			next    sql.NullString
		)
		if err := rows.Scan(&summary.ID, &summary.Name, &summary.Country, &grade, &updated, &last, &next); err != nil {
			return nil, fmt.Errorf("解析客户列表失败: %w", err)
		}

		summary.Grade = "UNKNOWN"
		if grade.Valid && strings.TrimSpace(grade.String) != "" {
			summary.Grade = strings.ToUpper(strings.TrimSpace(grade.String))
		}

		if updated.Valid {
			summary.UpdatedAt = updated.String
		}
		if last.Valid {
			summary.LastFollowupAt = last.String
		}
		if next.Valid {
			summary.NextFollowupAt = next.String
		}

		summary.Status = deriveCustomerStatus(summary, now)
		if filter.Status != "" && filter.Status != summary.Status {
			continue
		}

		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历客户列表失败: %w", err)
	}
	return summaries, nil
}

func deriveCustomerStatus(summary domain.CustomerSummary, now time.Time) string {
	grade := strings.ToUpper(strings.TrimSpace(summary.Grade))
	if grade == "S" {
		return "won"
	}
	if summary.NextFollowupAt != "" {
		if due, err := time.Parse(time.RFC3339, summary.NextFollowupAt); err == nil {
			if due.After(now) || due.IsZero() {
				return "pending"
			}
		}
		return "pending"
	}
	if summary.LastFollowupAt != "" {
		return "in_progress"
	}
	return "pending"
}

// GetCustomerDetail loads all persisted information for a specific customer.
func (s *Store) GetCustomerDetail(ctx context.Context, customerID int64) (*domain.CustomerDetail, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer id")
	}

	customer, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	contacts, err := s.ListContacts(ctx, customerID)
	if err != nil {
		return nil, err
	}

	detail := &domain.CustomerDetail{
		ID:          customer.ID,
		Name:        customer.Name,
		Website:     customer.Website,
		Country:     customer.Country,
		Summary:     customer.Summary,
		Grade:       strings.ToUpper(strings.TrimSpace(customer.Grade)),
		GradeReason: customer.GradeReason,
		Contacts:    contacts,
		SourceJSON:  customer.SourceJSON,
		CreatedAt:   customer.CreatedAt,
		UpdatedAt:   customer.UpdatedAt,
	}

	if detail.Grade == "" {
		detail.Grade = "UNKNOWN"
	}

	if analysis, err := s.GetLatestAnalysis(ctx, customerID); err == nil {
		detail.Analysis = analysis
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if emailDraft, err := s.GetLatestEmailDraft(ctx, customerID, "initial"); err == nil {
		detail.EmailDraft = emailDraft
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if followupID, err := s.GetLatestFollowupID(ctx, customerID); err == nil {
		detail.FollowupID = followupID
	} else {
		return nil, err
	}

	if task, err := s.GetLatestScheduledTask(ctx, customerID); err == nil {
		detail.ScheduledTask = task
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if job, err := s.GetLatestAutomationJob(ctx, customerID); err == nil {
		detail.AutomationJob = job
	} else if err != nil {
		return nil, err
	}

	return detail, nil
}

// DeleteCustomer permanently removes a customer and related records.
func (s *Store) DeleteCustomer(ctx context.Context, customerID int64) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return fmt.Errorf("invalid customer id")
	}
	result, err := s.DB.ExecContext(ctx, `DELETE FROM customers WHERE id = ?`, customerID)
	if err != nil {
		return fmt.Errorf("删除客户失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("删除客户失败: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FindCustomerByQuery attempts to match an existing customer by name or website.
func (s *Store) FindCustomerByQuery(ctx context.Context, query string) (*domain.Customer, []domain.Contact, error) {
	if s == nil || s.DB == nil {
		return nil, nil, fmt.Errorf("store not initialized")
	}
	needle := strings.TrimSpace(query)
	if needle == "" {
		return nil, nil, nil
	}

	lowerNeedle := strings.ToLower(needle)
	var (
		customer domain.Customer
		source   sql.NullString
	)

	scan := func(condition string, args ...any) error {
		row := s.DB.QueryRowContext(ctx, `
			SELECT id, name, website, country, grade, grade_reason, summary, source_json, created_at, updated_at
			FROM customers
			WHERE `+condition+`
			ORDER BY updated_at DESC
			LIMIT 1
		`, args...)
		return row.Scan(
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
		)
	}

	var err error
	err = scan("lower(name) = ?", lowerNeedle)
	if errors.Is(err, sql.ErrNoRows) {
		err = scan("lower(website) = ?", lowerNeedle)
	}
	if errors.Is(err, sql.ErrNoRows) {
		normalized := normalizeWebsiteForCompare(needle)
		if normalized != "" {
			err = scan("lower(rtrim(replace(replace(replace(website, 'https://', ''), 'http://', ''), 'www.', ''), '/')) = ?", normalized)
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	if source.Valid {
		customer.SourceJSON = json.RawMessage(source.String)
	}

	contacts, err := s.ListContacts(ctx, customer.ID)
	if err != nil {
		return nil, nil, err
	}
	return &customer, contacts, nil
}

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

// UpdateCustomer updates persisted company info and contacts.
func (s *Store) UpdateCustomer(ctx context.Context, customerID int64, req *domain.CreateCompanyRequest) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return fmt.Errorf("invalid customer id")
	}
	if req == nil {
		return fmt.Errorf("payload is nil")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("客户名称不能为空")
	}

	source := req.SourceJSON
	if len(source) == 0 {
		source = []byte("{}")
	}

	now := Now()
	return s.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE customers
             SET name = ?, website = ?, country = ?, summary = ?, source_json = ?, updated_at = ?
             WHERE id = ?`,
			req.Name,
			req.Website,
			req.Country,
			req.Summary,
			string(source),
			now,
			customerID,
		)
		if err != nil {
			return fmt.Errorf("更新客户失败: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("客户不存在或未更新")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM contacts WHERE customer_id = ?`, customerID); err != nil {
			return fmt.Errorf("清理旧联系人失败: %w", err)
		}
		return insertContactsTx(ctx, tx, customerID, req.Contacts)
	})
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
			return nil, fmt.Errorf("尚未生成切入点分析: %w", sql.ErrNoRows)
		}
		return nil, fmt.Errorf("查询分析报告失败: %w", err)
	}
	return &resp, nil
}

// GetLatestEmailDraft returns the newest email draft of the specified type.
func (s *Store) GetLatestEmailDraft(ctx context.Context, customerID int64, emailType string) (*domain.EmailDraftResponse, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if emailType == "" {
		emailType = "initial"
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id, subject, body FROM emails WHERE customer_id = ? AND type = ? ORDER BY id DESC LIMIT 1`,
		customerID,
		emailType,
	)
	var resp domain.EmailDraftResponse
	if err := row.Scan(&resp.EmailID, &resp.Subject, &resp.Body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("未找到邮件草稿: %w", sql.ErrNoRows)
		}
		return nil, fmt.Errorf("查询邮件草稿失败: %w", err)
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

// GetLatestFollowupID fetches the most recent follow-up record id for a customer.
func (s *Store) GetLatestFollowupID(ctx context.Context, customerID int64) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	if customerID <= 0 {
		return 0, fmt.Errorf("invalid customer id")
	}
	row := s.DB.QueryRowContext(ctx,
		`SELECT id FROM followups WHERE customer_id = ? ORDER BY id DESC LIMIT 1`,
		customerID,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("查询跟进记录失败: %w", err)
	}
	return id, nil
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

func normalizeWebsiteForCompare(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.TrimPrefix(trimmed, "www.")
	return strings.TrimSuffix(trimmed, "/")
}
