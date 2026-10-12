package main

import (
	"context"
	"log"

	"carenav_backend_go/internal/config"
	"carenav_backend_go/internal/db"
	"carenav_backend_go/internal/handlers"
	"carenav_backend_go/internal/mail"
	"carenav_backend_go/internal/router"
	"carenav_backend_go/internal/streamchat"
)

// @title						CareNav API
// @version					1.0
// @description				Backend API for the CareNav application.
// @BasePath					/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and the JWT.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	client, err := db.Connect(cfg.MongoURL)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer client.Disconnect(context.Background())
	log.Println("DB connected")

	database := client.Database(cfg.MongoDBName)

	stream, err := streamchat.New(cfg.StreamAPIKey, cfg.StreamAPISecret)
	if err != nil {
		log.Fatalf("stream client error: %v", err)
	}

	mailer := mail.NewMailer(cfg.EmailUser, cfg.EmailPass)

	authHandler := handlers.NewAuthHandler(database, stream, mailer, cfg)
	userHandler := handlers.NewUserHandler(database)

	r := router.New(authHandler, userHandler)

	log.Printf("Server is listening on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
