package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Platon223/DevSwell/api/internal/email"
	"github.com/Platon223/DevSwell/api/internal/httpapi"
	"github.com/Platon223/DevSwell/api/internal/mongodb"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("DevSwell API starting...")

	_ = godotenv.Load("api/.env")

	ctx := context.Background()

	client, err := mongodb.Connect(ctx)
	if err != nil {
		log.Fatalf("connecting to mongodb: %v", err)
	}
	defer client.Disconnect(ctx)

	users := mongodb.NewUserStore(client)
	if err := users.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensuring indexes: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	mailer := email.NewClient(os.Getenv("RESEND_API_KEY"), os.Getenv("RESEND_FROM"))

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	router := httpapi.NewRouter(users, mailer, baseURL, []byte(jwtSecret))
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
