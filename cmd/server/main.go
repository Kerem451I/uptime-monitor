package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kerem451I/uptime-monitor/internal/api"
	"github.com/Kerem451I/uptime-monitor/internal/checker"
	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	// after godotenv.Load(), the program can access DATABASE_URL as an environment variable.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	required := map[string]string{
		"DB_USER":     dbUser,
		"DB_PASSWORD": dbPassword,
		"DB_NAME":     dbName,
		"DB_HOST":     dbHost,
	}

	for key, val := range required {
		if val == "" {
			log.Fatalf("%s must be set", key)
		}
	}

	if dbPort == "" {
		dbPort = "5432" // default
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	pool, err := db.New(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("connected to database successfully")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := api.NewHandler(pool)
	router := api.NewRouter(handler)

	server := http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	chkr := checker.New(pool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go chkr.Run(ctx)

	log.Printf("server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
