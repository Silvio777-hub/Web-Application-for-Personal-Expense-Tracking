package models

import "time"

// User represents an application user with role-based access.
type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	Email              string    `json:"email"`
	Password           string    `json:"-"`
	Role               string    `json:"role"` // admin or user
	ProfilePictureURL  string    `json:"profile_picture_url,omitempty"`
	Currency           string    `json:"currency,omitempty"`
	Language           string    `json:"language,omitempty"`
	DateFormat         string    `json:"date_format,omitempty"`
	NotifyEnabled      bool      `json:"notify_enabled"`
	MonthlySavingsGoal float64   `json:"monthly_savings_goal"`
	ResetToken         string    `json:"-"`
	ResetExpiry        time.Time `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Active             bool      `json:"active"`
	MinTokenIssuedAt   time.Time `json:"-"`
}

// Category represents an expense category.
type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`      // Material Icon name or CSS class
	Color     string    `json:"color"`     // Hex color code
	Enabled   bool      `json:"enabled"`   // Admin can disable global categories
	IsGlobal  bool      `json:"is_global"` // True if added by admin
	UserID    int64     `json:"user_id"`   // Owner of the category
	CreatedAt time.Time `json:"created_at"`
}

// Expense represents a single financial transaction.
type Expense struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	CategoryID  int64     `json:"category_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description,omitempty"`
	Date        time.Time `json:"date"`
	ReceiptURL  string    `json:"receipt_url,omitempty"` // URL to uploaded receipt image
	CreatedAt   time.Time `json:"created_at"`
}

// Debt represents money owed to or by the user.
type Debt struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Person    string    `json:"person"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Type      string    `json:"type"` // owed_to_me, owed_by_me
	DueDate   time.Time `json:"due_date,omitempty"`
	Paid      bool      `json:"paid"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog tracks system and user activities for admins.
type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemConfig stores global application settings.
type SystemConfig struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
