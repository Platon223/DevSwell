package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type digestItem struct {
	Title       string    `json:"title"`
	Source      string    `json:"source"`
	Project     string    `json:"project"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Technology  string    `json:"technology"`
}

type digestResponse struct {
	Items []digestItem `json:"items"`
}

func digestHandler(users *mongodb.UserStore, newsItems *mongodb.NewsItemReader) http.HandlerFunc {
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

		items, err := newsItems.FilterByStack(r.Context(), user.Stack)
		if err != nil {
			http.Error(w, "could not load digest", http.StatusInternalServerError)
			return
		}

		resp := digestResponse{Items: make([]digestItem, 0, len(items))}
		for _, item := range items {
			resp.Items = append(resp.Items, digestItem{
				Title:       item.Title,
				Source:      item.Source,
				Project:     item.Project,
				URL:         item.URL,
				PublishedAt: item.PublishedAt,
				Type:        item.Type,
				Severity:    item.Severity,
				Technology:  item.Technology,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
