package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// extractUserIDFromSession validates the session cookie's JWT and returns the
// authenticated user's ID (the token's subject claim).
func extractUserIDFromSession(r *http.Request, jwtSecret []byte) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", errors.New("no session cookie")
	}

	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired session")
	}
	return claims.Subject, nil
}

// requireAuth is for JSON API endpoints: responds 401 if not authenticated.
func requireAuth(next http.HandlerFunc, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := extractUserIDFromSession(r, jwtSecret)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(withUserID(r.Context(), userID)))
	}
}

// requirePageAuth is for browser page routes: redirects to the landing page
// instead of returning a raw JSON error, since a page navigation should land
// somewhere useful rather than showing "401 Unauthorized" as plain text.
func requirePageAuth(next http.HandlerFunc, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := extractUserIDFromSession(r, jwtSecret)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next(w, r.WithContext(withUserID(r.Context(), userID)))
	}
}
