package sqlite

import (
	"database/sql"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

type auditRepo struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) repository.AuditLogRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(log *models.AuditLog) error {
	query := `INSERT INTO audit_logs (user_id, action, details, timestamp) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := r.db.Exec(query, log.UserID, log.Action, log.Details)
	return err
}

func (r *auditRepo) List(limit int) ([]*models.AuditLog, error) {
	query := `SELECT id, user_id, action, details, timestamp FROM audit_logs ORDER BY timestamp DESC LIMIT ?`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []*models.AuditLog{}
	for rows.Next() {
		log := &models.AuditLog{}
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Details, &log.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *auditRepo) ClearAll() error {
	query := `DELETE FROM audit_logs`
	_, err := r.db.Exec(query)
	return err
}
