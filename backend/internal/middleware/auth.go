package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"task-manager/internal/service"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			if header == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing token")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
			if tokenString == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing token")
				return
			}

			userID, err := authService.ParseToken(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) int64 {
	userID, _ := r.Context().Value(UserIDKey).(int64)
	return userID
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		return
	}
}
