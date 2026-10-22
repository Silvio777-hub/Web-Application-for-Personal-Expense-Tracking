package sqlite

import (
	"database/sql"
	"expense-tracker/internal/logger"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/utils"
)

type userRepo struct {
	db          *sql.DB
	keyProvider utils.KeyProvider
}

func NewUserRepository(db *sql.DB, kp utils.KeyProvider) repository.UserRepository {
	return &userRepo{db: db, keyProvider: kp}
}

func (r *userRepo) encrypt(val string) string {
	if val == "" {
		return ""
	}
	key, err := r.keyProvider.GetMasterKey()
	if err != nil {
		logger.Error("Key retrieval failed for encryption", err)
		return ""
	}
	encrypted, err := utils.Encrypt(val, key)
	if err != nil {
		logger.Error("Encryption failed", err)
		return ""
	}
	return encrypted
}

func (r *userRepo) decrypt(val string) string {
	if val == "" {
		return ""
	}
	key, err := r.keyProvider.GetMasterKey()
	if err != nil {
		logger.Error("Key retrieval failed for decryption", err)
		return ""
	}
	decrypted, err := utils.Decrypt(val, key)
	if err != nil {
		logger.Error("Decryption failed", err)
		return ""
	}
	return decrypted
}

// Create inserts a new user into the database and retrieves the generated ID.
func (r *userRepo) Create(user *models.User) error {
	query := `INSERT INTO users (username, email, password, role, active, profile_picture_url, currency, language, date_format, notify_enabled, monthly_savings_goal, reset_token, reset_expiry, min_token_issued_at, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	encToken := r.encrypt(user.ResetToken)

	result, err := r.db.Exec(query, user.Username, user.Email, user.Password, user.Role, user.Active,
		user.ProfilePictureURL, user.Currency, user.Language, user.DateFormat, user.NotifyEnabled,
		user.MonthlySavingsGoal, encToken, user.ResetExpiry, user.MinTokenIssuedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

// GetByID fetches a user record by their database ID.
func (r *userRepo) GetByID(id int64) (*models.User, error) {
	user := &models.User{}
	var resetExpiry sql.NullTime
	query := `SELECT id, username, email, password, role, active, 
			  COALESCE(profile_picture_url, ''), COALESCE(currency, 'USD'), COALESCE(language, 'en'), COALESCE(date_format, 'YYYY-MM-DD'), 
			  COALESCE(notify_enabled, 1), COALESCE(monthly_savings_goal, 0), COALESCE(reset_token, ''), reset_expiry, min_token_issued_at, 
			  created_at, updated_at 
			  FROM users WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Active,
		&user.ProfilePictureURL, &user.Currency, &user.Language, &user.DateFormat, &user.NotifyEnabled, &user.MonthlySavingsGoal,
		&user.ResetToken, &resetExpiry, &user.MinTokenIssuedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	user.ResetToken = r.decrypt(user.ResetToken)
	if resetExpiry.Valid {
		user.ResetExpiry = resetExpiry.Time
	}
	return user, nil
}

func (r *userRepo) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	var resetExpiry sql.NullTime
	query := `SELECT id, username, email, password, role, active, 
			  COALESCE(profile_picture_url, ''), COALESCE(currency, 'USD'), COALESCE(language, 'en'), COALESCE(date_format, 'YYYY-MM-DD'), 
			  COALESCE(notify_enabled, 1), COALESCE(monthly_savings_goal, 0), COALESCE(reset_token, ''), reset_expiry, min_token_issued_at, 
			  created_at, updated_at 
			  FROM users WHERE username = ?`
	err := r.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Active,
		&user.ProfilePictureURL, &user.Currency, &user.Language, &user.DateFormat, &user.NotifyEnabled, &user.MonthlySavingsGoal,
		&user.ResetToken, &resetExpiry, &user.MinTokenIssuedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	user.ResetToken = r.decrypt(user.ResetToken)
	if resetExpiry.Valid {
		user.ResetExpiry = resetExpiry.Time
	}
	return user, nil
}

func (r *userRepo) Update(user *models.User) error {
	query := `UPDATE users SET username = ?, email = ?, password = ?, role = ?, active = ?, profile_picture_url = ?, currency = ?, language = ?, date_format = ?, notify_enabled = ?, monthly_savings_goal = ?, reset_token = ?, reset_expiry = ?, min_token_issued_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	encToken := r.encrypt(user.ResetToken)

	_, err := r.db.Exec(query, user.Username, user.Email, user.Password, user.Role, user.Active,
		user.ProfilePictureURL, user.Currency, user.Language, user.DateFormat, user.NotifyEnabled,
		user.MonthlySavingsGoal, encToken, user.ResetExpiry, user.MinTokenIssuedAt, user.ID)
	return err
}

func (r *userRepo) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	var resetExpiry sql.NullTime
	query := `SELECT id, username, email, password, role, active, 
			  COALESCE(profile_picture_url, ''), COALESCE(currency, 'USD'), COALESCE(language, 'en'), COALESCE(date_format, 'YYYY-MM-DD'), 
			  COALESCE(notify_enabled, 1), COALESCE(monthly_savings_goal, 0), COALESCE(reset_token, ''), reset_expiry, min_token_issued_at, 
			  created_at, updated_at 
			  FROM users WHERE email = ?`
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Active,
		&user.ProfilePictureURL, &user.Currency, &user.Language, &user.DateFormat, &user.NotifyEnabled, &user.MonthlySavingsGoal,
		&user.ResetToken, &resetExpiry, &user.MinTokenIssuedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	user.ResetToken = r.decrypt(user.ResetToken)
	if resetExpiry.Valid {
		user.ResetExpiry = resetExpiry.Time
	}
	return user, nil
}

func (r *userRepo) Delete(id int64) error {
	query := `UPDATE users SET active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *userRepo) Recover(id int64) error {
	query := `UPDATE users SET active = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// HardDelete permanently removes a user from the database.
func (r *userRepo) HardDelete(id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *userRepo) ListAll() ([]*models.User, error) {
	query := `SELECT id, username, email, password, role, active, 
			  COALESCE(profile_picture_url, ''), COALESCE(currency, 'USD'), COALESCE(language, 'en'), COALESCE(date_format, 'YYYY-MM-DD'), 
			  COALESCE(notify_enabled, 1), COALESCE(monthly_savings_goal, 0), COALESCE(reset_token, ''), reset_expiry, min_token_issued_at, 
			  created_at, updated_at 
			  FROM users ORDER BY role ASC, username ASC`
	return r.listQuery(query)
}

func (r *userRepo) RevokeSessions(id int64) error {
	query := `UPDATE users SET min_token_issued_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *userRepo) listQuery(query string) ([]*models.User, error) {
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var resetExpiry sql.NullTime
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Active,
			&user.ProfilePictureURL, &user.Currency, &user.Language, &user.DateFormat, &user.NotifyEnabled, &user.MonthlySavingsGoal,
			&user.ResetToken, &resetExpiry, &user.MinTokenIssuedAt, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		user.ResetToken = r.decrypt(user.ResetToken)
		if resetExpiry.Valid {
			user.ResetExpiry = resetExpiry.Time
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
