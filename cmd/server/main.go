package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kerem451I/uptime-monitor/internal/api"
	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}
	// after godotenv.Load(), the program can access DATABASE_URL as an environment variable.

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL is not set")
	}

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

	log.Printf("server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
