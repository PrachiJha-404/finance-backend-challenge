package middleware

import (
	"net/http"

	"finance-backend/internal/apierr"
	"finance-backend/internal/domain/user"
)

func RequireRoles(roles ...user.Role) func(http.Handler) http.Handler {

	allowed := make(map[user.Role]struct{}, len(roles))
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
	return RequireRoles(user.RoleAdmin)
}

func AnalystAndAbove() func(http.Handler) http.Handler {
	return RequireRoles(user.RoleAnalyst, user.RoleAdmin)
}

func AnyRole() func(http.Handler) http.Handler {
	return RequireRoles(user.RoleViewer, user.RoleAnalyst, user.RoleAdmin)
}
