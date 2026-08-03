package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type meResponse struct {
	Email string   `json:"email"`
	Plan  string   `json:"plan"`
	Stack []string `json:"stack"`
}

func meHandler(users *mongodb.UserStore) http.HandlerFunc {
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meResponse{Email: user.Email, Plan: user.Plan, Stack: user.Stack})
	}
}
