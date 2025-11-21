package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store wraps a SQLite connection and provides high-level helpers.
type Store struct {
	DB *sql.DB
}

// Open initializes a new SQLite database in WAL mode for better concurrency.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &Store{DB: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// InitSchema ensures all tables exist. It is idempotent and safe to call on every start.
func (s *Store) InitSchema(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}

	statements := []string{
		`PRAGMA journal_mode = WAL;`,
        `CREATE TABLE IF NOT EXISTS settings (
        id INTEGER PRIMARY KEY CHECK (id = 1),
        llm_base_url TEXT,
        llm_api_key TEXT,
        llm_model TEXT,
        my_company_name TEXT,
        my_product_profile TEXT,
        smtp_host TEXT,
        smtp_port INTEGER,
        smtp_username TEXT,
        smtp_password TEXT,
        smtp_security TEXT DEFAULT 'auto',
        admin_email TEXT,
        rating_guideline TEXT,
        search_provider TEXT,
        search_api_key TEXT,
        automation_enabled INTEGER DEFAULT 0,
        automation_followup_days INTEGER DEFAULT 3,
        automation_required_grade TEXT DEFAULT 'A',
        login_password_hash TEXT,
        login_password_version INTEGER DEFAULT 1,
        created_at TEXT,
        updated_at TEXT
    );`,
		`INSERT INTO settings (id, created_at, updated_at)
			SELECT 1, datetime('now'), datetime('now')
			WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1);`,
		`CREATE TABLE IF NOT EXISTS customers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			website TEXT,
			country TEXT,
			grade TEXT,
			grade_reason TEXT,
			summary TEXT,
			followup_sent INTEGER DEFAULT 0,
			source_json TEXT,
			created_at TEXT,
			updated_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS contacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER,
			name TEXT,
			title TEXT,
			email TEXT,
			phone TEXT,
			source TEXT,
			is_key INTEGER DEFAULT 0,
			created_at TEXT,
			updated_at TEXT,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS analyses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER,
			core_business TEXT,
			pain_points TEXT,
			my_entry_points TEXT,
			full_report TEXT,
			created_at TEXT,
			updated_at TEXT,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER,
			type TEXT,
			subject TEXT,
			body TEXT,
			status TEXT,
			sent_at TEXT,
			smtp_message_id TEXT,
			created_at TEXT,
			updated_at TEXT,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS followups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER,
			initial_email_id INTEGER,
			notes TEXT,
			created_at TEXT,
			updated_at TEXT,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			FOREIGN KEY(initial_email_id) REFERENCES emails(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS scheduled_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER,
			due_at TEXT,
			status TEXT,
			last_error TEXT,
			context_email_id INTEGER,
			generated_email_id INTEGER,
			schedule_mode TEXT DEFAULT 'simple',
			delay_value INTEGER DEFAULT 0,
			delay_unit TEXT,
			cron_expression TEXT,
			attempts INTEGER DEFAULT 0,
			created_at TEXT,
			updated_at TEXT,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			FOREIGN KEY(context_email_id) REFERENCES emails(id) ON DELETE SET NULL,
			FOREIGN KEY(generated_email_id) REFERENCES emails(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS automation_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			stage TEXT NOT NULL,
			last_error TEXT,
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(customer_id) REFERENCES customers(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS todo_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT NOT NULL,
			status TEXT NOT NULL,
			last_error TEXT,
			customer_id INTEGER,
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT,
			message TEXT,
			meta_json TEXT,
			created_at TEXT
		);`,
	}

	for _, stmt := range statements {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}

	// 追加列（向后兼容）
	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE customers ADD COLUMN summary TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure summary column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN automation_enabled INTEGER DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure automation_enabled column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN automation_followup_days INTEGER DEFAULT 3`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure automation_followup_days column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN automation_required_grade TEXT DEFAULT 'A'`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure automation_required_grade column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN smtp_security TEXT DEFAULT 'auto'`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure smtp_security column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN login_password_hash TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure login_password_hash column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE settings ADD COLUMN login_password_version INTEGER DEFAULT 1`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure login_password_version column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE scheduled_tasks ADD COLUMN schedule_mode TEXT DEFAULT 'simple'`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure schedule_mode column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE scheduled_tasks ADD COLUMN delay_value INTEGER DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure delay_value column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE scheduled_tasks ADD COLUMN delay_unit TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure delay_unit column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE scheduled_tasks ADD COLUMN cron_expression TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure cron_expression column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE scheduled_tasks ADD COLUMN attempts INTEGER DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure attempts column: %w", err)
		}
	}

	if _, err := s.DB.ExecContext(ctx, `ALTER TABLE customers ADD COLUMN followup_sent INTEGER DEFAULT 0`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("ensure followup_sent column: %w", err)
		}
	}

	return nil
}

// TouchUpdatedAt updates the updated_at column for a given table and row id.
func (s *Store) TouchUpdatedAt(ctx context.Context, table string, id int64) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET updated_at = ? WHERE id = ?", table), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// Now returns ISO8601 UTC timestamp helper.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// MustBeginTx begins a transaction with sensible defaults.
func (s *Store) MustBeginTx(ctx context.Context) (*sql.Tx, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// WithTx is a helper to execute fn within a transaction.
func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.MustBeginTx(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("txn rollback error: %v", rbErr)
		}
		return err
	}
	return tx.Commit()
}

// GetLoginPassword retrieves hashed login password and version.
func (s *Store) GetLoginPassword(ctx context.Context) (string, int, error) {
	if s == nil || s.DB == nil {
		return "", 0, fmt.Errorf("store not initialized")
	}
	var hash sql.NullString
	var version sql.NullInt64
	if err := s.DB.QueryRowContext(ctx, `SELECT login_password_hash, login_password_version FROM settings WHERE id = 1`).
		Scan(&hash, &version); err != nil {
		return "", 0, fmt.Errorf("query login password: %w", err)
	}
	v := 1
	if version.Valid && version.Int64 > 0 {
		v = int(version.Int64)
	}
	return hash.String, v, nil
}

// UpdateLoginPassword stores the hashed password and version.
func (s *Store) UpdateLoginPassword(ctx context.Context, hash string, version int) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	if version <= 0 {
		version = 1
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE settings SET login_password_hash = ?, login_password_version = ?, updated_at = datetime('now')
		WHERE id = 1;
	`, hash, version)
	if err != nil {
		return fmt.Errorf("update login password: %w", err)
	}
	return nil
}
