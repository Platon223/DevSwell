package httpapi

import (
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type supportRequest struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type supportResponse struct {
	Message string `json:"message"`
}

func supportHandler(users *mongodb.UserStore, mailer *email.Client, supportEmail string) http.HandlerFunc {
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

		var req supportRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}
		if req.Category == "" {
			req.Category = "Other"
		}
		if supportEmail == "" {
			http.Error(w, "support is not configured yet", http.StatusServiceUnavailable)
			return
		}

		subject := fmt.Sprintf("[DevSwell Support] %s from %s", req.Category, user.Email)
		body := supportEmailHTML(user.Email, req.Category, req.Message)
		if err := mailer.Send(r.Context(), supportEmail, subject, body); err != nil {
			http.Error(w, "could not send your message, please try again", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(supportResponse{Message: "Thanks — we got your message and will get back to you soon."})
	}
}

// supportEmailHTML escapes all user-supplied fields since this becomes the
// HTML body of a real email opened in a real client — unescaped input here
// would be an HTML/script injection vector into the recipient's inbox.
func supportEmailHTML(fromEmail, category, message string) string {
	return fmt.Sprintf(`
<div style="font-family: sans-serif; padding: 24px;">
  <h2>New support message</h2>
  <p><strong>From:</strong> %s</p>
  <p><strong>Category:</strong> %s</p>
  <p><strong>Message:</strong></p>
  <p style="white-space: pre-wrap;">%s</p>
</div>`, htmlpkg.EscapeString(fromEmail), htmlpkg.EscapeString(category), htmlpkg.EscapeString(message))
}
