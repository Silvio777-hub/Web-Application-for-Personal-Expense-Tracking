package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

type categoryRepo struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) repository.CategoryRepository {
	return &categoryRepo{db: db}
}

func (r *categoryRepo) List(userID int64) ([]*models.Category, error) {
	query := `SELECT id, name, icon, color, enabled, is_global, user_id FROM categories WHERE user_id = ? OR is_global = 1`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []*models.Category{}
	for rows.Next() {
		cat := &models.Category{}
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Icon, &cat.Color, &cat.Enabled, &cat.IsGlobal, &cat.UserID)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func (r *categoryRepo) GetByID(id int64) (*models.Category, error) {
	cat := &models.Category{}
	query := `SELECT id, name, icon, color, enabled, is_global, user_id FROM categories WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&cat.ID, &cat.Name, &cat.Icon, &cat.Color, &cat.Enabled, &cat.IsGlobal, &cat.UserID)
	if err != nil {
		return nil, err
	}
	return cat, nil
}

func (r *categoryRepo) Create(cat *models.Category) error {
	query := `INSERT INTO categories (name, icon, color, enabled, is_global, user_id, created_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	result, err := r.db.Exec(query, cat.Name, cat.Icon, cat.Color, cat.Enabled, cat.IsGlobal, cat.UserID)
	if err != nil {
		return err
	}
	cat.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}

func (r *categoryRepo) Update(cat *models.Category) error {
	query := `UPDATE categories SET name = ?, icon = ?, color = ?, enabled = ? WHERE id = ?`
	_, err := r.db.Exec(query, cat.Name, cat.Icon, cat.Color, cat.Enabled, cat.ID)
	return err
}

func (r *categoryRepo) Delete(id int64) error {
	query := `DELETE FROM categories WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
