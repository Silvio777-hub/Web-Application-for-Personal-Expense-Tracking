package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

type systemConfigRepo struct {
	db *sql.DB
}

func NewSystemConfigRepository(db *sql.DB) repository.SystemConfigRepository {
	return &systemConfigRepo{db: db}
}

func (r *systemConfigRepo) Get(key string) (string, error) {
	var value string
	query := `SELECT value FROM system_configs WHERE key = ?`
	err := r.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *systemConfigRepo) Set(key string, value string) error {
	query := `INSERT INTO system_configs (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			  ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.Exec(query, key, value)
	return err
}

func (r *systemConfigRepo) List() ([]*models.SystemConfig, error) {
	query := `SELECT key, value, updated_at FROM system_configs`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := []*models.SystemConfig{}
	for rows.Next() {
		cfg := &models.SystemConfig{}
		err := rows.Scan(&cfg.Key, &cfg.Value, &cfg.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}
