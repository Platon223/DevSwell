package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type authClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func loginHandler(users *mongodb.UserStore, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
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

		user, err := users.FindByEmail(r.Context(), req.Email)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "invalid email or password", http.StatusUnauthorized)
				return
			}
			http.Error(w, "could not process login", http.StatusInternalServerError)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		if !user.Verified {
			http.Error(w, "please verify your email before logging in", http.StatusForbidden)
			return
		}

		now := time.Now()
		claims := authClaims{
			Email: user.Email,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   user.ID.Hex(),
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			},
		}

		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
		if err != nil {
			http.Error(w, "could not create token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{Token: signed})
	}
}
