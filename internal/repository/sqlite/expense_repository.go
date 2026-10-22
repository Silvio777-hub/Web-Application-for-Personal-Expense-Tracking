package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"fmt"
	"strings"
)

type expenseRepo struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) repository.ExpenseRepository {
	return &expenseRepo{db: db}
}

// Create adds a new expense record to the database and populates the ID.
func (r *expenseRepo) Create(expense *models.Expense) error {
	query := `INSERT INTO expenses (user_id, category_id, amount, currency, description, date, receipt_url, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	result, err := r.db.Exec(query, expense.UserID, expense.CategoryID, expense.Amount, expense.Currency, expense.Description, expense.Date, expense.ReceiptURL)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	expense.ID = id
	return nil
}

// GetByID retrieves a specific expense by its unique identifier.
func (r *expenseRepo) GetByID(id int64) (*models.Expense, error) {
	expense := &models.Expense{}
	query := `SELECT id, user_id, category_id, amount, currency, description, date, receipt_url, created_at FROM expenses WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&expense.ID, &expense.UserID, &expense.CategoryID, &expense.Amount, &expense.Currency, &expense.Description, &expense.Date, &expense.ReceiptURL, &expense.CreatedAt)
	if err != nil {
		return nil, err
	}
	return expense, nil
}

func (r *expenseRepo) Update(expense *models.Expense) error {
	query := `UPDATE expenses SET category_id = ?, amount = ?, currency = ?, description = ?, date = ?, receipt_url = ? WHERE id = ?`
	_, err := r.db.Exec(query, expense.CategoryID, expense.Amount, expense.Currency, expense.Description, expense.Date, expense.ReceiptURL, expense.ID)
	return err
}

func (r *expenseRepo) Delete(id int64) error {
	query := `UPDATE expenses SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *expenseRepo) HardDelete(id int64) error {
	query := `DELETE FROM expenses WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *expenseRepo) ClearAll() error {
	query := `DELETE FROM expenses`
	_, err := r.db.Exec(query)
	return err
}

func (r *expenseRepo) BulkDelete(userID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	// Prepare placeholders for the IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)

	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`UPDATE expenses SET deleted_at = CURRENT_TIMESTAMP WHERE id IN (%s)`,
		strings.Join(placeholders, ","))

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *expenseRepo) Restore(id int64) error {
	query := `UPDATE expenses SET deleted_at = NULL WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *expenseRepo) GetGlobalTotalSpent() (float64, error) {
	var total float64
	query := `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE deleted_at IS NULL`
	err := r.db.QueryRow(query).Scan(&total)
	return total, err
}

func (r *expenseRepo) GetSpendingPerUser() ([]repository.UserSpending, error) {
	query := `SELECT u.id, u.username, COALESCE(SUM(e.amount), 0) as total
			  FROM users u
			  LEFT JOIN expenses e ON u.id = e.user_id AND e.deleted_at IS NULL
			  WHERE u.active = 1
			  GROUP BY u.id, u.username
			  ORDER BY total DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []repository.UserSpending
	for rows.Next() {
		var s repository.UserSpending
		if err := rows.Scan(&s.UserID, &s.Username, &s.Total); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *expenseRepo) ListDeletedByUser(userID int64) ([]*models.Expense, error) {
	query := `SELECT id, user_id, category_id, amount, currency, description, date, receipt_url, created_at FROM expenses WHERE user_id = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := []*models.Expense{}
	for rows.Next() {
		expense := &models.Expense{}
		err := rows.Scan(&expense.ID, &expense.UserID, &expense.CategoryID, &expense.Amount, &expense.Currency, &expense.Description, &expense.Date, &expense.ReceiptURL, &expense.CreatedAt)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}

func (r *expenseRepo) Search(userID int64, filter, start, end, query string, min, max float64, limit, offset int) ([]*models.Expense, error) {
	sqlStr := `SELECT e.id, e.user_id, u.username, e.category_id, e.amount, e.currency, e.description, e.date, e.receipt_url, e.created_at FROM expenses e JOIN users u ON e.user_id = u.id WHERE 1=1 AND e.deleted_at IS NULL`
	var args []interface{}

	if userID > 0 {
		sqlStr += ` AND e.user_id = ?`
		args = append(args, userID)
	}

	if query != "" {
		sqlStr += ` AND e.description LIKE ?`
		args = append(args, "%"+query+"%")
	}

	if min > 0 {
		sqlStr += ` AND e.amount >= ?`
		args = append(args, min)
	}
	if max > 0 {
		sqlStr += ` AND e.amount <= ?`
		args = append(args, max)
	}

	if start != "" && end != "" {
		sqlStr += ` AND e.date BETWEEN ? AND ?`
		args = append(args, start, end)
	} else if filter != "" {
		switch filter {
		case "weekly":
			sqlStr += ` AND e.date >= date('now', '-7 days')`
		case "monthly":
			sqlStr += ` AND strftime('%Y-%m', e.date) = strftime('%Y-%m', 'now')`
		case "yearly":
			sqlStr += ` AND strftime('%Y', e.date) = strftime('%Y', 'now')`
		}
	}

	sqlStr += ` ORDER BY e.date DESC`

	if limit > 0 {
		sqlStr += ` LIMIT ?`
		args = append(args, limit)
	}
	if offset > 0 {
		sqlStr += ` OFFSET ?`
		args = append(args, offset)
	}

	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := []*models.Expense{}
	for rows.Next() {
		expense := &models.Expense{}
		err := rows.Scan(&expense.ID, &expense.UserID, &expense.Username, &expense.CategoryID, &expense.Amount, &expense.Currency, &expense.Description, &expense.Date, &expense.ReceiptURL, &expense.CreatedAt)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}

func (r *expenseRepo) ListAllDeleted() ([]*models.Expense, error) {
	query := `SELECT e.id, e.user_id, u.username, e.category_id, e.amount, e.currency, e.description, e.date, e.receipt_url, e.created_at FROM expenses e JOIN users u ON e.user_id = u.id WHERE e.deleted_at IS NOT NULL ORDER BY e.deleted_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := []*models.Expense{}
	for rows.Next() {
		expense := &models.Expense{}
		err := rows.Scan(&expense.ID, &expense.UserID, &expense.Username, &expense.CategoryID, &expense.Amount, &expense.Currency, &expense.Description, &expense.Date, &expense.ReceiptURL, &expense.CreatedAt)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}
func (r *expenseRepo) ListAll() ([]*models.Expense, error) {
	query := `SELECT e.id, e.user_id, u.username, e.category_id, e.amount, e.currency, e.description, e.date, e.receipt_url, e.created_at FROM expenses e JOIN users u ON e.user_id = u.id WHERE e.deleted_at IS NULL ORDER BY e.date DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := []*models.Expense{}
	for rows.Next() {
		expense := &models.Expense{}
		err := rows.Scan(&expense.ID, &expense.UserID, &expense.Username, &expense.CategoryID, &expense.Amount, &expense.Currency, &expense.Description, &expense.Date, &expense.ReceiptURL, &expense.CreatedAt)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}

func (r *expenseRepo) ExistsByReceiptURL(userID int64, url string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM expenses WHERE user_id = ? AND receipt_url = ? AND deleted_at IS NULL`
	err := r.db.QueryRow(query, userID, url).Scan(&count)
	return count > 0, err
}
