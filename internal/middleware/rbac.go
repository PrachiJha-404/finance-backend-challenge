package middleware

import (
	"net/http"

	"finance-backend-challenge/internal/apierr"
)

func RequireRoles(roles ...string) func(http.Handler) http.Handler {

	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {

				apierr.Write(w, apierr.Unauthorized(""))
				return
			}

			if _, ok := allowed[claims.Role]; !ok {
				apierr.Write(w, apierr.Forbidden(""))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AdminOnly() func(http.Handler) http.Handler {
	return RequireRoles("admin")
}

func AnalystAndAbove() func(http.Handler) http.Handler {
	return RequireRoles("analyst", "admin")
}

func AnyRole() func(http.Handler) http.Handler {
	return RequireRoles("viewer", "analyst", "admin")
}
