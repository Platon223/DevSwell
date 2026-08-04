package httpapi

import (
	"net/http"
	"time"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/hackernews"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"golang.org/x/time/rate"
)

func NewRouter(users *mongodb.UserStore, newsItems *mongodb.NewsItemReader, generalFeed *mongodb.GeneralFeedStore, mailer *email.Client, baseURL string, supportEmail string, jwtSecret []byte) http.Handler {
	mux := http.NewServeMux()

	signupLimiter := newIPRateLimiter(rate.Every(20*time.Minute), 3) // ~3/hour per IP
	loginLimiter := newIPRateLimiter(rate.Every(12*time.Second), 5)  // ~5/minute per IP
	deleteLimiter := newIPRateLimiter(rate.Every(20*time.Minute), 3) // ~3/hour per IP
	supportLimiter := newIPRateLimiter(rate.Every(2*time.Minute), 5) // ~5 per 10min per IP

	mux.HandleFunc("POST /signup", rateLimited(signupHandler(users, mailer, baseURL), signupLimiter))
	mux.HandleFunc("GET /verify", verifyHandler(users))
	mux.HandleFunc("POST /login", rateLimited(loginHandler(users, jwtSecret), loginLimiter))
	mux.HandleFunc("GET /me", requireAuth(meHandler(users), jwtSecret))
	mux.HandleFunc("PUT /me/stack", requireAuth(updateStackHandler(users), jwtSecret))
	mux.HandleFunc("PUT /me/preferences", requireAuth(updatePreferencesHandler(users), jwtSecret))
	mux.HandleFunc("GET /digest", requireAuth(digestHandler(users, newsItems), jwtSecret))
	mux.HandleFunc("POST /me/delete", rateLimited(requireAuth(requestDeletionHandler(users, mailer, baseURL), jwtSecret), deleteLimiter))
	mux.HandleFunc("GET /me/delete/confirm", confirmDeletionHandler(users))
	mux.HandleFunc("GET /unsubscribe", unsubscribeHandler(users))
	mux.HandleFunc("GET /feed", feedHandler(generalFeed, hackernews.FetchTopStories))
	mux.HandleFunc("POST /support", rateLimited(requireAuth(supportHandler(users, mailer, supportEmail), jwtSecret), supportLimiter))
	mux.HandleFunc("POST /logout", logoutHandler)

	mux.HandleFunc("GET /signup", signupPageHandler)
	mux.HandleFunc("GET /login", loginPageHandler)
	mux.HandleFunc("GET /privacy", privacyPageHandler)
	mux.HandleFunc("GET /dashboard", requirePageAuth(dashboardPageHandler("home"), jwtSecret))
	mux.HandleFunc("GET /dashboard/stack", requirePageAuth(dashboardPageHandler("stack"), jwtSecret))
	mux.HandleFunc("GET /dashboard/feed", requirePageAuth(dashboardPageHandler("feed"), jwtSecret))
	mux.HandleFunc("GET /dashboard/settings", requirePageAuth(dashboardPageHandler("settings"), jwtSecret))
	mux.HandleFunc("GET /dashboard/upgrade", requirePageAuth(dashboardPageHandler("upgrade"), jwtSecret))
	mux.HandleFunc("GET /dashboard/help", requirePageAuth(dashboardPageHandler("help"), jwtSecret))
	mux.HandleFunc("GET /dashboard/support", requirePageAuth(dashboardPageHandler("support"), jwtSecret))

	mux.Handle("GET /static/", staticFileServer())
	mux.HandleFunc("GET /", landingPageHandler)
	return mux
}
