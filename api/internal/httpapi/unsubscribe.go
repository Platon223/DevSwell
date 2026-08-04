package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// unsubscribeHandler is reached via the link in every stack-update digest
// email (no auth cookie required — the recipient may not be logged in when
// they click it). Unlike VerificationToken/DeletionToken, the token here is
// never cleared after use, since the same link is reused across every future
// email until the user re-subscribes from the dashboard.
func unsubscribeHandler(users *mongodb.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		user, err := users.FindByUnsubscribeToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid unsubscribe link", http.StatusBadRequest)
			return
		}

		if err := users.SetEmailNotificationsEnabled(r.Context(), user.ID, false); err != nil {
			http.Error(w, "could not process request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>Unsubscribed</h1><p>You won't receive any more DevSwell stack-update emails. You can turn them back on anytime from your dashboard's Settings page.</p>"))
	}
}

type preferencesRequest struct {
	EmailNotificationsEnabled bool `json:"email_notifications_enabled"`
}

// updatePreferencesHandler lets a logged-in user re-enable notifications
// after unsubscribing (or turn them off without needing an email link).
func updatePreferencesHandler(users *mongodb.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDHex, ok := userIDFromContext(r.Context())
		if !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		userID, err := bson.ObjectIDFromHex(userIDHex)
		if err != nil {
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}

		var req preferencesRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := users.SetEmailNotificationsEnabled(r.Context(), userID, req.EmailNotificationsEnabled); err != nil {
			http.Error(w, "could not update preferences", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
