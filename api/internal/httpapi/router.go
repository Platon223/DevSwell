package httpapi

import (
	"net/http"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
)

func NewRouter(users *mongodb.UserStore, mailer *email.Client, baseURL string, jwtSecret []byte) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup", signupHandler(users, mailer, baseURL))
	mux.HandleFunc("GET /verify", verifyHandler(users))
	mux.HandleFunc("POST /login", loginHandler(users, jwtSecret))
	return mux
}
