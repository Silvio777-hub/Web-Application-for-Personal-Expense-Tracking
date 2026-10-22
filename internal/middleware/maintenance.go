package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Maintenance blocks all non-admin requests when maintenance mode is active.
func Maintenance(configRepo interface {
	Get(string) (string, error)
}, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mMode, _ := configRepo.Get("maintenance_mode")
			if mMode != "true" {
				next.ServeHTTP(w, r)
				return
			}

			// If maintenance is on, check if user is an admin
			tokenString := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				cookie, err := r.Cookie("auth_token")
				if err == nil {
					tokenString = cookie.Value
				}
			}

			if tokenString != "" {
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return []byte(jwtSecret), nil
				})

				if err == nil && token.Valid {
					claims, ok := token.Claims.(jwt.MapClaims)
					if ok {
						role, _ := claims["role"].(string)
						if role == "admin" {
							next.ServeHTTP(w, r)
							return
						}
					}
				}
			}

			// Not an admin or not logged in during maintenance
			http.Error(w, "System is undergoing maintenance. Please try again later.", http.StatusServiceUnavailable)
		})
	}
}
