package api

import (
	"bytes"
	"encoding/json"
	"expense-tracker/internal/config"
	"expense-tracker/internal/database"
	"expense-tracker/internal/logger"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup: Init logger for tests
	logger.Init("info", os.Stdout)

	// Create in-memory DB for all api tests
	database.Init(":memory:")

	code := m.Run()
	os.Exit(code)
}

func TestLoginHandler(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}
	router := NewRouter(cfg)

	// 1. Get initial CSRF token by making a GET request
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)

	var csrfToken string
	for _, cookie := range getRR.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfToken = cookie.Value
		}
	}

	// 2. Try login with non-existent user + CSRF token
	loginReq := loginRequest{
		Username: "nonexistent",
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
