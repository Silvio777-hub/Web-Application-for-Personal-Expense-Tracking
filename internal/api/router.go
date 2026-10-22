package api

import (
	"context"
	"expense-tracker/internal/config"
	"expense-tracker/internal/database"
	"expense-tracker/internal/mail"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/ocr"
	"expense-tracker/internal/repository/sqlite"
	"expense-tracker/internal/service"
	"expense-tracker/internal/utils"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter sets up the HTTP router with routes and middleware.
func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Initialize repositories first to enable config-driven middleware
	kp := &utils.EnvKeyProvider{EnvVar: "MASTER_KEY"}
	userRepo := sqlite.NewUserRepository(database.DB, kp)
	expenseRepo := sqlite.NewExpenseRepository(database.DB)
	categoryRepo := sqlite.NewCategoryRepository(database.DB)
	debtRepo := sqlite.NewDebtRepository(database.DB)
	configRepo := sqlite.NewSystemConfigRepository(database.DB)
	auditRepo := sqlite.NewAuditLogRepository(database.DB)

	// Common middleware
	r.Use(chiMiddleware.RealIP) // Set correct IP early
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders) // Industry Standard Protection

	rl := middleware.NewRateLimiter(cfg.TrustedProxies)
	r.Use(rl.Limit("global", 60, 100)) // General throughput
	r.Use(middleware.NoCache)
	r.Use(middleware.Maintenance(configRepo, cfg.JWTSecret)) // Global Maintenance Lock
	r.Use(middleware.CSRF())                                 // End-to-end CSRF Protection
	r.Use(chiMiddleware.RequestID)

	// Initialize Services & Mailer
	var mailer mail.Mailer
	if cfg.SMTPUser != "" && cfg.SMTPPass != "" {
		mailer = mail.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	} else {
		mailer = &mail.LogMailer{}
	}

	userService := service.NewUserService(userRepo, mailer, cfg.JWTSecret)
	expenseService := service.NewExpenseService(expenseRepo)
	finService := service.NewFinancialService(expenseRepo, debtRepo)
	ocrProc := &ocr.MockProcessor{ErrorRate: 0.02} // 2% failure rate for realism

	// API Routes
	r.Route("/api", func(r chi.Router) {
		// Public Auth
		r.Route("/auth", func(r chi.Router) {
			r.Use(rl.Limit("auth", 5, 10))
			r.Post("/login", loginHandler(userService))
			r.Post("/logout", logoutHandler())
			r.Post("/reset-request", requestResetHandler(userRepo, mailer, cfg.AppBaseURL))
			r.Post("/reset-verify", verifyResetHandler(userRepo))
		})

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg.JWTSecret, userRepo, configRepo))
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), "auditRepo", auditRepo)
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})

			// User & Profile
			r.Route("/user", func(r chi.Router) {
				r.Put("/profile", updateProfileHandler(userRepo))
				r.Post("/upload-pic", uploadProfilePictureHandler(userRepo))
				r.Post("/change-password", changePasswordHandler(userRepo))
				r.Put("/preferences", updatePreferencesHandler(userRepo))
			})

			// Admin Operations
			r.Route("/admin", func(r chi.Router) {
				r.Post("/register", registerHandler(userService)) // Moving registration under admin
				r.Get("/users", listUsersHandler(userRepo, true))
				r.Delete("/users", deleteUserHandler(userRepo))
				r.Delete("/users/hard-delete", hardDeleteUserHandler(userRepo))
				r.Get("/stats", getSystemStatsHandler(expenseRepo, userRepo))
				r.Get("/configs", listConfigsHandler(configRepo))
				r.Post("/configs", updateConfigHandler(configRepo))
				r.Post("/users/revoke-sessions", revokeSessionsHandler(userRepo))
				r.Post("/users/impersonate", impersonateHandler(userService))
				r.Post("/users/recover", recoverUserHandler(userRepo))
				r.Get("/audit-logs", listAuditLogsHandler(auditRepo))
				r.Delete("/audit-logs", clearAuditLogsHandler(auditRepo))
				r.Post("/system/backup", systemBackupHandler())
				r.Delete("/expenses/all", clearExpensesHandler(expenseService))
				r.Delete("/expenses/hard", hardDeleteExpenseHandler(expenseService))
			})

			// Expenses & Finance
			r.Route("/expenses", func(r chi.Router) {
				r.Get("/", listExpensesHandler(expenseService))
				r.Get("/export", exportExpensesHandler(expenseService, categoryRepo))
				r.Post("/", createExpenseHandler(expenseService))
				r.Put("/", updateExpenseHandler(expenseService))
				r.Delete("/", deleteExpenseHandler(expenseService))
				r.Delete("/bulk", bulkDeleteHandler(expenseService))
				r.Get("/trash", listDeletedExpensesHandler(expenseService))
				r.Post("/restore", restoreExpenseHandler(expenseService))
				r.Post("/upload-receipt", uploadReceiptHandler(userRepo, ocrProc))
			})

			r.Get("/debts", listDebtsHandler(finService))
			r.Get("/categories", listCategoriesHandler(categoryRepo))
			r.Get("/exchange-rates/{base}", getExchangeRatesHandler())

			// Assets
			r.Get("/assets/{type}/{filename}", serveAssetHandler(expenseRepo, userRepo))
		})
	})

	// Static files
	fileServer := http.FileServer(http.Dir("./public"))
	r.Handle("/css/*", fileServer)
	r.Handle("/js/*", fileServer)
	r.Handle("/favicon.ico", fileServer)

	// Main Entry Point
	r.Get("/", indexHandler)

	return r
}
