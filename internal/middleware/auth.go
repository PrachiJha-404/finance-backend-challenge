package middleware

import (
	"context"
	"net/http"
	"strings"

	"finance-backend-challenge/internal/apierr"
)

type contextKey string

const userContextKey contextKey = "user"

type Claims struct {
	UserID int
	Email  string
	Role   string
}

func ClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(userContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

func Authenticate(userService interface {
	ValidateToken(string) (*struct {
		ID    int
		Email string
		Role  string
	}, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				apierr.Write(w, apierr.Unauthorized("missing authorization header"))
				return
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				apierr.Write(w, apierr.Unauthorized("invalid authorization header format"))
				return
			}

			token := tokenParts[1]
			u, err := userService.ValidateToken(token)
			if err != nil {
				apierr.Write(w, apierr.Unauthorized("invalid token"))
				return
			}

			claims := &Claims{
				UserID: u.ID,
				Email:  u.Email,
				Role:   u.Role,
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
