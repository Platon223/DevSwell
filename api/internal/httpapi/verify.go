package httpapi

import (
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
)

func verifyHandler(users *mongodb.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		user, err := users.FindByVerificationToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired verification link", http.StatusBadRequest)
			return
		}

		if err := users.MarkVerified(r.Context(), user.ID); err != nil {
			http.Error(w, "could not verify account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>Email confirmed</h1><p>Your DevSwell account is now active. You can log in.</p>"))
	}
}
