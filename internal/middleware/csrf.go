package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

// CSRF middleware protects against Cross-Site Request Forgery.
func CSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from cookie
			cookie, err := r.Cookie(CSRFCookieName)
			var token string

			if err != nil {
				// No cookie? Generate one
				token = generateRandomToken(32)
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // Must be readable by JS to send back in header
					Secure:   false, // Set true for HTTPS
					SameSite: http.SameSiteStrictMode,
					MaxAge:   86400,
				})
			} else {
				token = cookie.Value
			}

			// 2. Validate on state-changing requests
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
				headerToken := r.Header.Get(CSRFHeaderName)
				if headerToken == "" || headerToken != token {
					http.Error(w, "CSRF token mismatch", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateRandomToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
