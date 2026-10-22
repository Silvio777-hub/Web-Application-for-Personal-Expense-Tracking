package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Migration represents a single database schema change.
type Migration struct {
	ID      int
	SQL     string
	DownSQL string
}

var migrations = []Migration{
	{
		ID: 1,
		SQL: `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		DownSQL: `DROP TABLE IF EXISTS users;`,
	},
	{
		ID: 2,
		SQL: `
		CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			icon TEXT,
			color TEXT,
			enabled BOOLEAN DEFAULT 1,
			is_global BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		DownSQL: `DROP TABLE IF EXISTS categories;`,
	},
	{
		ID: 3,
		SQL: `
		CREATE TABLE IF NOT EXISTS expenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			amount REAL NOT NULL,
			currency TEXT NOT NULL,
			description TEXT,
			date DATETIME NOT NULL,
			receipt_url TEXT,
			deleted_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		);`,
		DownSQL: `DROP TABLE IF EXISTS expenses;`,
	},

	{
		ID: 6,
		SQL: `
		CREATE TABLE IF NOT EXISTS debts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			person TEXT NOT NULL,
			amount REAL NOT NULL,
			currency TEXT NOT NULL,
			type TEXT NOT NULL,
			due_date DATETIME,
			paid BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
	},
	{
		ID: 7,
		SQL: `
		CREATE TABLE IF NOT EXISTS system_configs (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	},
	{
		ID: 8,
		SQL: `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			action TEXT NOT NULL,
			details TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
	},
	{
		ID:  9,
		SQL: `ALTER TABLE users ADD COLUMN profile_picture_url TEXT;`,
	},
	{
		ID:  10,
		SQL: `ALTER TABLE users ADD COLUMN currency TEXT DEFAULT 'USD';`,
	},
	{
		ID:  11,
		SQL: `ALTER TABLE users ADD COLUMN language TEXT DEFAULT 'en';`,
	},
	{
		ID:  12,
		SQL: `ALTER TABLE users ADD COLUMN date_format TEXT DEFAULT 'YYYY-MM-DD';`,
	},
	{
		ID:  13,
		SQL: `ALTER TABLE users ADD COLUMN notify_enabled BOOLEAN DEFAULT 1;`,
	},
	{
		ID:  14,
		SQL: `ALTER TABLE users ADD COLUMN monthly_savings_goal REAL DEFAULT 0;`,
	},
	{
		ID:  15,
		SQL: `ALTER TABLE users ADD COLUMN reset_token TEXT;`,
	},
	{
		ID:  16,
		SQL: `ALTER TABLE users ADD COLUMN reset_expiry DATETIME;`,
	},
	{
		ID:  17,
		SQL: `ALTER TABLE users ADD COLUMN min_token_issued_at DATETIME DEFAULT '2026-01-12 00:00:00';`,
	},
	{
		ID: 18,
		SQL: `
		INSERT OR IGNORE INTO system_configs (key, value) VALUES ('global_announcement', '');
		`,
	},
	{
		ID:  19,
		SQL: `ALTER TABLE categories ADD COLUMN user_id INTEGER DEFAULT 0 REFERENCES users(id);`,
	},
	{
		ID:  20,
		SQL: `ALTER TABLE users ADD COLUMN two_fa_enabled BOOLEAN DEFAULT 0; ALTER TABLE users ADD COLUMN two_fa_secret TEXT;`,
	},
}

// Migrate runs all pending migrations.
func Migrate(db *sql.DB) error {
	// Create migrations table if not exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (id INTEGER PRIMARY KEY)`)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM _migrations WHERE id = ?)`, m.ID).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			log.Printf("Running migration %d...", m.ID)
			_, err := db.Exec(m.SQL)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate column name") {
					log.Printf("Migration %d: Column already exists, skipping.", m.ID)
				} else {
					return fmt.Errorf("migration %d failed: %w", m.ID, err)
				}
			}
			_, err = db.Exec(`INSERT INTO _migrations (id) VALUES (?)`, m.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RollbackLast reverts the last executed migration.
func RollbackLast(db *sql.DB) error {
	var lastID int
	err := db.QueryRow(`SELECT id FROM _migrations ORDER BY id DESC LIMIT 1`).Scan(&lastID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No migrations to rollback.")
			return nil
		}
		return err
	}

	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if m.ID == lastID {
			if m.DownSQL == "" {
				return fmt.Errorf("migration %d has no DownSQL defined", m.ID)
			}
			log.Printf("Rolling back migration %d...", m.ID)
			_, err := db.Exec(m.DownSQL)
			if err != nil {
				return fmt.Errorf("rollback of migration %d failed: %w", m.ID, err)
			}
			_, err = db.Exec(`DELETE FROM _migrations WHERE id = ?`, m.ID)
			return err
		}
	}

	return fmt.Errorf("migration definition for ID %d not found", lastID)
}
