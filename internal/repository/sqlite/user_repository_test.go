package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/utils"
	"testing"

	_ "modernc.org/sqlite"
)

const testMasterKey = "395DD669C782E5159C0DBC8CBFED373E8EBE423119A2C90B9C27338093AD8FED"

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	query := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'user',
		active BOOLEAN DEFAULT 1,
		profile_picture_url TEXT,
		currency TEXT DEFAULT 'USD',
		language TEXT DEFAULT 'en',
		date_format TEXT DEFAULT 'YYYY-MM-DD',
		notify_enabled BOOLEAN DEFAULT 1,
		monthly_savings_goal REAL DEFAULT 0,
		reset_token TEXT,
		reset_expiry DATETIME,
		min_token_issued_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		two_fa_enabled BOOLEAN DEFAULT 0,
		two_fa_secret TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db
}

func TestUserRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	kp := &utils.StaticKeyProvider{Key: testMasterKey}
	repo := NewUserRepository(db, kp)

	user := &models.User{
		Username:           "testuser",
		Email:              "test@example.com",
		Password:           "hashedpassword",
		Role:               "user",
		Active:             true,
		Currency:           "USD",
		NotifyEnabled:      true,
		MonthlySavingsGoal: 500.00,
	}

	// Test Create
	if err := repo.Create(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Errorf("Expected user ID to be set, got 0")
	}

	// Test GetByID
	fetchedUser, err := repo.GetByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if fetchedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, fetchedUser.Username)
	}
	if fetchedUser.MonthlySavingsGoal != 500.00 {
		t.Errorf("Expected savings goal 500.00, got %f", fetchedUser.MonthlySavingsGoal)
	}
}

func TestUserRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	kp := &utils.StaticKeyProvider{Key: testMasterKey}
	repo := NewUserRepository(db, kp)
	user := &models.User{
		Username: "updateuser",
		Email:    "update@example.com",
		Password: "pass",
		Role:     "user",
		Active:   true,
	}
	repo.Create(user)

	// Modify
	user.Username = "updated_name"
	user.Currency = "EUR"

	if err := repo.Update(user); err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	updated, _ := repo.GetByID(user.ID)
	if updated.Username != "updated_name" {
		t.Errorf("Expected updated username 'updated_name', got '%s'", updated.Username)
	}
	if updated.Currency != "EUR" {
		t.Errorf("Expected updated currency 'EUR', got '%s'", updated.Currency)
	}
}

func TestUserRepo_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	kp := &utils.StaticKeyProvider{Key: testMasterKey}
	repo := NewUserRepository(db, kp)
	user := &models.User{
		Username: "delete_me",
		Email:    "delete@example.com",
		Password: "pass",
		Role:     "user",
		Active:   true,
	}
	repo.Create(user)

	// Verify it exists
	if _, err := repo.GetByID(user.ID); err != nil {
		t.Fatalf("User should exist before hard delete: %v", err)
	}

	// Hard Delete
	if err := repo.HardDelete(user.ID); err != nil {
		t.Fatalf("Failed to hard delete user: %v", err)
	}

	// Verify it's gone
	_, err := repo.GetByID(user.ID)
	if err == nil {
		t.Error("Expected error when getting hard deleted user, got nil")
	}
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}
