package httpapi

import (
	"net/http"
	"time"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/hackernews"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"golang.org/x/time/rate"
)

func NewRouter(users *mongodb.UserStore, newsItems *mongodb.NewsItemReader, generalFeed *mongodb.GeneralFeedStore, mailer *email.Client, baseURL string, jwtSecret []byte) http.Handler {
	mux := http.NewServeMux()

	signupLimiter := newIPRateLimiter(rate.Every(20*time.Minute), 3) // ~3/hour per IP
	loginLimiter := newIPRateLimiter(rate.Every(12*time.Second), 5)  // ~5/minute per IP
	deleteLimiter := newIPRateLimiter(rate.Every(20*time.Minute), 3) // ~3/hour per IP

	mux.HandleFunc("POST /signup", rateLimited(signupHandler(users, mailer, baseURL), signupLimiter))
	mux.HandleFunc("GET /verify", verifyHandler(users))
	mux.HandleFunc("POST /login", rateLimited(loginHandler(users, jwtSecret), loginLimiter))
	mux.HandleFunc("GET /me", requireAuth(meHandler(users), jwtSecret))
	mux.HandleFunc("PUT /me/stack", requireAuth(updateStackHandler(users), jwtSecret))
	mux.HandleFunc("GET /digest", requireAuth(digestHandler(users, newsItems), jwtSecret))
	mux.HandleFunc("POST /me/delete", rateLimited(requireAuth(requestDeletionHandler(users, mailer, baseURL), jwtSecret), deleteLimiter))
	mux.HandleFunc("GET /me/delete/confirm", confirmDeletionHandler(users))
	mux.HandleFunc("GET /feed", feedHandler(generalFeed, hackernews.FetchTopStories))
	return mux
}
