package httpapi

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// requireAuth reads the session cookie, verifies the JWT, and puts the
// authenticated user's ID in the request context before calling next.
func requireAuth(next http.HandlerFunc, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		claims := &authClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired session", http.StatusUnauthorized)
			return
		}

		ctx := withUserID(r.Context(), claims.Subject)
		next(w, r.WithContext(ctx))
	}
}
