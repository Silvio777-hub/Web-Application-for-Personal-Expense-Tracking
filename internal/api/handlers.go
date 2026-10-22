package api

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"expense-tracker/internal/database"
	"expense-tracker/internal/logger"
	"expense-tracker/internal/mail"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/ocr"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/service"
	"expense-tracker/internal/utils"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token      string       `json:"token"`
	User       *models.User `json:"user"`
	IsReadOnly bool         `json:"is_read_only"` // Flag for session transparency
}

func loginHandler(us *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB Limit
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("Login decode failed", "error", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		token, user, err := us.Authenticate(req.Username, req.Password)
		if err != nil {
			logger.Warn("Authentication failed", "username", req.Username, "error", err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		logger.Info("Login successful", "username", req.Username)

		// Set JWT in HttpOnly cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true if using HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		jsonResponse(w, http.StatusOK, authResponse{
			Token:      token,
			User:       user,
			IsReadOnly: false,
		})
	}
}

func logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusOK)
	}
}

func registerHandler(us *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Only administrators can register new users", http.StatusForbidden)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB Limit
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		v := utils.NewValidator()
		v.Required(req.Username, "username")
		v.MinLength(req.Username, "username", 3)
		v.Required(req.Email, "email")
		v.Email(req.Email, "email")
		v.Required(req.Password, "password")
		v.MinLength(req.Password, "password", 6)

		if !v.Valid() {
			errorResponse(w, v.Error(), http.StatusBadRequest)
			return
		}

		user := &models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		}

		if err := us.Register(user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, http.StatusCreated, user)
	}
}

// listExpensesHandler retrieves expenses with optional filtering.
func listExpensesHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		filter := r.URL.Query().Get("filter")
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		queryStr := r.URL.Query().Get("q")
		minAmount := r.URL.Query().Get("min_amount")
		maxAmount := r.URL.Query().Get("max_amount")
		all := r.URL.Query().Get("all")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		searchID := userID
		if all == "true" && isAdmin(r) {
			searchID = 0
		}

		expenses, err := es.SearchExpenses(searchID, filter, startDate, endDate, queryStr, minAmount, maxAmount, limit, offset)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, expenses)
	}
}

// createExpenseHandler processes new expense submissions.
func createExpenseHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB Limit
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)

		var req struct {
			CategoryID  int64   `json:"category_id"`
			Amount      float64 `json:"amount"`
			Currency    string  `json:"currency"`
			Description string  `json:"description"`
			Date        string  `json:"date"`
			ReceiptURL  string  `json:"receipt_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		v := utils.NewValidator()
		v.Check(req.Amount > 0, "amount", "must be greater than zero")
		v.Check(req.CategoryID > 0, "category_id", "invalid category")
		v.Required(req.Date, "date")
		v.MaxLength(req.Description, "description", 500)

		if !v.Valid() {
			errorResponse(w, v.Error(), http.StatusBadRequest)
			return
		}

		date, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			date, err = time.Parse("2006-01-02", req.Date)
			if err != nil {
				date = time.Now()
			}
		}

		exp := models.Expense{
			UserID:      userID,
			CategoryID:  req.CategoryID,
			Amount:      req.Amount,
			Currency:    req.Currency,
			Description: req.Description,
			Date:        date,
			ReceiptURL:  req.ReceiptURL,
		}

		if err := es.AddExpense(&exp); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, exp)
	}
}

func updateExpenseHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB Limit
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)

		var req struct {
			ID          int64   `json:"id"`
			CategoryID  int64   `json:"category_id"`
			Amount      float64 `json:"amount"`
			Currency    string  `json:"currency"`
			Description string  `json:"description"`
			Date        string  `json:"date"`
			ReceiptURL  string  `json:"receipt_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "Amount must be positive", http.StatusBadRequest)
			return
		}

		// IDOR Protection: Verify ownership BEFORE updating
		existing, err := es.GetExpenseByID(req.ID)
		if err != nil {
			http.Error(w, "expense not found", http.StatusNotFound)
			return
		}
		if existing.UserID != userID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		date, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			date, err = time.Parse("2006-01-02", req.Date)
			if err != nil {
				date = time.Now()
			}
		}

		exp := models.Expense{
			ID:          req.ID,
			UserID:      userID,
			CategoryID:  req.CategoryID,
			Amount:      req.Amount,
			Currency:    req.Currency,
			Description: req.Description,
			Date:        date,
			ReceiptURL:  req.ReceiptURL,
		}

		if err := es.UpdateExpense(&exp); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, exp)
	}
}

func deleteExpenseHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		userID := r.Context().Value(middleware.UserIDKey).(int64)

		// IDOR Protection: Verify ownership or Admin role
		if !isAdmin(r) {
			exp, err := es.GetExpenseByID(req.ID)
			if err != nil {
				http.Error(w, "expense not found", http.StatusNotFound)
				return
			}
			if exp.UserID != userID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if err := es.DeleteExpense(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "expense deleted"})
	}
}

func bulkDeleteHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		userID := r.Context().Value(middleware.UserIDKey).(int64)
		deleteID := userID
		if isAdmin(r) {
			deleteID = 0 // Allow admin to delete any ID
		}

		if err := es.BulkDeleteExpenses(deleteID, req.IDs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "expenses deleted"})
	}
}

func listDeletedExpensesHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		var expenses []*models.Expense
		var err error

		if isAdmin(r) {
			expenses, err = es.ListAllDeletedExpenses()
		} else {
			expenses, err = es.ListDeletedExpenses(userID)
		}

		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, expenses)
	}
}

func restoreExpenseHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Security Check: Ensure user owns the expense before restoring, unless admin
		if !isAdmin(r) {
			exp, err := es.GetExpenseByID(req.ID)
			if err != nil {
				http.Error(w, "expense not found", http.StatusNotFound)
				return
			}
			if exp.UserID != userID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if err := es.RestoreExpense(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "expense restored"})
	}
}

func exportExpensesHandler(es *service.ExpenseService, cr repository.CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		all := r.URL.Query().Get("all") == "true"

		var expenses []*models.Expense
		var err error

		if all && isAdmin(r) {
			expenses, err = es.ListAllExpenses() // I need to ensure this is in service
		} else {
			expenses, err = es.SearchExpenses(userID, "", "", "", "", "", "", 0, 0)
		}

		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		// Fetch categories to map IDs to names
		categories, err := cr.List(userID)
		if err != nil {
			http.Error(w, "failed to fetch categories", http.StatusInternalServerError)
			return
		}
		catMap := make(map[int64]string)
		for _, cat := range categories {
			catMap[cat.ID] = cat.Name
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment;filename=expenses.csv")

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Write Header
		_ = writer.Write([]string{"Date", "Category", "Description", "Amount", "Currency"})

		for _, e := range expenses {
			catName := catMap[e.CategoryID]
			if catName == "" {
				catName = fmt.Sprintf("Unknown (%d)", e.CategoryID)
			}

			// Sanitize description for CSV Injection
			desc := e.Description
			if len(desc) > 0 && (desc[0] == '=' || desc[0] == '+' || desc[0] == '-' || desc[0] == '@') {
				desc = "'" + desc
			}

			_ = writer.Write([]string{
				e.Date.Format("2006-01-02"),
				catName,
				desc,
				fmt.Sprintf("%.2f", e.Amount),
				e.Currency,
			})
		}
	}
}

func listCategoriesHandler(repo repository.CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		categories, err := repo.List(userID)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, categories)
	}
}

// getForecastHandler removed as requested

func listDebtsHandler(fs *service.FinancialService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		debts, err := fs.ListDebts(userID)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, debts)
	}
}

func listUsersHandler(repo repository.UserRepository, all bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		users, err := repo.ListAll()

		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, users)
	}
}

func systemBackupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Create backups directory
		_ = os.MkdirAll("backups", 0755)

		filename := fmt.Sprintf("backup_%s.db", time.Now().Format("20060102_150405"))
		path := filepath.Join("backups", filename)

		if err := database.Backup(path); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "SYSTEM_BACKUP", "Manual database backup created: "+filename)
		jsonResponse(w, http.StatusOK, map[string]string{
			"message":  "Backup created successfully",
			"filename": filename,
		})
	}
}

// hardDeleteUserHandler handles permanent deletion of a user.
func hardDeleteUserHandler(userRepo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)

		if !isAdmin(r) && req.ID != userID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if err := userRepo.HardDelete(req.ID); err != nil {
			http.Error(w, "failed to hard delete user", http.StatusInternalServerError)
			return
		}

		// Log audit
		if ar, ok := r.Context().Value("auditRepo").(repository.AuditLogRepository); ok {
			details := fmt.Sprintf("Hard deleted user (ID: %d)", req.ID)
			if strings.HasPrefix(r.Header.Get("X-Audit-Reason"), "SELF_DELETE") {
				details = "User permanently deleted their own account"
			}
			ar.Create(&models.AuditLog{
				UserID:  userID,
				Action:  "USER_HARD_DELETE",
				Details: details,
			})
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "user permanently deleted"})
	}
}

func deleteUserHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		currentUserID := r.Context().Value(middleware.UserIDKey).(int64)
		if req.ID == currentUserID {
			http.Error(w, "cannot delete yourself", http.StatusBadRequest)
			return
		}

		if err := repo.Delete(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Log action
		if ar, ok := r.Context().Value("auditRepo").(repository.AuditLogRepository); ok {
			userID := r.Context().Value(middleware.UserIDKey).(int64)
			var username string
			if user, err := repo.GetByID(req.ID); err == nil {
				username = user.Username
			} else {
				username = "Unknown"
			}
			ar.Create(&models.AuditLog{
				UserID:  userID,
				Action:  "USER_DELETE",
				Details: fmt.Sprintf("Deleted user account: %s (ID: %d)", username, req.ID),
			})
		}
		jsonResponse(w, http.StatusOK, map[string]string{"message": "user deleted"})
	}
}

func recoverUserHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := repo.Recover(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Log action
		if ar, ok := r.Context().Value("auditRepo").(repository.AuditLogRepository); ok {
			userID := r.Context().Value(middleware.UserIDKey).(int64)
			user, _ := repo.GetByID(req.ID)
			ar.Create(&models.AuditLog{
				UserID:  userID,
				Action:  "USER_RESTORE",
				Details: fmt.Sprintf("Retrieved user account: %s (ID: %d)", user.Username, req.ID),
			})
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "user recovered"})
	}
}

func updateProfileHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB Limit
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		user, err := repo.GetByID(userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		var req struct {
			Email             string `json:"email"`
			ProfilePictureURL string `json:"profile_picture_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if req.Email != "" {
			user.Email = req.Email
		}
		if req.ProfilePictureURL != "" {
			user.ProfilePictureURL = req.ProfilePictureURL
		}

		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, user)
	}
}

func uploadProfilePictureHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey).(int64)

		// Parse multipart form (30MB limit)
		if err := r.ParseMultipartForm(30 << 20); err != nil {
			logger.Warn("Profile upload failed: File too large", "error", err)
			http.Error(w, "file too large (max 30MB)", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("profile_pic")
		if err != nil {
			http.Error(w, "invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
			http.Error(w, "unsupported file format", http.StatusBadRequest)
			return
		}

		// Magic Byte Verification (Deep Inspection)
		buffer := make([]byte, 512)
		if _, err := file.Read(buffer); err != nil {
			http.Error(w, "failed to read file", http.StatusInternalServerError)
			return
		}
		if _, err := file.Seek(0, 0); err != nil {
			http.Error(w, "failed to reset file pointer", http.StatusInternalServerError)
			return
		}

		contentType := http.DetectContentType(buffer)
		if !strings.HasPrefix(contentType, "image/") {
			logger.Warn("Malicious upload attempt: Magic bytes don't match image extension", "detected", contentType)
			http.Error(w, "invalid image content", http.StatusBadRequest)
			return
		}

		// Create unique filename and sanitize for Path Traversal
		filename := time.Now().Format("20060102150405") + "_" + filepath.Base(header.Filename)
		dir := filepath.Join("uploads", "profiles")
		os.MkdirAll(dir, 0755)
		savePath := filepath.Join(dir, filename)

		out, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "failed to save file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, "failed to copy file", http.StatusInternalServerError)
			return
		}

		// Update user record
		user, err := repo.GetByID(userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		user.ProfilePictureURL = "/api/assets/profiles/" + filename
		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"url": user.ProfilePictureURL})
	}
}

func changePasswordHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		var req struct {
			Current string `json:"current"`
			New     string `json:"new"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		user, err := repo.GetByID(userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		// Verify current
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Current)); err != nil {
			http.Error(w, "incorrect current password", http.StatusUnauthorized)
			return
		}

		// Hash and update
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.New), service.BcryptCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		user.Password = string(hashed)

		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{"message": "password updated"})
	}
}

func updatePreferencesHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isReadOnly(r) {
			http.Error(w, "Account is in READ-ONLY mode", http.StatusForbidden)
			return
		}
		userID := r.Context().Value(middleware.UserIDKey).(int64)
		var req struct {
			Currency           string  `json:"currency"`
			NotifyEnabled      bool    `json:"notify_enabled"`
			MonthlySavingsGoal float64 `json:"monthly_savings_goal"`
			Language           string  `json:"language"`
			DateFormat         string  `json:"date_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		user, err := repo.GetByID(userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		user.Currency = req.Currency
		user.NotifyEnabled = req.NotifyEnabled
		user.MonthlySavingsGoal = req.MonthlySavingsGoal
		user.Language = req.Language
		user.DateFormat = req.DateFormat

		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, user)
	}
}

func getSystemStatsHandler(er repository.ExpenseRepository, ur repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		users, err := ur.ListAll()
		if err != nil {
			http.Error(w, "failed to fetch users", http.StatusInternalServerError)
			return
		}

		totalSpent, err := er.GetGlobalTotalSpent()
		if err != nil {
			http.Error(w, "failed to fetch total stats", http.StatusInternalServerError)
			return
		}

		perUser, err := er.GetSpendingPerUser()
		if err != nil {
			// Log error but continue with empty list if granular stats fail
			log.Printf("Error fetching per-user spending: %v\n", err)
			perUser = []repository.UserSpending{}
		}

		stats := map[string]interface{}{
			"total_users":    len(users),
			"total_expenses": totalSpent,
			"user_breakdown": perUser,
			"health":         "Healthy",
		}
		jsonResponse(w, http.StatusOK, stats)
	}
}

func listAuditLogsHandler(repo repository.AuditLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		logs, err := repo.List(50)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, logs)
	}
}

func listConfigsHandler(repo repository.SystemConfigRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		configs, err := repo.List()
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, configs)
	}
}

func updateConfigHandler(repo repository.SystemConfigRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var cfg struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		err := repo.Set(cfg.Key, cfg.Value)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "CONFIG_UPDATE", fmt.Sprintf("Updated config %s to %s", cfg.Key, cfg.Value))
		w.WriteHeader(http.StatusOK)
	}
}

func revokeSessionsHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			UserID int64 `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		err := repo.RevokeSessions(req.UserID)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "REVOKE_SESSIONS", fmt.Sprintf("Revoked all sessions for user %d", req.UserID))
		w.WriteHeader(http.StatusOK)
	}
}

func impersonateHandler(us *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			UserID int64 `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		token, err := us.Impersonate(req.UserID)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "USER_IMPERSONATION", fmt.Sprintf("Admin impersonating user %d", req.UserID))

		jsonResponse(w, http.StatusOK, map[string]string{"token": token})
	}
}

func requestResetHandler(repo repository.UserRepository, mailer mail.Mailer, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		user, err := repo.GetByEmail(req.Email)
		if err != nil {
			// SECURITY: Timing Mitigation & User Enumeration Protection
			// Simulate timing variance (300ms + random 0-200ms)
			// Using crypto/rand since math/rand isn't imported
			b := make([]byte, 1)
			rand.Read(b)
			sleepTime := 300 + int(b[0]%200)
			time.Sleep(time.Millisecond * time.Duration(sleepTime))

			// Return 200 OK with the SAME GENERIC MESSAGE to confuse attackers
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Please check your Email Inbox or Spam to Create your New Password"})
			return
		}

		// Generate secure token (32 chars)
		tokenBytes := make([]byte, 16)
		rand.Read(tokenBytes)
		token := fmt.Sprintf("%x", tokenBytes)

		user.ResetToken = token
		user.ResetExpiry = time.Now().Add(1 * time.Hour)

		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send Real Email
		resetURL := fmt.Sprintf("%s/?token=%s", baseURL, token) // Use trusted baseURL
		subject := "Secure Password Recovery"
		body := fmt.Sprintf("Hello %s,\n\nA password reset request was made for your account. Please click the link below to set a new password:\n\n%s\n\nThis link will expire in 1 hour.", user.Username, resetURL)

		if err := mailer.Send(user.Email, subject, body); err != nil {
			log.Printf("Failed to send recovery email: %v", err)
			// Still return 200 so we don't reveal failure or user existence
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Please check your Email Inbox or Spam to Create your New Password"})
	}
}

func verifyResetHandler(repo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token    string `json:"token"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Find user by token instead of email to verify the link authenticity
		// In a real system we'd use a dedicated repository method,
		// but for now we list or query by token if implemented.
		// For simplicity in this demo environment, we'll iterate.
		users, _ := repo.ListAll()
		var targetUser *models.User
		for _, u := range users {
			if u.ResetToken == req.Token && u.ResetToken != "" {
				targetUser = u
				break
			}
		}

		if targetUser == nil || time.Now().After(targetUser.ResetExpiry) {
			http.Error(w, "invalid or expired recovery link", http.StatusUnauthorized)
			return
		}
		user := targetUser

		// Hash and update
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), service.BcryptCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		user.Password = string(hashed)
		user.ResetToken = "" // Clear token
		user.ResetExpiry = time.Time{}

		if err := repo.Update(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
	}
}

func isAdmin(r *http.Request) bool {
	role := r.Context().Value(middleware.UserRoleKey)
	return role == "admin"
}

func isReadOnly(r *http.Request) bool {
	ro, ok := r.Context().Value(middleware.IsReadOnlyKey).(bool)
	return ok && ro
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		filepath.Join("templates", "layout.html"),
		filepath.Join("templates", "index.html"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Timestamp int64
	}{
		Timestamp: time.Now().UnixNano(),
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func uploadReceiptHandler(repo repository.UserRepository, proc ocr.Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form (30MB limit)
		if err := r.ParseMultipartForm(30 << 20); err != nil {
			logger.Warn("Receipt upload failed: File too large", "error", err)
			http.Error(w, "file too large (max 30MB)", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("receipt")
		if err != nil {
			http.Error(w, "invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".pdf" {
			http.Error(w, "unsupported file format", http.StatusBadRequest)
			return
		}

		// Magic Byte Verification (Deep Inspection)
		buffer := make([]byte, 512)
		if _, err := file.Read(buffer); err != nil {
			http.Error(w, "failed to read file", http.StatusInternalServerError)
			return
		}
		if _, err := file.Seek(0, 0); err != nil {
			http.Error(w, "failed to reset file pointer", http.StatusInternalServerError)
			return
		}

		contentType := http.DetectContentType(buffer)
		if !strings.HasPrefix(contentType, "image/") && contentType != "application/pdf" {
			logger.Warn("Malicious upload attempt: Magic bytes don't match receipt extension", "detected", contentType)
			http.Error(w, "invalid file content", http.StatusBadRequest)
			return
		}

		// Create unique filename and sanitize for Path Traversal
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(header.Filename))
		// Ensure directory exists and define save path
		dir := filepath.Join("uploads", "receipts")
		os.MkdirAll(dir, 0755)
		savePath := filepath.Join(dir, filename)

		out, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "failed to save file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, "failed to copy file", http.StatusInternalServerError)
			return
		}

		// Run OCR Processing (Simulated/Real)
		ocrData, err := proc.Process(r.Context(), savePath)
		if err != nil {
			logger.Warn("OCR Processing failed", "path", savePath, "error", err)
			// Don't fail the whole request, just return empty data
			ocrData = &ocr.OCRResult{}
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"url":      "/api/assets/receipts/" + filename,
			"ai_data":  ocrData,
			"metadata": header.Filename,
		})
	}
}

func serveAssetHandler(er repository.ExpenseRepository, ur repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetType := chi.URLParam(r, "type")
		filename := filepath.Base(chi.URLParam(r, "filename"))
		userID := r.Context().Value(middleware.UserIDKey).(int64)

		dir := ""
		switch assetType {
		case "receipts":
			dir = filepath.Join("uploads", "receipts")
			// Optimized Security Check: Verify Ownership via DB index
			targetURL := "/api/assets/receipts/" + filename
			found, err := er.ExistsByReceiptURL(userID, targetURL)
			if err != nil {
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			if !found && !isAdmin(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		case "profiles":
			dir = filepath.Join("uploads", "profiles")
			// Users can only view their own profile picture through this API
			// (Admins can view anyone's)
			user, err := ur.GetByID(userID)
			if err != nil || user == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// Extract filename from user's record for comparison
			userFile := filepath.Base(user.ProfilePictureURL)
			if userFile != filename && !isAdmin(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		default:
			http.Error(w, "invalid asset type", http.StatusBadRequest)
			return
		}

		filePath := filepath.Join(dir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		if r.URL.Query().Get("download") == "true" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		http.ServeFile(w, r, filePath)
	}
}

func jsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func errorResponse(w http.ResponseWriter, err error, code int) {
	logger.Error("API Error response", err, "code", code)
	// For client errors (4xx), return the actual error message
	if code < 500 {
		http.Error(w, err.Error(), code)
	} else {
		http.Error(w, http.StatusText(code), code) // Hide internal server errors (5xx)
	}
}

func auditLog(r *http.Request, action, details string) {
	auditRepo, ok := r.Context().Value("auditRepo").(repository.AuditLogRepository)
	if !ok {
		return
	}
	userID, _ := r.Context().Value(middleware.UserIDKey).(int64)
	auditRepo.Create(&models.AuditLog{
		UserID:  userID,
		Action:  action,
		Details: details,
	})
}

func clearAuditLogsHandler(repo repository.AuditLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if err := repo.ClearAll(); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "AUDIT_CLEAR", "All audit logs cleared by administrator")
		jsonResponse(w, http.StatusOK, map[string]string{"message": "Audit logs cleared"})
	}
}

func clearExpensesHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if err := es.ClearAllExpenses(); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "EXPENSES_CLEAR", "All expenses cleared by administrator")
		jsonResponse(w, http.StatusOK, map[string]string{"message": "All expenses cleared"})
	}
}

func hardDeleteExpenseHandler(es *service.ExpenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if err := es.HardDeleteExpense(req.ID); err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}

		auditLog(r, "EXPENSE_HARD_DELETE", fmt.Sprintf("Expense ID %d permanently deleted by administrator", req.ID))
		jsonResponse(w, http.StatusOK, map[string]string{"message": "Expense permanently deleted"})
	}
}

func getExchangeRatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := chi.URLParam(r, "base")
		if base == "" {
			base = "USD"
		}

		// Use a client with a timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		req, err := http.NewRequestWithContext(r.Context(), "GET", fmt.Sprintf("https://open.er-api.com/v6/latest/%s", base), nil)
		if err != nil {
			logger.Error("Failed to create rate request", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Set a standard User-Agent to avoid early blocks
		req.Header.Set("User-Agent", "PersonalExpenseTracker/1.0")

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("Failed to fetch exchange rates via proxy", err)
			http.Error(w, "failed to fetch exchange rates", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Warn("External rate API returned error", "status", resp.Status)
			http.Error(w, "external api error", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
	}
}
