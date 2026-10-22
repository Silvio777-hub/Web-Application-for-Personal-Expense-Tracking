package repository

import "expense-tracker/internal/models"

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id int64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	Delete(id int64) error // Soft delete
	Recover(id int64) error
	HardDelete(id int64) error        // Permanent delete
	ListAll() ([]*models.User, error) // Including inactive
	GetByEmail(email string) (*models.User, error)
	RevokeSessions(id int64) error
}

type UserSpending struct {
	UserID   int64   `json:"user_id"`
	Username string  `json:"username"`
	Total    float64 `json:"total"`
}

type ExpenseRepository interface {
	Create(expense *models.Expense) error
	GetByID(id int64) (*models.Expense, error)
	Update(expense *models.Expense) error
	Delete(id int64) error
	HardDelete(id int64) error
	ClearAll() error
	GetGlobalTotalSpent() (float64, error)
	GetSpendingPerUser() ([]UserSpending, error)
	ListDeletedByUser(userID int64) ([]*models.Expense, error)
	ListAllDeleted() ([]*models.Expense, error)
	Restore(id int64) error
	ListAll() ([]*models.Expense, error)
	Search(userID int64, filter, start, end, query string, min, max float64, limit, offset int) ([]*models.Expense, error)
	BulkDelete(userID int64, ids []int64) error
	ExistsByReceiptURL(userID int64, url string) (bool, error)
}

type DebtRepository interface {
	Create(debt *models.Debt) error
	ListByUser(userID int64) ([]*models.Debt, error)
	Update(debt *models.Debt) error
	Delete(id int64) error
}

type CategoryRepository interface {
	List(userID int64) ([]*models.Category, error)
	GetByID(id int64) (*models.Category, error)
	Create(cat *models.Category) error
	Update(cat *models.Category) error
	Delete(id int64) error
}

type AuditLogRepository interface {
	Create(log *models.AuditLog) error
	List(limit int) ([]*models.AuditLog, error)
	ClearAll() error
}

type SystemConfigRepository interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	List() ([]*models.SystemConfig, error)
}
