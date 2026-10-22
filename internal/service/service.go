package service

import (
	"database/sql"
	"errors"
	"expense-tracker/internal/mail"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"strconv"
	"time"

	"regexp"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email address already in use")
)

const (
	BcryptCost = 12
	// Dummy hash for timing attack mitigation (cost matches BcryptCost)
	DummyHash = "$2a$12$L8c8V/Q6/S6Q/S6Q/S6Q/OuO/oO/oO/oO/oO/oO/oO/oO/oO/oO"
)

type UserService struct {
	repo      repository.UserRepository
	mailer    mail.Mailer
	jwtSecret []byte
}

func NewUserService(r repository.UserRepository, m mail.Mailer, secret string) *UserService {
	return &UserService{repo: r, mailer: m, jwtSecret: []byte(secret)}
}

func (s *UserService) Register(user *models.User) error {
	// Check if user exists
	existing, err := s.repo.GetByUsername(user.Username)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existing != nil {
		return ErrUsernameTaken
	}

	// Check if email exists
	existingEmail, err := s.repo.GetByEmail(user.Email)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingEmail != nil {
		return ErrEmailTaken
	}

	// Enforce Password Policy
	if err := s.validatePassword(user.Password); err != nil {
		return err
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), BcryptCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)
	user.Role = "user"
	user.Active = true

	return s.repo.Create(user)
}

func (s *UserService) Authenticate(username, password string) (string, *models.User, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			// Mitigation for Username Enumeration:
			// Perform a dummy hash comparison to ensure constant-time response.
			_ = bcrypt.CompareHashAndPassword([]byte(DummyHash), []byte(password))
			return "", nil, ErrUserNotFound
		}
		return "", nil, err
	}

	if !user.Active {
		return "", nil, errors.New("You're unable to Login because the Administrator SUSPENDED your account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, ErrInvalidPassword
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      user.ID,
		"username":     user.Username,
		"role":         user.Role,
		"iat":          time.Now().Unix(),
		"is_read_only": false,
		"exp":          time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return tokenString, user, nil
}

func (s *UserService) Impersonate(targetUserID int64) (string, error) {
	user, err := s.repo.GetByID(targetUserID)
	if err != nil {
		return "", err
	}

	// Generate Read-Only JWT for Impersonation
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      user.ID,
		"username":     user.Username,
		"role":         user.Role,
		"iat":          time.Now().Unix(),
		"is_read_only": true,
		"exp":          time.Now().Add(time.Hour * 1).Unix(), // Shorter duration
	})

	return token.SignedString(s.jwtSecret)
}

type FinancialService struct {
	expenseRepo repository.ExpenseRepository
	debtRepo    repository.DebtRepository
}

func NewFinancialService(e repository.ExpenseRepository, d repository.DebtRepository) *FinancialService {
	return &FinancialService{expenseRepo: e, debtRepo: d}
}

func (s *FinancialService) ListDebts(userID int64) ([]*models.Debt, error) {
	return s.debtRepo.ListByUser(userID)
}

type ExpenseService struct {
	repo repository.ExpenseRepository
}

func NewExpenseService(r repository.ExpenseRepository) *ExpenseService {
	return &ExpenseService{repo: r}
}

func (s *ExpenseService) AddExpense(exp *models.Expense) error {
	return s.repo.Create(exp)
}

func (s *ExpenseService) UpdateExpense(exp *models.Expense) error {
	return s.repo.Update(exp)
}

func (s *ExpenseService) DeleteExpense(id int64) error {
	return s.repo.Delete(id)
}

func (s *ExpenseService) BulkDeleteExpenses(userID int64, ids []int64) error {
	return s.repo.BulkDelete(userID, ids)
}

func (s *ExpenseService) GetExpenseByID(id int64) (*models.Expense, error) {
	return s.repo.GetByID(id)
}

func (s *ExpenseService) RestoreExpense(id int64) error {
	return s.repo.Restore(id)
}

func (s *ExpenseService) HardDeleteExpense(id int64) error {
	return s.repo.HardDelete(id)
}

func (s *ExpenseService) ClearAllExpenses() error {
	return s.repo.ClearAll()
}

func (s *ExpenseService) ListDeletedExpenses(userID int64) ([]*models.Expense, error) {
	return s.repo.ListDeletedByUser(userID)
}

func (s *ExpenseService) ListAllDeletedExpenses() ([]*models.Expense, error) {
	return s.repo.ListAllDeleted()
}

func (s *ExpenseService) ListAllExpenses() ([]*models.Expense, error) {
	return s.repo.ListAll()
}

func (s *ExpenseService) SearchExpenses(userID int64, filter, start, end, query, minStr, maxStr string, limit, offset int) ([]*models.Expense, error) {
	var min, max float64
	var err error
	if minStr != "" {
		min, err = strconv.ParseFloat(minStr, 64)
		if err != nil {
			return nil, errors.New("invalid min_amount")
		}
	}
	if maxStr != "" {
		max, err = strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return nil, errors.New("invalid max_amount")
		}
	}
	return s.repo.Search(userID, filter, start, end, query, min, max, limit, offset)
}

func (s *UserService) validatePassword(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(pw)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(pw)
	if !hasDigit || !hasSpecial {
		return errors.New("password must contain at least one digit and one special character")
	}
	return nil
}
