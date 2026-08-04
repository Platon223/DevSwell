package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type requestDeletionResponse struct {
	Message string `json:"message"`
}

// requestDeletionHandler generates a one-time deletion token, stores it on the
// user's record, and emails a confirmation link. The account is not deleted
// until that link is clicked (confirmDeletionHandler).
func requestDeletionHandler(users *mongodb.UserStore, mailer *email.Client, baseURL string) http.HandlerFunc {
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

		user, err := users.FindByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		token, err := generateToken()
		if err != nil {
			http.Error(w, "could not process request", http.StatusInternalServerError)
			return
		}

		if err := users.SetDeletionToken(r.Context(), userID, token); err != nil {
			http.Error(w, "could not process request", http.StatusInternalServerError)
			return
		}

		confirmLink := fmt.Sprintf("%s/me/delete/confirm?token=%s", baseURL, token)
		if err := mailer.Send(r.Context(), user.Email, "Confirm DevSwell account deletion", confirmDeletionHTML(baseURL, confirmLink)); err != nil {
			log.Printf("sending deletion confirmation email to %s: %v", user.Email, err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requestDeletionResponse{Message: "check your email to confirm account deletion"})
	}
}

// confirmDeletionHandler is reached via the emailed link (no auth cookie
// required, since it's clicked from the user's mail client) and permanently
// deletes the account.
func confirmDeletionHandler(users *mongodb.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		user, err := users.FindByDeletionToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired deletion link", http.StatusBadRequest)
			return
		}

		if err := users.Delete(r.Context(), user.ID); err != nil {
			http.Error(w, "could not delete account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>Account deleted</h1><p>Your DevSwell account and all associated data have been permanently deleted.</p>"))
	}
}

func confirmDeletionHTML(baseURL, confirmLink string) string {
	body := `<p style="font-size:14px;color:#3a3a3a;line-height:1.6;margin:0 0 8px;">We received a request to delete your DevSwell account. This action is permanent and cannot be undone.</p>` +
		`<p style="font-size:14px;color:#3a3a3a;line-height:1.6;margin:0 0 20px;">If you did not request this, you can safely ignore this email.</p>` +
		email.Button(confirmLink, "Permanently Delete My Account", "#cf222e")
	return email.BrandedHTML(baseURL+"/static/logo-mark.png", "Confirm account deletion", body)
}
