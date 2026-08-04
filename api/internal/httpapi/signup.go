package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupResponse struct {
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

func signupHandler(users *mongodb.UserStore, mailer *email.Client, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req signupRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "could not process password", http.StatusInternalServerError)
			return
		}

		token, err := generateToken()
		if err != nil {
			http.Error(w, "could not create account", http.StatusInternalServerError)
			return
		}

		unsubscribeToken, err := generateToken()
		if err != nil {
			http.Error(w, "could not create account", http.StatusInternalServerError)
			return
		}

		user := domain.User{
			Email:                     req.Email,
			PasswordHash:              string(hash),
			Stack:                     []string{},
			Plan:                      "free",
			Verified:                  false,
			VerificationToken:         token,
			EmailNotificationsEnabled: true,
			UnsubscribeToken:          unsubscribeToken,
		}

		if err := users.Insert(r.Context(), user); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				http.Error(w, "an account with that email already exists", http.StatusConflict)
				return
			}
			http.Error(w, "could not create account", http.StatusInternalServerError)
			return
		}

		verifyLink := fmt.Sprintf("%s/verify?token=%s", baseURL, token)
		if err := mailer.Send(r.Context(), user.Email, "Confirm your DevSwell account", confirmEmailHTML(baseURL, verifyLink)); err != nil {
			log.Printf("sending verification email to %s: %v", user.Email, err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(signupResponse{Email: user.Email, Plan: user.Plan})
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func confirmEmailHTML(baseURL, verifyLink string) string {
	body := `<p style="font-size:14px;color:#3a3a3a;line-height:1.6;margin:0 0 20px;">Click the button below to confirm your email address and activate your account.</p>` +
		email.Button(verifyLink, "Confirm Email", "#0b7bb8")
	return email.BrandedHTML(baseURL+"/static/logo-mark.png", "Confirm your DevSwell account", body)
}
