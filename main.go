package main

import (
	"expense-tracker/internal/api"
	"expense-tracker/internal/config"
	"expense-tracker/internal/database"
	"expense-tracker/internal/logger"
	"net/http"
	"os"
)

func main() {
	// Initialize Logger
	logger.Init("info", os.Stdout)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", err)
		os.Exit(1)
	}
	// Initialize SQLite database
	if err := database.Init(cfg.DBPath); err != nil {
		logger.Error("Database init failed", err)
		os.Exit(1)
	}
	router := api.NewRouter(cfg)
	logger.Info("Starting server [STRUCTURED LOGS ENABLED]", "port", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, router); err != nil {
		logger.Error("Server shutdown unexpectedly", err)
		os.Exit(1)
	}
}
