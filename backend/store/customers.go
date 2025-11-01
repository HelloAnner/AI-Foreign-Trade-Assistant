package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"

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

// CustomerListResult wraps paginated summaries.
type CustomerListResult struct {
	Items  []domain.CustomerSummary `json:"items"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// ListCustomers returns lightweight summaries for the management table.
func (s *Store) ListCustomers(ctx context.Context, filter CustomerListFilter) (*CustomerListResult, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 200 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
        SELECT
            c.id,
            c.name,
            c.website,
            c.country,
            COALESCE(NULLIF(upper(c.grade), ''), 'UNKNOWN') AS grade,
            c.created_at,
            c.updated_at,
            MAX(f.updated_at) AS last_followup_at,
            MIN(CASE WHEN st.status = 'scheduled' THEN st.due_at END) AS next_followup_at
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

	rows, err := s.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("查询客户列表失败: %w", err)
	}
	defer rows.Close()

	summaries := make([]domain.CustomerSummary, 0)
	now := time.Now().UTC()
	searchTerm := strings.TrimSpace(filter.Search)

	for rows.Next() {
		var (
			summary domain.CustomerSummary
			grade   sql.NullString
			created sql.NullString
			updated sql.NullString
			last    sql.NullString
			next    sql.NullString
			website sql.NullString
		)
		if err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&website,
			&summary.Country,
			&grade,
			&created,
			&updated,
			&last,
			&next,
		); err != nil {
			return nil, fmt.Errorf("解析客户列表失败: %w", err)
		}

		name := strings.TrimSpace(summary.Name)
		web := strings.TrimSpace(website.String)
		if searchTerm != "" && !matchesSearchTerm(name, web, searchTerm) {
			continue
		}

		summary.Grade = "UNKNOWN"
		if grade.Valid && strings.TrimSpace(grade.String) != "" {
			summary.Grade = strings.ToUpper(strings.TrimSpace(grade.String))
		}

		if created.Valid {
			summary.CreatedAt = strings.TrimSpace(created.String)
		}
		if updated.Valid {
			summary.UpdatedAt = strings.TrimSpace(updated.String)
		}
		if last.Valid {
			summary.LastFollowupAt = strings.TrimSpace(last.String)
		}
		if next.Valid {
			summary.NextFollowupAt = strings.TrimSpace(next.String)
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

	total := len(summaries)

	sortCustomers(summaries, filter.Sort)

	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	paged := make([]domain.CustomerSummary, 0)
	if start < end {
		paged = append(paged, summaries[start:end]...)
	}

	return &CustomerListResult{
		Items:  paged,
		Total:  total,
		Limit:  limit,
		Offset: start,
	}, nil
}

func deriveCustomerStatus(summary domain.CustomerSummary, now time.Time) string {
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

func sortCustomers(items []domain.CustomerSummary, sortKey string) {
	switch sortKey {
	case "name_asc":
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})
	case "name_desc":
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Name) > strings.ToLower(items[j].Name)
		})
	case "created_asc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].CreatedAt)
			tj := parseTimeValue(items[j].CreatedAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.Before(tj)
		})
	case "created_desc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].CreatedAt)
			tj := parseTimeValue(items[j].CreatedAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.After(tj)
		})
	case "updated_asc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].UpdatedAt)
			tj := parseTimeValue(items[j].UpdatedAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.Before(tj)
		})
	case "updated_desc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].UpdatedAt)
			tj := parseTimeValue(items[j].UpdatedAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.After(tj)
		})
	case "last_followup_asc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].LastFollowupAt)
			tj := parseTimeValue(items[j].LastFollowupAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.Before(tj)
		})
	case "last_followup_desc":
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].LastFollowupAt)
			tj := parseTimeValue(items[j].LastFollowupAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.After(tj)
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			ti := parseTimeValue(items[i].CreatedAt)
			tj := parseTimeValue(items[j].CreatedAt)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			if ti.Equal(tj) {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			}
			return ti.After(tj)
		})
	}
}

func parseTimeValue(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func matchesSearchTerm(name, website, query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return true
	}
	nameLower := strings.ToLower(strings.TrimSpace(name))
	if strings.Contains(nameLower, q) {
		return true
	}
	websiteLower := strings.ToLower(strings.TrimSpace(website))
	if websiteLower != "" && strings.Contains(websiteLower, q) {
		return true
	}
	compactQuery := normalizeSearchToken(q)
	if compactQuery == "" {
		return true
	}
	if strings.Contains(normalizeSearchToken(nameLower), compactQuery) {
		return true
	}
	if websiteLower != "" && strings.Contains(normalizeSearchToken(websiteLower), compactQuery) {
		return true
	}
	full, initials := buildPinyinVariants(nameLower)
	if full != "" && strings.Contains(full, compactQuery) {
		return true
	}
	if initials != "" && strings.Contains(initials, compactQuery) {
		return true
	}
	return false
}

func buildPinyinVariants(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ""
	}
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Heteronym = false
	syllables := pinyin.Pinyin(trimmed, args)
	if len(syllables) == 0 {
		return "", ""
	}
	var full strings.Builder
	var initials strings.Builder
	for _, s := range syllables {
		if len(s) == 0 {
			continue
		}
		sy := normalizeSearchToken(s[0])
		if sy == "" {
			continue
		}
		full.WriteString(sy)
		initials.WriteByte(sy[0])
	}
	return full.String(), initials.String()
}

func normalizeSearchToken(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	lowered = strings.ReplaceAll(lowered, " ", "")
	lowered = strings.ReplaceAll(lowered, "-", "")
	lowered = strings.ReplaceAll(lowered, "_", "")
	lowered = strings.ReplaceAll(lowered, ".", "")
	return lowered
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

func (s *Store) customerExistsByNameOrWebsite(ctx context.Context, name, website string) (bool, error) {
	if s == nil || s.DB == nil {
		return false, fmt.Errorf("store not initialized")
	}
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	normalizedWebsite := normalizeWebsiteForCompare(website)

	exists, err := s.existsWithQuery(ctx, "SELECT 1 FROM customers WHERE lower(name) = ? LIMIT 1", normalizedName != "", normalizedName)
	if err != nil || exists {
		return exists, err
	}
	exists, err = s.existsWithQuery(ctx,
		`SELECT 1 FROM customers
         WHERE lower(rtrim(replace(replace(replace(website, 'https://', ''), 'http://', ''), 'www.', ''), '/')) = ?
         LIMIT 1`,
		normalizedWebsite != "",
		normalizedWebsite,
	)
	if err != nil || exists {
		return exists, err
	}

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if normalizedName != "" {
		conditions = append(conditions, "lower(c.name) = ?")
		args = append(args, normalizedName)
	}
	if normalizedWebsite != "" {
		conditions = append(conditions, "lower(rtrim(replace(replace(replace(c.website, 'https://', ''), 'http://', ''), 'www.', ''), '/')) = ?")
		args = append(args, normalizedWebsite)
	}
	if len(conditions) == 0 {
		return false, nil
	}

	query := `
        SELECT 1
        FROM automation_jobs aj
        JOIN customers c ON c.id = aj.customer_id
        WHERE aj.status IN (?, ?)
          AND (` + strings.Join(conditions, " OR ") + `)
        LIMIT 1`
	params := append([]any{domain.AutomationStatusQueued, domain.AutomationStatusRunning}, args...)
	exists, err = s.existsWithQuery(ctx, query, true, params...)
	if err != nil || exists {
		return exists, err
	}

	return false, nil
}

func (s *Store) existsWithQuery(ctx context.Context, query string, enabled bool, args ...any) (bool, error) {
	if !enabled {
		return false, nil
	}
	row := s.DB.QueryRowContext(ctx, query, args...)
	var dummy int
	if err := row.Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query existence failed: %w", err)
	}
	return true, nil
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

	if exists, err := s.customerExistsByNameOrWebsite(ctx, req.Name, req.Website); err != nil {
		return 0, err
	} else if exists {
		return 0, fmt.Errorf("客户已存在")
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
