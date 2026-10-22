package service

import (
	"database/sql"
	"expense-tracker/internal/mail"
	"expense-tracker/internal/models"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
}

func (m *mockUserRepo) Create(user *models.User) error         { return nil }
func (m *mockUserRepo) GetByID(id int64) (*models.User, error) { return nil, nil }
func (m *mockUserRepo) GetByUsername(username string) (*models.User, error) {
	if username == "admin" {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin_password"), bcrypt.DefaultCost)
		return &models.User{
			ID:                 1,
			Username:           "admin",
			Password:           string(hash),
			Role:               "admin",
			Active:             true,
			Language:           "en",
			Currency:           "USD",
			DateFormat:         "YYYY-MM-DD",
			MonthlySavingsGoal: 0,
		}, nil
	}
	return nil, nil
}
func (m *mockUserRepo) Update(user *models.User) error   { return nil }
func (m *mockUserRepo) Delete(id int64) error            { return nil }
func (m *mockUserRepo) Recover(id int64) error           { return nil }
func (m *mockUserRepo) HardDelete(id int64) error        { return nil }
func (m *mockUserRepo) ListAll() ([]*models.User, error) { return nil, nil }
func (m *mockUserRepo) GetByEmail(email string) (*models.User, error) {
	if email == "existing@example.com" {
		return &models.User{Username: "existing", Email: email}, nil
	}
	return nil, sql.ErrNoRows
}
func (m *mockUserRepo) RevokeSessions(id int64) error { return nil }

func TestAuthenticate(t *testing.T) {
	repo := &mockUserRepo{}
	s := NewUserService(repo, &mail.LogMailer{}, "secret")

	// Test successful login
	token, user, err := s.Authenticate("admin", "admin_password")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if token == "" || user == nil {
		t.Error("Expected token and user")
	}

	// Test invalid password
	_, _, err = s.Authenticate("admin", "wrong")
	if err == nil {
		t.Error("Expected error for wrong password")
	}
}
func TestRegister_PasswordPolicy(t *testing.T) {
	repo := &mockUserRepo{}
	s := NewUserService(repo, &mail.LogMailer{}, "secret")

	// Test weak password (too short)
	err := s.Register(&models.User{Username: "new", Password: "123"})
	if err == nil || err.Error() != "password must be at least 8 characters long" {
		t.Errorf("Expected length error, got %v", err)
	}

	// Test weak password (no special/digit)
	err = s.Register(&models.User{Username: "new", Password: "password"})
	if err == nil || err.Error() != "password must contain at least one digit and one special character" {
		t.Errorf("Expected complexity error, got %v", err)
	}

	// Test valid password
	err = s.Register(&models.User{Username: "new", Password: "SecurePassword1!"})
	if err != nil {
		t.Errorf("Expected success for strong password, got %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{}
	s := NewUserService(repo, &mail.LogMailer{}, "secret")

	err := s.Register(&models.User{
		Username: "newuser",
		Email:    "existing@example.com",
		Password: "SecurePassword1!",
	})
	if err != ErrEmailTaken {
		t.Errorf("Expected ErrEmailTaken, got %v", err)
	}
}
