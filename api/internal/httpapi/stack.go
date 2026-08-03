package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type updateStackRequest struct {
	Stack []string `json:"stack"`
}

type updateStackResponse struct {
	Stack []string `json:"stack"`
}

func updateStackHandler(users *mongodb.UserStore) http.HandlerFunc {
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

		var req updateStackRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Stack == nil {
			req.Stack = []string{}
		}

		if err := users.UpdateStack(r.Context(), userID, req.Stack); err != nil {
			http.Error(w, "could not update stack", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updateStackResponse{Stack: req.Stack})
	}
}
