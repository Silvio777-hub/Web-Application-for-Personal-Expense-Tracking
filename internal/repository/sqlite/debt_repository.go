package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

type debtRepo struct {
	db *sql.DB
}

func NewDebtRepository(db *sql.DB) repository.DebtRepository {
	return &debtRepo{db: db}
}

func (r *debtRepo) Create(debt *models.Debt) error {
	query := `INSERT INTO debts (user_id, person, amount, currency, type, due_date, paid, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	result, err := r.db.Exec(query, debt.UserID, debt.Person, debt.Amount, debt.Currency, debt.Type, debt.DueDate, debt.Paid)
	if err != nil {
		return err
	}
	debt.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}

func (r *debtRepo) ListByUser(userID int64) ([]*models.Debt, error) {
	query := `SELECT id, user_id, person, amount, currency, type, due_date, paid, created_at FROM debts WHERE user_id = ?`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []*models.Debt{}
	for rows.Next() {
		d := &models.Debt{}
		err := rows.Scan(&d.ID, &d.UserID, &d.Person, &d.Amount, &d.Currency, &d.Type, &d.DueDate, &d.Paid, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	return debts, rows.Err()
}

func (r *debtRepo) Update(debt *models.Debt) error {
	query := `UPDATE debts SET person = ?, amount = ?, currency = ?, type = ?, due_date = ?, paid = ? WHERE id = ?`
	_, err := r.db.Exec(query, debt.Person, debt.Amount, debt.Currency, debt.Type, debt.DueDate, debt.Paid, debt.ID)
	return err
}

func (r *debtRepo) Delete(id int64) error {
	query := `DELETE FROM debts WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
