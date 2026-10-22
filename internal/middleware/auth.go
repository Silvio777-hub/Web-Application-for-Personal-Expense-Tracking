package middleware

import (
	"context"
	"expense-tracker/internal/models"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey     contextKey = "user_id"
	UserRoleKey   contextKey = "user_role"
	IsReadOnlyKey contextKey = "is_read_only"
)

// Authenticate validates the JWT in the Authorization header.
func Authenticate(jwtSecret string, userRepo interface {
	GetByID(int64) (*models.User, error)
}, configRepo interface {
	Get(string) (string, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authenticate validates the JWT in the Authorization header.
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

			if tokenString == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid claims", http.StatusUnauthorized)
				return
			}

			userIDF, ok := claims["user_id"].(float64)
			if !ok {
				http.Error(w, "User ID not found in token", http.StatusUnauthorized)
				return
			}
			userID := int64(userIDF)

			role, _ := claims["role"].(string)
			iat, _ := claims["iat"].(float64)
			isReadOnly, _ := claims["is_read_only"].(bool)

			// 2. Check Session Revocation (Revoke sessions if iat < min_token_issued_at)
			if userRepo != nil {
				user, err := userRepo.GetByID(userID)
				if err == nil && user != nil && !user.MinTokenIssuedAt.IsZero() {
					if int64(iat) < user.MinTokenIssuedAt.Unix() {
						http.Error(w, "Session expired. Please log in again.", http.StatusUnauthorized)
						return
					}
				}
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, role)
			ctx = context.WithValue(ctx, IsReadOnlyKey, isReadOnly)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
