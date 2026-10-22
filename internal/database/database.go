package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// SQLite Optimizations for Web Applications:
	// Set MaxOpenConns to 1 to eliminate "database is locked" errors in SQLite.
	// SQLite handles its own concurrency better with a single handle in serialized mode.
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	if err = DB.Ping(); err != nil {
		return err
	}

	// Enable WAL (Write Ahead Logging) and busy timeout
	_, _ = DB.Exec("PRAGMA journal_mode=WAL;")
	_, _ = DB.Exec("PRAGMA busy_timeout=10000;") // 10s timeout to survive high concurrency spikes
	_, _ = DB.Exec("PRAGMA synchronous=NORMAL;") // Balance reliability and performance
	_, _ = DB.Exec("PRAGMA cache_size=-64000;")  // Use ~64MB of memory for caching index data

	// Run migrations
	if err := Migrate(DB); err != nil {
		return err
	}

	// Ensure schema correctness (fallback for specific columns)
	ensureSchema()

	// Seed default data
	seedData()

	log.Println("Database initialized and migrations applied at", dbPath)
	return nil
}

// Backup creates an atomic, consistent copy of the database at the specified path.
func Backup(destPath string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	// Use VACUUM INTO for an atomic, safe online backup
	query := fmt.Sprintf("VACUUM INTO '%s';", destPath)
	_, err := DB.Exec(query)
	return err
}

func seedData() {
	// Seed Admin User (ID=1)
	// First try to update in case ID 1 exists but name/password changed
	_, err := DB.Exec(`UPDATE users SET username='admin', password=?, role='admin', active=1, updated_at=CURRENT_TIMESTAMP WHERE id=1`,
		"$2a$10$EY5RyNhBtSlUP6PtMcMQ2O/p.RISk/kGQLhzvz3hJU2/3azJuw7ha")

	// Then try to insert if it doesn't exist at all
	_, err = DB.Exec(`INSERT OR IGNORE INTO users (id, username, email, password, role, active) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "admin", "admin@example.com", "$2a$10$EY5RyNhBtSlUP6PtMcMQ2O/p.RISk/kGQLhzvz3hJU2/3azJuw7ha", "admin", 1)

	if err != nil {
		log.Printf("Warning: Admin seed check encountered error: %v\n", err)
	}

	// Seed Default Categories
	categories := []struct {
		Name  string
		Icon  string
		Color string
	}{
		{"Housing & Utilities", "", "#3357FF"},
		{"Food & Dining", "", "#4CAF50"},
		{"Transportation", "", "#FFC107"},
		{"Health & Wellness", "", "#E91E63"},
		{"Tech & Subscriptions", "", "#03A9F4"},
		{"Entertainment", "", "#9C27B0"},
		{"Personal Care & Style", "", "#FF5722"},
		{"Education & Work", "", "#3F51B5"},
		{"Savings & Debt", "", "#607D8B"},
		{"Miscellaneous", "", "#9E9E9E"},
	}

	for _, c := range categories {
		_, _ = DB.Exec(`INSERT OR IGNORE INTO categories (name, icon, color, is_global) VALUES (?, ?, ?, 1)`,
			c.Name, c.Icon, c.Color)
	}
}

func ensureSchema() {
	// SQLite doesn't support ADD COLUMN IF NOT EXISTS directly,
	// so we try to add each new column and ignore errors if they exist.
	columns := map[string]string{
		"profile_picture_url":  "TEXT",
		"currency":             "TEXT DEFAULT 'USD'",
		"language":             "TEXT DEFAULT 'en'",
		"date_format":          "TEXT DEFAULT 'YYYY-MM-DD'",
		"notify_enabled":       "BOOLEAN DEFAULT 1",
		"reset_token":          "TEXT",
		"reset_expiry":         "DATETIME",
		"monthly_savings_goal": "REAL DEFAULT 0",
		"min_token_issued_at":  "DATETIME DEFAULT '2026-01-12 00:00:00'",
	}

	for col, spec := range columns {
		query := fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", col, spec)
		if _, err := DB.Exec(query); err != nil {
			// Only ignore "already exists" errors
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") &&
				!strings.Contains(strings.ToLower(err.Error()), "exists") {
				log.Printf("Warning: Failed to ensure column %s: %v\n", col, err)
			}
		}
	}

	// Expenses migrations
	_, _ = DB.Exec("ALTER TABLE expenses ADD COLUMN deleted_at DATETIME")

	// Add Indices
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_expenses_user_date ON expenses(user_id, date)`)
	_, _ = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_expenses_user_cat_date ON expenses(user_id, category_id, date)`)
}
